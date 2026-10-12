package learning

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
)

// The day's plan puts the principles to work without the person having to
// think about them: what is due is reviewed a little every day (spacing),
// mixed (interleaving), by recalling rather than rereading (retrieval),
// whole notes are recalled on a blank page when their time comes, a short
// mixed test comes back every week, and the cards that keep slipping are
// named, with what to do about them (deliberate practice).

// Plan is what to do today to learn and remember.
type Plan struct {
	Cards struct {
		Due       int     `json:"due"`     // what a review now would offer
		Reviews   int     `json:"reviews"` // known cards due
		New       int     `json:"new"`     // new cards in today's allowance
		Minutes   int     `json:"minutes"` // estimated, from the person's own pace
		Leeches   int     `json:"leeches"`
		Retention float64 `json:"retention"` // last 30 days, 0 when too few reviews
		Reviewed  int     `json:"reviewed"`  // reviews behind the retention
		Target    float64 `json:"target"`    // the retention FSRS aims for
	} `json:"cards"`
	Recall struct {
		Due       []RecallNote `json:"due"`
		Suggested []RecallNote `json:"suggested"`
	} `json:"recall"`
	Test struct {
		Due    bool       `json:"due"` // none in the last 7 days, and cards to test
		LastAt *time.Time `json:"last_at"`
		Score  *float64   `json:"score"` // share right in the last test
	} `json:"test"`
	Fleeting   int `json:"fleeting"` // fleeting notes older than two days, to process
	Palaces    int `json:"palaces"`
	StreakDays int `json:"streak_days"` // consecutive days with a review or a recall, ending today or yesterday
}

