package progress

import (
	"fmt"
	"testing"
	"time"
)

func parseDays(t *testing.T, ds ...string) []time.Time {
	t.Helper()
	var out []time.Time
	for _, s := range ds {
		d, err := time.Parse(time.DateOnly, s)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, d)
	}
	return out
}

func TestTier(t *testing.T) {
	steps := [3]float64{7, 21, 66}
	for _, c := range []struct {
		v    float64
		tier string
		next string
	}{{0, "", "bronze"}, {6, "", "bronze"}, {7, "bronze", "silver"}, {30, "silver", "gold"}, {66, "gold", ""}, {400, "gold", ""}} {
		tr, next := tier(c.v, steps)
		got := ""
		if next != nil {
			got = next.Tier
		}
		if tr != c.tier || got != c.next {
			t.Errorf("tier(%v) = %q next %q, want %q next %q", c.v, tr, got, c.tier, c.next)
		}
	}
	if _, next := tier(8, steps); next.Target != 21 {
		t.Errorf("next target = %v", next.Target)
	}
}

func TestLongestRun(t *testing.T) {
	d := func(n int) string { return fmt.Sprintf("2026-10-%02d", n) }
	days := parseDays(t, d(1), d(2), d(3), d(5), d(6), d(7), d(8), d(10))
	if got := longestRun(days); got != 4 {
		t.Fatalf("longest = %d, want 4", got)
	}
	if got := longestRun(nil); got != 0 {
		t.Fatalf("none = %d", got)
	}
}
