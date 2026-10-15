package decisions

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestScore(t *testing.T) {
	if c := Score(nil); c.Count != 0 || c.Brier != nil || len(c.Buckets) != 0 {
		t.Fatalf("no forecasts = %+v", c)
	}

	// Nine of ten 90% forecasts came true: perfectly calibrated, Brier 0.09.
	var rs []Resolved
	for i := range 10 {
		rs = append(rs, Resolved{P: 0.9, Happened: i < 9})
	}
	c := Score(rs)
	if c.Count != 10 || !near(*c.Brier, 0.09) || !near(c.Predicted, 0.9) || !near(c.Happened, 0.9) {
		t.Fatalf("calibrated = %+v", c)
	}
	if len(c.Buckets) != 1 || c.Buckets[0].From != 0.9 || c.Buckets[0].Count != 10 || !near(c.Buckets[0].Happened, 0.9) {
		t.Fatalf("buckets = %+v", c.Buckets)
	}

	// Overconfident: 80% forecasts that came true half the time, and a coin
	// flip (50%) that did. Buckets are tenths, lowest first; 0.5 opens its own.
	c = Score([]Resolved{{0.8, true}, {0.8, false}, {0.5, true}})
	if want := (0.04 + 0.64 + 0.25) / 3; !near(*c.Brier, want) {
		t.Fatalf("brier = %v, want %v", *c.Brier, want)
	}
	if len(c.Buckets) != 2 || c.Buckets[0].From != 0.5 || c.Buckets[1].From != 0.8 || !near(c.Buckets[1].Happened, 0.5) {
		t.Fatalf("buckets = %+v", c.Buckets)
	}

	// A certainty (out of range for the API, but scored safely) lands in the top bucket.
	if c := Score([]Resolved{{1, true}}); len(c.Buckets) != 1 || c.Buckets[0].From != 0.9 || !near(*c.Brier, 0) {
		t.Fatalf("certain = %+v", c)
	}
}
