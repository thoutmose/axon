package progress

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/habits"
)

// The collection (EP-119): cards to collect, each raised from bronze to
// silver to gold by what the person does — a trophy for each roadmap
// skill, a card for each habit, and cards for problems solved, focus,
// reviews, the daily streak, and decisions reviewed. Everything is worked
// out from what is already recorded; the interface words it.

// Tiers, lowest first.
var Tiers = []string{"bronze", "silver", "gold"}

// Step is the next tier and what it takes.
type Step struct {
	Tier   string  `json:"tier"`
	Target float64 `json:"target"`
}

// Card is one collectible: its tier ("" while locked), the value it counts,
// and the next tier.
type Card struct {
	ID     string  `json:"id"`
	Family string  `json:"family"` // skill, habit, practice, focus, reviews, streak, decisions
	Title  string  `json:"title"`  // the skill's or habit's name; empty for the others
	Kind   string  `json:"kind"`   // practice: easy, medium, hard
	Tier   string  `json:"tier"`
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"` // days, hours, problems, reviews, decisions
	Next   *Step   `json:"next"`
}

// Collection is every card, with the tiers earned out of all there are.
type Collection struct {
	Cards  []Card `json:"cards"`
	Earned int    `json:"earned"` // bronze counts 1, silver 2, gold 3
	Total  int    `json:"total"`
}

// tier is the highest step value reaches, and the next one.
func tier(value float64, steps [3]float64) (string, *Step) {
	current := ""
	for i, s := range steps {
		if value < s {
			return current, &Step{Tiers[i], s}
		}
		current = Tiers[i]
	}
	return current, nil
}

// longestRun is the longest run of consecutive days (ascending).
func longestRun(days []time.Time) int {
	best, run := 0, 0
	for i, d := range days {
		if i > 0 && d.Equal(days[i-1].AddDate(0, 0, 1)) {
			run++
		} else {
			run = 1
		}
		best = max(best, run)
	}
	return best
}

// Thresholds of the cards counted in numbers.
var (
	habitSteps    = [3]float64{7, 21, 66} // 66 days: the median time to automaticity (Lally et al., 2010)
	practiceSteps = map[string][3]float64{"easy": {10, 30, 60}, "medium": {5, 25, 60}, "hard": {1, 10, 30}}
	focusSteps    = [3]float64{10, 50, 200}
	reviewSteps   = [3]float64{100, 1000, 5000}
	streakSteps   = [3]float64{7, 30, 100}
	decisionSteps = [3]float64{5, 20, 50}
	skillHours    = 10.0 // of focus for a skill's silver trophy
)

func card(id, family, title, kind, unit string, value float64, steps [3]float64) Card {
	tr, next := tier(value, steps)
	return Card{ID: id, Family: family, Title: title, Kind: kind, Tier: tr, Value: value, Unit: unit, Next: next}
}

