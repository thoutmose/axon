package health

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Water drunk (EP-101). The day's target is the drinks' share of the daily
// need worked out from the profile: EFSA's adequate intakes count water
// from drinks and food together, and drinks give about 80 % of it. Without
// a profile, or when no needs are worked out (under 18, pregnancy), there
// is no target, only what was drunk.

// DrinksShare is the part of the daily water need that comes from drinks.
const DrinksShare = 0.8

// WaterEntry is one drink.
type WaterEntry struct {
	ID        uuid.UUID `json:"id"`
	DrunkOn   string    `json:"drunk_on"`
	ML        int       `json:"ml"`
	CreatedAt time.Time `json:"created_at"`
}

// WaterDay is a day's drinks, its total and target, and the week to it.
type WaterDay struct {
	Date     string       `json:"date"`
	TotalML  int          `json:"total_ml"`
	TargetML *int         `json:"target_ml"`
	Entries  []WaterEntry `json:"entries"`
	Week     []WaterTotal `json:"week"`
}

// WaterTotal is what was drunk on a day.
type WaterTotal struct {
	Date    string `json:"date"`
	TotalML int    `json:"total_ml"`
}

// AddWater logs a drink on a day no later than today.
func AddWater(ctx context.Context, tx pgx.Tx, userID uuid.UUID, day, today time.Time, ml int) (WaterEntry, error) {
	if ml < 1 || ml > 3000 {
		return WaterEntry{}, ValidationError{"A drink is 1 to 3,000 ml."}
	}
	if day.Format(time.DateOnly) > today.Format(time.DateOnly) {
		return WaterEntry{}, ValidationError{"Log a drink for today or a day before it."}
	}
	e := WaterEntry{ID: uuid.Must(uuid.NewV7()), DrunkOn: day.Format(time.DateOnly), ML: ml}
	err := tx.QueryRow(ctx, `INSERT INTO water_entries (id, user_id, drunk_on, ml) VALUES ($1, $2, $3, $4) RETURNING created_at`,
		e.ID, userID, e.DrunkOn, ml).Scan(&e.CreatedAt)
	return e, err
}

// DeleteWater removes one of the person's drinks.
func DeleteWater(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID) error {
	tag, err := tx.Exec(ctx, `DELETE FROM water_entries WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// Water returns a day's drinks with the week ending on it.
func Water(ctx context.Context, q db.Querier, userID uuid.UUID, day time.Time) (WaterDay, error) {
	d := WaterDay{Date: day.Format(time.DateOnly), Entries: []WaterEntry{}}
	rows, err := q.Query(ctx, `SELECT id, drunk_on::text, ml, created_at FROM water_entries
		WHERE user_id = $1 AND drunk_on = $2 ORDER BY created_at`, userID, d.Date)
	if err != nil {
		return d, err
	}
	entries, err := pgx.CollectRows(rows, pgx.RowToStructByPos[WaterEntry])
	if err != nil {
		return d, err
	}
	for _, e := range entries {
		d.TotalML += e.ML
	}
	if entries != nil {
		d.Entries = entries
	}
	rows, err = q.Query(ctx, `SELECT g::date::text, coalesce(sum(w.ml), 0)::int
		FROM generate_series($2::date - 6, $2::date, interval '1 day') g
		LEFT JOIN water_entries w ON w.user_id = $1 AND w.drunk_on = g::date
		GROUP BY g ORDER BY g`, userID, d.Date)
	if err != nil {
		return d, err
	}
	if d.Week, err = pgx.CollectRows(rows, pgx.RowToStructByPos[WaterTotal]); err != nil {
		return d, err
	}
	p, err := GetProfile(ctx, q, userID)
	if err != nil || p == nil {
		return d, err
	}
	if n := Calc(*p, day.Year(), "en"); n.Available {
		target := int(math.Round(n.WaterL*DrinksShare*10) * 100)
		d.TargetML = &target
	}
	return d, nil
}

func exportWater(ctx context.Context, q db.Querier, userID uuid.UUID) ([]WaterEntry, error) {
	rows, err := q.Query(ctx, `SELECT id, drunk_on::text, ml, created_at FROM water_entries WHERE user_id = $1 ORDER BY drunk_on, created_at`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[WaterEntry])
	if out == nil && !errors.Is(err, pgx.ErrNoRows) {
		out = []WaterEntry{}
	}
	return out, err
}
