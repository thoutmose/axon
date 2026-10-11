package budget

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// MonthTotal is one month of income and expenses.
type MonthTotal struct {
	Month        string `json:"month"`
	IncomeCents  int64  `json:"income_cents"`
	ExpenseCents int64  `json:"expense_cents"`
}

// CategoryTotal is what a category added up to over the report.
type CategoryTotal struct {
	CategoryID *uuid.UUID `json:"category_id"` // null: uncategorised
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Cents      int64      `json:"cents"`
}

// Report covers the last Months months up to and including the current one.
type Report struct {
	Currency   string          `json:"currency"`
	From       string          `json:"from"`
	To         string          `json:"to"`
	Months     []MonthTotal    `json:"months"`
	Categories []CategoryTotal `json:"categories"`
	Accounts   []Account       `json:"accounts"`
	// NetWorthCents is assets minus what is owed: the sum of every account balance.
	NetWorthCents int64 `json:"net_worth_cents"`
}

// BuildReport sums income and expenses per month and per category.
func BuildReport(ctx context.Context, q db.Querier, userID uuid.UUID, today time.Time, months int) (Report, error) {
	first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1-months, 0)
	end := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	r := Report{From: first.Format(time.DateOnly), To: end.AddDate(0, 0, -1).Format(time.DateOnly), Months: []MonthTotal{}}
	var err error
	if r.Currency, err = Currency(ctx, q, userID); err != nil {
		return r, err
	}
	rows, err := q.Query(ctx, `SELECT to_char(m, 'YYYY-MM'),
			coalesce(sum(t.amount_cents) FILTER (WHERE t.kind = 'income'), 0)::bigint,
			coalesce(sum(t.amount_cents) FILTER (WHERE t.kind = 'expense'), 0)::bigint
		FROM generate_series($2::date, $3::date - 1, interval '1 month') m
		LEFT JOIN budget_transactions t ON t.user_id = $1 AND t.occurred_on >= m AND t.occurred_on < m + interval '1 month'
		GROUP BY m ORDER BY m`, userID, first, end)
	if err != nil {
		return r, err
	}
	if r.Months, err = pgx.CollectRows(rows, pgx.RowToStructByPos[MonthTotal]); err != nil {
		return r, err
	}
	rows, err = q.Query(ctx, `SELECT t.category_id, coalesce(c.name, 'Uncategorised'), t.kind, sum(t.amount_cents)::bigint
		FROM budget_transactions t LEFT JOIN budget_categories c ON c.id = t.category_id
		WHERE t.user_id = $1 AND t.occurred_on >= $2 AND t.occurred_on < $3 AND t.kind <> 'transfer'
		GROUP BY t.category_id, c.name, t.kind ORDER BY t.kind DESC, 4 DESC`, userID, first, end)
	if err != nil {
		return r, err
	}
	if r.Categories, err = pgx.CollectRows(rows, pgx.RowToStructByPos[CategoryTotal]); err != nil {
		return r, err
	}
	if r.Accounts, err = Accounts(ctx, q, userID); err != nil {
		return r, err
	}
	for _, a := range r.Accounts {
		r.NetWorthCents += a.BalanceCents
	}
	return r, nil
}
