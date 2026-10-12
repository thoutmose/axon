package media

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/privacy"
)

// lookupsPerMinute bounds catalogue lookups per person, cached or not.
const lookupsPerMinute = 30

// Handlers serves the media endpoints.
type Handlers struct {
	db       *pgxpool.Pool
	log      *slog.Logger
	catalogs *Catalogs
	now      func() time.Time

	mu    sync.Mutex
	quota map[uuid.UUID][]time.Time
}

// NewHandlers returns the media endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger, catalogs *Catalogs) *Handlers {
	return &Handlers{db: pool, log: log, catalogs: catalogs, now: time.Now, quota: map[uuid.UUID][]time.Time{}}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/media", h.list)
	r.Post("/v1/media", h.create)
	r.Get("/v1/media/catalogs", h.catalogInfo)
	r.Get("/v1/media/search", h.search)
	r.Get("/v1/media/recommendations", h.recommendations)
	r.Get("/v1/media/covers/{id}", h.cover)
	r.Post("/v1/media/covers/find", h.findCovers)
	r.Get("/v1/media/{id}", h.get)
	r.Patch("/v1/media/{id}", h.update)
	r.Delete("/v1/media/{id}", h.delete)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		httpx.Problem(w, http.StatusNotFound, "No such item.")
	case errors.Is(err, errNoConsent):
		httpx.Problem(w, http.StatusForbidden, "Catalogue lookups are off. Allow them to search Open Library and TMDB, or add items by hand.")
	case errors.Is(err, ErrUnavailable):
		h.log.Warn("media catalogue unavailable", "err", err)
		httpx.Problem(w, http.StatusServiceUnavailable, "The catalogue cannot be reached right now. You can still add it by hand.")
	default:
		h.log.Error("media request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func id(r *http.Request) (uuid.UUID, error) {
	v, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return v, graph.ErrNotFound
	}
	return v, nil
}

// Consent is the purpose that allows catalogue lookups: they send the search
// text to Open Library or TMDB.
const Consent = "external_metadata"

var errNoConsent = errors.New("no consent for catalogue lookups")

func (h *Handlers) consented(r *http.Request, user uuid.UUID) error {
	ok, err := privacy.Granted(r.Context(), h.db, user, Consent)
	if err == nil && !ok {
		err = errNoConsent
	}
	return err
}

// allow spends one lookup from the person's per-minute quota.
func (h *Handlers) allow(user uuid.UUID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	cutoff := h.now().Add(-time.Minute)
	recent := slices.DeleteFunc(h.quota[user], func(t time.Time) bool { return t.Before(cutoff) })
	if len(recent) >= lookupsPerMinute {
		h.quota[user] = recent
		return false
	}
	h.quota[user] = append(recent, h.now())
	return true
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f := Filter{Kind: r.URL.Query().Get("kind"), Status: r.URL.Query().Get("status")}
	if (f.Kind != "" && !slices.Contains(Kinds, f.Kind)) || (f.Status != "" && !slices.Contains(Statuses, f.Status)) {
		httpx.Problem(w, http.StatusBadRequest, "Unknown kind or status.")
		return
	}
	items, err := List(r.Context(), h.db, who.UserID, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) catalogInfo(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	err := h.consented(r, who.UserID)
	if err != nil && !errors.Is(err, errNoConsent) {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"books": true, "films_and_series": h.catalogs.TMDBEnabled(), "consented": err == nil})
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if err := h.consented(r, who.UserID); err != nil {
		h.fail(w, err)
		return
	}
	if !h.allow(who.UserID) {
		httpx.Problem(w, http.StatusTooManyRequests, "That is a lot of searches in a minute. Wait a moment and try again.")
		return
	}
	found, err := h.catalogs.Search(r.Context(), r.URL.Query().Get("kind"), r.URL.Query().Get("q"))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": found})
}

// findCovers gives covers to items that have none (added by hand, imported,
// or from the demo), with the person's consent to catalogue lookups.
func (h *Handlers) findCovers(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if err := h.consented(r, who.UserID); err != nil {
		h.fail(w, err)
		return
	}
	checked, found, err := h.catalogs.FindCovers(r.Context(), who.UserID, func() bool { return h.allow(who.UserID) })
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"checked": checked, "found": found, "films_and_series": h.catalogs.TMDBEnabled()})
}

func (h *Handlers) recommendations(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	kind := r.URL.Query().Get("kind")
	if kind != "" && !slices.Contains(Kinds, kind) {
		httpx.Problem(w, http.StatusBadRequest, "Unknown kind.")
		return
	}
	recs, err := Recommend(r.Context(), h.db, who.UserID, kind, h.now(), 10)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": recs})
}

func (h *Handlers) cover(w http.ResponseWriter, r *http.Request) {
	cid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	data, ctype, err := CoverImage(r.Context(), h.db, cid)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	iid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	it, err := Get(r.Context(), h.db, who.UserID, iid)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, it)
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Fields
		Catalog *struct {
			Source   string `json:"source"`
			SourceID string `json:"source_id"`
		} `json:"catalog"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	f := body.Fields
	var cat *Catalog
	if body.Catalog != nil {
		if f.Kind == nil {
			httpx.Problem(w, http.StatusUnprocessableEntity, "Say whether it is a film, a series, or a book.")
			return
		}
		if err := h.consented(r, who.UserID); err != nil {
			h.fail(w, err)
			return
		}
		if !h.allow(who.UserID) {
			httpx.Problem(w, http.StatusTooManyRequests, "That is a lot of lookups in a minute. Wait a moment and try again.")
			return
		}
		cand, err := h.catalogs.Details(r.Context(), body.Catalog.Source, *f.Kind, body.Catalog.SourceID)
		if err != nil {
			h.fail(w, err)
			return
		}
		// What the person typed wins over the catalogue.
		f.Title = or(f.Title, &cand.Title)
		if f.Year == nil && cand.Year != nil {
			f.Year = cand.Year
		}
		f.Creators = or(f.Creators, &cand.Creators)
		if f.Genres == nil {
			f.Genres = &cand.Genres
		}
		if f.Total == nil && cand.Total != nil {
			f.Total = cand.Total
		}
		cat = &Catalog{Source: cand.Source, SourceID: cand.SourceID, Overview: cand.Overview, CoverID: h.catalogs.Cover(r.Context(), cand)}
	}
	var it Item
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		it, err = Create(r.Context(), tx, who.UserID, graph.UserActor, f, cat, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, it)
}

func or(a, b *string) *string {
	if a != nil {
		return a
	}
	return b
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	iid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var f Fields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var it Item
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		it, err = Update(r.Context(), tx, who.UserID, iid, graph.UserActor, f, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, it)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	iid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		if _, err := Get(r.Context(), tx, who.UserID, iid); err != nil {
			return err
		}
		return graph.DeleteEntity(r.Context(), tx, who.UserID, iid, graph.UserActor)
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
