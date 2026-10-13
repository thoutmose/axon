// Package dashboard serves the daily view, the calendar, and read-only iCal
// feeds (EP-08). It only reads other domains' tables.
package dashboard

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/l10n"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Handlers serves the dashboard, calendar, and feeds.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the dashboard endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the signed-in endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/dashboard", h.dashboard)
	r.Get("/v1/dashboard/overview", h.overview)
	r.Get("/v1/calendar", h.calendar)
}

// FeedRoutes mounts feed management; the caller wraps them in auth.RequireSession.
func (h *Handlers) FeedRoutes(r chi.Router) {
	r.Get("/v1/me/calendar-feeds", h.listFeeds)
	r.Post("/v1/me/calendar-feeds", h.createFeed)
	r.Delete("/v1/me/calendar-feeds/{id}", h.revokeFeed)
}

// PublicRoutes mounts the iCal feed itself, authenticated by its URL token.
func (h *Handlers) PublicRoutes(r chi.Router) {
	r.Get("/v1/calendar-feeds/{token}.ics", h.feed)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	h.log.Error("dashboard request failed", "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

// today is the user's current date in their time zone.
func today(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (time.Time, error) {
	day, _, err := clock.Today(ctx, q, userID, now)
	return day, err
}

// KPI is one dashboard number with the definition of what it counts.
type KPI struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Value      int    `json:"value"`
	Definition string `json:"definition"`
}

// kpis are counted in one query; each definition says exactly what is
// counted, and each equals the size of the task view with the same name.
var kpis = []struct {
	id, label, definition, sql string
}{
	{"overdue", l10n.Mark("Overdue"), l10n.Mark("Open tasks whose due date is before today."),
		`count(*) FILTER (WHERE t.status NOT IN ('done', 'cancelled', 'someday') AND t.due_on < $2)`},
	{"today", l10n.Mark("For today"), l10n.Mark("Open tasks scheduled or due today or earlier, overdue included; someday tasks are left out."),
		`count(*) FILTER (WHERE t.status NOT IN ('done', 'cancelled', 'someday') AND (t.scheduled_on <= $2 OR t.due_on <= $2))`},
	{"inbox", l10n.Mark("Inbox"), l10n.Mark("Captured tasks you have not clarified yet."),
		`count(*) FILTER (WHERE t.status = 'inbox')`},
}

type noteSummary struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
}

type milestoneSummary struct {
	ID        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	GoalID    uuid.UUID  `json:"goal_id"`
	GoalTitle string     `json:"goal_title"`
	DueOn     tasks.Date `json:"due_on"`
}

func (h *Handlers) dashboard(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	ctx := r.Context()
	day, err := today(ctx, h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}

	// The counts: one indexed pass over the user's tasks.
	sql := `SELECT `
	for i, k := range kpis {
		if i > 0 {
			sql += `, `
		}
		sql += k.sql
	}
	sql += ` FROM tasks t JOIN entities e ON e.id = t.id WHERE e.user_id = $1`
	values := make([]int, len(kpis))
	dest := make([]any, len(kpis))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := h.db.QueryRow(ctx, sql, who.UserID, day).Scan(dest...); err != nil {
		h.fail(w, err)
		return
	}
	out := struct {
		Today      tasks.Date         `json:"today"`
		KPIs       []KPI              `json:"kpis"`
		Tasks      []tasks.Task       `json:"tasks"`
		Goals      []goals.Goal       `json:"goals"`
		Milestones []milestoneSummary `json:"milestones"`
		Notes      []noteSummary      `json:"notes"`
		CheckedIn  bool               `json:"checked_in"`
		Totals     map[string]int     `json:"totals"`
	}{Today: tasks.Date{Time: day}, Totals: map[string]int{}}
	for i, k := range kpis {
		out.KPIs = append(out.KPIs, KPI{ID: k.id, Label: l10n.T(ctx, k.label), Value: values[i], Definition: l10n.T(ctx, k.definition)})
	}

	// Today's tasks: the same set as the "today" count.
	if out.Tasks, err = tasks.DueBy(ctx, h.db, who.UserID, day, 10); err != nil {
		h.fail(w, err)
		return
	}

	active, err := goals.List(ctx, h.db, who.UserID, "active")
	if err != nil {
		h.fail(w, err)
		return
	}
	out.Goals = active[:min(len(active), 5)]

	rows, err := h.db.Query(ctx, `SELECT m.id, me.title, g.id, ge.title, m.due_on
		FROM milestones m JOIN entities me ON me.id = m.id JOIN goals g ON g.id = m.goal_id JOIN entities ge ON ge.id = g.id
		WHERE me.user_id = $1 AND m.done_at IS NULL AND g.status = 'active' AND m.due_on BETWEEN $2 AND $2 + 14
		ORDER BY m.due_on, m.id LIMIT 10`, who.UserID, day)
	if err != nil {
		h.fail(w, err)
		return
	}
	out.Milestones, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (milestoneSummary, error) {
		var m milestoneSummary
		err := row.Scan(&m.ID, &m.Title, &m.GoalID, &m.GoalTitle, &m.DueOn.Time)
		return m, err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	if out.Milestones == nil {
		out.Milestones = []milestoneSummary{}
	}

	rows, err = h.db.Query(ctx, `SELECT n.id, e.title, n.path, n.updated_at FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = n.id) ORDER BY n.updated_at DESC, n.id LIMIT 5`, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	out.Notes, err = pgx.CollectRows(rows, pgx.RowToStructByPos[noteSummary])
	if err != nil {
		h.fail(w, err)
		return
	}
	if out.Notes == nil {
		out.Notes = []noteSummary{}
	}

	// Totals drive the first-run guidance ("create your first task…").
	var nTasks, nNotes, nGoals int
	err = h.db.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM entities WHERE user_id = $1 AND type = 'task'),
		(SELECT count(*) FROM entities ne WHERE user_id = $1 AND type = 'note' AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = ne.id)),
		(SELECT count(*) FROM entities WHERE user_id = $1 AND type = 'goal'),
		EXISTS (SELECT 1 FROM checkins WHERE user_id = $1 AND kind = 'daily' AND day = $2)`, who.UserID, day).
		Scan(&nTasks, &nNotes, &nGoals, &out.CheckedIn)
	if err != nil {
		h.fail(w, err)
		return
	}
	out.Totals["tasks"], out.Totals["notes"], out.Totals["goals"] = nTasks, nNotes, nGoals
	httpx.JSON(w, http.StatusOK, out)
}

// --- Calendar ---

// Event is one dated item on the calendar.
type Event struct {
	Date  tasks.Date `json:"date"`
	Type  string     `json:"type"` // task, goal, milestone
	Kind  string     `json:"kind"` // scheduled, due, target, milestone
	ID    uuid.UUID  `json:"id"`
	Title string     `json:"title"`
	Done  bool       `json:"done"`
	// GoalID is the milestone's goal, so the calendar can link to it.
	GoalID *uuid.UUID `json:"goal_id,omitempty"`
}

var allTypes = []string{"task", "goal", "milestone"}

// events returns dated items of the given types between from and to
// (inclusive), oldest first. Cancelled tasks and archived goals are left out;
// openOnly also leaves out done tasks, achieved goals, and done milestones.
func events(ctx context.Context, q db.Querier, userID uuid.UUID, from, to time.Time, types []string, openOnly bool, limit int) ([]Event, error) {
	rows, err := q.Query(ctx, `
		SELECT d.day, d.type, d.kind, d.id, d.title, d.done, d.goal_id FROM (
			SELECT t.scheduled_on AS day, 'task' AS type, 'scheduled' AS kind, t.id, e.title, t.status = 'done' AS done, NULL::uuid AS goal_id
			  FROM tasks t JOIN entities e ON e.id = t.id
			 WHERE e.user_id = $1 AND t.status <> 'cancelled' AND t.scheduled_on BETWEEN $2 AND $3
			   AND (t.due_on IS NULL OR t.due_on <> t.scheduled_on)
			UNION ALL
			SELECT t.due_on, 'task', 'due', t.id, e.title, t.status = 'done', NULL
			  FROM tasks t JOIN entities e ON e.id = t.id
			 WHERE e.user_id = $1 AND t.status <> 'cancelled' AND t.due_on BETWEEN $2 AND $3
			UNION ALL
			SELECT g.target_date, 'goal', 'target', g.id, e.title, g.status = 'achieved', NULL
			  FROM goals g JOIN entities e ON e.id = g.id
			 WHERE e.user_id = $1 AND g.status <> 'archived' AND g.target_date BETWEEN $2 AND $3
			UNION ALL
			SELECT m.due_on, 'milestone', 'milestone', m.id, e.title, m.done_at IS NOT NULL, m.goal_id
			  FROM milestones m JOIN entities e ON e.id = m.id JOIN goals g ON g.id = m.goal_id
			 WHERE e.user_id = $1 AND g.status <> 'archived' AND m.due_on BETWEEN $2 AND $3
		) d
		WHERE d.type = ANY($4) AND NOT ($5 AND d.done)
		ORDER BY d.day, d.type, d.title, d.id LIMIT $6`, userID, from, to, types, openOnly, limit)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Event, error) {
		var ev Event
		err := row.Scan(&ev.Date.Time, &ev.Type, &ev.Kind, &ev.ID, &ev.Title, &ev.Done, &ev.GoalID)
		return ev, err
	})
	if out == nil {
		out = []Event{}
	}
	return out, err
}

func parseTypes(s string) ([]string, bool) {
	if s == "" {
		return allTypes, true
	}
	var out []string
	for _, t := range strings.Split(s, ",") {
		if !slices.Contains(allTypes, t) {
			return nil, false
		}
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out, len(out) > 0
}

func (h *Handlers) calendar(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	q := r.URL.Query()
	from, err1 := time.Parse(time.DateOnly, q.Get("from"))
	to, err2 := time.Parse(time.DateOnly, q.Get("to"))
	if err1 != nil || err2 != nil || to.Before(from) || to.Sub(from) > 92*24*time.Hour {
		httpx.Problem(w, http.StatusBadRequest, "Send ?from= and ?to= as dates like 2026-10-01, at most 92 days apart.")
		return
	}
	types, ok := parseTypes(q.Get("types"))
	if !ok {
		httpx.Problem(w, http.StatusBadRequest, "?types= is a comma-separated list of task, goal, and milestone.")
		return
	}
	items, err := events(r.Context(), h.db, who.UserID, from, to, types, false, 2000)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}
