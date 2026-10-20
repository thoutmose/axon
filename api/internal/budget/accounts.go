package budget

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Account is where money is: a bank account, cash, a card, a loan.
type Account struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"` // asset or liability
	Role         string    `json:"role"`
	OpeningCents int64     `json:"opening_cents"`
	BalanceCents int64     `json:"balance_cents"`
	Archived     bool      `json:"archived"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

var roles = []string{"checking", "savings", "cash", "credit_card", "loan", "investment", "other"}

// Movement is what Balances needs from a transaction.
type Movement struct {
	Kind        string
	AmountCents int64
	From, To    *uuid.UUID // From: AccountID (income uses it as the destination); To: ToAccountID
}

// Balances applies movements to opening balances: income adds to its
// account, an expense subtracts, a transfer moves money between two
// accounts. Transfers never change the total, so the total is always
// openings + income − expenses on accounts (property-tested).
func Balances(opening map[uuid.UUID]int64, moves []Movement) map[uuid.UUID]int64 {
	out := make(map[uuid.UUID]int64, len(opening))
	for id, c := range opening {
		out[id] = c
	}
	for _, m := range moves {
		if m.From == nil {
			continue
		}
		switch m.Kind {
		case "income":
			out[*m.From] += m.AmountCents
		case "expense":
			out[*m.From] -= m.AmountCents
		case "transfer":
			if m.To != nil {
				out[*m.From] -= m.AmountCents
				out[*m.To] += m.AmountCents
			}
		}
	}
	return out
}

// balanceSQLOver is Balances in SQL, for one account a, over the
// transactions that also match `and` (empty for all of them).
func balanceSQLOver(and string) string {
	return `a.opening_cents + coalesce((SELECT sum(CASE
		WHEN t.kind = 'income' AND t.account_id = a.id THEN t.amount_cents
		WHEN t.kind = 'expense' AND t.account_id = a.id THEN -t.amount_cents
		WHEN t.kind = 'transfer' AND t.account_id = a.id THEN -t.amount_cents
		WHEN t.kind = 'transfer' AND t.to_account_id = a.id THEN t.amount_cents
		ELSE 0 END) FROM budget_transactions t WHERE (t.account_id = a.id OR t.to_account_id = a.id)` + and + `), 0)::bigint`
}

var (
	balanceSQL     = balanceSQLOver("")
	accountColumns = `a.id, a.name, a.type, a.role, a.opening_cents, ` + balanceSQL + `, a.archived, a.created_at, a.updated_at`
	// accountColumnsBefore counts only the transactions dated before $2.
	accountColumnsBefore = `a.id, a.name, a.type, a.role, a.opening_cents, ` + balanceSQLOver(" AND t.occurred_on < $2") + `, a.archived, a.created_at, a.updated_at`
)

func scanAccount(r pgx.Row) (Account, error) {
	var a Account
	err := r.Scan(&a.ID, &a.Name, &a.Type, &a.Role, &a.OpeningCents, &a.BalanceCents, &a.Archived, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

// GetAccount returns one of the user's accounts with its balance.
func GetAccount(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Account, error) {
	a, err := scanAccount(q.QueryRow(ctx, `SELECT `+accountColumns+` FROM budget_accounts a WHERE a.id = $1 AND a.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

