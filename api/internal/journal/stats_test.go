package journal

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Reference values from Python's statistics.correlation(method="ranked").
func TestSpearmanMatchesTheReference(t *testing.T) {
	for _, c := range []struct {
		x, y []float64
		want float64
	}{
		{[]float64{1, 2, 3, 4, 5, 6, 7, 8}, []float64{2, 1, 4, 3, 6, 5, 8, 7}, 0.9047619047619048},
		{[]float64{7, 6.5, 8, 5, 7, 9, 6, 7.5}, []float64{4, 3, 4, 2, 3, 5, 3, 4}, 0.9191301614277292}, // ties
		{[]float64{0, 1, 0, 1, 1, 0, 0, 1}, []float64{3, 4, 2, 5, 4, 3, 2, 4}, 0.9058216273156765},     // a yes/no factor
	} {
		if got := spearman(c.x, c.y); !near(got, c.want) {
			t.Errorf("spearman = %v, want %v", got, c.want)
		}
	}
	if got := spearman([]float64{1, 1, 1}, []float64{1, 2, 3}); !math.IsNaN(got) {
		t.Errorf("constant factor = %v, want NaN", got)
	}
}

func TestPermutationSeparatesALinkFromNoise(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
	linked := []float64{1, 3, 2, 4, 5, 7, 6, 8, 9, 11, 10, 12, 13, 14}
	noise := []float64{7, 14, 2, 9, 4, 12, 1, 11, 6, 13, 3, 8, 5, 10} // ρ = −0.02
	if p := permutationP(x, linked, 2000, 1); p > 0.01 {
		t.Errorf("linked p = %v", p)
	}
	if p := permutationP(x, noise, 2000, 1); p < 0.2 {
		t.Errorf("noise p = %v", p)
	}
	if permutationP(x, linked, 2000, 1) != permutationP(x, linked, 2000, 1) {
		t.Error("not reproducible")
	}
}

func TestBenjaminiHochberg(t *testing.T) {
	q := benjaminiHochberg([]float64{0.01, 0.04, 0.03, 0.2})
	want := []float64{0.04, 0.04 * 4 / 3, 0.04 * 4 / 3, 0.2}
	for i := range q {
		if !near(q[i], want[i]) {
			t.Fatalf("q = %v, want %v", q, want)
		}
	}
}
