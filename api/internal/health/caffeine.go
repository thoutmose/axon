package health

import (
	"context"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Coffee and tea (EP-117). Caffeine leaves the body by half every half-life
// (about 5 hours on average; 1.5 to 9.5 in adults, longer in pregnancy).
// EFSA (2015) finds no safety concern up to 400 mg a day for adults, 200 mg
// when pregnant. Gardiner et al. (2023, Sleep Medicine Reviews) found that a
// coffee (107 mg) should be drunk at least 8.8 hours before bed to keep
// sleep intact: what it leaves then (about 32 mg) is the threshold here.

// Sip is caffeine taken at a moment.
type Sip struct {
	At time.Time
	MG int
}

// Caffeine thresholds.
const (
	DailyLimitMG    = 400 // EFSA, healthy adults
	PregnantLimitMG = 200 // EFSA in pregnancy; kept while breastfeeding too, the profile asks both at once
	CoffeeMG        = 107 // the coffee of Gardiner et al. (2023)
	HalfLifeHours   = 5.0 // the average adult
	coffeeGapHours  = 8.8 // how long before bed that coffee keeps sleep intact
)

// BedtimeMG is what that coffee leaves at bedtime, 8.8 hours later, at the
// average half-life: about 32 mg.
var BedtimeMG = CoffeeMG * math.Exp2(-coffeeGapHours/HalfLifeHours)

// Remaining is the caffeine left in the body at a moment, in mg.
func Remaining(sips []Sip, at time.Time, halfLife float64) float64 {
	var mg float64
	for _, s := range sips {
		if !s.At.After(at) {
			mg += float64(s.MG) * math.Exp2(-at.Sub(s.At).Hours()/halfLife)
		}
	}
	return mg
}

// Cutoff is the latest moment a coffee still leaves no more than BedtimeMG
// at bedtime, given what was already drunk; false when even a coffee now is
// too much.
func Cutoff(sips []Sip, bedtime time.Time, halfLife float64) (time.Time, bool) {
	room := BedtimeMG - Remaining(sips, bedtime, halfLife)
	if room <= 0 {
		return time.Time{}, false
	}
	gap := halfLife * math.Log2(CoffeeMG/room) // hours before bed
	return bedtime.Add(-time.Duration(gap * float64(time.Hour))), true
}

// BedtimeOn is the night's bedtime for a day: a time after midnight and
// before 6:00 belongs to the night after that day.
func BedtimeOn(day time.Time, hhmm string, loc *time.Location) time.Time {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		t, _ = time.Parse("15:04", "23:00")
	}
	b := time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, loc)
	if t.Hour() < 6 {
		b = b.AddDate(0, 0, 1)
	}
	return b
}

// Drink is a kind of drink and its usual caffeine, per serving (USDA
// FoodData Central and EFSA, rounded): a person can give their own.
type Drink struct {
	Kind string `json:"kind"`
	MG   int    `json:"mg"`
	ML   int    `json:"ml"`
}

// Drinks are the kinds offered, in order.
var Drinks = []Drink{
	{"espresso", 63, 30}, {"coffee", 95, 240}, {"instant", 62, 240}, {"tea", 47, 240}, {"green_tea", 28, 240},
	{"matcha", 70, 240}, {"cola", 32, 330}, {"energy", 80, 250}, {"decaf", 3, 240}, {"other", 0, 0},
}

// HalfLives are the half-lives, in hours, by metabolism.
var HalfLives = map[string]float64{"fast": 3, "average": HalfLifeHours, "slow": 7}

// CaffeineEntry is one drink taken.
type CaffeineEntry struct {
	ID   uuid.UUID `json:"id"`
	At   time.Time `json:"at"`
	Kind string    `json:"kind"`
	MG   int       `json:"mg"`
}

// CaffeineTotal is a day's caffeine.
type CaffeineTotal struct {
	Date    string `json:"date"`
	TotalMG int    `json:"total_mg"`
}

// CaffeineDay is a day's caffeine against the limit and the night after it.
type CaffeineDay struct {
	Date        string          `json:"date"`
	TotalMG     int             `json:"total_mg"`
	LimitMG     int             `json:"limit_mg"` // 400, or 200 when pregnant or breastfeeding
	Entries     []CaffeineEntry `json:"entries"`
	NowMG       float64         `json:"now_mg"`        // in the body now
	Bedtime     time.Time       `json:"bedtime"`       // the night after the day
	AtBedtimeMG float64         `json:"at_bedtime_mg"` // left then
	BedtimeMG   float64         `json:"bedtime_mg"`    // the threshold
	Cutoff      *time.Time      `json:"cutoff"`        // the last coffee that keeps sleep intact; null when already past
	HalfLife    float64         `json:"half_life_hours"`
	Week        []CaffeineTotal `json:"week"`
	Drinks      []Drink         `json:"drinks"`
}

