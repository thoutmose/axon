// Package progress turns what a person does in every area into transparent
// experience points, levels, badges, and one daily streak (EP-15). Points are
// computed from the data itself, never stored separately, so they cannot
// drift from reality, and every point can be traced back to its source.
package progress

import (
	"sort"
	"time"
)

// StreakState is where the daily streak stands (README: streak and freeze states).
type StreakState string

// Streak states. None of them is a failure: "broken" only means "start again".
const (
	Inactive StreakState = "inactive" // nothing yet, or long ago
	Active   StreakState = "active"   // today counts already
	AtRisk   StreakState = "at_risk"  // yesterday counted, today is still open
	Frozen   StreakState = "frozen"   // a freeze covered the last missed day
	Broken   StreakState = "broken"   // the last scheduled day was missed; a fresh start is one action away
)

// Freeze is a streak freeze bought with earned points.
type Freeze struct {
	ID       string
	BoughtOn time.Time  // local date
	UsedOn   *time.Time // local date it covered, if any
}

// Streak is the computed streak, with the freezes it newly consumed.
type Streak struct {
	State   StreakState `json:"state"`
	Days    int         `json:"days"`    // active days in the current run (frozen days are kept, not counted)
	Freezes int         `json:"freezes"` // unused freezes in stock
	Used    []Use       `json:"-"`       // freezes to mark used
}

// Use records that a freeze covered a missed day.
type Use struct {
	ID  string
	Day time.Time
}

func dayKey(d time.Time) string { return d.Format(time.DateOnly) }

// ComputeStreak walks back from today over local dates. Today never breaks
// the run: it is still open. A missed earlier day is covered by the oldest
// unused freeze bought on or before it, but only when the day before it
// continues the run — a freeze bridges a gap, it never revives a dead streak.
// It is pure arithmetic on local calendar days, so time zones and DST cannot
// shift it.
func ComputeStreak(active map[string]bool, today time.Time, freezes []Freeze) Streak {
	var stock []Freeze
	covered := map[string]bool{}
	for _, f := range freezes {
		if f.UsedOn == nil {
			stock = append(stock, f)
		} else {
			covered[dayKey(*f.UsedOn)] = true
		}
	}
	sort.Slice(stock, func(i, j int) bool { return stock[i].BoughtOn.Before(stock[j].BoughtOn) })

	st := Streak{}
	todayDone := active[dayKey(today)]
	if todayDone {
		st.Days = 1
	}
	frozeYesterday := false
	for d, i := today.AddDate(0, 0, -1), 0; i < 3660; d, i = d.AddDate(0, 0, -1), i+1 {
		k := dayKey(d)
		if active[k] {
			st.Days++
			continue
		}
		if covered[k] {
			frozeYesterday = frozeYesterday || i == 0
			continue
		}
		// A gap of missed days, newest first, up to a week.
		var gap []time.Time
		x := d
		for len(gap) < 7 && !active[dayKey(x)] && !covered[dayKey(x)] {
			gap = append(gap, x)
			x = x.AddDate(0, 0, -1)
		}
		if !active[dayKey(x)] && !covered[dayKey(x)] {
			break // nothing behind the gap to keep
		}
		// Cover the oldest gap days with the oldest freezes, each bought on or before its day.
		var picks []int
		taken := map[int]bool{}
		for g := len(gap) - 1; g >= 0; g-- {
			found := -1
			for j, f := range stock {
				if !taken[j] && !f.BoughtOn.After(gap[g]) {
					found = j
					break
				}
			}
			if found < 0 {
				break
			}
			taken[found] = true
			picks = append(picks, found)
		}
		if len(picks) < len(gap) {
			break
		}
		for n, j := range picks {
			day := gap[len(gap)-1-n]
			st.Used = append(st.Used, Use{stock[j].ID, day})
			covered[dayKey(day)] = true
		}
		var rest []Freeze
		for j, f := range stock {
			if !taken[j] {
				rest = append(rest, f)
			}
		}
		stock = rest
		frozeYesterday = frozeYesterday || i == 0
		d = gap[len(gap)-1] // continue from the oldest covered day
		i += len(gap) - 1
	}
	st.Freezes = len(stock)
	switch {
	case todayDone:
		st.State = Active
	case st.Days > 0 && frozeYesterday:
		st.State = Frozen
	case st.Days > 0:
		st.State = AtRisk
	case anyRecent(active, today, 30):
		st.State = Broken
	default:
		st.State = Inactive
	}
	return st
}

func anyRecent(active map[string]bool, today time.Time, days int) bool {
	for i := 1; i <= days; i++ {
		if active[dayKey(today.AddDate(0, 0, -i))] {
			return true
		}
	}
	return false
}
