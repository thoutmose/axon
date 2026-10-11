// Package budget keeps income, expenses, and monthly budgets per category
// (EP-39, first slice). Amounts are integer cents in one currency per
// person; nothing here is a graph entity, so search and agents never read
// it.
package budget

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/thoutmose/axon/api/internal/db"
)

// MaxCents bounds every amount: a billion in the currency's main unit.
const MaxCents = 100_000_000_000

// Errors callers translate to HTTP responses.
var ErrNotFound = errors.New("not found")

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Category groups transactions; expense categories can have a monthly budget.
type Category struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"` // income or expense
	MonthlyCents *int64    `json:"monthly_cents"`
	Archived     bool      `json:"archived"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Transaction is money in or out on a day.
type Transaction struct {
	ID          uuid.UUID  `json:"id"`
	OccurredOn  string     `json:"occurred_on"`
	Kind        string     `json:"kind"`
	AmountCents int64      `json:"amount_cents"`
	CategoryID  *uuid.UUID `json:"category_id"`
	Payee       string     `json:"payee"`
	Note        string     `json:"note"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Line is one category in a month: what was budgeted and what happened.
type Line struct {
	Category
	ActualCents int64 `json:"actual_cents"`
}

// Month is the budget view of one calendar month.
type Month struct {
	Month         string `json:"month"` // YYYY-MM
	Currency      string `json:"currency"`
	IncomeCents   int64  `json:"income_cents"`
	ExpenseCents  int64  `json:"expense_cents"`
	BudgetedCents int64  `json:"budgeted_cents"` // sum of expense budgets
	// Uncategorised totals, so the lines always add up to the totals.
	UncategorisedIncomeCents  int64  `json:"uncategorised_income_cents"`
	UncategorisedExpenseCents int64  `json:"uncategorised_expense_cents"`
	Lines                     []Line `json:"lines"`
}

var monthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// MonthRange returns the first day of month and of the next month.
func MonthRange(month string) (time.Time, time.Time, error) {
	if !monthRe.MatchString(month) {
		return time.Time{}, time.Time{}, ValidationError{"A month looks like 2026-10."}
	}
	start, err := time.Parse("2006-01", month)
	return start, start.AddDate(0, 1, 0), err
}

// Currency returns the person's currency (EUR until they choose).
func Currency(ctx context.Context, q db.Querier, userID uuid.UUID) (string, error) {
	var c string
	err := q.QueryRow(ctx, `SELECT currency FROM budget_profiles WHERE user_id = $1`, userID).Scan(&c)
	if errors.Is(err, pgx.ErrNoRows) {
		return "EUR", nil
	}
	return c, err
}

var currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)

// SetCurrency changes the display currency; amounts are not converted.
func SetCurrency(ctx context.Context, q db.Querier, userID uuid.UUID, currency string) error {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !currencyRe.MatchString(currency) {
		return ValidationError{"A currency is a three-letter ISO 4217 code, like EUR or USD."}
	}
	_, err := q.Exec(ctx, `INSERT INTO budget_profiles (user_id, currency) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET currency = $2, updated_at = now()`, userID, currency)
	return err
}

