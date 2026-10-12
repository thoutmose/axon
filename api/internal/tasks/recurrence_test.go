package tasks

import (
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseRule(t *testing.T) {
	for _, ok := range []string{"FREQ=DAILY", "freq=weekly;byday=mo,th", "RRULE:FREQ=MONTHLY;BYMONTHDAY=-1", "FREQ=YEARLY;UNTIL=20301231"} {
		if _, err := ParseRule(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "FREQ=HOURLY", "FREQ=DAILY;COUNT=3", "FREQ=DAILY;BYHOUR=9", "every day", "FREQ=SOMETIMES"} {
		if _, err := ParseRule(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestNextOccurrenceExamples(t *testing.T) {
	cases := []struct{ rule, current, want string }{
		{"FREQ=DAILY", "2026-03-28", "2026-03-29"},                       // across the EU DST change
		{"FREQ=DAILY;INTERVAL=3", "2026-10-31", "2026-11-03"},            // across the US DST change
		{"FREQ=WEEKLY;BYDAY=MO,TH", "2026-10-12", "2026-10-15"},          // Monday → Thursday
		{"FREQ=WEEKLY;BYDAY=MO,TH", "2026-10-15", "2026-10-19"},          // Thursday → Monday
		{"FREQ=WEEKLY;INTERVAL=2;BYDAY=FR", "2026-10-16", "2026-10-30"},  // every other Friday
		{"FREQ=MONTHLY;BYMONTHDAY=31", "2026-01-31", "2026-03-31"},       // February has no 31st
		{"FREQ=MONTHLY;BYMONTHDAY=-1", "2026-01-31", "2026-02-28"},       // last day of the month
		{"FREQ=MONTHLY;BYDAY=MO;BYSETPOS=1", "2026-10-05", "2026-11-02"}, // first Monday
		{"FREQ=YEARLY", "2028-02-29", "2032-02-29"},                      // leap day
	}
	for _, c := range cases {
		got, ok, err := NextOccurrence(c.rule, date(c.current))
		if err != nil || !ok || got.Format("2006-01-02") != c.want {
			t.Errorf("%s after %s = %v %v %v, want %s", c.rule, c.current, got.Format("2006-01-02"), ok, err, c.want)
		}
	}
	if _, ok, _ := NextOccurrence("FREQ=DAILY;UNTIL=20261010", date("2026-10-10")); ok {
		t.Error("series past UNTIL did not end")
	}
}

// Property: for random rules and start dates over many years, each next
// occurrence is a later date, at midnight, and satisfies the rule.
func TestNextOccurrenceProperties(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // deterministic test data, not security
	days := []string{"MO", "TU", "WE", "TH", "FR", "SA", "SU"}
	weekday := map[string]time.Weekday{"MO": 1, "TU": 2, "WE": 3, "TH": 4, "FR": 5, "SA": 6, "SU": 0}
	for range 2000 {
		start := date("2020-01-01").AddDate(0, 0, rng.IntN(365*15))
		interval := 1 + rng.IntN(4)
		cur := start
		switch rng.IntN(3) {
		case 0: // daily
			rule := "FREQ=DAILY;INTERVAL=" + strconv.Itoa(interval)
			for range 20 {
				next, ok, err := NextOccurrence(rule, cur)
				if err != nil || !ok || int(next.Sub(cur).Hours()) != 24*interval || next.Hour() != 0 {
					t.Fatalf("%s after %s = %s", rule, cur, next)
				}
				cur = next
			}
		case 1: // weekly on a random set of days
			set := slices.Clone(days)
			rng.Shuffle(len(set), func(i, j int) { set[i], set[j] = set[j], set[i] })
			set = set[:1+rng.IntN(3)]
			rule := "FREQ=WEEKLY;BYDAY=" + strings.Join(set, ",")
			cur = alignTo(cur, weekday[set[0]])
			for range 20 {
				next, ok, err := NextOccurrence(rule, cur)
				if err != nil || !ok || !next.After(cur) || next.Sub(cur) > 7*24*time.Hour {
					t.Fatalf("%s after %s = %s", rule, cur, next)
				}
				if !slices.ContainsFunc(set, func(d string) bool { return weekday[d] == next.Weekday() }) {
					t.Fatalf("%s produced a %s", rule, next.Weekday())
				}
				cur = next
			}
		case 2: // monthly on a day that some months lack
			dom := 1 + rng.IntN(31)
			rule := "FREQ=MONTHLY;BYMONTHDAY=" + strconv.Itoa(dom)
			cur = time.Date(cur.Year(), cur.Month(), 1, 0, 0, 0, 0, time.UTC)
			for range 24 {
				next, ok, err := NextOccurrence(rule, cur)
				if err != nil || !ok || !next.After(cur) || next.Day() != dom {
					t.Fatalf("%s after %s = %s", rule, cur, next)
				}
				cur = next
			}
		}
	}
}

func alignTo(t time.Time, wd time.Weekday) time.Time {
	for t.Weekday() != wd {
		t = t.AddDate(0, 0, 1)
	}
	return t
}