// Accounts lists the user's accounts with balances.
func Accounts(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Account, error) {
	rows, err := q.Query(ctx, `SELECT `+accountColumns+` FROM budget_accounts a WHERE a.user_id = $1
		ORDER BY a.archived, a.type, lower(a.name) LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Account, error) { return scanAccount(r) })
	if out == nil {
		out = []Account{}
	}
	return out, err
}

// AccountsBefore lists the user's accounts with their balance from the
// transactions dated before `day`: at the end of a month, pass the first
// day of the next one.
func AccountsBefore(ctx context.Context, q db.Querier, userID uuid.UUID, day time.Time) ([]Account, error) {
	rows, err := q.Query(ctx, `SELECT `+accountColumnsBefore+` FROM budget_accounts a WHERE a.user_id = $1
		ORDER BY a.archived, a.type, lower(a.name) LIMIT 200`, userID, day)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Account, error) { return scanAccount(r) })
	if out == nil {
		out = []Account{}
	}
	return out, err
}

// AccountFields are the editable parts; nil means "not provided".
type AccountFields struct {
	Name, Type, Role *string
	OpeningCents     *int64
	Archived         *bool
}

func (f *AccountFields) validate() error {
	if f.Name != nil {
		*f.Name = strings.TrimSpace(*f.Name)
		if *f.Name == "" || len([]rune(*f.Name)) > 100 {
			return ValidationError{"An account needs a name of 1 to 100 characters."}
		}
	}
	if f.Type != nil && *f.Type != "asset" && *f.Type != "liability" {
		return ValidationError{"An account is an asset (money you have) or a liability (money you owe)."}
	}
	if f.Role != nil && !slices.Contains(roles, *f.Role) {
		return ValidationError{"Role is checking, savings, cash, credit_card, loan, investment, or other."}
	}
	if f.OpeningCents != nil && (*f.OpeningCents > MaxCents || *f.OpeningCents < -MaxCents) {
		return ValidationError{"An opening balance is at most 1,000,000,000 either way."}
	}
	return nil
}

// CreateAccount adds an account.
func CreateAccount(ctx context.Context, q db.Querier, userID uuid.UUID, f AccountFields) (Account, error) {
	if f.Name == nil || f.Type == nil {
		return Account{}, ValidationError{"An account needs a name and a type (asset or liability)."}
	}
	if err := f.validate(); err != nil {
		return Account{}, err
	}
	role := "checking"
	if *f.Type == "liability" {
		role = "credit_card"
	}
	if f.Role != nil {
		role = *f.Role
	}
	var opening int64
	if f.OpeningCents != nil {
		opening = *f.OpeningCents
	}
	id := uuid.Must(uuid.NewV7())
	_, err := q.Exec(ctx, `INSERT INTO budget_accounts (id, user_id, name, type, role, opening_cents) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, userID, *f.Name, *f.Type, role, opening)
	if uniqueViolation(err) {
		return Account{}, ValidationError{"You already have an account with that name."}
	}
	if err != nil {
		return Account{}, err
	}
	return GetAccount(ctx, q, userID, id)
}

// UpdateAccount changes an account. Its type is fixed once created.
func UpdateAccount(ctx context.Context, q db.Querier, userID, id uuid.UUID, f AccountFields) (Account, error) {
	if f.Type != nil {
		return Account{}, ValidationError{"An account's type cannot change; create a new one."}
	}
	if err := f.validate(); err != nil {
		return Account{}, err
	}
	a, err := GetAccount(ctx, q, userID, id)
	if err != nil {
		return a, err
	}
	if f.Name != nil {
		a.Name = *f.Name
	}
	if f.Role != nil {
		a.Role = *f.Role
	}
	if f.OpeningCents != nil {
		a.OpeningCents = *f.OpeningCents
	}
	if f.Archived != nil {
		a.Archived = *f.Archived
	}
	_, err = q.Exec(ctx, `UPDATE budget_accounts SET name = $3, role = $4, opening_cents = $5, archived = $6, updated_at = now()
		WHERE id = $1 AND user_id = $2`, id, userID, a.Name, a.Role, a.OpeningCents, a.Archived)
	if uniqueViolation(err) {
		return Account{}, ValidationError{"You already have an account with that name."}
	}
	if err != nil {
		return Account{}, err
	}
	return GetAccount(ctx, q, userID, id)
}

// DeleteAccount removes an account that no transaction uses; otherwise
// archive it, so balances and history stay true.
func DeleteAccount(ctx context.Context, q db.Querier, userID, id uuid.UUID) error {
	var used bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM budget_transactions WHERE (account_id = $1 OR to_account_id = $1) AND user_id = $2)
		OR EXISTS (SELECT 1 FROM budget_recurring WHERE (account_id = $1 OR to_account_id = $1) AND user_id = $2)`, id, userID).Scan(&used); err != nil {
		return err
	}
	if used {
		return ValidationError{"This account has transactions. Archive it instead, so balances and history stay true."}
	}
	tag, err := q.Exec(ctx, `DELETE FROM budget_accounts WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
