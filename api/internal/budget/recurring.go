package budget

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Recurring is a transaction that repeats on an RRULE (rent, salary,
// subscriptions). Due occurrences are recorded when the budget is opened.
type Recurring struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind"`
	AmountCents int64      `json:"amount_cents"`
	AccountID   *uuid.UUID `json:"account_id"`
	ToAccountID *uuid.UUID `json:"to_account_id"`
	CategoryID  *uuid.UUID `json:"category_id"`
	Payee       string     `json:"payee"`
	Note        string     `json:"note"`
	Tags        []string   `json:"tags"`
	Rule        string     `json:"rule"`
	NextOn      *string    `json:"next_on"` // null once the series ended
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

const recurringColumns = `id, name, kind, amount_cents, account_id, to_account_id, category_id, payee, note, tags, rule,
	next_on::text, active, created_at, updated_at`

func scanRecurring(r pgx.Row) (Recurring, error) {
	var x Recurring
	err := r.Scan(&x.ID, &x.Name, &x.Kind, &x.AmountCents, &x.AccountID, &x.ToAccountID, &x.CategoryID, &x.Payee, &x.Note,
		&x.Tags, &x.Rule, &x.NextOn, &x.Active, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

// ListRecurring lists the user's recurring transactions, next due first.
func ListRecurring(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Recurring, error) {
	rows, err := q.Query(ctx, `SELECT `+recurringColumns+` FROM budget_recurring WHERE user_id = $1
		ORDER BY active DESC, next_on NULLS LAST, lower(name) LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Recurring, error) { return scanRecurring(r) })
	if out == nil {
		out = []Recurring{}
	}
	return out, err
}

func getRecurring(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Recurring, error) {
	r, err := scanRecurring(q.QueryRow(ctx, `SELECT `+recurringColumns+` FROM budget_recurring WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// RecurringFields are the editable parts; Start is the first occurrence.
type RecurringFields struct {
	Name, Rule, Start *string
	Active            *bool
	TransactionFields
}

func (r *Recurring) asTransaction() Transaction {
	return Transaction{Kind: r.Kind, AmountCents: r.AmountCents, AccountID: r.AccountID, ToAccountID: r.ToAccountID,
		CategoryID: r.CategoryID, Payee: r.Payee, Note: r.Note, Tags: r.Tags}
}

func (f *RecurringFields) apply(r *Recurring) error {
	t := &f.TransactionFields
	if err := t.validate(); err != nil {
		return err
	}
	if f.Name != nil {
		r.Name = strings.TrimSpace(*f.Name)
	}
	if r.Name == "" || len([]rune(r.Name)) > 100 {
		return ValidationError{"A recurring transaction needs a name of 1 to 100 characters."}
	}
	if f.Rule != nil {
		rule, err := tasks.ParseRule(*f.Rule)
		if err != nil {
			return ValidationError{err.Error()}
		}
		r.Rule = rule
	}
	if f.Start != nil {
		if _, err := time.Parse(time.DateOnly, *f.Start); err != nil {
			return ValidationError{"A start date looks like 2026-10-11."}
		}
		r.NextOn = f.Start
	}
	if f.Active != nil {
		r.Active = *f.Active
	}
	if t.Kind != nil {
		r.Kind = *t.Kind
	}
	if t.AmountCents != nil {
		r.AmountCents = *t.AmountCents
	}
	if t.AccountID != nil || t.ClearAccount {
		r.AccountID = t.AccountID
	}
	if t.ToAccountID != nil || t.ClearToAccount {
		r.ToAccountID = t.ToAccountID
	}
	if t.CategoryID != nil || t.ClearCategory {
		r.CategoryID = t.CategoryID
	}
	if t.Payee != nil {
		r.Payee = strings.TrimSpace(*t.Payee)
	}
	if t.Note != nil {
		r.Note = *t.Note
	}
	if t.Tags != nil {
		r.Tags = *t.Tags
	}
	return nil
}

func checkRecurring(ctx context.Context, q db.Querier, userID uuid.UUID, r Recurring) error {
	if r.Rule == "" || r.AmountCents <= 0 || r.Kind == "" {
		return ValidationError{"A recurring transaction needs a kind, an amount, a rule, and a start date."}
	}
	t := r.asTransaction()
	if err := checkCategory(ctx, q, userID, t.CategoryID, t.Kind); err != nil {
		return err
	}
	return checkAccounts(ctx, q, userID, t)
}

// CreateRecurring adds a recurring transaction starting on f.Start.
func CreateRecurring(ctx context.Context, q db.Querier, userID uuid.UUID, f RecurringFields) (Recurring, error) {
	if f.Start == nil || f.Rule == nil {
		return Recurring{}, ValidationError{"A recurring transaction needs a rule and a start date."}
	}
	r := Recurring{Active: true, Tags: []string{}}
	if err := f.apply(&r); err != nil {
		return r, err
	}
	if err := checkRecurring(ctx, q, userID, r); err != nil {
		return r, err
	}
	r.ID = uuid.Must(uuid.NewV7())
	if _, err := q.Exec(ctx, `INSERT INTO budget_recurring (id, user_id, name, kind, amount_cents, account_id, to_account_id, category_id,
			payee, note, tags, rule, next_on, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`, r.ID, userID, r.Name, r.Kind, r.AmountCents, r.AccountID,
		r.ToAccountID, r.CategoryID, r.Payee, r.Note, r.Tags, r.Rule, r.NextOn, r.Active); err != nil {
		return r, err
	}
	return getRecurring(ctx, q, userID, r.ID)
}

// UpdateRecurring changes a recurring transaction; already recorded ones stay.
func UpdateRecurring(ctx context.Context, q db.Querier, userID, id uuid.UUID, f RecurringFields) (Recurring, error) {
	r, err := getRecurring(ctx, q, userID, id)
	if err != nil {
		return r, err
	}
	if err := f.apply(&r); err != nil {
		return r, err
	}
	if err := checkRecurring(ctx, q, userID, r); err != nil {
		return r, err
	}
	if _, err := q.Exec(ctx, `UPDATE budget_recurring SET name = $3, kind = $4, amount_cents = $5, account_id = $6, to_account_id = $7,
		category_id = $8, payee = $9, note = $10, tags = $11, rule = $12, next_on = $13, active = $14, updated_at = now()
		WHERE id = $1 AND user_id = $2`, id, userID, r.Name, r.Kind, r.AmountCents, r.AccountID, r.ToAccountID, r.CategoryID,
		r.Payee, r.Note, r.Tags, r.Rule, r.NextOn, r.Active); err != nil {
		return r, err
	}
	return getRecurring(ctx, q, userID, id)
}

// DeleteRecurring stops a series; its recorded transactions stay.
func DeleteRecurring(ctx context.Context, q db.Querier, userID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM budget_recurring WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// maxCatchUp bounds how many occurrences one run records per series.
const maxCatchUp = 400

// RunRecurring records every due occurrence up to today. A unique index on
// (series, date) makes it idempotent: running it twice, or twice at once,
// records each occurrence once.
func RunRecurring(ctx context.Context, q db.Querier, userID uuid.UUID, today time.Time) (int, error) {
	rows, err := q.Query(ctx, `SELECT `+recurringColumns+` FROM budget_recurring
		WHERE user_id = $1 AND active AND next_on IS NOT NULL AND next_on <= $2`, userID, today)
	if err != nil {
		return 0, err
	}
	due, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Recurring, error) { return scanRecurring(r) })
	if err != nil {
		return 0, err
	}
	created := 0
	for _, r := range due {
		next, _ := time.Parse(time.DateOnly, *r.NextOn)
		ended := false
		for i := 0; i < maxCatchUp && !next.After(today); i++ {
			t := r.asTransaction()
			on := next.Format(time.DateOnly)
			f := TransactionFields{OccurredOn: &on, Kind: &t.Kind, AmountCents: &t.AmountCents, CategoryID: t.CategoryID,
				AccountID: t.AccountID, ToAccountID: t.ToAccountID, Tags: &t.Tags, Payee: &t.Payee, Note: &t.Note}
			id := r.ID
			switch _, err := createTransaction(ctx, q, userID, f, &id, nil); {
			case err == nil:
				created++
			case errors.Is(err, errDuplicate):
			default:
				return created, err
			}
			n, ok, err := tasks.NextOccurrence(r.Rule, next)
			if err != nil {
				return created, err
			}
			if !ok {
				ended = true
				break
			}
			next = n
		}
		var nextOn *string
		if !ended {
			s := next.Format(time.DateOnly)
			nextOn = &s
		}
		if _, err := q.Exec(ctx, `UPDATE budget_recurring SET next_on = $2, updated_at = now() WHERE id = $1`, r.ID, nextOn); err != nil {
			return created, err
		}
	}
	return created, nil
}

// Upcoming is one occurrence of a recurring income or bill still to come.
type Upcoming struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind"` // income or expense
	AmountCents int64      `json:"amount_cents"`
	AccountID   *uuid.UUID `json:"account_id"`
	On          string     `json:"on"` // YYYY-MM-DD
}

// UpcomingBefore lists each occurrence of the active recurring income and
// bills due after today and before `day`, soonest first. Transfers change
// nothing the person has, so they are left out. Due occurrences are
// recorded by RunRecurring before this is asked, so next_on is after today.
func UpcomingBefore(ctx context.Context, q db.Querier, userID uuid.UUID, today, day time.Time) ([]Upcoming, error) {
	rows, err := q.Query(ctx, `SELECT `+recurringColumns+` FROM budget_recurring
		WHERE user_id = $1 AND active AND kind <> 'transfer' AND next_on IS NOT NULL AND next_on > $2 AND next_on < $3`,
		userID, today, day)
	if err != nil {
		return nil, err
	}
	series, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Recurring, error) { return scanRecurring(r) })
	if err != nil {
		return nil, err
	}
	out := []Upcoming{}
	for _, r := range series {
		next, _ := time.Parse(time.DateOnly, *r.NextOn)
		for i := 0; i < maxCatchUp && next.Before(day); i++ {
			out = append(out, Upcoming{ID: r.ID, Name: r.Name, Kind: r.Kind, AmountCents: r.AmountCents, AccountID: r.AccountID,
				On: next.Format(time.DateOnly)})
			n, ok, err := tasks.NextOccurrence(r.Rule, next)
			if err != nil {
				return nil, err
			}
			if !ok {
				break
			}
			next = n
		}
	}
	slices.SortStableFunc(out, func(a, b Upcoming) int { return strings.Compare(a.On, b.On) })
	return out, nil
}

// Today is the user's calendar date in their time zone.
func Today(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (time.Time, error) {
	day, _, err := clock.Today(ctx, q, userID, now)
	return day, err
}
