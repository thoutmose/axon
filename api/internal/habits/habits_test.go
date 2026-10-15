package habits

import (
	"strings"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/clock"
)

var everyDay = []int{1, 2, 3, 4, 5, 6, 7}

func d(s string) time.Time { t, _ := time.Parse(time.DateOnly, s); return t }

func checks(days ...string) map[string]string {
	m := map[string]string{}
	for _, day := range days {
		m[day] = "done"
	}
	return m
}

// Copy reviewed against the product quality questions: a missed day leads to
// recovery, never punishment.
func TestRecoveryCopyNeverShames(t *testing.T) {
	banned := []string{"fail", "lost", "broke", "shame", "lazy", "should have", "streak", "again?", "!"}
	for missed := 1; missed <= 7; missed++ {
		msg := recoveryMessage(missed, d("2026-10-09"), d("2026-10-10"))
		for _, w := range banned {
			if strings.Contains(strings.ToLower(msg), w) {
				t.Errorf("recovery copy %q contains %q", msg, w)
			}
		}
	}
}

func TestComputeRecoveryAndStreak(t *testing.T) {
	created := d("2026-09-01")
	today := d("2026-10-10") // a Saturday

	st := Compute(everyDay, checks("2026-10-07", "2026-10-08"), today, created)
	if st.Recovery == nil || st.Recovery.Missed != 1 || !strings.Contains(st.Recovery.Message, "yesterday") || st.Streak != 0 {
		t.Fatalf("missed yesterday: %+v", st)
	}
	st = Compute(everyDay, checks("2026-10-05"), today, created)
	if st.Recovery == nil || st.Recovery.Missed != 4 || !strings.Contains(st.Recovery.Message, "smallest version") {
		t.Fatalf("missed several: %+v", st)
	}
	// Checking in today clears the prompt and counts.
	st = Compute(everyDay, checks("2026-10-08", "2026-10-09", "2026-10-10"), today, created)
	if st.Recovery != nil || st.Streak != 3 || st.CheckedIn == nil {
		t.Fatalf("checked in today: %+v", st)
	}
	// A planned skip neither breaks nor counts; the two-minute version counts.
	c := checks("2026-10-07")
	c["2026-10-08"], c["2026-10-09"] = "skipped", "two_minute"
	if st = Compute(everyDay, c, today, created); st.Recovery != nil || st.Streak != 2 {
		t.Fatalf("skip and two-minute: %+v", st)
	}
	// Rest days on a Mon/Wed/Fri schedule are not misses.
	mwf := []int{1, 3, 5}
	if st = Compute(mwf, checks("2026-10-05", "2026-10-07", "2026-10-09"), today, created); st.Recovery != nil || st.Streak != 3 || st.ScheduledToday {
		t.Fatalf("rest days: %+v", st)
	}
	// Missing Friday on that schedule names the day, not "yesterday".
	if st = Compute(mwf, checks("2026-10-07"), d("2026-10-11"), created); st.Recovery == nil || !strings.Contains(st.Recovery.Message, "Friday") {
		t.Fatalf("named day: %+v", st)
	}
	// A habit created today has nothing to recover from.
	if st = Compute(everyDay, nil, today, today); st.Recovery != nil {
		t.Fatalf("new habit: %+v", st)
	}
}

// Daylight-saving changes never add or drop a day: a daily habit kept every
// day for 300 days, across two DST transitions, has a 300-day streak.
func TestDaylightSavingNeverShiftsDays(t *testing.T) {
	for _, tz := range []string{"Europe/Paris", "America/New_York", "Australia/Sydney"} {
		loc, _ := time.LoadLocation(tz)
		start := time.Date(2026, 1, 1, 23, 30, 0, 0, loc)
		c := map[string]string{}
		var today time.Time
		for i := range 300 {
			now := start.AddDate(0, 0, i) // 23:30 local every evening
			today = clock.Day(now, loc)
			c[today.Format(time.DateOnly)] = "done"
		}
		st := Compute(everyDay, c, today, clock.Day(start, loc))
		if st.Streak != 300 || st.Recovery != nil {
			t.Errorf("%s: streak %d, recovery %+v", tz, st.Streak, st.Recovery)
		}
	}
}

// Travelling east: checked in at 23:30 in Paris, then it is already the next
// morning in Tokyo. Yesterday counts, nothing was missed, today is open.
func TestTravelAcrossTimeZones(t *testing.T) {
	paris, _ := time.LoadLocation("Europe/Paris")
	checkedAt := time.Date(2026, 10, 9, 23, 30, 0, 0, paris)
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	day := clock.Day(checkedAt, paris)
	later := checkedAt.Add(30 * time.Minute)
	todayTokyo := clock.Day(later, tokyo)
	if !todayTokyo.Equal(day.AddDate(0, 0, 1)) {
		t.Fatalf("Tokyo date = %s", todayTokyo)
	}
	st := Compute(everyDay, checks(day.Format(time.DateOnly)), todayTokyo, d("2026-09-01"))
	if st.Recovery != nil || st.Streak != 1 || st.CheckedIn != nil {
		t.Fatalf("after travel: %+v", st)
	}
	// Flying back west the same instant is still the Paris day: no double count.
	if st := Compute(everyDay, checks(day.Format(time.DateOnly)), clock.Day(later.Add(-time.Hour), paris), d("2026-09-01")); st.CheckedIn == nil {
		t.Fatalf("back home: %+v", st)
	}
}

func TestBestRun(t *testing.T) {
	day := func(n int) time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, n) }
	key := func(n int) string { return day(n).Format(time.DateOnly) }
	every := []int{1, 2, 3, 4, 5, 6, 7}
	// Kept days 0–4, missed 5, kept 6–7: the best run is 5.
	c := map[string]string{}
	for _, n := range []int{0, 1, 2, 3, 4, 6, 7} {
		c[key(n)] = "done"
	}
	if got := BestRun(every, c, day(0), day(7)); got != 5 {
		t.Fatalf("best = %d, want 5", got)
	}
	// A planned rest day neither counts nor breaks it; a two-minute version counts.
	c[key(5)] = "skipped"
	c[key(6)] = "two_minute"
	if got := BestRun(every, c, day(0), day(7)); got != 7 {
		t.Fatalf("with a rest day = %d, want 7", got)
	}
	// Days off the schedule are not misses: Mondays, Wednesdays, Fridays.
	mwf := map[string]string{}
	for d := day(0); !d.After(day(27)); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd == time.Monday || wd == time.Wednesday || wd == time.Friday {
			mwf[d.Format(time.DateOnly)] = "done"
		}
	}
	if got := BestRun([]int{1, 3, 5}, mwf, day(0), day(27)); got != 12 {
		t.Fatalf("three days a week for four weeks = %d, want 12", got)
	}
	if got := BestRun(every, map[string]string{}, day(0), day(7)); got != 0 {
		t.Fatalf("nothing = %d", got)
	}
}
