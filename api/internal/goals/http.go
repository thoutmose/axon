package goals

import (
	"encoding/json"
	"errors"
	"fmt"
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

// Handlers serves goals and milestones.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the goal endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/goals", h.list)
	r.Post("/v1/goals", h.create)
	r.Get("/v1/goals/{id}", h.get)
	r.Patch("/v1/goals/{id}", h.update)
	r.Delete("/v1/goals/{id}", h.delete)
	r.Get("/v1/goals/{id}/progress", h.progress)
	r.Post("/v1/goals/{id}/milestones", h.addMilestone)
	r.Get("/v1/milestones/{id}", h.getMilestone)
	r.Patch("/v1/milestones/{id}", h.updateMilestone)
	r.Delete("/v1/milestones/{id}", h.deleteMilestone)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such goal or milestone.")
	case errors.Is(err, ErrStale):
		httpx.Problem(w, http.StatusPreconditionFailed, "This changed since you loaded it. Reload and try again.")
	default:
		h.log.Error("goals request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// fields reads a JSON object, telling "absent" from "null".
type fields map[string]json.RawMessage

func decode(r *http.Request) (fields, error) {
	var raw fields
	if err := httpx.Decode(r, &raw); err != nil {
		return nil, ValidationError{err.Error()}
	}
	return raw, nil
}

// take decodes key into dst. It reports whether the key was null, which is
// only allowed when nullable.
func (f fields) take(key string, dst any, nullable bool) (null bool, err error) {
	v, ok := f[key]
	if !ok {
		return false, nil
	}
	delete(f, key)
	if string(v) == "null" {
		if !nullable {
			return false, ValidationError{fmt.Sprintf("%s cannot be null.", key)}
		}
		return true, nil
	}
	if err := json.Unmarshal(v, dst); err != nil {
		return false, ValidationError{fmt.Sprintf("%s has the wrong type.", key)}
	}
	return false, nil
}

func (f fields) date(key string, dst **time.Time) (null bool, err error) {
	var s *string
	if null, err = f.take(key, &s, true); err != nil || s == nil {
		return null, err
	}
	t, err := time.Parse(time.DateOnly, *s)
	if err != nil {
		return false, ValidationError{fmt.Sprintf("%s must be a date like 2026-10-10.", key)}
	}
	*dst = &t
	return false, nil
}

func (f fields) rest() error {
	for key := range f {
		return ValidationError{fmt.Sprintf("Unknown field %q.", key)}
	}
	return nil
}

func goalFields(r *http.Request) (Fields, error) {
	raw, err := decode(r)
	if err != nil {
		return Fields{}, err
	}
	var g Fields
	steps := []func() error{
		func() error { _, err := raw.take("title", &g.Title, false); return err },
		func() error { _, err := raw.take("outcome", &g.Outcome, false); return err },
		func() error { _, err := raw.take("status", &g.Status, false); return err },
		func() (err error) { g.ClearTargetDate, err = raw.date("target_date", &g.TargetDate); return err },
		func() (err error) { g.ClearMeasure, err = raw.take("measure", &g.Measure, true); return err },
		raw.rest,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return Fields{}, err
		}
	}
	return g, nil
}

func milestoneFields(r *http.Request) (MilestoneFields, error) {
	raw, err := decode(r)
	if err != nil {
		return MilestoneFields{}, err
	}
	var m MilestoneFields
	steps := []func() error{
		func() error { _, err := raw.take("title", &m.Title, false); return err },
		func() (err error) { m.ClearDue, err = raw.date("due_on", &m.DueOn); return err },
		func() error { _, err := raw.take("done", &m.Done, false); return err },
		raw.rest,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return MilestoneFields{}, err
		}
	}
	return m, nil
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
	if !contains(statuses, status) {
		httpx.Problem(w, http.StatusBadRequest, "?status= must be active, achieved, or archived.")
		return
	}
	items, err := List(r.Context(), h.db, who.UserID, status)
	if err != nil {
		h.fail(w, err)
		return
	}
	if items == nil {
		items = []Goal{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := goalFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var g Goal
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		g, err = Create(r.Context(), tx, who.UserID, graph.UserActor, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(g.UpdatedAt))
	httpx.JSON(w, http.StatusCreated, g)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	g, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(g.UpdatedAt))
	httpx.JSON(w, http.StatusOK, g)
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
	f, err := goalFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var g Goal
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		g, err = Update(r.Context(), tx, who.UserID, id, graph.UserActor, version, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(g.UpdatedAt))
	httpx.JSON(w, http.StatusOK, g)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return Delete(r.Context(), tx, who.UserID, id, graph.UserActor)
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) progress(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	g, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	ex, err := Explain(r.Context(), h.db, who.UserID, g)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ex)
}

func (h *Handlers) addMilestone(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	goalID, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	f, err := milestoneFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var m Milestone
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		m, err = AddMilestone(r.Context(), tx, who.UserID, goalID, graph.UserActor, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(m.UpdatedAt))
	httpx.JSON(w, http.StatusCreated, m)
}

func (h *Handlers) getMilestone(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	m, err := GetMilestone(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(m.UpdatedAt))
	httpx.JSON(w, http.StatusOK, m)
}

func (h *Handlers) updateMilestone(w http.ResponseWriter, r *http.Request) {
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
	f, err := milestoneFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var m Milestone
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		m, err = UpdateMilestone(r.Context(), tx, who.UserID, id, graph.UserActor, version, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(m.UpdatedAt))
	httpx.JSON(w, http.StatusOK, m)
}

func (h *Handlers) deleteMilestone(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return DeleteMilestone(r.Context(), tx, who.UserID, id, graph.UserActor)
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
