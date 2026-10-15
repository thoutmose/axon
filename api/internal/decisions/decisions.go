// Package decisions is the decision journal (EP-114): a decision with the
// outcome expected and the chance given to it, reviewed on its day, and
// scored against what happened, so a person learns how far to trust their
// own forecasts.
package decisions

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// ErrNotFound means no such decision for this person.
var ErrNotFound = errors.New("not found")

// Decision is a decision and, once reviewed, what happened.
type Decision struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Context     string    `json:"context"`
	Choice      string    `json:"choice"`
	Expected    string    `json:"expected"`
	Probability float64   `json:"probability"` // the chance that the expected outcome happens
	DecidedOn   string    `json:"decided_on"`
	ReviewOn    string    `json:"review_on"`
	Happened    *bool     `json:"happened"` // null until reviewed
	Outcome     string    `json:"outcome"`
	Lessons     string    `json:"lessons"`
	ReviewedOn  *string   `json:"reviewed_on"`
	Due         bool      `json:"due"` // not reviewed, and its day has come
	CreatedAt   time.Time `json:"created_at"`
}

// Input is a new decision. DecidedOn is today when empty.
type Input struct {
	Title       string   `json:"title"`
	Context     string   `json:"context"`
	Choice      string   `json:"choice"`
	Expected    string   `json:"expected"`
	Probability *float64 `json:"probability"`
	DecidedOn   string   `json:"decided_on"`
	ReviewOn    string   `json:"review_on"`
}

// Patch changes what can change: not the forecast.
type Patch struct {
	Title       *string  `json:"title"`
	Context     *string  `json:"context"`
	Choice      *string  `json:"choice"`
	ReviewOn    *string  `json:"review_on"`
	Expected    *string  `json:"expected"`
	Probability *float64 `json:"probability"`
}

// Review is what happened.
type Review struct {
	Happened *bool  `json:"happened"`
	Outcome  string `json:"outcome"`
	Lessons  string `json:"lessons"`
}

func text(s string, max int, tooLong ValidationError) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > max {
		return "", tooLong
	}
	return s, nil
}

func date(s string) (time.Time, bool) {
	d, err := time.Parse(time.DateOnly, s)
	return d, err == nil
}

func checkTitle(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > 200 {
		return "", ValidationError{"A decision needs a title of 1 to 200 characters."}
	}
	return s, nil
}

func checkReview(decided time.Time, s string) error {
	r, ok := date(s)
	if !ok || !r.After(decided) || r.After(decided.AddDate(10, 0, 0)) {
		return ValidationError{"The review day is after the decision, within ten years."}
	}
	return nil
}

func (in *Input) validate(today time.Time) (err error) {
	if in.Title, err = checkTitle(in.Title); err != nil {
		return err
	}
	if in.Context, err = text(in.Context, 5000, ValidationError{"The context is up to 5,000 characters."}); err != nil {
		return err
	}
	if in.Choice, err = text(in.Choice, 2000, ValidationError{"The choice is up to 2,000 characters."}); err != nil {
		return err
	}
	in.Expected = strings.TrimSpace(in.Expected)
	if in.Expected == "" || utf8.RuneCountInString(in.Expected) > 500 {
		return ValidationError{"Say what you expect to happen, in 1 to 500 characters."}
	}
	if in.Probability == nil || *in.Probability < 0.01 || *in.Probability > 0.99 {
		return ValidationError{"Give the chance that it happens, from 1 to 99 percent."}
	}
	decided := today
	if in.DecidedOn != "" {
		d, ok := date(in.DecidedOn)
		if !ok || d.After(today) {
			return ValidationError{"The decision is made today or a day before."}
		}
		decided = d
	}
	in.DecidedOn = decided.Format(time.DateOnly)
	return checkReview(decided, in.ReviewOn)
}

const columns = `id, title, context, choice, expected, probability, decided_on::text, review_on::text, happened, outcome, lessons, reviewed_on::text,
	happened IS NULL AND review_on <= $2::date, created_at`

func scan(r pgx.CollectableRow) (Decision, error) {
	var d Decision
	err := r.Scan(&d.ID, &d.Title, &d.Context, &d.Choice, &d.Expected, &d.Probability, &d.DecidedOn, &d.ReviewOn, &d.Happened,
		&d.Outcome, &d.Lessons, &d.ReviewedOn, &d.Due, &d.CreatedAt)
	return d, err
}

