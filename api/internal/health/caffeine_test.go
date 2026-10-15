package health

import (
	"math"
	"testing"
	"time"
)

func TestCaffeineModel(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 15, h, m, 0, 0, time.UTC) }
	coffee := []Sip{{At: at(8, 0), MG: 100}}

	// Half of it is left after one half-life, a quarter after two; nothing before it was drunk.
	if r := Remaining(coffee, at(13, 0), 5); math.Abs(r-50) > 1e-9 {
		t.Fatalf("after 5 h = %v", r)
	}
	if r := Remaining(coffee, at(18, 0), 5); math.Abs(r-25) > 1e-9 {
		t.Fatalf("after 10 h = %v", r)
	}
	if r := Remaining(coffee, at(7, 0), 5); r != 0 {
		t.Fatalf("before = %v", r)
	}

	// The last coffee (107 mg, as in Gardiner et al. 2023) that leaves no
	// more than the threshold at 23:00, with nothing drunk yet: 8.8 hours before.
	cut, ok := Cutoff(nil, at(23, 0), 5)
	if !ok {
		t.Fatal("no cut-off with nothing drunk")
	}
	if gap := at(23, 0).Sub(cut).Hours(); math.Abs(gap-8.8) > 0.1 {
		t.Fatalf("cut-off %v before bed, want about 8.8 h", gap)
	}
	// A morning coffee leaves a little at bedtime: the cut-off comes earlier.
	cut2, ok := Cutoff(coffee, at(23, 0), 5)
	if !ok || !cut2.Before(cut) {
		t.Fatalf("cut-off with a morning coffee = %v, %v", cut2, ok)
	}
	// Too much already: no more coffee today.
	if _, ok := Cutoff([]Sip{{At: at(20, 0), MG: 200}}, at(23, 0), 5); ok {
		t.Fatal("a cut-off after a late strong coffee")
	}
	// A slower metabolism (half-life 7 h) means stopping earlier.
	slow, _ := Cutoff(nil, at(23, 0), 7)
	if !slow.Before(cut) {
		t.Fatalf("slow cut-off %v not before %v", slow, cut)
	}
}

func TestBedtimeOf(t *testing.T) {
	paris, _ := time.LoadLocation("Europe/Paris")
	day := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	if b := BedtimeOn(day, "23:00", paris); b.Format(time.RFC3339) != "2026-10-15T23:00:00+02:00" {
		t.Fatalf("23:00 = %v", b)
	}
	// After midnight is the night of that day.
	if b := BedtimeOn(day, "00:30", paris); b.Format(time.RFC3339) != "2026-10-16T00:30:00+02:00" {
		t.Fatalf("00:30 = %v", b)
	}
}
