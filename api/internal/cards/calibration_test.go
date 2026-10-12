package cards

import (
	"math"
	"testing"
)

func TestCalibrateComparesPredictionWithRecall(t *testing.T) {
	var rs []reviewed
	add := func(n int, p float64, remembered int) {
		for i := range n {
			rs = append(rs, reviewed{predicted: p, remembered: i < remembered})
		}
	}
	add(10, 0.92, 9) // predicted 92%, remembered 9 of 10
	add(10, 0.97, 8) // predicted 97%, remembered 8 of 10: the model is optimistic there
	add(4, 0.6, 1)
	c := calibrate(rs, 0.9)
	if c.Reviews != 24 || c.Target != 0.9 {
		t.Fatalf("calibration = %+v", c)
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	if !near(c.Remembered, 18.0/24) || !near(c.Predicted, (10*0.92+10*0.97+4*0.6)/24) {
		t.Errorf("overall = predicted %.4f, remembered %.4f", c.Predicted, c.Remembered)
	}
	// Buckets cover 0 to 1 in order; empty ones are left out.
	want := []struct {
		from, to   float64
		n          int
		remembered float64
	}{{0.5, 0.7, 4, 0.25}, {0.9, 0.95, 10, 0.9}, {0.95, 1, 10, 0.8}}
	if len(c.Buckets) != len(want) {
		t.Fatalf("buckets = %+v", c.Buckets)
	}
	for i, w := range want {
		b := c.Buckets[i]
		if b.From != w.from || b.To != w.to || b.Reviews != w.n || !near(b.Remembered, w.remembered) {
			t.Errorf("bucket %d = %+v, want %+v", i, b, w)
		}
	}
}

func TestCalibrateWithNoReviews(t *testing.T) {
	c := calibrate(nil, 0.85)
	if c.Reviews != 0 || c.Buckets == nil || len(c.Buckets) != 0 || c.Target != 0.85 {
		t.Errorf("empty calibration = %+v", c)
	}
}
