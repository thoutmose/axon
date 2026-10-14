package practice

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the practice log.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the practice handlers.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/practice", h.list)
	r.Post("/v1/practice", h.logAttempt)
	r.Get("/v1/practice/stats", h.stats)
	r.Post("/v1/practice/{id}/attempts", h.attempt)
	r.Delete("/v1/practice/{id}", h.delete)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such problem.")
	default:
		h.log.Error("practice request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func (h *Handlers) who(r *http.Request) (uuid.UUID, time.Time, error) {
	who, _ := auth.FromContext(r.Context())
	today, _, err := clock.Today(r.Context(), h.db, who.UserID, h.now())
	return who.UserID, today, err
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	user, today, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := List(r.Context(), h.db, user, today, r.URL.Query().Get("due") == "true")
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "today": today.Format(time.DateOnly)})
}

func (h *Handlers) stats(w http.ResponseWriter, r *http.Request) {
	user, today, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := List(r.Context(), h.db, user, today, false)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, Summarise(items, today))
}

func (h *Handlers) logAttempt(w http.ResponseWriter, r *http.Request) {
	user, today, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var in Input
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var p Problem
	var created bool
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) (err error) {
		p, created, err = Log(r.Context(), tx, user, in, today)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	status := http.StatusOK // another attempt at a problem already logged
	if created {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, p)
}

func (h *Handlers) attempt(w http.ResponseWriter, r *http.Request) {
	user, today, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, ErrNotFound)
		return
	}
	var in AttemptInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var p Problem
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) (err error) {
		p, err = AddAttempt(r.Context(), tx, user, id, in, today)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	user, _, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err == nil {
		err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return Delete(r.Context(), tx, user, id) })
	} else {
		err = ErrNotFound
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
