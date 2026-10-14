package practice

import (
	"testing"
	"time"
)

func d(s string) time.Time { t, _ := time.Parse(time.DateOnly, s); return t }

// Re-solving at growing intervals (EP-103): a problem comes back sooner the
// harder it went, and leaves once it has been solved unaided at long gaps.
func TestNextReSolve(t *testing.T) {
	for _, c := range []struct {
		name       string
		difficulty string
		attempts   []Attempt
		want       string // "" when no re-solve is due
	}{
		{"not solved: tomorrow", "medium", []Attempt{{d("2026-10-01"), "not_solved"}}, "2026-10-02"},
		{"with help: three days", "hard", []Attempt{{d("2026-10-01"), "with_help"}}, "2026-10-04"},
		{"first unaided solve: a week", "medium", []Attempt{{d("2026-10-01"), "solved"}}, "2026-10-08"},
		{"easy, unaided: done", "easy", []Attempt{{d("2026-10-01"), "solved"}}, ""},
		{"easy after help: three days", "easy", []Attempt{{d("2026-10-01"), "with_help"}}, "2026-10-04"},
		{"after help, then unaided: a week", "hard", []Attempt{{d("2026-10-01"), "with_help"}, {d("2026-10-04"), "solved"}}, "2026-10-11"},
		{"two unaided solves a week apart: 2.5 weeks", "medium", []Attempt{{d("2026-10-01"), "solved"}, {d("2026-10-08"), "solved"}}, "2026-10-26"},
		{"unaided at a 40-day gap: done (would pass 90 days)", "hard", []Attempt{{d("2026-08-01"), "solved"}, {d("2026-09-10"), "solved"}}, ""},
		{"a miss after solves starts over", "medium", []Attempt{{d("2026-09-01"), "solved"}, {d("2026-09-08"), "solved"}, {d("2026-10-01"), "not_solved"}}, "2026-10-02"},
		{"no attempt", "medium", nil, ""},
	} {
		got := ""
		if n := Next(c.difficulty, c.attempts); n != nil {
			got = n.Format(time.DateOnly)
		}
		if got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