// CaffeineInput is a drink to log: a kind, its caffeine if not the usual,
// and when (now when empty).
type CaffeineInput struct {
	Kind string     `json:"kind"`
	MG   *int       `json:"mg"`
	At   *time.Time `json:"at"`
}

// AddCaffeine logs a drink, now or in the last 30 days.
func AddCaffeine(ctx context.Context, tx pgx.Tx, userID uuid.UUID, in CaffeineInput, now time.Time) (CaffeineEntry, error) {
	i := slices.IndexFunc(Drinks, func(d Drink) bool { return d.Kind == in.Kind })
	if i < 0 {
		return CaffeineEntry{}, ValidationError{"Choose a drink from the list, or other."}
	}
	e := CaffeineEntry{ID: uuid.Must(uuid.NewV7()), Kind: in.Kind, MG: Drinks[i].MG, At: now}
	if in.MG != nil {
		e.MG = *in.MG
	}
	if e.MG < 1 || e.MG > 1000 {
		return CaffeineEntry{}, ValidationError{"A drink has 1 to 1,000 mg of caffeine."}
	}
	if in.At != nil {
		if in.At.After(now.Add(5*time.Minute)) || in.At.Before(now.AddDate(0, 0, -30)) {
			return CaffeineEntry{}, ValidationError{"Log a drink taken in the last 30 days."}
		}
		e.At = *in.At
	}
	_, err := tx.Exec(ctx, `INSERT INTO caffeine_entries (id, user_id, drunk_at, kind, mg) VALUES ($1, $2, $3, $4, $5)`,
		e.ID, userID, e.At, e.Kind, e.MG)
	return e, err
}

// DeleteCaffeine removes one of the person's drinks.
func DeleteCaffeine(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID) error {
	tag, err := tx.Exec(ctx, `DELETE FROM caffeine_entries WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// CaffeineOn returns a day's caffeine, in the person's time zone, with what
// is left now and at the bedtime after it, the cut-off, and the week to it.
func CaffeineOn(ctx context.Context, q db.Querier, userID uuid.UUID, day time.Time, loc *time.Location, now time.Time) (CaffeineDay, error) {
	var bedtime, metabolism string
	var pregnant bool
	err := q.QueryRow(ctx, `SELECT coalesce((SELECT doc->'caffeine'->>'bedtime' FROM user_settings WHERE user_id = $1), ''),
		coalesce((SELECT doc->'caffeine'->>'metabolism' FROM user_settings WHERE user_id = $1), ''),
		coalesce((SELECT pregnant FROM health_profiles WHERE user_id = $1), false)`, userID).Scan(&bedtime, &metabolism, &pregnant)
	if err != nil {
		return CaffeineDay{}, err
	}
	half, ok := HalfLives[metabolism]
	if !ok {
		half = HalfLifeHours
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	d := CaffeineDay{Date: day.Format(time.DateOnly), LimitMG: DailyLimitMG, Entries: []CaffeineEntry{}, BedtimeMG: BedtimeMG,
		HalfLife: half, Bedtime: BedtimeOn(day, bedtime, loc), Drinks: Drinks}
	if pregnant {
		d.LimitMG = PregnantLimitMG
	}
	// The day's drinks, and the two days before for what is still in the body.
	rows, err := q.Query(ctx, `SELECT id, drunk_at, kind, mg FROM caffeine_entries
		WHERE user_id = $1 AND drunk_at >= $2 AND drunk_at < $3 ORDER BY drunk_at`, userID, start.AddDate(0, 0, -2), d.Bedtime)
	if err != nil {
		return d, err
	}
	all, err := pgx.CollectRows(rows, pgx.RowToStructByPos[CaffeineEntry])
	if err != nil {
		return d, err
	}
	var sips []Sip
	for _, e := range all {
		sips = append(sips, Sip{e.At, e.MG})
		if !e.At.Before(start) && e.At.Before(start.AddDate(0, 0, 1)) {
			d.Entries = append(d.Entries, e)
			d.TotalMG += e.MG
		}
	}
	d.NowMG = math.Round(Remaining(sips, now, half))
	d.AtBedtimeMG = math.Round(Remaining(sips, d.Bedtime, half))
	if cut, ok := Cutoff(sips, d.Bedtime, half); ok {
		d.Cutoff = &cut
	}
	rows, err = q.Query(ctx, `SELECT g::date::text, coalesce(sum(c.mg), 0)::int FROM generate_series($2::date - 6, $2::date, '1 day') g
		LEFT JOIN caffeine_entries c ON c.user_id = $1 AND (c.drunk_at AT TIME ZONE $3)::date = g::date
		GROUP BY g ORDER BY g`, userID, d.Date, loc.String())
	if err != nil {
		return d, err
	}
	d.Week, err = pgx.CollectRows(rows, pgx.RowToStructByPos[CaffeineTotal])
	return d, err
}
