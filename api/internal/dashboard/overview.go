package dashboard

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/l10n"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// The command dashboard (EP-30): KPIs over a period with the previous period
// for comparison, daily series, a year of activity, the tables that need
// attention, and the skills network. Every KPI says what it counts and links
// to the items behind it.

// Trend is a KPI compared with the period before.
type Trend struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Value      int    `json:"value"`
	Previous   *int   `json:"previous"` // nil when a comparison makes no sense
	Unit       string `json:"unit"`     // count or percent
	Definition string `json:"definition"`
	Href       string `json:"href"`
}

// Point is one day of a series.
type Point struct {
	Day   tasks.Date `json:"day"`
	Value int        `json:"value"`
}

// Series is a named daily series.
type Series struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Points []Point `json:"points"`
}

type window struct{ from, to time.Time }

// counts returns the period KPIs for one window, in the user's time zone.
func counts(ctx context.Context, q db.Querier, userID uuid.UUID, tz string, w window) (map[string]int, error) {
	var tasksDone, notes, skills, met, due, focusMin int
	err := q.QueryRow(ctx, `
		WITH hab AS (
			SELECT h.id, h.days, greatest($3::date, least((e.created_at AT TIME ZONE $2)::date,
				coalesce((SELECT min(day) FROM habit_checkins c WHERE c.habit_id = h.id), 'infinity'::date))) AS start
			FROM habits h JOIN entities e ON e.id = h.id WHERE e.user_id = $1 AND h.status = 'active'
		), sched AS (
			SELECT hab.id, d::date AS day FROM hab, generate_series(hab.start, $4::date, interval '1 day') d
			WHERE extract(isodow FROM d)::smallint = ANY (hab.days)
		)
		SELECT
			(SELECT count(*) FROM tasks t JOIN entities e ON e.id = t.id
			 WHERE e.user_id = $1 AND t.status = 'done' AND (t.completed_at AT TIME ZONE $2)::date BETWEEN $3 AND $4),
			(SELECT count(*) FROM entities ne WHERE user_id = $1 AND type = 'note' AND (created_at AT TIME ZONE $2)::date BETWEEN $3 AND $4 AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = ne.id)),
			(SELECT count(*) FROM skills s JOIN entities e ON e.id = s.id
			 WHERE e.user_id = $1 AND s.status = 'done' AND (s.updated_at AT TIME ZONE $2)::date BETWEEN $3 AND $4),
			(SELECT count(*) FROM sched JOIN habit_checkins c ON c.habit_id = sched.id AND c.day = sched.day AND c.status IN ('done', 'two_minute')),
			(SELECT count(*) FROM sched),
			(SELECT coalesce(sum(f.worked_minutes), 0) FROM focus_sessions f JOIN entities e ON e.id = f.id
			 WHERE e.user_id = $1 AND (f.started_at AT TIME ZONE $2)::date BETWEEN $3 AND $4)`,
		userID, tz, w.from, w.to).Scan(&tasksDone, &notes, &skills, &met, &due, &focusMin)
	if err != nil {
		return nil, err
	}
	consistency := 0
	if due > 0 {
		consistency = met * 100 / due
	}
	return map[string]int{"tasks_done": tasksDone, "notes_written": notes, "skills_done": skills, "habit_consistency": consistency, "habit_due": due, "focus_minutes": focusMin}, nil
}

