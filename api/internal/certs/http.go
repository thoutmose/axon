package certs

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves certifications.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the certification handlers.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/certifications", h.list)
	r.Post("/v1/certifications", h.create)
	r.Put("/v1/certifications/{id}", h.replace)
	r.Delete("/v1/certifications/{id}", h.delete)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such certification.")
	default:
		h.log.Error("certification request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := List(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var f Fields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	c, err := Create(r.Context(), h.db, who.UserID, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handlers) replace(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, ErrNotFound)
		return
	}
	var f Fields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	c, err := Replace(r.Context(), h.db, who.UserID, id, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err == nil {
		err = Delete(r.Context(), h.db, who.UserID, id)
	} else {
		err = ErrNotFound
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
