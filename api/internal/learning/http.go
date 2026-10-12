package learning

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the learning engine.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the learning endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/learning/plan", h.plan)
	r.Post("/v1/learning/tests", h.recordTest)
	r.Get("/v1/palaces", h.palaces)
	r.Post("/v1/palaces", h.createPalace)
	r.Get("/v1/palaces/{id}", h.palace)
	r.Patch("/v1/palaces/{id}", h.updatePalace)
	r.Delete("/v1/palaces/{id}", h.deletePalace)
	r.Post("/v1/notes/{id}/recall", h.recall)
	r.Get("/v1/notes/{id}/recall", h.history)
	r.Get("/v1/notes/{id}/card-drafts", h.drafts)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "Not found.")
	default:
		h.log.Error("learning request failed", "err", err)
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

func (h *Handlers) tx(ctx context.Context, f func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, h.db, f)
}

func (h *Handlers) plan(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	p, err := Today(r.Context(), h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handlers) recordTest(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		Deck      string `json:"deck"`
		Questions int    `json:"questions"`
		Correct   int    `json:"correct"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := RecordTest(r.Context(), h.db, who.UserID, b.Deck, b.Questions, b.Correct, h.now()); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) palaces(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := Palaces(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) palace(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	pid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	p, err := GetPalace(r.Context(), h.db, who.UserID, pid)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handlers) createPalace(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var f PalaceFields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var p Palace
	err := h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		p, err = CreatePalace(r.Context(), tx, who.UserID, graph.UserActor, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handlers) updatePalace(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	pid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var f PalaceFields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var p Palace
	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		p, err = UpdatePalace(r.Context(), tx, who.UserID, pid, graph.UserActor, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handlers) deletePalace(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	pid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		if _, err := GetPalace(r.Context(), tx, who.UserID, pid); err != nil {
			return err
		}
		return graph.DeleteEntity(r.Context(), tx, who.UserID, pid, graph.UserActor)
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) recall(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	nid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var b struct {
		Text string `json:"text"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var res RecallResult
	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		res, err = Recall(r.Context(), tx, who.UserID, nid, b.Text, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handlers) history(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	nid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := History(r.Context(), h.db, who.UserID, nid)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) drafts(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	nid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := Drafts(r.Context(), h.db, who.UserID, nid)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}