func (h *Handlers) overview(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	ctx := r.Context()
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 7 || n > 365 {
			httpx.Problem(w, http.StatusBadRequest, "?days= is a number from 7 to 365.")
			return
		}
		days = n
	}
	day, loc, err := clock.Today(ctx, h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	tz := loc.String()
	cur := window{day.AddDate(0, 0, 1-days), day}
	prev := window{day.AddDate(0, 0, 1-2*days), day.AddDate(0, 0, -days)}
	now, err := counts(ctx, h.db, who.UserID, tz, cur)
	if err != nil {
		h.fail(w, err)
		return
	}
	before, err := counts(ctx, h.db, who.UserID, tz, prev)
	if err != nil {
		h.fail(w, err)
		return
	}
	active, err := goals.List(ctx, h.db, who.UserID, "active")
	if err != nil {
		h.fail(w, err)
		return
	}
	goalSum, measured := 0, 0
	for _, g := range active {
		if g.Progress.Percent != nil {
			goalSum += *g.Progress.Percent
			measured++
		}
	}
	goalAvg := 0
	if measured > 0 {
		goalAvg = goalSum / measured
	}
	var projectsActive, learning, cardsDue int
	if err := h.db.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM projects p JOIN entities e ON e.id = p.id WHERE e.user_id = $1 AND p.status = 'active'),
		(SELECT count(*) FROM skills s JOIN entities e ON e.id = s.id WHERE e.user_id = $1 AND s.status = 'learning')`, who.UserID).
		Scan(&projectsActive, &learning); err != nil {
		h.fail(w, err)
		return
	}
	if cardsDue, err = cards.DueCount(ctx, h.db, who.UserID, time.Now()); err != nil {
		h.fail(w, err)
		return
	}
	p := func(v int) *int { return &v }
	period := l10n.T(ctx, "%d days", days)
	consistencyPrev := p(before["habit_consistency"])
	if before["habit_due"] == 0 {
		consistencyPrev = nil
	}
	kpis := []Trend{
		{"tasks_done", l10n.T(ctx, "Tasks done"), now["tasks_done"], p(before["tasks_done"]), "count",
			l10n.T(ctx, "Tasks you completed in the last %s, by completion date in your time zone.", period), "/tasks?view=done"},
		{"focus_minutes", l10n.T(ctx, "Focus time"), now["focus_minutes"], p(before["focus_minutes"]), "minutes",
			l10n.T(ctx, "Minutes worked in focus sessions started in the last %s, breaks left out.", period), "/focus"},
		{"habit_consistency", l10n.T(ctx, "Habit consistency"), now["habit_consistency"], consistencyPrev, "percent",
			l10n.T(ctx, "Check-ins (done or two-minute) on scheduled days, out of the scheduled days of your active habits in the last %s, counted from each habit's start.", period), "/habits"},
		{"goal_progress", l10n.T(ctx, "Goal progress"), goalAvg, nil, "percent",
			l10n.T(ctx, "The average progress of your active goals that have something to measure, as each goal explains it."), "/goals"},
		{"skills_done", l10n.T(ctx, "Skills learned"), now["skills_done"], p(before["skills_done"]), "count",
			l10n.T(ctx, "Roadmap skills marked done in the last %s, by their last change.", period), "/roadmaps"},
		{"learning", l10n.T(ctx, "Learning now"), learning, nil, "count", l10n.T(ctx, "Roadmap skills with the status Learning."), "/roadmaps"},
		{"cards_due", l10n.T(ctx, "Cards to review"), cardsDue, nil, "count", l10n.T(ctx, "Flashcards a review would offer now: known cards FSRS has scheduled, and new cards within the day's allowance; suspended cards are left out."), "/learn"},
		{"notes_written", l10n.T(ctx, "Notes written"), now["notes_written"], p(before["notes_written"]), "count",
			l10n.T(ctx, "Notes created in the last %s.", period), "/notes"},
		{"projects_active", l10n.T(ctx, "Active projects"), projectsActive, nil, "count", l10n.T(ctx, "Projects with the status Active."), "/projects"},
	}

	series, err := h.series(ctx, who.UserID, tz, cur)
	if err != nil {
		h.fail(w, err)
		return
	}
	heat, err := h.daily(ctx, `SELECT (created_at AT TIME ZONE $2)::date AS d, count(*) FROM events
		WHERE user_id = $1 AND actor <> 'watch' AND (created_at AT TIME ZONE $2)::date BETWEEN $3 AND $4 GROUP BY d`, who.UserID, tz, window{day.AddDate(0, 0, -364), day})
	if err != nil {
		h.fail(w, err)
		return
	}
	tables, err := h.attention(ctx, who.UserID, day)
	if err != nil {
		h.fail(w, err)
		return
	}
	network, err := skillsNetwork(ctx, h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"today": tasks.Date{Time: day}, "days": days, "kpis": kpis, "series": series,
		"activity": heat, "tables": tables, "skills": network,
	})
}

// daily runs a "day, count" query and fills every day of the window.
func (h *Handlers) daily(ctx context.Context, sql string, userID uuid.UUID, tz string, w window) ([]Point, error) {
	rows, err := h.db.Query(ctx, sql, userID, tz, w.from, w.to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byDay := map[string]int{}
	for rows.Next() {
		var d time.Time
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			return nil, err
		}
		byDay[d.Format(time.DateOnly)] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := []Point{}
	for d := w.from; !d.After(w.to); d = d.AddDate(0, 0, 1) {
		out = append(out, Point{tasks.Date{Time: d}, byDay[d.Format(time.DateOnly)]})
	}
	return out, nil
}

func (h *Handlers) series(ctx context.Context, userID uuid.UUID, tz string, w window) ([]Series, error) {
	defs := []struct{ id, label, sql string }{
		{"tasks_done", "Tasks done", `SELECT (t.completed_at AT TIME ZONE $2)::date AS d, count(*) FROM tasks t JOIN entities e ON e.id = t.id
			WHERE e.user_id = $1 AND t.status = 'done' AND (t.completed_at AT TIME ZONE $2)::date BETWEEN $3 AND $4 GROUP BY d`},
		{"habit_checkins", "Habit check-ins", `SELECT c.day AS d, count(*) FROM habit_checkins c JOIN entities e ON e.id = c.habit_id
			WHERE e.user_id = $1 AND c.status IN ('done', 'two_minute') AND c.day BETWEEN $3 AND $4
			AND $2::text IS NOT NULL GROUP BY d`}, // check-in days are already local; $2 keeps the parameters uniform
		{"focus_minutes", "Focus minutes", `SELECT (f.started_at AT TIME ZONE $2)::date AS d, sum(f.worked_minutes)::int FROM focus_sessions f JOIN entities e ON e.id = f.id
			WHERE e.user_id = $1 AND (f.started_at AT TIME ZONE $2)::date BETWEEN $3 AND $4 GROUP BY d`},
		{"notes_written", "Notes written", `SELECT (created_at AT TIME ZONE $2)::date AS d, count(*) FROM entities ne
			WHERE user_id = $1 AND type = 'note' AND (created_at AT TIME ZONE $2)::date BETWEEN $3 AND $4 AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = ne.id) GROUP BY d`},
	}
	out := []Series{}
	for _, d := range defs {
		pts, err := h.daily(ctx, d.sql, userID, tz, w)
		if err != nil {
			return nil, err
		}
		out = append(out, Series{d.id, d.label, pts})
	}
	return out, nil
}

type attentionTask struct {
	ID    uuid.UUID  `json:"id"`
	Title string     `json:"title"`
	DueOn tasks.Date `json:"due_on"`
}

type stalledProject struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	LastActivity time.Time `json:"last_activity"`
}

type learningSkill struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	RoadmapID    uuid.UUID `json:"roadmap_id"`
	RoadmapTitle string    `json:"roadmap_title"`
}

// attention lists what needs a look: tasks due in the next 7 days, active
// projects with no activity for 14 days, and skills being learned.
func (h *Handlers) attention(ctx context.Context, userID uuid.UUID, day time.Time) (map[string]any, error) {
	rows, err := h.db.Query(ctx, `SELECT t.id, e.title, t.due_on FROM tasks t JOIN entities e ON e.id = t.id
		WHERE e.user_id = $1 AND t.status NOT IN ('done', 'cancelled') AND t.due_on BETWEEN $2 AND $2 + 7
		ORDER BY t.due_on, t.id LIMIT 20`, userID, day)
	if err != nil {
		return nil, err
	}
	due, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (attentionTask, error) {
		var t attentionTask
		return t, r.Scan(&t.ID, &t.Title, &t.DueOn.Time)
	})
	if err != nil {
		return nil, err
	}
	rows, err = h.db.Query(ctx, `SELECT p.id, e.title, greatest(p.updated_at, coalesce(
			(SELECT max(ev.created_at) FROM project_cards pc JOIN events ev ON ev.entity_id = pc.task_id WHERE pc.project_id = p.id), p.updated_at)) AS last
		FROM projects p JOIN entities e ON e.id = p.id
		WHERE e.user_id = $1 AND p.status = 'active'
		ORDER BY last LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	projects, err := pgx.CollectRows(rows, pgx.RowToStructByPos[stalledProject])
	if err != nil {
		return nil, err
	}
	stalled := []stalledProject{}
	for _, p := range projects {
		if p.LastActivity.Before(h.now().AddDate(0, 0, -14)) {
			stalled = append(stalled, p)
		}
	}
	rows, err = h.db.Query(ctx, `SELECT s.id, e.title, r.id, re.title FROM skills s JOIN entities e ON e.id = s.id
		JOIN roadmaps r ON r.id = s.roadmap_id JOIN entities re ON re.id = r.id
		WHERE e.user_id = $1 AND s.status = 'learning' ORDER BY s.updated_at DESC LIMIT 20`, userID)
	if err != nil {
		return nil, err
	}
	learning, err := pgx.CollectRows(rows, pgx.RowToStructByPos[learningSkill])
	if err != nil {
		return nil, err
	}
	if due == nil {
		due = []attentionTask{}
	}
	if learning == nil {
		learning = []learningSkill{}
	}
	return map[string]any{"due_soon": due, "stalled_projects": stalled, "learning": learning}, nil
}
