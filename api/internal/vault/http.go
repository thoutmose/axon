package vault

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the vault server endpoints.
type Handlers struct {
	db      *pgxpool.Pool
	log     *slog.Logger
	base    string // AXON_BASE_URL, written into the setup commands
	version string // the image tag the docker command runs
	now     func() time.Time
}

// NewHandlers returns the endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger, base, version string) *Handlers {
	return &Handlers{db: pool, log: log, base: strings.TrimRight(base, "/"), version: version, now: time.Now}
}

// SessionRoutes manage the vault server: only from the web app, because
// pairing hands out a token.
func (h *Handlers) SessionRoutes(r chi.Router) {
	r.Get("/v1/vault/source", h.get)
	r.With(auth.NotDemo).Put("/v1/vault/source", h.put)
	r.Delete("/v1/vault/source", h.remove)
}

// Routes take a session or a token: the vault server reports with its token.
func (h *Handlers) Routes(r chi.Router) {
	r.Post("/v1/vault/heartbeat", h.heartbeat)
	r.Post("/v1/vault/scaffold", h.scaffold)
}

// PublicRoutes need no sign-in: the pairing code is the credential, and the
// install script and program are the same for everyone. The caller wraps
// pairing in a rate limit and leaves out the same-origin check (servers
// send no Origin).
func (h *Handlers) PublicRoutes(r chi.Router) {
	r.Get("/v1/vault/install.sh", h.installScript)
	r.Get("/v1/vault/agent", h.agent)
}

// PairRoute is mounted on its own, rate limited.
func (h *Handlers) PairRoute(r chi.Router) { r.Post("/v1/vault/pair", h.pair) }

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No vault server is set: your notes live in Axon only.")
	case errors.Is(err, ErrBadCode):
		httpx.Problem(w, http.StatusUnauthorized, "The pairing code is wrong or has expired. Make a new one in Axon: Account → Preferences → Markdown vault.")
	case errors.Is(err, ErrInvalid):
		msg := err.Error()
		if i := strings.LastIndex(msg, "\n"); i >= 0 {
			msg = msg[i+1:]
		}
		httpx.Problem(w, http.StatusUnprocessableEntity, strings.ToUpper(msg[:1])+msg[1:]+".")
	default:
		h.log.Error("vault request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// Setup is how to start the sync on the server, ready to copy.
type Setup struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	Install   string    `json:"install"` // one line: downloads axon, pairs, installs a service
	Docker    string    `json:"docker"`  // the same with the axon-api image
}

func (h *Handlers) setup(code, dir string, expires time.Time) Setup {
	q := func(s string) string { return "'" + s + "'" } // clean() refuses quotes
	var hostDir string
	if strings.HasPrefix(dir, "~/") {
		hostDir = `"$HOME"/` + q(dir[2:])
	} else {
		hostDir = q(dir)
	}
	return Setup{
		Code:      code,
		ExpiresAt: expires,
		Install:   fmt.Sprintf("curl -fsSL %s | sh -s -- --code %s --dir %s", q(h.base+"/api/v1/vault/install.sh"), code, q(dir)),
		Docker: fmt.Sprintf("mkdir -p \"$HOME/.config/axon\" %s && docker run -d --name axon-vault --restart unless-stopped --user \"$(id -u):$(id -g)\" "+
			"-v %s:/vault -v \"$HOME/.config/axon\":/config axon-api:%s notes-sync -dir /vault -state /config/vault-sync.json "+
			"-url %s -token-file /config/vault-token -pair %s -watch", hostDir, hostDir, h.version, q(h.base), code),
	}
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	s, err := Get(r.Context(), h.db, who.UserID, h.now())
	if errors.Is(err, ErrNotFound) {
		httpx.JSON(w, http.StatusOK, map[string]any{"source": nil})
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"source": s})
}

// put names the server and folder and returns a fresh pairing code with the
// commands that use it.
func (h *Handlers) put(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		Host     string `json:"host"`
		Dir      string `json:"dir"`
		Scaffold *bool  `json:"scaffold"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	scaffold := b.Scaffold == nil || *b.Scaffold
	now := h.now()
	code, err := Put(r.Context(), h.db, who.UserID, b.Host, b.Dir, scaffold, now)
	if err != nil {
		h.fail(w, err)
		return
	}
	s, err := Get(r.Context(), h.db, who.UserID, now)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"source": s, "setup": h.setup(code, s.Dir, now.Add(PairTTL))})
}

func (h *Handlers) remove(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return Remove(r.Context(), tx, who.UserID) })
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) pair(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Code  string `json:"code"`
		Host  string `json:"host"`
		Empty bool   `json:"empty"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var out Paired
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		out, err = Pair(r.Context(), tx, b.Code, b.Host, b.Empty, h.now(), httpx.ClientIP(r), r.UserAgent())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handlers) heartbeat(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var rep Report
	if err := httpx.Decode(r, &rep); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := Heartbeat(r.Context(), h.db, who.UserID, rep, h.now()); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) scaffold(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var added []string
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		added, err = Scaffold(r.Context(), tx, who.UserID)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"added": added, "folders": Folders})
}

//go:embed install.sh
var installSh string

var installTmpl = template.Must(template.New("install").Parse(installSh))

// installScript is the one-line setup's script, with this Axon's address.
func (h *Handlers) installScript(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = installTmpl.Execute(w, map[string]string{"Base": h.base})
}

// agent serves this program for the server's system, when it is the one
// Axon runs on: the image holds a single static binary.
func (h *Handlers) agent(w http.ResponseWriter, r *http.Request) {
	goos, goarch := r.URL.Query().Get("os"), r.URL.Query().Get("arch")
	if goos != runtime.GOOS || goarch != runtime.GOARCH {
		httpx.Problem(w, http.StatusNotFound, fmt.Sprintf("This Axon runs on %s/%s and can only hand out that build. Build axon for %s/%s from the source (cd api && GOOS=%[3]s GOARCH=%[4]s go build ./cmd/axon), or use the Docker command.",
			runtime.GOOS, runtime.GOARCH, goos, goarch))
		return
	}
	exe, err := os.Executable()
	if err != nil {
		h.fail(w, err)
		return
	}
	f, err := os.Open(exe) //nolint:gosec // our own executable
	if err != nil {
		h.fail(w, err)
		return
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="axon"`)
	http.ServeContent(w, r, "axon", info.ModTime(), f)
}
