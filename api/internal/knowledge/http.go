package knowledge

import (
	"errors"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the knowledge explorer.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the explorer endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/notes/explore", h.explore)
	r.Get("/v1/notes/graph", h.graph)
	r.Get("/v1/notes/analysis", h.analysis)
	r.Get("/v1/notes/{id}/analysis", h.noteAnalysis)
}

func (h *Handlers) analysis(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	kinds := AnalysedKinds
	switch k := r.URL.Query().Get("kind"); k {
	case "":
	case "literature", "permanent":
		kinds = []string{k}
	default:
		httpx.Problem(w, http.StatusBadRequest, "kind is literature or permanent.")
		return
	}
	a, err := Analyse(r.Context(), h.db, who.UserID, kinds)
	if err != nil {
		h.log.Error("note analysis failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}

func (h *Handlers) noteAnalysis(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such note.")
		return
	}
	a, err := AnalyseNote(r.Context(), h.db, who.UserID, id)
	switch {
	case errors.Is(err, errNoNote):
		httpx.Problem(w, http.StatusNotFound, "No such note.")
	case err != nil:
		h.log.Error("note analysis failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	default:
		httpx.JSON(w, http.StatusOK, a)
	}
}

var paras = []string{"inbox", "projects", "areas", "resources", "archive", "other"}

func filter(w http.ResponseWriter, r *http.Request) (Filter, bool) {
	q := r.URL.Query()
	f := Filter{Para: q.Get("para"), Kind: q.Get("kind"), Tag: q.Get("tag"), Query: q.Get("q"), Orphans: q.Get("orphans") == "true"}
	if f.Para != "" && !slices.Contains(paras, f.Para) {
		httpx.Problem(w, http.StatusBadRequest, "?para= is inbox, projects, areas, resources, archive, or other.")
		return f, false
	}
	if len(f.Query) > 200 || len(f.Kind) > 50 || len(f.Tag) > 100 {
		httpx.Problem(w, http.StatusBadRequest, "A filter is too long.")
		return f, false
	}
	for key, dst := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
		if v := q.Get(key); v != "" {
			t, err := time.Parse(time.DateOnly, v)
			if err != nil {
				httpx.Problem(w, http.StatusBadRequest, "?"+key+"= is a date like 2026-10-11.")
				return f, false
			}
			*dst = &t
		}
	}
	return f, true
}

func (h *Handlers) explore(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, ok := filter(w, r)
	if !ok {
		return
	}
	ex, err := Explore(r.Context(), h.db, who.UserID, f)
	if err != nil {
		h.log.Error("explore failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	httpx.JSON(w, http.StatusOK, ex)
}

func (h *Handlers) graph(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, ok := filter(w, r)
	if !ok {
		return
	}
	g, err := BuildGraph(r.Context(), h.db, who.UserID, f)
	if err != nil {
		h.log.Error("knowledge graph failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	httpx.JSON(w, http.StatusOK, g)
}
