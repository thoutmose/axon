package habits

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves habits and check-ins.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the habit endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/habits", h.list)
	r.Post("/v1/habits", h.create)
	r.Get("/v1/habits/{id}", h.get)
	r.Patch("/v1/habits/{id}", h.update)
	r.Delete("/v1/habits/{id}", h.delete)
	r.Get("/v1/habits/{id}/checkins", h.history)
	r.Put("/v1/habits/{id}/checkins/{day}", h.checkIn)
	r.Delete("/v1/habits/{id}/checkins/{day}", h.uncheck)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such habit.")
	case errors.Is(err, ErrStale):
		httpx.Problem(w, http.StatusPreconditionFailed, "This habit changed since you loaded it. Reload and try again.")
	default:
		h.log.Error("habits request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// DecodeFields reads a JSON object into Fields; unknown fields are an error.
func DecodeFields(b []byte) (Fields, error) {
	var raw struct {
		Title      *string `json:"title"`
		Cue        *string `json:"cue"`
		Craving    *string `json:"craving"`
		Reward     *string `json:"reward"`
		StackAfter *string `json:"stack_after"`
		TwoMinute  *string `json:"two_minute"`
		Status     *string `json:"status"`
		Days       []int   `json:"days"`
	}
	if err := httpx.DecodeStrict(b, &raw); err != nil {
		return Fields{}, ValidationError{"The request body is not valid JSON for this endpoint."}
	}
	return Fields{raw.Title, raw.Cue, raw.Craving, raw.Reward, raw.StackAfter, raw.TwoMinute, raw.Status, raw.Days}, nil
}

func (h *Handlers) fields(r *http.Request) (Fields, error) {
	var raw json.RawMessage
	if err := httpx.Decode(r, &raw); err != nil {
		return Fields{}, ValidationError{err.Error()}
	}
	return DecodeFields(raw)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id, graph.ErrNotFound
	}
	return id, nil
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}
	if !slices.Contains(statuses, status) {
		httpx.Problem(w, http.StatusBadRequest, "?status= must be active, paused, or archived.")
		return
	}
	list, err := List(r.Context(), h.db, who.UserID, status, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := h.fields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var out Habit
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		out, err = Create(r.Context(), tx, who.UserID, graph.UserActor, f, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(out.UpdatedAt))
	httpx.JSON(w, http.StatusCreated, out)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	out, err := Get(r.Context(), h.db, who.UserID, id, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(out.UpdatedAt))
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	version, ok := httpx.IfMatch(w, r)
	if !ok {
		return
	}
	f, err := h.fields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var out Habit
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		out, err = Update(r.Context(), tx, who.UserID, id, graph.UserActor, version, f, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(out.UpdatedAt))
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return Delete(r.Context(), tx, who.UserID, id, graph.UserActor, h.now())
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pathDay(r *http.Request) (time.Time, error) {
	d, err := time.Parse(time.DateOnly, chi.URLParam(r, "day"))
	if err != nil {
		return d, ValidationError{"The day looks like 2026-10-12."}
	}
	return d, nil
}

func (h *Handlers) checkIn(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	day, err := pathDay(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	c, err := CheckIn(r.Context(), h.db, who.UserID, id, day, body.Status, body.Note, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) uncheck(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	day, err := pathDay(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := Uncheck(r.Context(), h.db, who.UserID, id, day, h.now()); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) history(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	today, _, err := Today(r.Context(), h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	from, to := today.AddDate(0, 0, -27), today
	q := r.URL.Query()
	var err1, err2 error
	if v := q.Get("from"); v != "" {
		from, err1 = time.Parse(time.DateOnly, v)
	}
	if v := q.Get("to"); v != "" {
		to, err2 = time.Parse(time.DateOnly, v)
	}
	if err1 != nil || err2 != nil || to.Before(from) || to.Sub(from) > 366*24*time.Hour {
		httpx.Problem(w, http.StatusBadRequest, fmt.Sprintf("Send ?from= and ?to= as dates at most a year apart (default: the last 28 days to %s).", today.Format(time.DateOnly)))
		return
	}
	items, err := History(r.Context(), h.db, who.UserID, id, from, to, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}
