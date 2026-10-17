package roadmaps

import (
	"strings"
	"testing"
	"time"
)

// The roadmap's weekly rhythm on the calendar (EP-97): each day's block
// with the tracks of the phase that day falls in, only within the plan.
func TestTheWeeklyRhythmFollowsThePhases(t *testing.T) {
	p := *Library()[0].Plan
	at := func(s string) time.Time { return day(s) }
	got := map[string]string{}
	for _, b := range p.Blocks(0, at("2026-10-18"), at("2027-01-11")) {
		got[b.Date.Format(time.DateOnly)] = b.Title
	}
	for date, want := range map[string]string{
		"2026-10-19": "Theory: MIT 6.042J discrete maths · 2–3 h",
		"2026-10-20": "Core: MIT 6.006 algorithms + Stanford Algorithms 1 · 2–3 h",
		"2026-10-23": "Spaced tests and 3 DSA problems · 2 h",
		"2026-10-24": "Project pipekit 4 h, DSA 2 h, notes 1 h · 7 h",
		"2026-10-25": "Project pipekit 3 h, reading 2 h: TLCL + BashGuide + Missing Semester, Zettel 1 h, weekly reflection 1 h · 7 h",
		"2027-01-11": "Theory: 18.01SC → 18.02SC calculus (+ Exo7) · 2–3 h", // phase 2
	} {
		if got[date] != want {
			t.Errorf("%s = %q, want %q", date, got[date], want)
		}
	}
	if _, ok := got["2026-10-18"]; ok {
		t.Error("a block before the plan starts")
	}
	if len(got) != 85 {
		t.Errorf("%d blocks from 19 October to 11 January", len(got))
	}
	// Phase 13 reads Full Stack Open on Sundays; shifted plans move with it.
	for _, b := range p.Blocks(7, at("2029-08-05"), at("2029-08-05")) {
		if !strings.Contains(b.Title, "Full Stack Open") {
			t.Errorf("phase 13 Sunday = %q", b.Title)
		}
	}
	if len(p.Blocks(0, at("2031-01-06"), at("2031-02-28"))) != 0 {
		t.Error("blocks after the plan ends")
	}
}