// Summary computes a month: totals, and actual against budget per category.
// Every sum is done by the database in integers.
func Summary(ctx context.Context, q db.Querier, userID uuid.UUID, month string) (Month, error) {
	start, end, err := MonthRange(month)
	if err != nil {
		return Month{}, err
	}
	m := Month{Month: month, Lines: []Line{}}
	if m.Currency, err = Currency(ctx, q, userID); err != nil {
		return m, err
	}
	rows, err := q.Query(ctx, `SELECT `+categoryColumns+`, coalesce((SELECT sum(t.amount_cents) FROM budget_transactions t
			WHERE t.category_id = c.id AND t.occurred_on >= $2 AND t.occurred_on < $3), 0)::bigint
		FROM budget_categories c WHERE c.user_id = $1
		ORDER BY c.kind DESC, c.archived, lower(c.name)`, userID, start, end)
	if err != nil {
		return m, err
	}
	lines, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Line, error) {
		var l Line
		err := r.Scan(&l.ID, &l.Name, &l.Kind, &l.MonthlyCents, &l.Archived, &l.CreatedAt, &l.UpdatedAt, &l.ActualCents)
		return l, err
	})
	if err != nil {
		return m, err
	}
	for _, l := range lines {
		if l.Archived && l.ActualCents == 0 {
			continue // an old category with nothing this month
		}
		m.Lines = append(m.Lines, l)
		if l.Kind == "expense" && l.MonthlyCents != nil && !l.Archived {
			m.BudgetedCents += *l.MonthlyCents
		}
	}
	err = q.QueryRow(ctx, `SELECT
			coalesce(sum(amount_cents) FILTER (WHERE kind = 'income'), 0)::bigint,
			coalesce(sum(amount_cents) FILTER (WHERE kind = 'expense'), 0)::bigint,
			coalesce(sum(amount_cents) FILTER (WHERE kind = 'income' AND category_id IS NULL), 0)::bigint,
			coalesce(sum(amount_cents) FILTER (WHERE kind = 'expense' AND category_id IS NULL), 0)::bigint
		FROM budget_transactions WHERE user_id = $1 AND occurred_on >= $2 AND occurred_on < $3`, userID, start, end).
		Scan(&m.IncomeCents, &m.ExpenseCents, &m.UncategorisedIncomeCents, &m.UncategorisedExpenseCents)
	return m, err
}

const categoryColumns = `c.id, c.name, c.kind, c.monthly_cents, c.archived, c.created_at, c.updated_at`

// CategoryFields are the editable parts; nil means "not provided".
type CategoryFields struct {
	Name         *string
	Kind         *string
	MonthlyCents *int64
	ClearMonthly bool
	Archived     *bool
}

func (f *CategoryFields) validate() error {
	if f.Name != nil {
		*f.Name = strings.TrimSpace(*f.Name)
		if *f.Name == "" || len([]rune(*f.Name)) > 100 {
			return ValidationError{"A category needs a name of 1 to 100 characters."}
		}
	}
	if f.Kind != nil && *f.Kind != "income" && *f.Kind != "expense" {
		return ValidationError{"A category is for income or expense."}
	}
	if f.MonthlyCents != nil && (*f.MonthlyCents < 0 || *f.MonthlyCents > MaxCents) {
		return ValidationError{"A monthly budget is between 0 and 1,000,000,000."}
	}
	return nil
}

