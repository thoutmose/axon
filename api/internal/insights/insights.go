// Package insights computes explainable personal metrics and weekly reviews
// (EP-20), on the server only. Every metric says how it is counted and can
// list the items behind its number. Weeks are ISO weeks (Monday first) in
// the person's time zone.
package insights

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Metric is a documented weekly measure.
type Metric struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Unit    string `json:"unit"`    // count, minutes, or percent
	Formula string `json:"formula"` // how it is counted, in plain words
	Source  string `json:"source"`  // where the facts come from
	// weekly returns (week, value) rows for $1 user, $2 time zone, $3 first
	// day (inclusive), $4 last day (exclusive). items lists what is behind
	// the number: (id, type, title, when).
	weekly, items string
}

// local turns a timestamp column into the person's local date.
func local(col string) string { return "(" + col + " AT TIME ZONE $2)::date" }

// notMirrored leaves out the notes the vault mirror keeps for items (EP-48):
// they would count every task as a note written.
const notMirrored = ` AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = ne.id)`

func week(day string) string { return "date_trunc('week', " + day + ")::date" }

// Metrics are the definitions, in display order.
var Metrics = []Metric{
	{
		ID: "tasks_done", Title: "Tasks completed", Unit: "count",
		Formula: "Tasks whose completion time falls in the week.", Source: "tasks.completed_at",
		weekly: `SELECT ` + week(local("t.completed_at")) + `, count(*) FROM tasks t JOIN entities e ON e.id = t.id
			WHERE e.user_id = $1 AND ` + local("t.completed_at") + ` >= $3 AND ` + local("t.completed_at") + ` < $4 GROUP BY 1`,
		items: `SELECT t.id, 'task', e.title, t.completed_at FROM tasks t JOIN entities e ON e.id = t.id
			WHERE e.user_id = $1 AND ` + local("t.completed_at") + ` >= $3 AND ` + local("t.completed_at") + ` < $4 ORDER BY t.completed_at DESC`,
	},
	{
		ID: "tasks_captured", Title: "Tasks captured", Unit: "count",
		Formula: "Tasks created in the week, whatever their status now.", Source: "entities.created_at (type task)",
		weekly: `SELECT ` + week(local("created_at")) + `, count(*) FROM entities
			WHERE user_id = $1 AND type = 'task' AND ` + local("created_at") + ` >= $3 AND ` + local("created_at") + ` < $4 GROUP BY 1`,
		items: `SELECT id, type, title, created_at FROM entities
			WHERE user_id = $1 AND type = 'task' AND ` + local("created_at") + ` >= $3 AND ` + local("created_at") + ` < $4 ORDER BY created_at DESC`,
	},
	{
		ID: "habit_checkins", Title: "Habit check-ins", Unit: "count",
		Formula: "Days a habit was checked in as done or as its two-minute version (skips do not count).", Source: "habit_checkins.day",
		weekly: `SELECT ` + week("c.day") + `, count(*) FROM habit_checkins c JOIN entities e ON e.id = c.habit_id
			WHERE e.user_id = $1 AND c.status IN ('done', 'two_minute') AND c.day >= $3 AND c.day < $4 AND $2::text <> '' GROUP BY 1`,
		items: `SELECT e.id, 'habit', e.title || ' · ' || to_char(c.day, 'Dy DD Mon') || CASE WHEN c.status = 'two_minute' THEN ' (two-minute version)' ELSE '' END,
			c.day::timestamptz FROM habit_checkins c JOIN entities e ON e.id = c.habit_id
			WHERE e.user_id = $1 AND c.status IN ('done', 'two_minute') AND c.day >= $3 AND c.day < $4 AND $2::text <> '' ORDER BY c.day DESC, e.title`,
	},
	{
		ID: "milestones_done", Title: "Milestones reached", Unit: "count",
		Formula: "Goal milestones marked done in the week.", Source: "milestones.done_at",
		weekly: `SELECT ` + week(local("m.done_at")) + `, count(*) FROM milestones m JOIN entities e ON e.id = m.id
			WHERE e.user_id = $1 AND ` + local("m.done_at") + ` >= $3 AND ` + local("m.done_at") + ` < $4 GROUP BY 1`,
		items: `SELECT m.id, 'milestone', e.title, m.done_at FROM milestones m JOIN entities e ON e.id = m.id
			WHERE e.user_id = $1 AND ` + local("m.done_at") + ` >= $3 AND ` + local("m.done_at") + ` < $4 ORDER BY m.done_at DESC`,
	},
	{
		ID: "focus_minutes", Title: "Focus minutes", Unit: "minutes",
		Formula: "Minutes worked in Pomodoro sessions started in the week.", Source: "focus_sessions.worked_minutes",
		weekly: `SELECT ` + week(local("f.started_at")) + `, sum(f.worked_minutes) FROM focus_sessions f JOIN entities e ON e.id = f.id
			WHERE e.user_id = $1 AND f.worked_minutes > 0 AND ` + local("f.started_at") + ` >= $3 AND ` + local("f.started_at") + ` < $4 GROUP BY 1`,
		items: `SELECT f.id, 'focus', e.title || ' · ' || f.worked_minutes || ' min', f.started_at FROM focus_sessions f JOIN entities e ON e.id = f.id
			WHERE e.user_id = $1 AND f.worked_minutes > 0 AND ` + local("f.started_at") + ` >= $3 AND ` + local("f.started_at") + ` < $4 ORDER BY f.started_at DESC`,
	},
	{
		ID: "notes_written", Title: "Notes written", Unit: "count",
		Formula: "Notes created in the week (synced vault files count when they first arrive; the notes Axon keeps for items do not).", Source: "entities.created_at (type note)",
		weekly: `SELECT ` + week(local("created_at")) + `, count(*) FROM entities ne
			WHERE user_id = $1 AND type = 'note' AND ` + local("created_at") + ` >= $3 AND ` + local("created_at") + ` < $4` + notMirrored + ` GROUP BY 1`,
		items: `SELECT id, type, title, created_at FROM entities ne
			WHERE user_id = $1 AND type = 'note' AND ` + local("created_at") + ` >= $3 AND ` + local("created_at") + ` < $4` + notMirrored + ` ORDER BY created_at DESC`,
	},
	{
		ID: "cards_reviewed", Title: "Flashcards reviewed", Unit: "count",
		Formula: "Flashcard reviews in the week (one card reviewed twice counts twice).", Source: "card_reviews.reviewed_at",
		weekly: `SELECT ` + week(local("r.reviewed_at")) + `, count(*) FROM card_reviews r JOIN entities e ON e.id = r.card_id
			WHERE e.user_id = $1 AND ` + local("r.reviewed_at") + ` >= $3 AND ` + local("r.reviewed_at") + ` < $4 GROUP BY 1`,
		items: `SELECT e.id, 'card', e.title || ' · ' || (ARRAY['again', 'hard', 'good', 'easy'])[r.rating], r.reviewed_at FROM card_reviews r JOIN entities e ON e.id = r.card_id
			WHERE e.user_id = $1 AND ` + local("r.reviewed_at") + ` >= $3 AND ` + local("r.reviewed_at") + ` < $4 ORDER BY r.reviewed_at DESC`,
	},
	{
		ID: "recall_rate", Title: "Recall rate", Unit: "percent",
		Formula: "Share of flashcard reviews answered good or easy, out of all reviews in the week.", Source: "card_reviews.rating",
		weekly: `SELECT ` + week(local("r.reviewed_at")) + `, round(100.0 * count(*) FILTER (WHERE r.rating >= 3) / count(*)) FROM card_reviews r JOIN entities e ON e.id = r.card_id
			WHERE e.user_id = $1 AND ` + local("r.reviewed_at") + ` >= $3 AND ` + local("r.reviewed_at") + ` < $4 GROUP BY 1`,
		items: `SELECT e.id, 'card', e.title || ' · ' || (ARRAY['again', 'hard', 'good', 'easy'])[r.rating], r.reviewed_at FROM card_reviews r JOIN entities e ON e.id = r.card_id
			WHERE e.user_id = $1 AND r.rating < 3 AND ` + local("r.reviewed_at") + ` >= $3 AND ` + local("r.reviewed_at") + ` < $4 ORDER BY r.reviewed_at DESC`,
	},
	{
		ID: "skills_learned", Title: "Skills learned", Unit: "count",
		Formula: "Skills marked learned, dated by their last change.", Source: "skills.status = done, skills.updated_at",
		weekly: `SELECT ` + week(local("s.updated_at")) + `, count(*) FROM skills s JOIN entities e ON e.id = s.id
			WHERE e.user_id = $1 AND s.status = 'done' AND ` + local("s.updated_at") + ` >= $3 AND ` + local("s.updated_at") + ` < $4 GROUP BY 1`,
		items: `SELECT s.id, 'skill', e.title, s.updated_at FROM skills s JOIN entities e ON e.id = s.id
			WHERE e.user_id = $1 AND s.status = 'done' AND ` + local("s.updated_at") + ` >= $3 AND ` + local("s.updated_at") + ` < $4 ORDER BY s.updated_at DESC`,
	},
}

