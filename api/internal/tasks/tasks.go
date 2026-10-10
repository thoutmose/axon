// Package tasks implements tasks, GTD statuses and contexts, date-based
// recurrence, and daily/weekly/monthly check-ins (EP-09).
package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// Type is the graph entity type of a task.
const Type = "task"

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "tasks", Export: export})
	// A task can block another task.
	graph.RegisterRelation(graph.Relation{Name: "blocks", Sources: []string{Type}, Targets: []string{Type}})
}

// Statuses follow GTD: capture to the inbox, then clarify.
var statuses = []string{"inbox", "next", "waiting", "someday", "done", "cancelled"}

// Date is a calendar day, serialized as "YYYY-MM-DD".
type Date struct{ time.Time }

// MarshalJSON writes the date only.
func (d Date) MarshalJSON() ([]byte, error) { return json.Marshal(d.Format(time.DateOnly)) }

// UnmarshalJSON reads "YYYY-MM-DD".
func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return fmt.Errorf("dates use YYYY-MM-DD")
	}
	d.Time = t
	return nil
}

func datePtr(t *time.Time) *Date {
	if t == nil {
		return nil
	}
	return &Date{day(*t)}
}

// Task is the API representation.
type Task struct {
	ID           uuid.UUID  `json:"id"`
	Title        string     `json:"title"`
	Notes        string     `json:"notes"`
	Status       string     `json:"status"`
	Context      *string    `json:"context"`
	ScheduledOn  *Date      `json:"scheduled_on"`
	DueOn        *Date      `json:"due_on"`
	Recurrence   *string    `json:"recurrence"`
	SeriesID     *uuid.UUID `json:"series_id"`
	OccurrenceOn *Date      `json:"occurrence_on"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

const columns = `t.id, e.title, t.notes, t.status, t.context, t.scheduled_on, t.due_on, t.recurrence,
	t.series_id, t.occurrence_on, t.completed_at, e.created_at, t.updated_at`

func scan(row pgx.Row) (Task, error) {
	var t Task
	var scheduled, due, occurrence *time.Time
	err := row.Scan(&t.ID, &t.Title, &t.Notes, &t.Status, &t.Context, &scheduled, &due, &t.Recurrence,
		&t.SeriesID, &occurrence, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt)
	t.ScheduledOn, t.DueOn, t.OccurrenceOn = datePtr(scheduled), datePtr(due), datePtr(occurrence)
	return t, err
}

// Get returns one of the user's tasks.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Task, error) {
	t, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM tasks t JOIN entities e ON e.id = t.id
		WHERE t.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, graph.ErrNotFound
	}
	return t, err
}

// Fields are the editable parts of a task. Nil means "not provided"; for the
// nullable fields, Clear* removes the value.
type Fields struct {
	Title       *string
	Notes       *string
	Status      *string
	Context     *string
	ScheduledOn *time.Time
	DueOn       *time.Time
	Recurrence  *string

	ClearContext, ClearScheduled, ClearDue, ClearRecurrence bool
}

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

func (f *Fields) validate() error {
	if f.Title != nil {
		*f.Title = strings.TrimSpace(*f.Title)
		if *f.Title == "" || len([]rune(*f.Title)) > 500 {
			return ValidationError{"A task needs a title of 1 to 500 characters."}
		}
	}
	if f.Notes != nil && len(*f.Notes) > 20000 {
		return ValidationError{"Notes are limited to 20,000 characters."}
	}
	if f.Status != nil && !contains(statuses, *f.Status) {
		return ValidationError{"Status must be one of inbox, next, waiting, someday, done, cancelled."}
	}
	if f.Context != nil {
		c := strings.ToLower(strings.TrimSpace(*f.Context))
		if !strings.HasPrefix(c, "@") {
			c = "@" + c
		}
		if !validContext(c) {
			return ValidationError{"A context looks like @home or @phone: letters, digits, - and _, up to 40."}
		}
		f.Context = &c
	}
	if f.Recurrence != nil {
		rule, err := ParseRule(*f.Recurrence)
		if err != nil {
			return ValidationError{err.Error()}
		}
		f.Recurrence = &rule
	}
	return nil
}

func validContext(c string) bool {
	if len(c) < 2 || len(c) > 41 {
		return false
	}
	for _, r := range c[1:] {
		allowed := r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_'
		if !allowed {
			return false
		}
	}
	return true
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Create adds a task. Recurring tasks need a scheduled or due date to anchor the series.
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f Fields) (Task, error) {
	if f.Title == nil {
		return Task{}, ValidationError{"A task needs a title."}
	}
	if err := f.validate(); err != nil {
		return Task{}, err
	}
	status := "inbox"
	if f.Status != nil {
		status = *f.Status
	}
	notes := ""
	if f.Notes != nil {
		notes = *f.Notes
	}
	var occurrence *time.Time
	if f.Recurrence != nil {
		occurrence = firstNonNil(f.ScheduledOn, f.DueOn)
		if occurrence == nil {
			return Task{}, ValidationError{"A recurring task needs a scheduled or due date to start from."}
		}
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, *f.Title, notes)
	if err != nil {
		return Task{}, err
	}
	var series *uuid.UUID
	if f.Recurrence != nil {
		series = &id
	}
	var completed *time.Time
	if status == "done" || status == "cancelled" {
		now := time.Now()
		completed = &now
	}
	_, err = tx.Exec(ctx, `INSERT INTO tasks (id, status, notes, context, scheduled_on, due_on, recurrence, series_id, occurrence_on, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, status, notes, f.Context, f.ScheduledOn, f.DueOn, f.Recurrence, series, occurrence, completed)
	if err != nil {
		return Task{}, err
	}
	return Get(ctx, tx, userID, id)
}

