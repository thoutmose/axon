package tasks

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves tasks, contexts, and check-ins.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the task endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/tasks", h.list)
	r.Post("/v1/tasks", h.create)
	r.Get("/v1/tasks/{id}", h.get)
	r.Patch("/v1/tasks/{id}", h.update)
	r.Delete("/v1/tasks/{id}", h.delete)
	r.Get("/v1/contexts", h.contexts)
	r.Get("/v1/checkins", h.listCheckins)
	r.Put("/v1/checkins/{kind}/{day}", h.checkIn)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such task.")
	case errors.Is(err, ErrStale):
		httpx.Problem(w, http.StatusPreconditionFailed, "This task changed since you loaded it. Reload and try again.")
	default:
		h.log.Error("tasks request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// decodeFields reads a JSON object into Fields, telling "absent" from "null".
func decodeFields(r *http.Request) (Fields, error) {
	var raw map[string]json.RawMessage
	if err := httpx.Decode(r, &raw); err != nil {
		return Fields{}, ValidationError{err.Error()}
	}
	var f Fields
	str := func(key string, dst **string, clear *bool) error {
		v, ok := raw[key]
		if !ok {
			return nil
		}
		delete(raw, key)
		if string(v) == "null" {
			if clear == nil {
				return ValidationError{fmt.Sprintf("%s cannot be null.", key)}
			}
			*clear = true
			return nil
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return ValidationError{fmt.Sprintf("%s must be a string.", key)}
		}
		*dst = &s
		return nil
	}
	date := func(key string, dst **time.Time, clear *bool) error {
		var s *string
		if err := str(key, &s, clear); err != nil || s == nil {
			return err
		}
		t, err := time.Parse(time.DateOnly, *s)
		if err != nil {
			return ValidationError{fmt.Sprintf("%s must be a date like 2026-10-10.", key)}
		}
		*dst = &t
		return nil
	}
	for _, step := range []error{
		str("title", &f.Title, nil),
		str("notes", &f.Notes, nil),
		str("status", &f.Status, nil),
		str("context", &f.Context, &f.ClearContext),
		str("recurrence", &f.Recurrence, &f.ClearRecurrence),
		date("scheduled_on", &f.ScheduledOn, &f.ClearScheduled),
		date("due_on", &f.DueOn, &f.ClearDue),
	} {
		if step != nil {
			return Fields{}, step
		}
	}
	for key := range raw {
		return Fields{}, ValidationError{fmt.Sprintf("Unknown field %q.", key)}
	}
	return f, nil
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := decodeFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var t Task
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		t, err = Create(r.Context(), tx, who.UserID, graph.UserActor, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(t.UpdatedAt))
	httpx.JSON(w, http.StatusCreated, t)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, graph.ErrNotFound)
		return
	}
	t, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(t.UpdatedAt))
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, graph.ErrNotFound)
		return
	}
	version, ok := httpx.IfMatch(w, r)
	if !ok {
		return
	}
	f, err := decodeFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var t Task
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		t, err = Update(r.Context(), tx, who.UserID, id, graph.UserActor, version, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(t.UpdatedAt))
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, graph.ErrNotFound)
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

// --- Views ---

// view is one way of slicing the task list.
type view struct {
	where string // SQL predicate on t (tasks) using $today, never user input
	done  bool   // ordered by completion time, newest first
}

const open = `t.status NOT IN ('done', 'cancelled')`

var views = map[string]view{
	"all":     {where: open},
	"inbox":   {where: `t.status = 'inbox'`},
	"next":    {where: `t.status = 'next'`},
	"doing":   {where: `t.status = 'doing'`},
	"waiting": {where: `t.status = 'waiting'`},
	"someday": {where: `t.status = 'someday'`},
	"today":   {where: open + ` AND t.status <> 'someday' AND (t.scheduled_on <= $2 OR t.due_on <= $2)`},
	"overdue": {where: open + ` AND t.status <> 'someday' AND t.due_on < $2`},
	"week":    {where: open + ` AND t.status <> 'someday' AND (t.scheduled_on BETWEEN $2 AND $2 + 6 OR t.due_on BETWEEN $2 AND $2 + 6)`},
	"month":   {where: open + ` AND t.status <> 'someday' AND (t.scheduled_on BETWEEN $2 AND $2 + 30 OR t.due_on BETWEEN $2 AND $2 + 30)`},
	"done":    {where: `t.status IN ('done', 'cancelled')`, done: true},
}