// Find returns a metric by id.
func Find(id string) (Metric, bool) {
	for _, m := range Metrics {
		if m.ID == id {
			return m, true
		}
	}
	return Metric{}, false
}

// Monday returns the Monday of the ISO week containing day.
func Monday(day time.Time) time.Time {
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// ParseWeek reads "2026-W41" and returns its Monday.
func ParseWeek(s string) (time.Time, bool) {
	var y, w int
	if _, err := fmt.Sscanf(s, "%d-W%d", &y, &w); err != nil || w < 1 || w > 53 || y < 1970 || y > 9999 {
		return time.Time{}, false
	}
	// ISO week 1 contains 4 January.
	m := Monday(time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)).AddDate(0, 0, 7*(w-1))
	if yy, ww := m.ISOWeek(); yy != y || ww != w {
		return time.Time{}, false
	}
	return m, true
}

// Label is a week's ISO name, like "2026-W41".
func Label(monday time.Time) string {
	y, w := monday.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

// values runs a metric over [from, to) and returns a value per Monday.
func values(ctx context.Context, q db.Querier, m Metric, userID uuid.UUID, tz string, from, to time.Time) (map[string]float64, error) {
	rows, err := q.Query(ctx, `SELECT w, coalesce(v, 0)::float8 FROM (`+m.weekly+`) AS x(w, v)`, userID, tz, from.Format(time.DateOnly), to.Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("metric %s: %w", m.ID, err)
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var wk time.Time
		var v float64
		if err := rows.Scan(&wk, &v); err != nil {
			return nil, err
		}
		out[wk.Format(time.DateOnly)] = v
	}
	return out, rows.Err()
}

// Point is one week's value.
type Point struct {
	Week   string  `json:"week"`   // ISO name
	Monday string  `json:"monday"` // its first day
	Value  float64 `json:"value"`
}

// Trend compares the last four weeks with the four before.
type Trend struct {
	Direction string  `json:"direction"` // up, down, flat, or new
	Change    float64 `json:"change"`    // percent, when there is a before
	Recent    float64 `json:"recent"`    // weekly average, last four weeks
	Before    float64 `json:"before"`    // weekly average, the four weeks before
	Sentence  string  `json:"sentence"`
}

// Series is a metric with its weekly values and trend.
type Series struct {
	Metric
	Points []Point `json:"points"`
	Total  float64 `json:"total"` // sum of the counts, or the mean of a percentage over weeks with data
	Trend  Trend   `json:"trend"`
}

// Weekly computes every metric for the weeks ending with the one containing today.
func Weekly(ctx context.Context, q db.Querier, userID uuid.UUID, tz string, today time.Time, weeks int) ([]Series, error) {
	last := Monday(today)
	first := last.AddDate(0, 0, -7*(weeks-1))
	out := make([]Series, 0, len(Metrics))
	for _, m := range Metrics {
		vals, err := values(ctx, q, m, userID, tz, first, last.AddDate(0, 0, 7))
		if err != nil {
			return nil, err
		}
		s := Series{Metric: m, Points: make([]Point, weeks)}
		withData := 0
		for i := range weeks {
			mon := first.AddDate(0, 0, 7*i)
			v, ok := vals[mon.Format(time.DateOnly)]
			s.Points[i] = Point{Week: Label(mon), Monday: mon.Format(time.DateOnly), Value: v}
			s.Total += v
			if ok {
				withData++
			}
		}
		if m.Unit == "percent" {
			s.Total = 0
			if withData > 0 {
				sum := 0.0
				for _, p := range s.Points {
					sum += p.Value
				}
				s.Total = math.Round(sum / float64(withData))
			}
		}
		s.Trend = trend(s.Points, m.Unit)
		out = append(out, s)
	}
	return out, nil
}

// trend compares the averages of the last four points and the four before.
func trend(points []Point, unit string) Trend {
	avg := func(ps []Point) float64 {
		if len(ps) == 0 {
			return 0
		}
		sum := 0.0
		for _, p := range ps {
			sum += p.Value
		}
		return sum / float64(len(ps))
	}
	n := len(points)
	recent := avg(points[max(0, n-4):])
	before := avg(points[max(0, n-8):max(0, n-4)])
	t := Trend{Recent: math.Round(recent*10) / 10, Before: math.Round(before*10) / 10}
	per := " a week"
	if unit == "percent" {
		per = " %"
	}
	switch {
	case before == 0 && recent == 0:
		t.Direction, t.Sentence = "flat", "Nothing in the last eight weeks."
	case before == 0:
		t.Direction, t.Sentence = "new", fmt.Sprintf("New in the last four weeks: %.1f%s on average.", t.Recent, per)
	default:
		t.Change = math.Round((recent - before) / before * 100)
		t.Direction = "flat"
		if t.Change >= 10 {
			t.Direction = "up"
		} else if t.Change <= -10 {
			t.Direction = "down"
		}
		word := map[string]string{"up": "Up", "down": "Down", "flat": "Steady"}[t.Direction]
		t.Sentence = fmt.Sprintf("%s %+.0f %% on the four weeks before (%.1f → %.1f%s).", word, t.Change, t.Before, t.Recent, per)
	}
	return t
}

// Item is something behind a number.
type Item struct {
	ID    uuid.UUID `json:"id"`
	Type  string    `json:"type"`
	Title string    `json:"title"`
	When  time.Time `json:"when"`
}

// Items lists what a metric counted in [from, to), newest first, up to limit.
func Items(ctx context.Context, q db.Querier, m Metric, userID uuid.UUID, tz string, from, to time.Time, limit int) ([]Item, error) {
	rows, err := q.Query(ctx, m.items+fmt.Sprintf(" LIMIT %d", limit), userID, tz, from.Format(time.DateOnly), to.Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("metric %s items: %w", m.ID, err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Item])
	if out == nil {
		out = []Item{}
	}
	return out, err
}
