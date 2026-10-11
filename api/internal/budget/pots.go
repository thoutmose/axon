package budget

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Pot is a savings pot (the "tirelire"): money put aside for something to
// buy later.
type Pot struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	TargetCents int64     `json:"target_cents"`
	TargetDate  *string   `json:"target_date"`
	Note        string    `json:"note"`
	Archived    bool      `json:"archived"`
	SavedCents  int64     `json:"saved_cents"`
	// MonthlyCents is what to put in each month, from this month on, to
	// reach the target by its date; null without a date or once reached.
	MonthlyCents *int64    `json:"monthly_cents"`
	Moves        []PotMove `json:"moves"`
	CreatedAt    time.Time `json:"created_at"`
}

// PotMove is money put in (positive) or taken out (negative).
type PotMove struct {
	ID          uuid.UUID `json:"id"`
	OccurredOn  string    `json:"occurred_on"`
	AmountCents int64     `json:"amount_cents"`
	Note        string    `json:"note"`
}

// PotFields are the changeable fields; nil leaves a field as it is.
type PotFields struct {
	Name        *string `json:"name"`
	TargetCents *int64  `json:"target_cents"`
	TargetDate  *string `json:"target_date"` // "" clears it
	Note        *string `json:"note"`
	Archived    *bool   `json:"archived"`
}

// monthsLeft counts the months from today's month to the target's, both included.
func monthsLeft(today time.Time, target string) int {
	t, err := time.Parse(time.DateOnly, target)
	if err != nil {
		return 0
	}
	return (t.Year()-today.Year())*12 + int(t.Month()-today.Month()) + 1
}

// Monthly is what to save each month to go from saved to target by the
// target date, rounded up to the cent; with no month left it is the rest.
func Monthly(saved, target int64, today time.Time, targetDate string) *int64 {
	rest := target - saved
	if rest <= 0 || targetDate == "" {
		return nil
	}
	n := int64(max(monthsLeft(today, targetDate), 1))
	m := (rest + n - 1) / n
	return &m
}

// Pots returns the person's pots with their moves, newest first.
func Pots(ctx context.Context, q db.Querier, userID uuid.UUID, today time.Time) ([]Pot, error) {
	rows, err := q.Query(ctx, `SELECT p.id, p.name, p.target_cents, to_char(p.target_date, 'YYYY-MM-DD'), p.note, p.archived, p.created_at,
		coalesce((SELECT json_agg(json_build_object('id', m.id, 'occurred_on', to_char(m.occurred_on, 'YYYY-MM-DD'), 'amount_cents', m.amount_cents, 'note', m.note)
			ORDER BY m.occurred_on DESC, m.created_at DESC) FROM budget_pot_moves m WHERE m.pot_id = p.id), '[]')
		FROM budget_pots p WHERE p.user_id = $1 ORDER BY p.archived, p.created_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Pot, error) {
		var p Pot
		err := r.Scan(&p.ID, &p.Name, &p.TargetCents, &p.TargetDate, &p.Note, &p.Archived, &p.CreatedAt, &p.Moves)
		for _, m := range p.Moves {
			p.SavedCents += m.AmountCents
		}
		date := ""
		if p.TargetDate != nil {
			date = *p.TargetDate
		}
		p.MonthlyCents = Monthly(p.SavedCents, p.TargetCents, today, date)
		return p, err
	})
	if out == nil {
		out = []Pot{}
	}
	return out, err
}

// GetPot returns one of the person's pots.
func GetPot(ctx context.Context, q db.Querier, userID, id uuid.UUID, today time.Time) (Pot, error) {
	all, err := Pots(ctx, q, userID, today)
	if err != nil {
		return Pot{}, err
	}
	for _, p := range all {
		if p.ID == id {
			return p, nil
		}
	}
	return Pot{}, ErrNotFound
}

func (f PotFields) validate(creating bool) error {
	switch {
	case creating && (f.Name == nil || f.TargetCents == nil):
		return ValidationError{"Name the pot and give it a target."}
	case f.Name != nil && (strings.TrimSpace(*f.Name) == "" || len([]rune(*f.Name)) > 100):
		return ValidationError{"The name is 1 to 100 characters."}
	case f.TargetCents != nil && (*f.TargetCents <= 0 || *f.TargetCents > 100000000000):
		return ValidationError{"The target is more than zero."}
	case f.TargetDate != nil && *f.TargetDate != "" && !validDay(*f.TargetDate):
		return ValidationError{"The date is YYYY-MM-DD."}
	case f.Note != nil && len([]rune(*f.Note)) > 2000:
		return ValidationError{"The note is up to 2,000 characters."}
	}
	return nil
}

func validDay(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

func nullDate(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// CreatePot adds a pot.
func CreatePot(ctx context.Context, q db.Querier, userID uuid.UUID, f PotFields, today time.Time) (Pot, error) {
	if err := f.validate(true); err != nil {
		return Pot{}, err
	}
	note := ""
	if f.Note != nil {
		note = *f.Note
	}
	id := uuid.Must(uuid.NewV7())
	if _, err := q.Exec(ctx, `INSERT INTO budget_pots (id, user_id, name, target_cents, target_date, note) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, userID, strings.TrimSpace(*f.Name), *f.TargetCents, nullDate(f.TargetDate), note); err != nil {
		return Pot{}, err
	}
	return GetPot(ctx, q, userID, id, today)
}