// GetCollection works out every card at today.
func GetCollection(ctx context.Context, q db.Querier, userID uuid.UUID, tz string, today time.Time) (Collection, error) {
	c := Collection{Cards: []Card{}}

	// Roadmap skills: started (bronze), 10 hours of focus (silver), learned (gold).
	rows, err := q.Query(ctx, `SELECT s.id, e.title, s.status, coalesce((SELECT sum(f.worked_minutes) FROM links l
			JOIN focus_sessions f ON f.id = l.source_id WHERE l.target_id = s.id AND l.relation = 'focused_on'), 0)::float8 / 60
		FROM skills s JOIN entities e ON e.id = s.id WHERE e.user_id = $1 AND s.status <> 'skipped' ORDER BY e.title`, userID)
	if err != nil {
		return c, err
	}
	type skill struct {
		ID     uuid.UUID
		Title  string
		Status string
		Hours  float64
	}
	skills, err := pgx.CollectRows(rows, pgx.RowToStructByPos[skill])
	if err != nil {
		return c, err
	}
	for _, s := range skills {
		k := Card{ID: "skill:" + s.ID.String(), Family: "skill", Title: s.Title, Value: s.Hours, Unit: "hours"}
		switch {
		case s.Status == "done":
			k.Tier = "gold"
		case s.Hours >= skillHours:
			k.Tier, k.Next = "silver", &Step{"gold", 0} // learned, not counted
		case s.Status == "learning" || s.Hours > 0:
			k.Tier, k.Next = "bronze", &Step{"silver", skillHours}
		default:
			k.Next = &Step{"bronze", 0} // started
		}
		c.Cards = append(c.Cards, k)
	}

	// Habits: the best run of scheduled days kept.
	rows, err = q.Query(ctx, `SELECT h.id, e.title, h.days::int[], (e.created_at AT TIME ZONE $2)::date,
			coalesce((SELECT json_object_agg(c.day::text, c.status) FROM habit_checkins c WHERE c.habit_id = h.id), '{}')
		FROM habits h JOIN entities e ON e.id = h.id WHERE e.user_id = $1 AND h.status <> 'archived' ORDER BY e.title`, userID, tz)
	if err != nil {
		return c, err
	}
	type habit struct {
		ID       uuid.UUID
		Title    string
		Days     []int
		Created  time.Time
		Checkins map[string]string
	}
	hs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[habit])
	if err != nil {
		return c, err
	}
	for _, h := range hs {
		// Check-ins made before the habit (imported history) count too.
		from := h.Created
		for d := range h.Checkins {
			if t, err := time.Parse(time.DateOnly, d); err == nil && t.Before(from) {
				from = t
			}
		}
		run := habits.BestRun(h.Days, h.Checkins, from, today)
		c.Cards = append(c.Cards, card("habit:"+h.ID.String(), "habit", h.Title, "", "days", float64(run), habitSteps))
	}

	// Problems solved unaided, by difficulty; focus hours; reviews; decisions reviewed.
	var easy, medium, hard, reviews, decided int
	var focusHours float64
	if err := q.QueryRow(ctx, `SELECT
		(SELECT count(*) FILTER (WHERE p.difficulty = 'easy') FROM practice_problems p WHERE p.user_id = $1
			AND EXISTS (SELECT 1 FROM practice_attempts a WHERE a.problem_id = p.id AND a.outcome = 'solved'))::int,
		(SELECT count(*) FILTER (WHERE p.difficulty = 'medium') FROM practice_problems p WHERE p.user_id = $1
			AND EXISTS (SELECT 1 FROM practice_attempts a WHERE a.problem_id = p.id AND a.outcome = 'solved'))::int,
		(SELECT count(*) FILTER (WHERE p.difficulty = 'hard') FROM practice_problems p WHERE p.user_id = $1
			AND EXISTS (SELECT 1 FROM practice_attempts a WHERE a.problem_id = p.id AND a.outcome = 'solved'))::int,
		(SELECT coalesce(sum(f.worked_minutes), 0) FROM focus_sessions f JOIN entities e ON e.id = f.id WHERE e.user_id = $1)::float8 / 60,
		(SELECT count(*) FROM card_reviews r JOIN entities e ON e.id = r.card_id WHERE e.user_id = $1)::int,
		(SELECT count(*) FROM decisions d WHERE d.user_id = $1 AND d.happened IS NOT NULL)::int`, userID).
		Scan(&easy, &medium, &hard, &focusHours, &reviews, &decided); err != nil {
		return c, err
	}
	for _, p := range []struct {
		kind string
		n    int
	}{{"easy", easy}, {"medium", medium}, {"hard", hard}} {
		c.Cards = append(c.Cards, card("practice:"+p.kind, "practice", "", p.kind, "problems", float64(p.n), practiceSteps[p.kind]))
	}
	c.Cards = append(c.Cards, card("focus", "focus", "", "", "hours", focusHours, focusSteps))
	c.Cards = append(c.Cards, card("reviews", "reviews", "", "", "reviews", float64(reviews), reviewSteps))

	// The longest run of active days, as the daily streak counts them.
	rows, err = q.Query(ctx, `SELECT DISTINCT (at AT TIME ZONE $4)::date AS d FROM (`+ledger+`) l(at, rule, points, id, title)
		WHERE points > 0 ORDER BY d`, userID, epoch, far(), tz)
	if err != nil {
		return c, err
	}
	days, err := pgx.CollectRows(rows, pgx.RowTo[time.Time])
	if err != nil {
		return c, err
	}
	c.Cards = append(c.Cards, card("streak", "streak", "", "", "days", float64(longestRun(days)), streakSteps))
	c.Cards = append(c.Cards, card("decisions", "decisions", "", "", "decisions", float64(decided), decisionSteps))

	for _, k := range c.Cards {
		c.Total += len(Tiers)
		for i, t := range Tiers {
			if k.Tier == t {
				c.Earned += i + 1
			}
		}
	}
	return c, nil
}
