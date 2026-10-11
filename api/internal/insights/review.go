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

// ReviewMetric is a metric's value for the week and the week before.
type ReviewMetric struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Unit     string  `json:"unit"`
	Formula  string  `json:"formula"`
	Value    float64 `json:"value"`
	Previous float64 `json:"previous"`
}

// Section lists what a metric counted this week.
type Section struct {
	Metric string `json:"metric"`
	Title  string `json:"title"`
	Items  []Item `json:"items"`
	More   int    `json:"more"` // how many more items there were
}

// Review is the weekly review of one ISO week.
type Review struct {
	Week       string         `json:"week"`
	From       string         `json:"from"`
	To         string         `json:"to"` // last day, inclusive
	Previous   string         `json:"previous"`
	Next       *string        `json:"next"` // null for the current week
	Metrics    []ReviewMetric `json:"metrics"`
	Highlights []string       `json:"highlights"`
	Sections   []Section      `json:"sections"`
	CarryOver  []Item         `json:"carry_over"` // open tasks due by the end of the week
}

const sectionItems = 25

// WeeklyReview builds the review of the week starting on monday.
func WeeklyReview(ctx context.Context, q db.Querier, userID uuid.UUID, tz string, monday, today time.Time) (Review, error) {
	end := monday.AddDate(0, 0, 7)
	prev := monday.AddDate(0, 0, -7)
	r := Review{Week: Label(monday), From: monday.Format(time.DateOnly), To: end.AddDate(0, 0, -1).Format(time.DateOnly), Previous: Label(prev),
		Metrics: []ReviewMetric{}, Highlights: []string{}, Sections: []Section{}}
	if end.Before(Monday(today).AddDate(0, 0, 1)) {
		n := Label(end)
		r.Next = &n
	}
	for _, m := range Metrics {
		vals, err := values(ctx, q, m, userID, tz, prev, end)
		if err != nil {
			return r, err
		}
		rm := ReviewMetric{ID: m.ID, Title: m.Title, Unit: m.Unit, Formula: m.Formula,
			Value: vals[monday.Format(time.DateOnly)], Previous: vals[prev.Format(time.DateOnly)]}
		r.Metrics = append(r.Metrics, rm)
		if h := highlight(rm); h != "" {
			r.Highlights = append(r.Highlights, h)
		}
		if rm.Value == 0 || m.Unit == "percent" {
			continue
		}
		items, err := Items(ctx, q, m, userID, tz, monday, end, sectionItems)
		if err != nil {
			return r, err
		}
		more := 0
		if m.Unit == "count" {
			more = max(0, int(rm.Value)-len(items))
		}
		r.Sections = append(r.Sections, Section{Metric: m.ID, Title: m.Title, Items: items, More: more})
	}
	rows, err := q.Query(ctx, `SELECT t.id, 'task', e.title, t.due_on::timestamptz FROM tasks t JOIN entities e ON e.id = t.id
		WHERE e.user_id = $1 AND t.status NOT IN ('done', 'cancelled') AND t.due_on < $2 ORDER BY t.due_on, e.title LIMIT 20`,
		userID, end.Format(time.DateOnly))
	if err != nil {
		return r, err
	}
	r.CarryOver, err = pgx.CollectRows(rows, pgx.RowToStructByPos[Item])
	if r.CarryOver == nil {
		r.CarryOver = []Item{}
	}
	return r, err
}

// highlight says what changed most, without praise or blame.
func highlight(m ReviewMetric) string {
	switch {
	case m.Value == 0 && m.Previous == 0:
		return ""
	case m.Unit == "percent":
		return fmt.Sprintf("%s: %.0f %% (%.0f %% the week before).", m.Title, m.Value, m.Previous)
	case m.Previous == 0:
		return fmt.Sprintf("%s: %.0f, none the week before.", m.Title, m.Value)
	}
	change := math.Round((m.Value - m.Previous) / m.Previous * 100)
	if math.Abs(change) < 25 {
		return ""
	}
	return fmt.Sprintf("%s: %.0f, %+.0f %% on the week before (%.0f).", m.Title, m.Value, change, m.Previous)
}
