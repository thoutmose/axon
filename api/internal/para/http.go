package para

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves areas and resources.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the PARA endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	for _, k := range []Kind{Areas, Resources} {
		base := "/v1/" + k.Table
		r.Get(base, h.list(k))
		r.Post(base, h.create(k))
		r.Get(base+"/{id}", h.get(k))
		r.Patch(base+"/{id}", h.update(k))
		r.Delete(base+"/{id}", h.delete(k))
	}
}

func (h *Handlers) fail(w http.ResponseWriter, k Kind, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, fmt.Sprintf("No such %s.", k.Name))
	case errors.Is(err, ErrStale):
		httpx.Problem(w, http.StatusPreconditionFailed, "This changed since you loaded it. Reload and try again.")
	default:
		h.log.Error("para request failed", "kind", k.Table, "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func decode(r *http.Request) (Fields, error) {
	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		URL         *string `json:"url"`
		Status      *string `json:"status"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		return Fields{}, ValidationError{err.Error()}
	}
	return Fields(body), nil
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id, graph.ErrNotFound
	}
	return id, nil
}

func (h *Handlers) list(k Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		who, _ := auth.FromContext(r.Context())
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "active"
		}
		if !slices.Contains(statuses, status) {
			httpx.Problem(w, http.StatusBadRequest, "?status= must be active or archived.")
			return
		}
		items, err := k.List(r.Context(), h.db, who.UserID, status)
		if err != nil {
			h.fail(w, k, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func (h *Handlers) write(w http.ResponseWriter, r *http.Request, k Kind, code int, fn func(tx pgx.Tx) (Item, error)) {
	var it Item
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) (err error) {
		it, err = fn(tx)
		return err
	})
	if err != nil {
		h.fail(w, k, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(it.UpdatedAt))
	httpx.JSON(w, code, it)
}

func (h *Handlers) create(k Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		who, _ := auth.FromContext(r.Context())
		f, err := decode(r)
		if err != nil {
			h.fail(w, k, err)
			return
		}
		h.write(w, r, k, http.StatusCreated, func(tx pgx.Tx) (Item, error) {
			return k.Create(r.Context(), tx, who.UserID, graph.UserActor, f)
		})
	}
}

func (h *Handlers) get(k Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		who, _ := auth.FromContext(r.Context())
		id, err := pathID(r)
		if err != nil {
			h.fail(w, k, err)
			return
		}
		it, err := k.Get(r.Context(), h.db, who.UserID, id)
		if err != nil {
			h.fail(w, k, err)
			return
		}
		w.Header().Set("ETag", httpx.ETag(it.UpdatedAt))
		httpx.JSON(w, http.StatusOK, it)
	}
}

func (h *Handlers) update(k Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		who, _ := auth.FromContext(r.Context())
		id, err := pathID(r)
		if err != nil {
			h.fail(w, k, err)
			return
		}
		version, ok := httpx.IfMatch(w, r)
		if !ok {
			return
		}
		f, err := decode(r)
		if err != nil {
			h.fail(w, k, err)
			return
		}
		h.write(w, r, k, http.StatusOK, func(tx pgx.Tx) (Item, error) {
			return k.Update(r.Context(), tx, who.UserID, id, graph.UserActor, version, f)
		})
	}
}

func (h *Handlers) delete(k Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		who, _ := auth.FromContext(r.Context())
		id, err := pathID(r)
		if err != nil {
			h.fail(w, k, err)
			return
		}
		err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
			return k.Delete(r.Context(), tx, who.UserID, id, graph.UserActor)
		})
		if err != nil {
			h.fail(w, k, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
