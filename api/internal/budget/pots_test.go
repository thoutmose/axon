package budget

import (
	"testing"
	"time"
)

func TestMonthlyReachesTheTargetOnTime(t *testing.T) {
	oct := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		saved, target int64
		date          string
		want          int64 // -1 for none
	}{
		{0, 120000, "2027-09-30", 10000},    // 12 months, October to September
		{20000, 120000, "2027-09-30", 8334}, // rounded up so the target is reached
		{0, 5000, "2026-10-31", 5000},       // this month only
		{0, 5000, "2026-01-01", 5000},       // past date: the rest, now
		{5000, 5000, "2027-01-01", -1},      // reached
		{0, 5000, "", -1},                   // no date
	} {
		got := Monthly(c.saved, c.target, oct, c.date)
		if (got == nil) != (c.want == -1) || (got != nil && *got != c.want) {
			t.Errorf("Monthly(%d, %d, %q) = %v, want %d", c.saved, c.target, c.date, got, c.want)
		}
		if got != nil && c.date != "" {
			n := int64(max(monthsLeft(oct, c.date), 1))
			if c.saved+*got*n < c.target {
				t.Errorf("%+v does not reach the target", c)
			}
		}
	}
}
