// Package health logs food and training (EP-18). It is special-category
// data: nothing is stored or read without the health_data consent, and
// withdrawing it deletes everything. The data stays out of the graph, so it
// never shows up in search, links, or agent tools.
package health

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/privacy"
)

// Consents: storing health data, and looking foods up outside Axon.
const (
	Consent       = "health_data"
	LookupConsent = "food_lookups"
)

// ErrNotFound means no such entry for this person.
var ErrNotFound = errors.New("not found")

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

func init() {
	privacy.OnWithdraw(Consent, func(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
		return DeleteAll(ctx, tx, userID)
	})
}

// OnChange, when set, hears which health log changed, in the same
// transaction: "health:food:YYYY-MM-DD", "health:workout:<id>", or
// "health:*" when everything was deleted (the vault mirror then removes the
// health notes it wrote).
var OnChange func(ctx context.Context, q db.Querier, userID uuid.UUID, key string) error

func changed(ctx context.Context, q db.Querier, userID uuid.UUID, key string) error {
	if OnChange == nil {
		return nil
	}
	return OnChange(ctx, q, userID, key)
}

// DeleteAll removes every food entry and workout of the person.
func DeleteAll(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM food_entries WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM workouts WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return changed(ctx, tx, userID, "health:*")
}

// Meals and workout kinds.
var (
	Meals = []string{"breakfast", "lunch", "dinner", "snack"}
	Kinds = []string{"strength", "cardio", "mobility", "sport", "other"}
)

// Nutrients are amounts for what was eaten.
type Nutrients struct {
	Kcal    float64 `json:"kcal"`
	Protein float64 `json:"protein_g"`
	Carbs   float64 `json:"carbs_g"`
	Fat     float64 `json:"fat_g"`
}

// Entry is one thing eaten.
type Entry struct {
	ID      uuid.UUID  `json:"id"`
	EatenOn string     `json:"eaten_on"`
	Meal    string     `json:"meal"`
	Name    string     `json:"name"`
	Grams   *float64   `json:"grams"`
	Kcal    *float64   `json:"kcal"`
	Protein *float64   `json:"protein_g"`
	Carbs   *float64   `json:"carbs_g"`
	Fat     *float64   `json:"fat_g"`
	FoodID  *uuid.UUID `json:"food_id"`
}

// EntryInput adds an entry: either from a food and an amount in grams, or
// with the numbers typed in.
type EntryInput struct {
	EatenOn string     `json:"eaten_on"`
	Meal    string     `json:"meal"`
	Name    string     `json:"name"`
	Grams   *float64   `json:"grams"`
	Kcal    *float64   `json:"kcal"`
	Protein *float64   `json:"protein_g"`
	Carbs   *float64   `json:"carbs_g"`
	Fat     *float64   `json:"fat_g"`
	FoodID  *uuid.UUID `json:"food_id"`
}

const entryColumns = `id, to_char(eaten_on, 'YYYY-MM-DD'), meal, name, grams::float8, kcal::float8, protein_g::float8, carbs_g::float8, fat_g::float8, food_id`

func scanEntry(row pgx.CollectableRow) (Entry, error) {
	var e Entry
	err := row.Scan(&e.ID, &e.EatenOn, &e.Meal, &e.Name, &e.Grams, &e.Kcal, &e.Protein, &e.Carbs, &e.Fat, &e.FoodID)
	return e, err
}

// Day is one day of food: entries by meal and the totals.
type Day struct {
	Date    string    `json:"date"`
	Entries []Entry   `json:"entries"`
	Totals  Nutrients `json:"totals"`
}

// DayTotal is a day's totals, for charts.
type DayTotal struct {
	Date string `json:"date"`
	Nutrients
}

// FoodDay returns the entries of one day.
func FoodDay(ctx context.Context, q db.Querier, userID uuid.UUID, date string) (Day, error) {
	d := Day{Date: date, Entries: []Entry{}}
	rows, err := q.Query(ctx, `SELECT `+entryColumns+` FROM food_entries WHERE user_id = $1 AND eaten_on = $2
		ORDER BY array_position(ARRAY['breakfast','lunch','dinner','snack'], meal), created_at, id`, userID, date)
	if err != nil {
		return d, err
	}
	d.Entries, err = pgx.CollectRows(rows, scanEntry)
	if d.Entries == nil {
		d.Entries = []Entry{}
	}
	for _, e := range d.Entries {
		d.Totals.Kcal += deref(e.Kcal)
		d.Totals.Protein += deref(e.Protein)
		d.Totals.Carbs += deref(e.Carbs)
		d.Totals.Fat += deref(e.Fat)
	}
	d.Totals = round(d.Totals)
	return d, err
}

