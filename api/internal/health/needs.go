package health

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/l10n"
)

// Daily needs (EP-88): the energy, protein, fat, carbohydrates, fibre, and
// water a healthy adult needs to maintain themselves, worked out from
// published references and shown with how. They are reference intakes, never
// a diet: there is no weight-loss target, no numbers before 18 or during
// pregnancy or breastfeeding, and every view says to check with a dietitian
// (ADR 0028).

// Profile is what the needs are worked out from: special-category data,
// kept only with the health_data consent.
type Profile struct {
	BirthYear int       `json:"birth_year"`
	Sex       string    `json:"sex"` // female, male, or unspecified
	HeightCm  float64   `json:"height_cm"`
	WeightKg  float64   `json:"weight_kg"`
	Activity  string    `json:"activity"`
	Goal      string    `json:"goal"` // maintain, lose, or gain (EP-191); empty means maintain
	Pregnant  bool      `json:"pregnant_or_breastfeeding"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Goals are what the needs aim at (ADR 0039): keeping one's weight, losing
// it gently (15 % under maintenance, never under the basal rate or
// minKcal), or gaining slowly (10 % over). Both move protein up to keep muscle.
var Goals = []string{"maintain", "lose", "gain"}

const (
	minKcal   = 1200.0 // the floor the agents' safety guard also keeps
	loseShare = 0.85
	gainShare = 1.10
	kcalPerKg = 7700.0 // of body fat, for the pace in words
)

// Activity levels: the factor on the basal rate, and protein per kg of body
// weight (EFSA's 0.83 g/kg for adults, rising with training as the ISSN
// position stand advises).
var activities = map[string]struct{ factor, proteinPerKg float64 }{
	"sedentary":   {1.2, 0.83},
	"light":       {1.375, 1.0},
	"moderate":    {1.55, 1.2},
	"active":      {1.725, 1.4},
	"very_active": {1.9, 1.6},
}

// Activities lists the levels from least to most active.
var Activities = []string{"sedentary", "light", "moderate", "active", "very_active"}

// Sexes are the choices for the basal-rate equation.
var Sexes = []string{"female", "male", "unspecified"}

// Source is a reference the figures come from.
type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

var sources = []Source{
	{"Mifflin et al. (1990), A new predictive equation for resting energy expenditure in healthy individuals", "https://doi.org/10.1093/ajcn/51.2.241"},
	{"EFSA, Dietary Reference Values for nutrients (2017)", "https://www.efsa.europa.eu/en/topics/topic/dietary-reference-values"},
	{"Jäger et al. (2017), ISSN position stand: protein and exercise", "https://doi.org/10.1186/s12970-017-0177-8"},
	{"Deutz et al. (2014), Protein intake and exercise for optimal muscle function with aging (ESPEN)", "https://doi.org/10.1016/j.clnu.2014.04.007"},
}

// Needs are a day's reference intakes, with how each was worked out.
type Needs struct {
	Available    bool     `json:"available"`
	Reason       *string  `json:"reason"` // under_18 or pregnant: no numbers, ask a professional
	Age          int      `json:"age"`
	BMR          int      `json:"bmr_kcal"`
	Kcal         int      `json:"kcal"`
	ProteinG     int      `json:"protein_g"`
	FatG         int      `json:"fat_g"`
	CarbsG       int      `json:"carbs_g"`
	FiberG       int      `json:"fiber_g"`
	WaterL       float64  `json:"water_l"`
	Explanation  []string `json:"explanation"`
	Sources      []Source `json:"sources"`
	ProteinPerKg float64  `json:"protein_g_per_kg"`
	// Goal is what the figures aim at, and MaintenanceKcal what would keep
	// the weight, which Kcal equals for the maintain goal (EP-191).
	Goal            string `json:"goal"`
	MaintenanceKcal int    `json:"maintenance_kcal"`
}

func (p Profile) validate(year int) error {
	age := year - p.BirthYear
	switch {
	case age < 1 || age > 120:
		return ValidationError{"Give a birth year that makes you 1 to 120 years old."}
	case !slices.Contains(Sexes, p.Sex):
		return ValidationError{"Sex is female, male, or unspecified."}
	case p.HeightCm < 100 || p.HeightCm > 250:
		return ValidationError{"Height is 100 to 250 cm."}
	case p.WeightKg < 25 || p.WeightKg > 350:
		return ValidationError{"Weight is 25 to 350 kg."}
	case activities[p.Activity] == struct{ factor, proteinPerKg float64 }{}:
		return ValidationError{"Activity is sedentary, light, moderate, active, or very_active."}
	case p.Goal != "" && !slices.Contains(Goals, p.Goal):
		return ValidationError{"Goal is maintain, lose, or gain."}
	}
	return nil
}

// Calc works out the needs for the profile in the given year.
func Calc(p Profile, year int, lang string) Needs {
	n := Needs{Age: year - p.BirthYear, Explanation: []string{}, Sources: sources, Goal: p.Goal}
	if n.Goal == "" {
		n.Goal = "maintain"
	}
	reason := ""
	switch {
	case n.Age < 18:
		reason = "under_18"
	case p.Pregnant:
		reason = "pregnant"
	}
	if reason != "" {
		n.Reason = &reason
		return n
	}
	n.Available = true
	// Mifflin–St Jeor; without a sex, the middle of the two constants.
	constant, sexTerm := -78.0, l10n.Tr(lang, "− 78, the middle of the female and male constants")
	switch p.Sex {
	case "female":
		constant, sexTerm = -161, l10n.Tr(lang, "− 161 for women")
	case "male":
		constant, sexTerm = 5, l10n.Tr(lang, "+ 5 for men")
	}
	bmr := 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(n.Age) + constant
	a := activities[p.Activity]
	n.BMR = int(math.Round(bmr))
	n.MaintenanceKcal = int(math.Round(bmr*a.factor/10) * 10)
	n.Kcal = n.MaintenanceKcal
	perKg := a.proteinPerKg
	if n.Age >= 65 {
		perKg = math.Max(perKg, 1.0)
	}
	num := func(f float64, prec int) string { return l10n.Num(lang, f, prec) }
	energy := l10n.Tr(lang, "Energy: %d kcal, the basal rate × %s for your activity.", n.Kcal, num(a.factor, 3))
	protein := func() string {
		return l10n.Tr(lang, "Protein: %d g, %s g per kg of body weight.", n.ProteinG, num(perKg, 2))
	}
	// A goal moves the energy by a bounded share and protein up (ADR 0039).
	round10 := func(f float64) int { return int(math.Round(f/10) * 10) }
	perWeek := func(kcal int) string { return num(math.Abs(float64(kcal))*7/kcalPerKg, 1) }
	switch n.Goal {
	case "lose":
		floor := int(math.Max(math.Round(bmr), minKcal))
		n.Kcal = round10(float64(n.MaintenanceKcal) * loseShare)
		if n.Kcal < floor {
			n.Kcal = floor
			energy = l10n.Tr(lang, "Energy: %d kcal: the 15 %% under the %d kcal that keep your weight stops here, at your basal rate or 1,200 kcal, the floor Axon never goes under.", n.Kcal, n.MaintenanceKcal)
		} else {
			energy = l10n.Tr(lang, "Energy: %d kcal, 15 %% under the %d kcal that keep your weight: a gentle loss of about %s kg a week, never under your basal rate or 1,200 kcal.", n.Kcal, n.MaintenanceKcal, perWeek(n.MaintenanceKcal-n.Kcal))
		}
		perKg = math.Min(2.0, math.Max(1.6, perKg+0.4))
		protein = func() string {
			return l10n.Tr(lang, "Protein: %d g, %s g per kg of body weight: more while your weight changes, to keep and build muscle.", n.ProteinG, num(perKg, 2))
		}
	case "gain":
		n.Kcal = round10(float64(n.MaintenanceKcal) * gainShare)
		energy = l10n.Tr(lang, "Energy: %d kcal, 10 %% over the %d kcal that keep your weight, to gain slowly (about %s kg a week).", n.Kcal, n.MaintenanceKcal, perWeek(n.Kcal-n.MaintenanceKcal))
		perKg = math.Min(2.0, math.Max(1.6, perKg+0.4))
		protein = func() string {
			return l10n.Tr(lang, "Protein: %d g, %s g per kg of body weight: more while your weight changes, to keep and build muscle.", n.ProteinG, num(perKg, 2))
		}
	}
	n.ProteinPerKg = perKg
	n.ProteinG = int(math.Round(perKg * p.WeightKg))
	n.FatG = int(math.Round(float64(n.Kcal) * 0.30 / 9))
	n.CarbsG = max(0, int(math.Round(float64(n.Kcal-n.ProteinG*4-n.FatG*9)/4)))
	n.FiberG = 25
	water := map[string]float64{"female": 2.0, "male": 2.5}[p.Sex]
	if water == 0 {
		water = 2.25
	}
	n.WaterL = math.Round(water*10) / 10
	carbShare := int(math.Round(float64(n.CarbsG*4) * 100 / float64(n.Kcal)))
	n.Explanation = []string{
		l10n.Tr(lang, "Basal metabolic rate: %d kcal (Mifflin–St Jeor: 10 × weight + 6.25 × height − 5 × age %s).", n.BMR, sexTerm),
		energy,
		protein(),
		l10n.Tr(lang, "Fat: %d g, 30 %% of energy (reference intake 20–35 %%).", n.FatG),
		l10n.Tr(lang, "Carbohydrates: %d g, the rest of the energy (%d %%; reference intake 45–60 %%).", n.CarbsG, carbShare),
		l10n.Tr(lang, "Fibre: 25 g (adequate intake for adults)."),
		l10n.Tr(lang, "Water: %s L a day from drinks and food (adequate intake).", num(n.WaterL, 1)),
	}
	return n
}

// GetProfile returns the person's profile, or nil.
func GetProfile(ctx context.Context, q db.Querier, userID uuid.UUID) (*Profile, error) {
	var p Profile
	err := q.QueryRow(ctx, `SELECT birth_year, sex, height_cm, weight_kg, activity, goal, pregnant, updated_at
		FROM health_profiles WHERE user_id = $1`, userID).
		Scan(&p.BirthYear, &p.Sex, &p.HeightCm, &p.WeightKg, &p.Activity, &p.Goal, &p.Pregnant, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// PutProfile saves the person's profile.
func PutProfile(ctx context.Context, q db.Querier, userID uuid.UUID, p Profile, now time.Time) (Profile, error) {
	if p.Goal == "" {
		p.Goal = "maintain"
	}
	if err := p.validate(now.Year()); err != nil {
		return p, err
	}
	_, err := q.Exec(ctx, `INSERT INTO health_profiles (user_id, birth_year, sex, height_cm, weight_kg, activity, goal, pregnant, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET birth_year = $2, sex = $3, height_cm = $4, weight_kg = $5, activity = $6, goal = $7, pregnant = $8, updated_at = $9`,
		userID, p.BirthYear, p.Sex, p.HeightCm, p.WeightKg, p.Activity, p.Goal, p.Pregnant, now)
	p.UpdatedAt = now
	return p, err
}