func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// GetCategory returns one of the user's categories.
func GetCategory(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Category, error) {
	var c Category
	err := q.QueryRow(ctx, `SELECT `+categoryColumns+` FROM budget_categories c WHERE c.id = $1 AND c.user_id = $2`, id, userID).
		Scan(&c.ID, &c.Name, &c.Kind, &c.MonthlyCents, &c.Archived, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// CreateCategory adds a category.
func CreateCategory(ctx context.Context, q db.Querier, userID uuid.UUID, f CategoryFields) (Category, error) {
	if f.Name == nil || f.Kind == nil {
		return Category{}, ValidationError{"A category needs a name and a kind (income or expense)."}
	}
	if err := f.validate(); err != nil {
		return Category{}, err
	}
	id := uuid.Must(uuid.NewV7())
	_, err := q.Exec(ctx, `INSERT INTO budget_categories (id, user_id, name, kind, monthly_cents) VALUES ($1, $2, $3, $4, $5)`,
		id, userID, *f.Name, *f.Kind, f.MonthlyCents)
	if uniqueViolation(err) {
		return Category{}, ValidationError{"You already have a category with that name."}
	}
	if err != nil {
		return Category{}, err
	}
	return GetCategory(ctx, q, userID, id)
}

// UpdateCategory changes a category. Its kind is fixed once created.
func UpdateCategory(ctx context.Context, q db.Querier, userID, id uuid.UUID, f CategoryFields) (Category, error) {
	if f.Kind != nil {
		return Category{}, ValidationError{"A category's kind cannot change; create a new one."}
	}
	if err := f.validate(); err != nil {
		return Category{}, err
	}
	c, err := GetCategory(ctx, q, userID, id)
	if err != nil {
		return c, err
	}
	if f.Name != nil {
		c.Name = *f.Name
	}
	if f.MonthlyCents != nil || f.ClearMonthly {
		c.MonthlyCents = f.MonthlyCents
	}
	if f.Archived != nil {
		c.Archived = *f.Archived
	}
	_, err = q.Exec(ctx, `UPDATE budget_categories SET name = $3, monthly_cents = $4, archived = $5, updated_at = now()
		WHERE id = $1 AND user_id = $2`, id, userID, c.Name, c.MonthlyCents, c.Archived)
	if uniqueViolation(err) {
		return Category{}, ValidationError{"You already have a category with that name."}
	}
	if err != nil {
		return Category{}, err
	}
	return GetCategory(ctx, q, userID, id)
}

// DeleteCategory removes a category; its transactions become uncategorised.
func DeleteCategory(ctx context.Context, q db.Querier, userID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM budget_categories WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// Categories lists all of the user's categories.
func Categories(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Category, error) {
	rows, err := q.Query(ctx, `SELECT `+categoryColumns+` FROM budget_categories c WHERE c.user_id = $1
		ORDER BY c.kind DESC, c.archived, lower(c.name) LIMIT 500`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Category, error) {
		var c Category
		err := r.Scan(&c.ID, &c.Name, &c.Kind, &c.MonthlyCents, &c.Archived, &c.CreatedAt, &c.UpdatedAt)
		return c, err
	})
	if out == nil {
		out = []Category{}
	}
	return out, err
}

// TransactionFields are the editable parts; nil means "not provided".
type TransactionFields struct {
	OccurredOn    *string
	Kind          *string
	AmountCents   *int64
	CategoryID    *uuid.UUID
	ClearCategory bool
	Payee, Note   *string
}

func (f *TransactionFields) validate() error {
	if f.OccurredOn != nil {
		if _, err := time.Parse(time.DateOnly, *f.OccurredOn); err != nil {
			return ValidationError{"A date looks like 2026-10-11."}
		}
	}
	if f.Kind != nil && *f.Kind != "income" && *f.Kind != "expense" {
		return ValidationError{"A transaction is income or expense."}
	}
	if f.AmountCents != nil && (*f.AmountCents <= 0 || *f.AmountCents > MaxCents) {
		return ValidationError{"An amount is more than 0 and at most 1,000,000,000."}
	}
	if f.Payee != nil && len([]rune(*f.Payee)) > 200 {
		return ValidationError{"A payee is up to 200 characters."}
	}
	if f.Note != nil && len([]rune(*f.Note)) > 2000 {
		return ValidationError{"A note is up to 2,000 characters."}
	}
	return nil
}

const transactionColumns = `id, occurred_on::text, kind, amount_cents, category_id, payee, note, created_at, updated_at`

func scanTransaction(r pgx.Row) (Transaction, error) {
	var t Transaction
	err := r.Scan(&t.ID, &t.OccurredOn, &t.Kind, &t.AmountCents, &t.CategoryID, &t.Payee, &t.Note, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

// GetTransaction returns one of the user's transactions.
func GetTransaction(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Transaction, error) {
	t, err := scanTransaction(q.QueryRow(ctx, `SELECT `+transactionColumns+` FROM budget_transactions WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

// checkCategory makes sure a category is the user's and matches the kind.
func checkCategory(ctx context.Context, q db.Querier, userID uuid.UUID, id *uuid.UUID, kind string) error {
	if id == nil {
		return nil
	}
	c, err := GetCategory(ctx, q, userID, *id)
	if errors.Is(err, ErrNotFound) {
		return ValidationError{"No such category."}
	}
	if err != nil {
		return err
	}
	if c.Kind != kind {
		return ValidationError{"Use an " + kind + " category for an " + kind + "."}
	}
	return nil
}

// CreateTransaction records money in or out.
func CreateTransaction(ctx context.Context, q db.Querier, userID uuid.UUID, f TransactionFields) (Transaction, error) {
	if f.OccurredOn == nil || f.Kind == nil || f.AmountCents == nil {
		return Transaction{}, ValidationError{"A transaction needs a date, a kind, and an amount."}
	}
	if err := f.validate(); err != nil {
		return Transaction{}, err
	}
	if err := checkCategory(ctx, q, userID, f.CategoryID, *f.Kind); err != nil {
		return Transaction{}, err
	}
	payee, note := "", ""
	if f.Payee != nil {
		payee = strings.TrimSpace(*f.Payee)
	}
	if f.Note != nil {
		note = *f.Note
	}
	id := uuid.Must(uuid.NewV7())
	if _, err := q.Exec(ctx, `INSERT INTO budget_transactions (id, user_id, occurred_on, kind, amount_cents, category_id, payee, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, userID, *f.OccurredOn, *f.Kind, *f.AmountCents, f.CategoryID, payee, note); err != nil {
		return Transaction{}, err
	}
	return GetTransaction(ctx, q, userID, id)
}

// UpdateTransaction changes a transaction.
func UpdateTransaction(ctx context.Context, q db.Querier, userID, id uuid.UUID, f TransactionFields) (Transaction, error) {
	if err := f.validate(); err != nil {
		return Transaction{}, err
	}
	t, err := GetTransaction(ctx, q, userID, id)
	if err != nil {
		return t, err
	}
	if f.OccurredOn != nil {
		t.OccurredOn = *f.OccurredOn
	}
	if f.Kind != nil {
		t.Kind = *f.Kind
	}
	if f.AmountCents != nil {
		t.AmountCents = *f.AmountCents
	}
	if f.CategoryID != nil || f.ClearCategory {
		t.CategoryID = f.CategoryID
	}
	if f.Payee != nil {
		t.Payee = strings.TrimSpace(*f.Payee)
	}
	if f.Note != nil {
		t.Note = *f.Note
	}
	if err := checkCategory(ctx, q, userID, t.CategoryID, t.Kind); err != nil {
		return Transaction{}, err
	}
	if _, err := q.Exec(ctx, `UPDATE budget_transactions SET occurred_on = $3, kind = $4, amount_cents = $5, category_id = $6,
		payee = $7, note = $8, updated_at = now() WHERE id = $1 AND user_id = $2`,
		id, userID, t.OccurredOn, t.Kind, t.AmountCents, t.CategoryID, t.Payee, t.Note); err != nil {
		return Transaction{}, err
	}
	return GetTransaction(ctx, q, userID, id)
}

// DeleteTransaction removes a transaction.
func DeleteTransaction(ctx context.Context, q db.Querier, userID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM budget_transactions WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// Transactions lists a month's transactions, newest first.
func Transactions(ctx context.Context, q db.Querier, userID uuid.UUID, month string, category *uuid.UUID) ([]Transaction, error) {
	start, end, err := MonthRange(month)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT `+transactionColumns+` FROM budget_transactions
		WHERE user_id = $1 AND occurred_on >= $2 AND occurred_on < $3 AND ($4::uuid IS NULL OR category_id = $4)
		ORDER BY occurred_on DESC, id DESC LIMIT 1000`, userID, start, end, category)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Transaction, error) { return scanTransaction(r) })
	if out == nil {
		out = []Transaction{}
	}
	return out, err
}

// Export returns everything for the person's data export.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	currency, err := Currency(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	cats, err := Categories(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT `+transactionColumns+` FROM budget_transactions WHERE user_id = $1 ORDER BY occurred_on, id`, userID)
	if err != nil {
		return nil, err
	}
	txs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Transaction, error) { return scanTransaction(r) })
	return map[string]any{"currency": currency, "categories": cats, "transactions": txs}, err
}