// FoodWeek returns the totals of the seven days ending on date, oldest first.
func FoodWeek(ctx context.Context, q db.Querier, userID uuid.UUID, end time.Time) ([]DayTotal, error) {
	rows, err := q.Query(ctx, `SELECT to_char(d, 'YYYY-MM-DD'), coalesce(sum(kcal), 0)::float8, coalesce(sum(protein_g), 0)::float8,
		coalesce(sum(carbs_g), 0)::float8, coalesce(sum(fat_g), 0)::float8
		FROM generate_series($2::date - 6, $2::date, interval '1 day') d
		LEFT JOIN food_entries f ON f.user_id = $1 AND f.eaten_on = d::date
		GROUP BY d ORDER BY d`, userID, end.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (DayTotal, error) {
		var t DayTotal
		err := r.Scan(&t.Date, &t.Kcal, &t.Protein, &t.Carbs, &t.Fat)
		t.Nutrients = round(t.Nutrients)
		return t, err
	})
}

// AddEntry records what was eaten.
func AddEntry(ctx context.Context, tx pgx.Tx, userID uuid.UUID, in EntryInput) (Entry, error) {
	if _, err := time.Parse(time.DateOnly, in.EatenOn); err != nil {
		return Entry{}, ValidationError{"The date is YYYY-MM-DD."}
	}
	if !slices.Contains(Meals, in.Meal) {
		return Entry{}, ValidationError{"Meal is breakfast, lunch, dinner, or snack."}
	}
	if in.Grams != nil && (*in.Grams <= 0 || *in.Grams > 10000) {
		return Entry{}, ValidationError{"The amount is up to 10,000 g."}
	}
	if in.FoodID != nil {
		f, err := GetFood(ctx, tx, *in.FoodID)
		if err != nil {
			return Entry{}, ValidationError{"That food does not exist."}
		}
		if in.Grams == nil {
			if f.ServingG == nil {
				return Entry{}, ValidationError{"Say how many grams."}
			}
			in.Grams = f.ServingG
		}
		share := *in.Grams / 100
		scale := func(per100 *float64) *float64 {
			if per100 == nil {
				return nil
			}
			v := math.Round(*per100*share*10) / 10
			return &v
		}
		in.Kcal, in.Protein, in.Carbs, in.Fat = scale(f.Kcal), scale(f.Protein), scale(f.Carbs), scale(f.Fat)
		if strings.TrimSpace(in.Name) == "" {
			in.Name = f.Name
			if f.Brand != "" {
				in.Name += " (" + f.Brand + ")"
			}
		}
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len([]rune(in.Name)) > 300 {
		return Entry{}, ValidationError{"Name what you ate (up to 300 characters)."}
	}
	for _, v := range []*float64{in.Protein, in.Carbs, in.Fat} {
		if v != nil && (*v < 0 || *v > 2000) {
			return Entry{}, ValidationError{"Nutrients are 0 to 2,000 g."}
		}
	}
	if in.Kcal != nil && (*in.Kcal < 0 || *in.Kcal > 20000) {
		return Entry{}, ValidationError{"Energy is 0 to 20,000 kcal."}
	}
	id := uuid.Must(uuid.NewV7())
	rows, err := tx.Query(ctx, `INSERT INTO food_entries (id, user_id, eaten_on, meal, name, grams, kcal, protein_g, carbs_g, fat_g, food_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING `+entryColumns,
		id, userID, in.EatenOn, in.Meal, in.Name, in.Grams, in.Kcal, in.Protein, in.Carbs, in.Fat, in.FoodID)
	if err != nil {
		return Entry{}, err
	}
	e, err := pgx.CollectExactlyOneRow(rows, scanEntry)
	if err != nil {
		return e, err
	}
	return e, changed(ctx, tx, userID, "health:food:"+e.EatenOn)
}

// DeleteEntry removes one of the person's entries.
func DeleteEntry(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID) error {
	var day string
	err := tx.QueryRow(ctx, `DELETE FROM food_entries WHERE id = $1 AND user_id = $2 RETURNING to_char(eaten_on, 'YYYY-MM-DD')`, id, userID).Scan(&day)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return changed(ctx, tx, userID, "health:food:"+day)
}

func deref(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func round(n Nutrients) Nutrients {
	r := func(v float64) float64 { return math.Round(v*10) / 10 }
	return Nutrients{r(n.Kcal), r(n.Protein), r(n.Carbs), r(n.Fat)}
}
