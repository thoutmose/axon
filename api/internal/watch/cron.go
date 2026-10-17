package watch

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Schedule is a five-field cron expression (minute hour day-of-month month
// day-of-week), read in the person's time zone. The default is midnight.
type Schedule struct {
	minute, hour, dom, month, dow uint64 // bit sets
	domStar, dowStar              bool
}

// DefaultSchedule fetches each source once a day, at midnight.
const DefaultSchedule = "0 0 * * *"

// MinInterval keeps a source from being fetched more often than this.
const MinInterval = 15 * time.Minute

var macros = map[string]string{"@daily": "0 0 * * *", "@midnight": "0 0 * * *", "@hourly": "0 * * * *", "@weekly": "0 0 * * 0", "@monthly": "0 0 1 * *"}

// ParseSchedule reads a cron expression; "@daily", "@hourly", "@weekly",
// and "@monthly" are accepted too.
func ParseSchedule(expr string) (Schedule, error) {
	expr = strings.TrimSpace(expr)
	if m, ok := macros[strings.ToLower(expr)]; ok {
		expr = m
	}
	f := strings.Fields(expr)
	if len(f) != 5 {
		return Schedule{}, errors.New("a schedule has five fields: minute hour day month weekday, like 0 0 * * *")
	}
	var s Schedule
	var err error
	parts := []struct {
		dst    *uint64
		lo, hi int
		name   string
	}{{&s.minute, 0, 59, "minute"}, {&s.hour, 0, 23, "hour"}, {&s.dom, 1, 31, "day of the month"}, {&s.month, 1, 12, "month"}, {&s.dow, 0, 7, "weekday"}}
	for i, p := range parts {
		if *p.dst, err = field(f[i], p.lo, p.hi); err != nil {
			return Schedule{}, fmt.Errorf("%s: %w", p.name, err)
		}
	}
	if s.dow&(1<<7) != 0 { // 7 is Sunday too
		s.dow |= 1
	}
	s.domStar, s.dowStar = f[2] == "*", f[4] == "*"
	// Too frequent schedules are refused: two runs at least MinInterval apart.
	t := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	for range 200 {
		n := s.Next(t, time.UTC)
		if nn := s.Next(n, time.UTC); nn.Sub(n) < MinInterval {
			return Schedule{}, fmt.Errorf("runs are at least %d minutes apart", int(MinInterval.Minutes()))
		}
		t = n
	}
	return s, nil
}

// field reads "*", "5", "1-5", "*/15", "1-10/2", "1,15,30".
func field(f string, lo, hi int) (uint64, error) {
	var set uint64
	for _, part := range strings.Split(f, ",") {
		rng, stepStr, hasStep := strings.Cut(part, "/")
		step := 1
		if hasStep {
			n, err := strconv.Atoi(stepStr)
			if err != nil || n < 1 {
				return 0, fmt.Errorf("bad step %q", stepStr)
			}
			step = n
		}
		a, b := lo, hi
		switch {
		case rng == "*":
		case strings.Contains(rng, "-"):
			x, y, _ := strings.Cut(rng, "-")
			var e1, e2 error
			a, e1 = strconv.Atoi(x)
			b, e2 = strconv.Atoi(y)
			if e1 != nil || e2 != nil {
				return 0, fmt.Errorf("bad range %q", rng)
			}
		default:
			n, err := strconv.Atoi(rng)
			if err != nil {
				return 0, fmt.Errorf("bad value %q", rng)
			}
			a, b = n, n
			if hasStep {
				b = hi
			}
		}
		if a < lo || b > hi || a > b {
			return 0, fmt.Errorf("%q is outside %d-%d", rng, lo, hi)
		}
		for i := a; i <= b; i += step {
			set |= 1 << uint(i)
		}
	}
	return set, nil
}

// Next is the first time after t that matches, in loc.
func (s Schedule) Next(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc).Truncate(time.Minute).Add(time.Minute)
	// A matching minute exists within five years for any valid expression.
	for range 366 * 5 * 24 {
		if s.month&(1<<uint(t.Month())) == 0 {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if !s.dayMatches(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
			continue
		}
		if s.hour&(1<<uint(t.Hour())) == 0 {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, loc)
			continue
		}
		for m := t.Minute(); m < 60; m++ {
			if s.minute&(1<<uint(m)) != 0 {
				return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), m, 0, 0, loc)
			}
		}
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, loc)
	}
	return t
}

// dayMatches follows cron: with both day fields restricted, either matches.
func (s Schedule) dayMatches(t time.Time) bool {
	dom := s.dom&(1<<uint(t.Day())) != 0
	dow := s.dow&(1<<uint(t.Weekday())) != 0
	switch {
	case s.domStar && s.dowStar:
		return true
	case s.domStar:
		return dow
	case s.dowStar:
		return dom
	}
	return dom || dow
}