func firstNonNil(ts ...*time.Time) *time.Time {
	for _, t := range ts {
		if t != nil {
			return t
		}
	}
	return nil
}

// ErrStale means the task changed since the caller read it (If-Match failed).
var ErrStale = errors.New("stale version")

// Update applies f if the task's version still equals version, and creates
// the next occurrence when a recurring task is completed.
func Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, version time.Time, f Fields) (Task, error) {
	if err := f.validate(); err != nil {
		return Task{}, err
	}
	cur, err := Get(ctx, tx, userID, id)
	if err != nil {
		return Task{}, err
	}
	if !cur.UpdatedAt.Equal(version) {
		return Task{}, ErrStale
	}
	next := cur
	var changed []string
	set := func(name string, cond bool, apply func()) {
		if cond {
			apply()
			changed = append(changed, name)
		}
	}
	set("title", f.Title != nil && *f.Title != cur.Title, func() { next.Title = *f.Title })
	set("notes", f.Notes != nil && *f.Notes != cur.Notes, func() { next.Notes = *f.Notes })
	set("status", f.Status != nil && *f.Status != cur.Status, func() { next.Status = *f.Status })
	set("context", f.Context != nil || (f.ClearContext && cur.Context != nil), func() { next.Context = f.Context })
	set("scheduled_on", f.ScheduledOn != nil || (f.ClearScheduled && cur.ScheduledOn != nil), func() { next.ScheduledOn = datePtr(f.ScheduledOn) })
	set("due_on", f.DueOn != nil || (f.ClearDue && cur.DueOn != nil), func() { next.DueOn = datePtr(f.DueOn) })
	set("recurrence", f.Recurrence != nil || (f.ClearRecurrence && cur.Recurrence != nil), func() { next.Recurrence = f.Recurrence })
	if len(changed) == 0 {
		return cur, nil
	}

	finished := func(s string) bool { return s == "done" || s == "cancelled" }
	completedAt := cur.CompletedAt
	switch {
	case finished(next.Status) && !finished(cur.Status):
		now := time.Now()
		completedAt = &now
	case !finished(next.Status):
		completedAt = nil
	}
	if next.Recurrence != nil && next.ScheduledOn == nil && next.DueOn == nil {
		return Task{}, ValidationError{"A recurring task needs a scheduled or due date."}
	}
	series, occurrence := cur.SeriesID, unwrap(cur.OccurrenceOn)
	if next.Recurrence != nil && series == nil { // becoming recurring
		series = &cur.ID
		occurrence = unwrap(firstDate(next.ScheduledOn, next.DueOn))
	}

	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, next.Title, next.Notes, changed); err != nil {
		return Task{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE tasks SET status = $2, notes = $3, context = $4, scheduled_on = $5, due_on = $6,
		recurrence = $7, series_id = $8, occurrence_on = $9, completed_at = $10, updated_at = clock_timestamp() WHERE id = $1`,
		id, next.Status, next.Notes, next.Context, unwrap(next.ScheduledOn), unwrap(next.DueOn), next.Recurrence,
		series, occurrence, completedAt)
	if err != nil {
		return Task{}, err
	}
	if next.Status == "done" && cur.Status != "done" && next.Recurrence != nil {
		if err := createNextOccurrence(ctx, tx, userID, actor, next, *series, *occurrence); err != nil {
			return Task{}, err
		}
	}
	return Get(ctx, tx, userID, id)
}

func unwrap(d *Date) *time.Time {
	if d == nil {
		return nil
	}
	return &d.Time
}

func firstDate(ds ...*Date) *Date {
	for _, d := range ds {
		if d != nil {
			return d
		}
	}
	return nil
}

// createNextOccurrence adds the series' next task once: completing the same
// occurrence twice (a retry, a double click, two tabs) finds it already there.
func createNextOccurrence(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, done Task, series uuid.UUID, occurrence time.Time) error {
	nextDay, ok, err := NextOccurrence(*done.Recurrence, occurrence)
	if err != nil || !ok {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, series.String()); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tasks WHERE series_id = $1 AND occurrence_on = $2)`, series, nextDay).Scan(&exists); err != nil || exists {
		return err
	}
	shift := nextDay.Sub(occurrence)
	moved := func(d *Date) *time.Time {
		if d == nil {
			return nil
		}
		t := d.Add(shift)
		return &t
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, done.Title, done.Notes)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO tasks (id, status, notes, context, scheduled_on, due_on, recurrence, series_id, occurrence_on)
		VALUES ($1, 'next', $2, $3, $4, $5, $6, $7, $8)`,
		id, done.Notes, done.Context, moved(done.ScheduledOn), moved(done.DueOn), done.Recurrence, series, nextDay)
	return err
}

// Delete removes a task (and its links) through the graph.
func Delete(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	if _, err := Get(ctx, tx, userID, id); err != nil {
		return err
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM tasks t JOIN entities e ON e.id = t.id WHERE e.user_id = $1 ORDER BY t.id`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Task, error) { return scan(row) })
}
