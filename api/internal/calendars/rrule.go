package calendars

import (
	"slices"
	"strconv"
	"strings"
	"time"
)

// expand lists the starts of a repeating event (RFC 5545 RRULE) that fall
// in [from, to). It covers what calendars write in practice: FREQ daily,
// weekly, monthly, or yearly, with INTERVAL, COUNT, UNTIL, BYDAY (with an
// ordinal such as 2TU or -1FR for monthly rules), BYMONTHDAY, and BYMONTH.
// Times repeat at the same wall-clock time in the event's zone, across DST.
func expand(start time.Time, rule string, from, to time.Time) []time.Time {
	r := map[string]string{}
	for _, part := range strings.Split(rule, ";") {
		if k, v, ok := strings.Cut(part, "="); ok {
			r[strings.ToUpper(k)] = strings.ToUpper(v)
		}
	}
	interval, _ := strconv.Atoi(r["INTERVAL"])
	if interval < 1 {
		interval = 1
	}
	count, _ := strconv.Atoi(r["COUNT"])
	var until time.Time
	if u := r["UNTIL"]; u != "" {
		if t, err := time.Parse("20060102T150405Z", u); err == nil {
			until = t
		} else if t, err := time.ParseInLocation("20060102", u, start.Location()); err == nil {
			until = t.AddDate(0, 0, 1).Add(-time.Second) // the whole last day
		}
	}
	type byday struct {
		n  int // 0: every such weekday in the period; 2: the second; -1: the last
		wd time.Weekday
	}
	var days []byday
	for _, d := range strings.Split(r["BYDAY"], ",") {
		if len(d) < 2 {
			continue
		}
		wd, ok := map[string]time.Weekday{"SU": 0, "MO": 1, "TU": 2, "WE": 3, "TH": 4, "FR": 5, "SA": 6}[d[len(d)-2:]]
		if !ok {
			continue
		}
		n, _ := strconv.Atoi(d[:len(d)-2])
		days = append(days, byday{n, wd})
	}
	var monthDays, months []int
	for _, v := range strings.Split(r["BYMONTHDAY"], ",") {
		if n, err := strconv.Atoi(v); err == nil && n != 0 {
			monthDays = append(monthDays, n)
		}
	}
	for _, v := range strings.Split(r["BYMONTH"], ",") {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 12 {
			months = append(months, n)
		}
	}

	loc := start.Location()
	at := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, start.Hour(), start.Minute(), start.Second(), 0, loc)
	}
	inMonth := func(y int, m time.Month) []time.Time {
		last := time.Date(y, m+1, 0, 0, 0, 0, 0, loc).Day()
		var out []time.Time
		switch {
		case len(monthDays) > 0:
			for _, d := range monthDays {
				if d < 0 {
					d = last + 1 + d
				}
				if d >= 1 && d <= last {
					out = append(out, at(y, m, d))
				}
			}
		case len(days) > 0:
			for _, bd := range days {
				var hits []int
				for d := 1; d <= last; d++ {
					if time.Date(y, m, d, 0, 0, 0, 0, loc).Weekday() == bd.wd {
						hits = append(hits, d)
					}
				}
				switch {
				case bd.n == 0:
					for _, d := range hits {
						out = append(out, at(y, m, d))
					}
				case bd.n > 0 && bd.n <= len(hits):
					out = append(out, at(y, m, hits[bd.n-1]))
				case bd.n < 0 && -bd.n <= len(hits):
					out = append(out, at(y, m, hits[len(hits)+bd.n]))
				}
			}
		default:
			if start.Day() <= last {
				out = append(out, at(y, m, start.Day()))
			}
		}
		return out
	}

	var out []time.Time
	seen := 0
	for k := 0; k < 5000; k++ {
		var cands []time.Time
		switch r["FREQ"] {
		case "DAILY":
			cands = []time.Time{start.AddDate(0, 0, k*interval)}
		case "WEEKLY":
			// The week of the start, Monday first, then every interval weeks.
			monday := start.AddDate(0, 0, -((int(start.Weekday())+6)%7)+7*k*interval)
			if len(days) == 0 {
				cands = []time.Time{start.AddDate(0, 0, 7*k*interval)}
			}
			for _, bd := range days {
				cands = append(cands, at(monday.Year(), monday.Month(), monday.Day()+(int(bd.wd)+6)%7))
			}
		case "MONTHLY":
			first := time.Date(start.Year(), start.Month()+time.Month(k*interval), 1, 0, 0, 0, 0, loc)
			cands = inMonth(first.Year(), first.Month())
		case "YEARLY":
			y := start.Year() + k*interval
			ms := months
			if len(ms) == 0 {
				ms = []int{int(start.Month())}
			}
			for _, m := range ms {
				if len(monthDays) == 0 && len(days) == 0 {
					if start.Day() <= time.Date(y, time.Month(m)+1, 0, 0, 0, 0, 0, loc).Day() {
						cands = append(cands, at(y, time.Month(m), start.Day()))
					}
				} else {
					cands = append(cands, inMonth(y, time.Month(m))...)
				}
			}
		default:
			return []time.Time{start} // an unknown rule: the first occurrence only
		}
		slices.SortFunc(cands, func(a, b time.Time) int { return a.Compare(b) })
		for _, c := range cands {
			if c.Before(start) {
				continue
			}
			if (count > 0 && seen >= count) || (!until.IsZero() && c.After(until)) || !c.Before(to) {
				return out
			}
			seen++
			if !c.Before(from) {
				out = append(out, c)
			}
		}
	}
	return out
}
