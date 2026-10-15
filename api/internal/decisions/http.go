package decisions

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the decision journal.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the decision journal handlers.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/decisions", h.list)
	r.Post("/v1/decisions", h.create)
	r.Get("/v1/decisions/calibration", h.calibration)
	r.Patch("/v1/decisions/{id}", h.update)
	r.Post("/v1/decisions/{id}/review", h.review)
	r.Delete("/v1/decisions/{id}", h.delete)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such decision.")
	default:
		h.log.Error("decision request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func (h *Handlers) who(r *http.Request) (uuid.UUID, time.Time, error) {
	who, _ := auth.FromContext(r.Context())
	today, _, err := clock.Today(r.Context(), h.db, who.UserID, h.now())
	return who.UserID, today, err
}

// id reads the decision in the path; a malformed one is not found.
func id(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id, ErrNotFound
	}
	return id, nil
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	user, today, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	status := r.URL.Query().Get("status")
	if !slices.Contains(Statuses, status) {
		httpx.Problem(w, http.StatusUnprocessableEntity, "The status is open, due, or reviewed.")
		return
	}
	items, err := List(r.Context(), h.db, user, today, status)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "today": today.Format(time.DateOnly)})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
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
	d, err := Create(r.Context(), h.db, user, in, today)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, d)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	user, today, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	id, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var p Patch
	if err := httpx.Decode(r, &p); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	d, err := Update(r.Context(), h.db, user, id, p, today)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (h *Handlers) review(w http.ResponseWriter, r *http.Request) {
	user, today, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	id, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var in Review
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	d, err := Resolve(r.Context(), h.db, user, id, in, today)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	user, _, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	id, err := id(r)
	if err == nil {
		err = Delete(r.Context(), h.db, user, id)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) calibration(w http.ResponseWriter, r *http.Request) {
	user, _, err := h.who(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	c, err := GetCalibration(r.Context(), h.db, user)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}
