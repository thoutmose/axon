package health

import (
	"strings"
	"testing"
)

func TestNeedsFollowTheReferences(t *testing.T) {
	// Mifflin–St Jeor for a man of 30, 180 cm, 80 kg: 10·80 + 6.25·180 − 5·30 + 5 = 1780.
	n := Calc(Profile{BirthYear: 1996, Sex: "male", HeightCm: 180, WeightKg: 80, Activity: "moderate"}, 2026, "en")
	if !n.Available || n.Age != 30 || n.BMR != 1780 {
		t.Fatalf("bmr = %+v", n)
	}
	// × 1.55 = 2759 → 2760 kcal; protein 1.2 g/kg; fat 30 % of energy; carbohydrates the rest.
	if n.Kcal != 2760 || n.ProteinG != 96 || n.FatG != 92 || n.CarbsG != 387 || n.FiberG != 25 || n.WaterL != 2.5 {
		t.Fatalf("needs = %+v", n)
	}
	// A woman of 40, 165 cm, 60 kg, sedentary: 600 + 1031.25 − 200 − 161 = 1270.25, × 1.2 → 1520.
	n = Calc(Profile{BirthYear: 1986, Sex: "female", HeightCm: 165, WeightKg: 60, Activity: "sedentary"}, 2026, "en")
	if n.BMR != 1270 || n.Kcal != 1520 || n.ProteinG != 50 || n.FatG != 51 || n.CarbsG != 215 || n.WaterL != 2.0 {
		t.Fatalf("needs = %+v", n)
	}
	if len(n.Explanation) != 7 || len(n.Sources) < 3 {
		t.Fatalf("explanation %d lines, %d sources", len(n.Explanation), len(n.Sources))
	}
}

func TestNeedsWithoutASexAreTheMiddleOfBoth(t *testing.T) {
	n := Calc(Profile{BirthYear: 1986, Sex: "unspecified", HeightCm: 165, WeightKg: 60, Activity: "sedentary"}, 2026, "en")
	// 1270.25 + 83 (half of the 166 between −161 and +5) = 1353.25.
	if n.BMR != 1353 || n.WaterL != 2.3 {
		t.Fatalf("needs = %+v", n)
	}
}

func TestOlderAdultsGetMoreProtein(t *testing.T) {
	n := Calc(Profile{BirthYear: 1950, Sex: "female", HeightCm: 160, WeightKg: 60, Activity: "sedentary"}, 2026, "en")
	if n.ProteinG != 60 { // 1.0 g/kg from 65, not 0.83
		t.Fatalf("protein = %d", n.ProteinG)
	}
}

func TestNoNumbersWhereAProfessionalIsNeeded(t *testing.T) {
	for p, reason := range map[Profile]string{
		{BirthYear: 2010, Sex: "female", HeightCm: 160, WeightKg: 50, Activity: "light"}:                 "under_18",
		{BirthYear: 1995, Sex: "female", HeightCm: 165, WeightKg: 62, Activity: "light", Pregnant: true}: "pregnant",
	} {
		n := Calc(p, 2026, "en")
		if n.Available || n.Reason == nil || *n.Reason != reason || n.Kcal != 0 {
			t.Errorf("%+v: %+v", p, n)
		}
	}
}

func TestNeedsExplainThemselvesInFrench(t *testing.T) {
	n := Calc(Profile{BirthYear: 1996, Sex: "male", HeightCm: 180, WeightKg: 80, Activity: "moderate"}, 2026, "fr")
	if !strings.Contains(strings.Join(n.Explanation, "\n"), "Protéines") {
		t.Fatalf("fr = %v", n.Explanation)
	}
}