func (h *Handlers) today(r *http.Request, userID uuid.UUID) (time.Time, error) {
	var tz string
	if err := h.db.QueryRow(r.Context(), `SELECT timezone FROM users WHERE id = $1`, userID).Scan(&tz); err != nil {
		return time.Time{}, err
	}
	return Today(h.now(), tz), nil
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	q := r.URL.Query()
	name := q.Get("view")
	if name == "" {
		name = "all"
	}
	v, ok := views[name]
	if !ok {
		httpx.Problem(w, http.StatusBadRequest, "?view= must be one of all, inbox, next, doing, waiting, someday, today, overdue, week, month, done.")
		return
	}
	limit, ok := httpx.Limit(r)
	if !ok {
		httpx.Problem(w, http.StatusBadRequest, "?limit= must be between 1 and 100.")
		return
	}
	var context *string
	if c := q.Get("context"); c != "" {
		context = &c
	}
	today, err := h.today(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}

	// Keyset pagination on (sort key, id); the cursor is base64url("key|id").
	sortKey := `coalesce(t.scheduled_on, t.due_on, 'infinity'::date)`
	order := sortKey + `, t.id`
	cmp := `(` + sortKey + `, t.id) > ($5::date, $6::uuid)`
	if v.done {
		sortKey = `coalesce(t.completed_at, 'epoch'::timestamptz)`
		order = sortKey + ` DESC, t.id DESC`
		cmp = `(` + sortKey + `, t.id) < ($5::timestamptz, $6::uuid)`
	}
	var afterKey, afterID any
	if c := q.Get("after"); c != "" {
		raw, _ := base64.RawURLEncoding.DecodeString(c)
		key, id, found := strings.Cut(string(raw), "|")
		uid, err := uuid.Parse(id)
		if !found || err != nil {
			httpx.Problem(w, http.StatusBadRequest, "The ?after= cursor is not valid. Use next_cursor from the previous page.")
			return
		}
		afterKey, afterID = key, uid
	}
	sql := `SELECT ` + columns + `, ` + sortKey + `::text FROM tasks t JOIN entities e ON e.id = t.id
		WHERE e.user_id = $1 AND $2::date IS NOT NULL -- every view receives today; not all use it
		AND ` + v.where + ` AND ($3::text IS NULL OR t.context = $3)
		AND ($6::uuid IS NULL OR ` + cmp + `)
		ORDER BY ` + order + ` LIMIT $4`
	rows, err := h.db.Query(r.Context(), sql, who.UserID, today, context, limit, afterKey, afterID)
	if err != nil {
		h.fail(w, err)
		return
	}
	var keys []string
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Task, error) {
		var t Task
		var scheduled, due, occurrence *time.Time
		var key string // the sort key as PostgreSQL text, which parses back losslessly
		err := row.Scan(&t.ID, &t.Title, &t.Notes, &t.Status, &t.Context, &scheduled, &due, &t.Recurrence,
			&t.SeriesID, &occurrence, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &key)
		t.ScheduledOn, t.DueOn, t.OccurrenceOn = datePtr(scheduled), datePtr(due), datePtr(occurrence)
		keys = append(keys, key)
		return t, err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	out := map[string]any{"items": items, "today": Date{today}, "next_cursor": nil}
	if items == nil {
		out["items"] = []Task{}
	}
	if len(items) == limit {
		out["next_cursor"] = base64.RawURLEncoding.EncodeToString([]byte(keys[len(keys)-1] + "|" + items[len(items)-1].ID.String()))
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handlers) contexts(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rows, err := h.db.Query(r.Context(), `SELECT t.context, count(*) FROM tasks t JOIN entities e ON e.id = t.id
		WHERE e.user_id = $1 AND t.context IS NOT NULL AND `+open+` GROUP BY t.context ORDER BY t.context LIMIT 100`, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	type ctxCount struct {
		Context string `json:"context"`
		Open    int    `json:"open"`
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[ctxCount])
	if err != nil {
		h.fail(w, err)
		return
	}
	if list == nil {
		list = []ctxCount{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// --- Check-ins ---

type checkin struct {
	Kind      string    `json:"kind"`
	Day       Date      `json:"day"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// period normalizes a check-in day: weekly check-ins cover the ISO week
// (Monday), monthly ones the first of the month.
func period(kind string, d time.Time) (time.Time, bool) {
	switch kind {
	case "daily":
		return d, true
	case "weekly":
		return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7)), true
	case "monthly":
		return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC), true
	}
	return time.Time{}, false
}

func (h *Handlers) checkIn(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	d, err := time.Parse(time.DateOnly, chi.URLParam(r, "day"))
	if err != nil {
		httpx.Problem(w, http.StatusBadRequest, "The day must look like 2026-10-10.")
		return
	}
	day, ok := period(chi.URLParam(r, "kind"), d)
	if !ok {
		httpx.Problem(w, http.StatusNotFound, "Check-ins are daily, weekly, or monthly.")
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(body.Note) > 2000 {
		httpx.Problem(w, http.StatusUnprocessableEntity, "Check-in notes are limited to 2,000 characters.")
		return
	}
	var c checkin
	var stored time.Time
	err = h.db.QueryRow(r.Context(), `INSERT INTO checkins (user_id, kind, day, note) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, kind, day) DO UPDATE SET note = excluded.note
		RETURNING kind, day, note, created_at`, who.UserID, chi.URLParam(r, "kind"), day, body.Note).
		Scan(&c.Kind, &stored, &c.Note, &c.CreatedAt)
	if err != nil {
		h.fail(w, err)
		return
	}
	c.Day = Date{day}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) listCheckins(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	today, err := h.today(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	from, to := today.AddDate(0, 0, -30), today
	for name, dst := range map[string]*time.Time{"from": &from, "to": &to} {
		if v := r.URL.Query().Get(name); v != "" {
			if *dst, err = time.Parse(time.DateOnly, v); err != nil {
				httpx.Problem(w, http.StatusBadRequest, "?from= and ?to= must look like 2026-10-10.")
				return
			}
		}
	}
	if to.Sub(from) > 400*24*time.Hour || to.Before(from) {
		httpx.Problem(w, http.StatusBadRequest, "Ask for at most 400 days of check-ins at a time.")
		return
	}
	rows, err := h.db.Query(r.Context(), `SELECT kind, day, note, created_at FROM checkins
		WHERE user_id = $1 AND day BETWEEN $2 AND $3 ORDER BY day DESC, kind`, who.UserID, from, to)
	if err != nil {
		h.fail(w, err)
		return
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (checkin, error) {
		var c checkin
		var d time.Time
		err := row.Scan(&c.Kind, &d, &c.Note, &c.CreatedAt)
		c.Day = Date{day(d)}
		return c, err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	if list == nil {
		list = []checkin{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list, "today": Date{today}})
}