// Today builds the plan.
func Today(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (Plan, error) {
	var p Plan
	prefs, err := cards.LoadPrefs(ctx, q, userID)
	if err != nil {
		return p, err
	}
	p.Cards.Target = prefs.Retention
	if p.Cards.Due, err = cards.DueCount(ctx, q, userID, now); err != nil {
		return p, err
	}
	var total int
	var avgMs *float64
	if err := q.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM cards c JOIN entities e ON e.id = c.id WHERE e.user_id = $1 AND NOT c.suspended AND c.reps > 0 AND c.due <= $2),
		(SELECT count(*) FROM cards c JOIN entities e ON e.id = c.id WHERE e.user_id = $1 AND NOT c.suspended AND c.lapses >= $3),
		(SELECT count(*) FROM cards c JOIN entities e ON e.id = c.id WHERE e.user_id = $1 AND NOT c.suspended),
		(SELECT avg(duration_ms)::float8 FROM (SELECT r.duration_ms FROM card_reviews r JOIN entities e ON e.id = r.card_id
			WHERE e.user_id = $1 AND r.duration_ms IS NOT NULL ORDER BY r.reviewed_at DESC LIMIT 200) x),
		(SELECT count(*) FROM memory_palaces WHERE user_id = $1),
		(SELECT count(*) FROM notes n JOIN entities e ON e.id = n.id WHERE n.user_id = $1 AND n.kind = 'fleeting' AND e.created_at < $4)`,
		userID, now, cards.LeechLapses, now.Add(-48*time.Hour)).
		Scan(&p.Cards.Reviews, &p.Cards.Leeches, &total, &avgMs, &p.Palaces, &p.Fleeting); err != nil {
		return p, err
	}
	p.Cards.New = max(0, p.Cards.Due-p.Cards.Reviews)
	perCard := 10.0 // seconds, until the person's own pace is known
	if avgMs != nil {
		perCard = min(120, *avgMs/1000)
	}
	if p.Cards.Due > 0 {
		p.Cards.Minutes = max(1, int(float64(p.Cards.Due)*perCard/60+0.5))
	}
	if p.Cards.Retention, p.Cards.Reviewed, err = cards.Retention(ctx, q, userID, now); err != nil {
		return p, err
	}
	if p.Recall.Due, p.Recall.Suggested, err = RecallDue(ctx, q, userID, now, 3); err != nil {
		return p, err
	}
	var last time.Time
	var questions, correct int
	err = q.QueryRow(ctx, `SELECT taken_at, questions, correct FROM practice_tests WHERE user_id = $1 ORDER BY taken_at DESC LIMIT 1`, userID).
		Scan(&last, &questions, &correct)
	switch {
	case err == nil:
		score := float64(correct) / float64(questions)
		p.Test.LastAt, p.Test.Score = &last, &score
		p.Test.Due = total > 0 && now.Sub(last) >= 7*24*time.Hour
	case errors.Is(err, pgx.ErrNoRows):
		p.Test.Due = total >= 5
	default:
		return p, err
	}
	loc, _ := clock.Zone(ctx, q, userID) // UTC when it cannot be read
	rows, err := q.Query(ctx, `SELECT DISTINCT d FROM (
			SELECT (r.reviewed_at AT TIME ZONE $3)::date AS d FROM card_reviews r JOIN entities e ON e.id = r.card_id WHERE e.user_id = $1 AND r.reviewed_at > $2
			UNION SELECT (recalled_at AT TIME ZONE $3)::date FROM note_recalls WHERE user_id = $1 AND recalled_at > $2
			UNION SELECT (taken_at AT TIME ZONE $3)::date FROM practice_tests WHERE user_id = $1 AND taken_at > $2) x ORDER BY d DESC`,
		userID, now.AddDate(0, 0, -400), loc.String())
	if err != nil {
		return p, err
	}
	days, err := pgx.CollectRows(rows, pgx.RowTo[time.Time])
	if err != nil {
		return p, err
	}
	p.StreakDays = streak(days, clock.Day(now, loc))
	return p, nil
}

// streak counts consecutive days, newest first, ending today or yesterday.
func streak(days []time.Time, today time.Time) int {
	want := today
	if len(days) > 0 && !days[0].Equal(today) {
		want = today.AddDate(0, 0, -1)
	}
	n := 0
	for _, d := range days {
		if !d.Equal(want) {
			break
		}
		n++
		want = want.AddDate(0, 0, -1)
	}
	return n
}

// RecordTest stores a practice test's result.
func RecordTest(ctx context.Context, q db.Querier, userID uuid.UUID, deck string, questions, correct int, now time.Time) error {
	if questions < 1 || questions > 100 || correct < 0 || correct > questions || len([]rune(deck)) > 200 {
		return ValidationError{"A test has 1 to 100 questions, and no more right answers than questions."}
	}
	_, err := q.Exec(ctx, `INSERT INTO practice_tests (id, user_id, deck, questions, correct, taken_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.Must(uuid.NewV7()), userID, deck, questions, correct, now)
	return err
}

// Export is the recall attempts and practice tests, for a personal data export.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	type recall struct {
		NoteID uuid.UUID `json:"note_id"`
		Score  float64   `json:"score"`
		At     time.Time `json:"recalled_at"`
		DueOn  string    `json:"due_on"`
	}
	type test struct {
		Deck      string    `json:"deck"`
		Questions int       `json:"questions"`
		Correct   int       `json:"correct"`
		At        time.Time `json:"taken_at"`
	}
	rows, err := q.Query(ctx, `SELECT note_id, score::float8, recalled_at, to_char(due_on, 'YYYY-MM-DD') FROM note_recalls WHERE user_id = $1 ORDER BY recalled_at`, userID)
	if err != nil {
		return nil, err
	}
	recalls, err := pgx.CollectRows(rows, pgx.RowToStructByPos[recall])
	if err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT deck, questions, correct, taken_at FROM practice_tests WHERE user_id = $1 ORDER BY taken_at`, userID)
	if err != nil {
		return nil, err
	}
	tests, err := pgx.CollectRows(rows, pgx.RowToStructByPos[test])
	if err != nil {
		return nil, err
	}
	return map[string]any{"note_recalls": recalls, "practice_tests": tests}, nil
}