// UpdatePot changes the given fields.
func UpdatePot(ctx context.Context, q db.Querier, userID, id uuid.UUID, f PotFields, today time.Time) (Pot, error) {
	if err := f.validate(false); err != nil {
		return Pot{}, err
	}
	var name *string
	if f.Name != nil {
		n := strings.TrimSpace(*f.Name)
		name = &n
	}
	tag, err := q.Exec(ctx, `UPDATE budget_pots SET name = coalesce($3, name), target_cents = coalesce($4, target_cents),
		target_date = CASE WHEN $5::boolean THEN $6::date ELSE target_date END, note = coalesce($7, note), archived = coalesce($8, archived),
		updated_at = now() WHERE id = $1 AND user_id = $2`,
		id, userID, name, f.TargetCents, f.TargetDate != nil, nullDate(f.TargetDate), f.Note, f.Archived)
	if err != nil {
		return Pot{}, err
	}
	if tag.RowsAffected() == 0 {
		return Pot{}, ErrNotFound
	}
	return GetPot(ctx, q, userID, id, today)
}

// DeletePot removes a pot and its history.
func DeletePot(ctx context.Context, q db.Querier, userID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM budget_pots WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		err = ErrNotFound
	}
	return err
}

// MovePot puts money in (positive) or takes it out (negative). A pot never
// goes below zero.
func MovePot(ctx context.Context, tx pgx.Tx, userID, potID uuid.UUID, m PotMove, today time.Time) (Pot, error) {
	if m.OccurredOn == "" {
		m.OccurredOn = today.Format(time.DateOnly)
	}
	switch {
	case !validDay(m.OccurredOn):
		return Pot{}, ValidationError{"The date is YYYY-MM-DD."}
	case m.AmountCents == 0 || m.AmountCents > 100000000000 || m.AmountCents < -100000000000:
		return Pot{}, ValidationError{"Put in or take out an amount other than zero."}
	case len([]rune(m.Note)) > 500:
		return Pot{}, ValidationError{"The note is up to 500 characters."}
	}
	var saved int64
	err := tx.QueryRow(ctx, `SELECT coalesce((SELECT sum(amount_cents) FROM budget_pot_moves WHERE pot_id = p.id), 0)::bigint
		FROM budget_pots p WHERE p.id = $1 AND p.user_id = $2 FOR UPDATE`, potID, userID).Scan(&saved)
	if errors.Is(err, pgx.ErrNoRows) {
		return Pot{}, ErrNotFound
	}
	if err != nil {
		return Pot{}, err
	}
	if saved+m.AmountCents < 0 {
		return Pot{}, ValidationError{"There is not that much in the pot."}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO budget_pot_moves (id, pot_id, occurred_on, amount_cents, note) VALUES ($1, $2, $3, $4, $5)`,
		uuid.Must(uuid.NewV7()), potID, m.OccurredOn, m.AmountCents, m.Note); err != nil {
		return Pot{}, err
	}
	return GetPot(ctx, tx, userID, potID, today)
}

// DeletePotMove removes one move, unless that would leave the pot below zero.
func DeletePotMove(ctx context.Context, tx pgx.Tx, userID, potID, moveID uuid.UUID, today time.Time) (Pot, error) {
	var amount, saved int64
	err := tx.QueryRow(ctx, `SELECT m.amount_cents, (SELECT sum(amount_cents) FROM budget_pot_moves WHERE pot_id = p.id)::bigint
		FROM budget_pot_moves m JOIN budget_pots p ON p.id = m.pot_id WHERE m.id = $1 AND p.id = $2 AND p.user_id = $3`, moveID, potID, userID).Scan(&amount, &saved)
	if errors.Is(err, pgx.ErrNoRows) {
		return Pot{}, ErrNotFound
	}
	if err != nil {
		return Pot{}, err
	}
	if saved-amount < 0 {
		return Pot{}, ValidationError{"Removing this would leave the pot below zero."}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM budget_pot_moves WHERE id = $1`, moveID); err != nil {
		return Pot{}, err
	}
	return GetPot(ctx, tx, userID, potID, today)
}
