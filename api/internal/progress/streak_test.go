package progress

import (
	"math/rand"
	"testing"
	"time"
)

func day(s string) time.Time { t, _ := time.Parse(time.DateOnly, s); return t }

func days(list ...string) map[string]bool {
	m := map[string]bool{}
	for _, d := range list {
		m[d] = true
	}
	return m
}

func TestStreakExamples(t *testing.T) {
	today := day("2026-03-10")
	cases := []struct {
		name    string
		active  map[string]bool
		freezes []Freeze
		state   StreakState
		days    int
		used    int
	}{
		{"nothing yet", days(), nil, Inactive, 0, 0},
		{"today only", days("2026-03-10"), nil, Active, 1, 0},
		{"three in a row, today open", days("2026-03-07", "2026-03-08", "2026-03-09"), nil, AtRisk, 3, 0},
		{"three in a row, today done", days("2026-03-08", "2026-03-09", "2026-03-10"), nil, Active, 3, 0},
		{"missed yesterday", days("2026-03-07", "2026-03-08"), nil, Broken, 0, 0},
		{"missed yesterday, freeze saves it", days("2026-03-07", "2026-03-08"), []Freeze{{ID: "f", BoughtOn: day("2026-03-01")}}, Frozen, 2, 1},
		{"freeze bought after the miss does not apply", days("2026-03-07", "2026-03-08"), []Freeze{{ID: "f", BoughtOn: day("2026-03-10")}}, Broken, 0, 0},
		{"one freeze, two missed days", days("2026-03-06", "2026-03-07"), []Freeze{{ID: "f", BoughtOn: day("2026-03-01")}}, Broken, 0, 0},
		{"two freezes bridge two missed days", days("2026-03-06", "2026-03-07"), []Freeze{{ID: "a", BoughtOn: day("2026-03-01")}, {ID: "b", BoughtOn: day("2026-03-02")}}, Frozen, 2, 2},
		{"long ago", days("2025-12-01"), nil, Inactive, 0, 0},
		{"gap in the middle of a run", days("2026-03-05", "2026-03-06", "2026-03-08", "2026-03-09", "2026-03-10"), []Freeze{{ID: "f", BoughtOn: day("2026-03-01")}}, Active, 5, 1},
	}
	for _, c := range cases {
		st := ComputeStreak(c.active, today, c.freezes)
		if st.State != c.state || st.Days != c.days || len(st.Used) != c.used {
			t.Errorf("%s: got %s/%d used %d, want %s/%d used %d", c.name, st.State, st.Days, len(st.Used), c.state, c.days, c.used)
		}
	}
}

// randomHistory builds random active days and freezes over the last 120 days.
func randomHistory(r *rand.Rand, today time.Time) (map[string]bool, []Freeze) {
	active := map[string]bool{}
	density := r.Float64()
	for i := 0; i < 120; i++ {
		if r.Float64() < density {
			active[dayKey(today.AddDate(0, 0, -i))] = true
		}
	}
	var freezes []Freeze
	for i := 0; i < r.Intn(4); i++ {
		freezes = append(freezes, Freeze{ID: string(rune('a' + i)), BoughtOn: today.AddDate(0, 0, -r.Intn(120))})
	}
	return active, freezes
}

func TestStreakProperties(t *testing.T) {
	r := rand.New(rand.NewSource(15)) //nolint:gosec // reproducible test data
	for n := 0; n < 3000; n++ {
		today := day("2026-01-01").AddDate(0, 0, r.Intn(3650))
		active, freezes := randomHistory(r, today)
		st := ComputeStreak(active, today, freezes)

		// Bounded: never negative, never more than the active days, never
		// more freezes used than owned.
		count := 0
		for range active {
			count++
		}
		if st.Days < 0 || st.Days > count || len(st.Used)+st.Freezes != len(freezes) {
			t.Fatalf("bounds: %+v with %d active days, %d freezes", st, count, len(freezes))
		}
		// Each freeze covers one day, on or after the day it was bought.
		seen := map[string]bool{}
		for _, u := range st.Used {
			if seen[u.ID] {
				t.Fatalf("freeze %s used twice", u.ID)
			}
			seen[u.ID] = true
			for _, f := range freezes {
				if f.ID == u.ID && f.BoughtOn.After(u.Day) {
					t.Fatalf("freeze %s used before it was bought", u.ID)
				}
			}
		}
		// Acting today never lowers the streak, and makes it active.
		withToday := map[string]bool{dayKey(today): true}
		for k := range active {
			withToday[k] = true
		}
		after := ComputeStreak(withToday, today, freezes)
		if after.Days < st.Days || after.State != Active {
			t.Fatalf("acting today lowered the streak: %+v then %+v", st, after)
		}
		// Persisting the freezes it used gives the same answer and uses none again.
		saved := append([]Freeze(nil), freezes...)
		for i := range saved {
			for _, u := range st.Used {
				if saved[i].ID == u.ID {
					d := u.Day
					saved[i].UsedOn = &d
				}
			}
		}
		again := ComputeStreak(active, today, saved)
		if again.Days != st.Days || again.State != st.State || len(again.Used) != 0 {
			t.Fatalf("not stable after saving freezes: %+v then %+v", st, again)
		}
	}
}

// The same local calendar, seen from different time zones, gives the same
// streak: the engine works on local dates, which the caller derives once.
func TestStreakIsTheSameInEveryTimeZone(t *testing.T) {
	zones := []string{"UTC", "Pacific/Kiritimati", "Pacific/Pago_Pago", "Europe/Paris", "America/Sao_Paulo", "Asia/Kathmandu"}
	r := rand.New(rand.NewSource(7)) //nolint:gosec // reproducible test data
	for n := 0; n < 200; n++ {
		base := day("2026-03-28").AddDate(0, 0, r.Intn(10)) // around a DST change in Europe
		var moments []time.Time
		for i := 0; i < 30; i++ {
			if r.Intn(2) == 0 {
				moments = append(moments, base.AddDate(0, 0, -i).Add(time.Duration(r.Intn(24*60))*time.Minute))
			}
		}
		var want *Streak
		for _, z := range zones {
			loc, err := time.LoadLocation(z)
			if err != nil {
				t.Fatal(err)
			}
			// Interpret each moment as local wall-clock time in that zone.
			active := map[string]bool{}
			for _, m := range moments {
				local := time.Date(m.Year(), m.Month(), m.Day(), m.Hour(), m.Minute(), 0, 0, loc)
				active[local.Format(time.DateOnly)] = true
			}
			got := ComputeStreak(active, base, nil)
			if want == nil {
				want = &got
			} else if got.Days != want.Days || got.State != want.State {
				t.Fatalf("%s: %+v, UTC: %+v", z, got, *want)
			}
		}
	}
}
