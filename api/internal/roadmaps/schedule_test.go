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
	for _, b := range p.Blocks(0, at("2026-10-04"), at("2026-12-28")) {
		got[b.Date.Format(time.DateOnly)] = b.Title
	}
	for date, want := range map[string]string{
		"2026-10-05": "Maths: MIT 6.042J discrete maths · 2–3 h",
		"2026-10-06": "Computer science: MIT 6.006 algorithms · 2–3 h",
		"2026-10-09": "Spaced tests and 3 DSA problems · 2 h",
		"2026-10-10": "Project pipekit 4 h, DSA 2 h, notes 1 h · 7 h",
		"2026-10-11": "Project pipekit 3 h, reading 2 h: TLCL + BashGuide + Missing Semester, Zettel 1 h, weekly reflection 1 h · 7 h",
		"2026-12-28": "Maths: 18.01SC → 18.02SC calculus (+ Exo7) · 2–3 h", // phase 2
	} {
		if got[date] != want {
			t.Errorf("%s = %q, want %q", date, got[date], want)
		}
	}
	if _, ok := got["2026-10-04"]; ok {
		t.Error("a block before the plan starts")
	}
	if len(got) != 85 {
		t.Errorf("%d blocks from 5 October to 28 December", len(got))
	}
	// Phase 10 reads Full Stack Open on Sundays; shifted plans move with it.
	for _, b := range p.Blocks(7, at("2028-11-12"), at("2028-11-12")) {
		if !strings.Contains(b.Title, "Full Stack Open") {
			t.Errorf("phase 10 Sunday = %q", b.Title)
		}
	}
	if len(p.Blocks(0, at("2030-02-25"), at("2030-03-31"))) != 0 {
		t.Error("blocks after the plan ends")
	}
}