func get(ctx context.Context, q db.Querier, userID, id uuid.UUID, today time.Time) (Decision, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM decisions WHERE user_id = $1 AND id = $3`, userID, today, id)
	if err != nil {
		return Decision{}, err
	}
	d, err := pgx.CollectExactlyOneRow(rows, scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	return d, err
}

// Statuses filter the list.
var Statuses = []string{"", "open", "due", "reviewed"}

// List returns decisions: those to review first, by their day, then the
// reviewed ones, last reviewed first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, today time.Time, status string) ([]Decision, error) {
	where := map[string]string{
		"":         "",
		"open":     " AND happened IS NULL",
		"due":      " AND happened IS NULL AND review_on <= $2::date",
		"reviewed": " AND happened IS NOT NULL",
	}[status]
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM decisions WHERE user_id = $1`+where+`
		ORDER BY happened IS NOT NULL, CASE WHEN happened IS NULL THEN review_on END, reviewed_on DESC, created_at DESC`, userID, today)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, scan)
	if out == nil {
		out = []Decision{}
	}
	return out, err
}

// Create records a decision.
func Create(ctx context.Context, q db.Querier, userID uuid.UUID, in Input, today time.Time) (Decision, error) {
	if err := in.validate(today); err != nil {
		return Decision{}, err
	}
	id := uuid.Must(uuid.NewV7())
	if _, err := q.Exec(ctx, `INSERT INTO decisions (id, user_id, title, context, choice, expected, probability, decided_on, review_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, userID, in.Title, in.Context, in.Choice, in.Expected, *in.Probability, in.DecidedOn, in.ReviewOn); err != nil {
		return Decision{}, err
	}
	return get(ctx, q, userID, id, today)
}

// Update changes a decision's title, context, choice, or review day. The
// forecast stays as it was made.
func Update(ctx context.Context, q db.Querier, userID, id uuid.UUID, p Patch, today time.Time) (Decision, error) {
	d, err := get(ctx, q, userID, id, today)
	if err != nil {
		return d, err
	}
	if p.Expected != nil || p.Probability != nil {
		return d, ValidationError{"A forecast stays as it was made, so your score stays honest. Record a new decision instead."}
	}
	if p.Title != nil {
		if d.Title, err = checkTitle(*p.Title); err != nil {
			return d, err
		}
	}
	if p.Context != nil {
		if d.Context, err = text(*p.Context, 5000, ValidationError{"The context is up to 5,000 characters."}); err != nil {
			return d, err
		}
	}
	if p.Choice != nil {
		if d.Choice, err = text(*p.Choice, 2000, ValidationError{"The choice is up to 2,000 characters."}); err != nil {
			return d, err
		}
	}
	if p.ReviewOn != nil {
		decided, _ := date(d.DecidedOn)
		if err := checkReview(decided, *p.ReviewOn); err != nil {
			return d, err
		}
		d.ReviewOn = *p.ReviewOn
	}
	if _, err := q.Exec(ctx, `UPDATE decisions SET title = $3, context = $4, choice = $5, review_on = $6 WHERE user_id = $1 AND id = $2`,
		userID, id, d.Title, d.Context, d.Choice, d.ReviewOn); err != nil {
		return d, err
	}
	return get(ctx, q, userID, id, today)
}

// Resolve records what happened. It can be corrected later.
func Resolve(ctx context.Context, q db.Querier, userID, id uuid.UUID, r Review, today time.Time) (Decision, error) {
	if r.Happened == nil {
		return Decision{}, ValidationError{"Say whether what you expected happened."}
	}
	var err error
	if r.Outcome, err = text(r.Outcome, 5000, ValidationError{"What happened is up to 5,000 characters."}); err != nil {
		return Decision{}, err
	}
	if r.Lessons, err = text(r.Lessons, 5000, ValidationError{"The lessons are up to 5,000 characters."}); err != nil {
		return Decision{}, err
	}
	tag, err := q.Exec(ctx, `UPDATE decisions SET happened = $3, outcome = $4, lessons = $5, reviewed_on = coalesce(reviewed_on, $6)
		WHERE user_id = $1 AND id = $2`, userID, id, *r.Happened, r.Outcome, r.Lessons, today)
	if err != nil {
		return Decision{}, err
	}
	if tag.RowsAffected() == 0 {
		return Decision{}, ErrNotFound
	}
	return get(ctx, q, userID, id, today)
}

// Delete removes a decision.
func Delete(ctx context.Context, q db.Querier, userID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM decisions WHERE user_id = $1 AND id = $2`, userID, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// GetCalibration scores every reviewed forecast.
func GetCalibration(ctx context.Context, q db.Querier, userID uuid.UUID) (Calibration, error) {
	rows, err := q.Query(ctx, `SELECT probability, happened FROM decisions WHERE user_id = $1 AND happened IS NOT NULL`, userID)
	if err != nil {
		return Calibration{}, err
	}
	rs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Resolved])
	if err != nil {
		return Calibration{}, err
	}
	return Score(rs), nil
}

// Export is the person's decisions, for the data export.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return List(ctx, q, userID, time.Now(), "")
}
