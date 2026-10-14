package notify

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/digest"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/settings"
)

// Handlers serves channels and the digest preview.
type Handlers struct {
	db    *pgxpool.Pool
	log   *slog.Logger
	store *Store
	base  string
	now   func() time.Time
}

// NewHandlers returns the endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer, base string) *Handlers {
	return &Handlers{db: pool, log: log, store: NewStore(pool, sealer), base: strings.TrimRight(base, "/"), now: time.Now}
}

// SessionRoutes manage channels: only from the web app, as they hold credentials.
func (h *Handlers) SessionRoutes(r chi.Router) {
	r.Get("/v1/me/channels", h.list)
	r.With(auth.NotDemo).Post("/v1/me/channels", h.create)
	r.With(auth.NotDemo).Put("/v1/me/channels/{id}", h.update)
	r.Delete("/v1/me/channels/{id}", h.remove)
	r.With(auth.NotDemo).Post("/v1/me/channels/{id}/link", h.relink)
	r.With(auth.NotDemo).Post("/v1/me/channels/{id}/test", h.test)
}

// Routes read the digest, with a session or a token.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/me/digest", h.preview)
	r.Get("/v1/me/digest/picture", h.picture)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var inv InvalidError
	switch {
	case errors.As(err, &inv):
		httpx.Problem(w, http.StatusUnprocessableEntity, inv.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such channel.")
	case errors.Is(err, ErrNoKey):
		httpx.Problem(w, http.StatusServiceUnavailable, "This channel was saved with an encryption key the server no longer has.")
	default:
		h.log.Error("channels request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	list, err := h.store.List(r.Context(), who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// saved answers with the channel and, for a bot to link, the code to send it.
func (h *Handlers) saved(w http.ResponseWriter, status int, c Channel, code string) {
	out := map[string]any{"channel": c}
	if code != "" {
		out["link_code"] = code
	}
	httpx.JSON(w, status, out)
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var in Input
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var n int
	if err := h.db.QueryRow(r.Context(), `SELECT count(*) FROM notify_channels WHERE user_id = $1`, who.UserID).Scan(&n); err != nil {
		h.fail(w, err)
		return
	}
	if n >= 20 {
		httpx.Problem(w, http.StatusUnprocessableEntity, "You have 20 channels already: remove one first.")
		return
	}
	c, code, err := h.store.Save(r.Context(), who.UserID, uuid.Nil, in, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	h.saved(w, http.StatusCreated, c, code)
}

func channelID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such channel.")
		return id, false
	}
	return id, true
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := channelID(w, r)
	if !ok {
		return
	}
	var in Input
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	c, code, err := h.store.Save(r.Context(), who.UserID, id, in, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	h.saved(w, http.StatusOK, c, code)
}

func (h *Handlers) remove(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := channelID(w, r)
	if !ok {
		return
	}
	if err := h.store.Remove(r.Context(), who.UserID, id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) relink(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := channelID(w, r)
	if !ok {
		return
	}
	c, err := h.store.Get(r.Context(), who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if c.Mode != "bot" {
		httpx.Problem(w, http.StatusUnprocessableEntity, "Only a bot is linked to a chat; a webhook already knows where to post.")
		return
	}
	code, err := h.store.Relink(r.Context(), who.UserID, id, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"link_code": code, "expires_at": h.now().Add(LinkTTL)})
}

// test sends today's digest to one channel, and says what went wrong.
func (h *Handlers) test(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := channelID(w, r)
	if !ok {
		return
	}
	doc, err := settings.Get(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	targets, err := h.store.Targets(r.Context(), who.UserID, digest.Daily)
	if err != nil {
		h.fail(w, err)
		return
	}
	var target *Target
	for i := range targets {
		if targets[i].ID == id {
			target = &targets[i]
		}
	}
	if target == nil {
		if _, err := h.store.Get(r.Context(), who.UserID, id); err != nil {
			h.fail(w, err)
			return
		}
		httpx.Problem(w, http.StatusUnprocessableEntity, "This channel is off, takes no daily digest, or its bot is not linked to your chat yet.")
		return
	}
	d, err := digest.Build(r.Context(), h.db, who.UserID, digest.Daily, doc.Language, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	err = SendDigest(r.Context(), *target, d, h.base, digest.Scene(d))
	h.store.Sent(r.Context(), id, err, h.now())
	if err != nil {
		h.log.Info("channel test failed", "channel", id, "err", err)
		httpx.Problem(w, http.StatusBadGateway, "The channel did not take it: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) build(w http.ResponseWriter, r *http.Request) (digest.Digest, bool) {
	who, _ := auth.FromContext(r.Context())
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = digest.Daily
	}
	if !digest.Valid(kind) {
		httpx.Problem(w, http.StatusBadRequest, "kind is daily, weekly, or monthly.")
		return digest.Digest{}, false
	}
	doc, err := settings.Get(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return digest.Digest{}, false
	}
	d, err := digest.Build(r.Context(), h.db, who.UserID, kind, doc.Language, h.now())
	if err != nil {
		h.fail(w, err)
		return d, false
	}
	return d, true
}

// preview is the digest as it would go out now.
func (h *Handlers) preview(w http.ResponseWriter, r *http.Request) {
	if d, ok := h.build(w, r); ok {
		httpx.JSON(w, http.StatusOK, d)
	}
}

// picture is the digest's illustration.
func (h *Handlers) picture(w http.ResponseWriter, r *http.Request) {
	d, ok := h.build(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(digest.Scene(d)) //nolint:gosec // a PNG Axon drew, served as image/png
}