func TestProfileValidation(t *testing.T) {
	ok := Profile{BirthYear: 1990, Sex: "female", HeightCm: 165, WeightKg: 60, Activity: "light"}
	if err := ok.validate(2026); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Profile{
		{BirthYear: 1990, Sex: "other", HeightCm: 165, WeightKg: 60, Activity: "light"},
		{BirthYear: 1990, Sex: "female", HeightCm: 90, WeightKg: 60, Activity: "light"},
		{BirthYear: 1990, Sex: "female", HeightCm: 165, WeightKg: 400, Activity: "light"},
		{BirthYear: 1990, Sex: "female", HeightCm: 165, WeightKg: 60, Activity: "couch"},
		{BirthYear: 1890, Sex: "female", HeightCm: 165, WeightKg: 60, Activity: "light"},
		{BirthYear: 2030, Sex: "female", HeightCm: 165, WeightKg: 60, Activity: "light"},
	} {
		if bad.validate(2026) == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}

// A goal moves the needs by a bounded share (EP-191, ADR 0039): 15 % under
// maintenance to lose, never under the basal rate or 1,200 kcal; 10 % over to
// gain; protein up to keep muscle either way.
func TestNeedsFollowABoundedGoal(t *testing.T) {
	man := Profile{BirthYear: 1996, Sex: "male", HeightCm: 180, WeightKg: 80, Activity: "moderate"}
	keep := Calc(man, 2026, "en")
	if keep.Goal != "maintain" || keep.MaintenanceKcal != 2760 || keep.Kcal != 2760 {
		t.Fatalf("maintain = %+v", keep)
	}
	man.Goal = "lose"
	lose := Calc(man, 2026, "en")
	// 2760 × 0.85 = 2346 → 2350; protein 1.2 + 0.4 = 1.6 g/kg → 128 g; fat 30 %; carbohydrates the rest.
	if lose.Kcal != 2350 || lose.MaintenanceKcal != 2760 || lose.ProteinG != 128 || lose.FatG != 78 || lose.CarbsG != 284 {
		t.Fatalf("lose = %+v", lose)
	}
	if !strings.Contains(lose.Explanation[1], "15 % under the 2760 kcal") || !strings.Contains(lose.Explanation[1], "0.4 kg a week") {
		t.Fatalf("energy line = %q", lose.Explanation[1])
	}
	man.Goal = "gain"
	gain := Calc(man, 2026, "en")
	if gain.Kcal != 3040 || gain.ProteinG != 128 || !strings.Contains(gain.Explanation[1], "10 % over the 2760 kcal") {
		t.Fatalf("gain = %+v", gain)
	}
	// Very active people already at 1.6 g/kg go to 2.0, and no further.
	man.Activity, man.Goal = "very_active", "lose"
	if n := Calc(man, 2026, "en"); n.ProteinPerKg != 2.0 || n.ProteinG != 160 {
		t.Fatalf("very active lose = %+v", n)
	}
	// A small sedentary woman: 450 + 937.5 − 200 − 161 = 1026.5 → maintenance 1230; the
	// 15 % would give 1046, so the floor of 1,200 kcal holds and says so.
	small := Calc(Profile{BirthYear: 1986, Sex: "female", HeightCm: 150, WeightKg: 45, Activity: "sedentary", Goal: "lose"}, 2026, "en")
	if small.MaintenanceKcal != 1230 || small.Kcal != 1200 || !strings.Contains(small.Explanation[1], "stops here") {
		t.Fatalf("floored = %+v", small)
	}
	// A heavier basal rate is the floor when it is above 1,200.
	big := Calc(Profile{BirthYear: 1996, Sex: "male", HeightCm: 190, WeightKg: 120, Activity: "sedentary", Goal: "lose"}, 2026, "en")
	// 1200 + 1187.5 − 150 + 5 = 2242.5 → maintenance 2690; 85 % = 2286.5 → 2290 ≥ 2243: not floored.
	if big.Kcal != 2290 || big.BMR != 2243 {
		t.Fatalf("big = %+v", big)
	}
	if fr := Calc(man, 2026, "fr"); !strings.Contains(strings.Join(fr.Explanation, "\n"), "par semaine") {
		t.Fatalf("fr = %v", fr.Explanation)
	}
	bad := Profile{BirthYear: 1990, Sex: "female", HeightCm: 165, WeightKg: 60, Activity: "light", Goal: "bulk"}
	if bad.validate(2026) == nil {
		t.Error("accepted an unknown goal")
	}
}
