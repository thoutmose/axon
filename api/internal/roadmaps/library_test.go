package roadmaps

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// The library is one roadmap, the Top-5% plan (EP-95, EP-141): its skills are
// competencies with resources, prerequisites point to earlier skills, and
// every hard skill is placed by a domain of the skills assessment.
func TestTheLibraryIsTheTop5Roadmap(t *testing.T) {
	lib := Library()
	if len(lib) != 1 || lib[0].ID != "top5-engineer" || lib[0].Plan == nil {
		t.Fatalf("library = %d roadmaps", len(lib))
	}
	d := lib[0]
	seen := map[string]bool{}
	resources := map[string]int{}
	urls := map[string]bool{}
	for _, r := range d.Resources {
		resources[r.Skill]++
		if !strings.HasPrefix(r.URL, "https://") && !strings.HasPrefix(r.URL, "http://") || urls[r.URL] {
			t.Errorf("%q: link %q missing or repeated", r.Title, r.URL)
		}
		urls[r.URL] = true
	}
	hard := 0
	for _, s := range d.Skills {
		for _, req := range s.Requires {
			if !seen[req] {
				t.Errorf("%q requires %q, which is not an earlier skill", s.Title, req)
			}
		}
		if seen[s.Title] || s.Section == "" {
			t.Errorf("skill %q repeated or without a section", s.Title)
		}
		seen[s.Title] = true
		if s.Kind != "soft" {
			hard++
			if resources[s.Title] == 0 || s.Assess == "" || domain(s.Assess) == nil {
				t.Errorf("hard skill %q needs a resource and an assessment domain (%q)", s.Title, s.Assess)
			}
		}
	}
	if hard != 37 {
		t.Errorf("%d hard skills, want the 37 domains", hard)
	}
	for skill := range resources {
		if !seen[skill] {
			t.Errorf("resources name unknown skill %q", skill)
		}
	}
	if len(Sources()) != 1 {
		t.Errorf("sources = %+v", Sources())
	}
}

// Phases follow each other day after day from the plan's start to its end,
// each with a milestone, a project, an exit check, and known domains.
func TestThePlansPhasesFollowEachOther(t *testing.T) {
	p := *Library()[0].Plan
	skills := map[string]bool{}
	for _, s := range Library()[0].Skills {
		skills[s.Title] = true
	}
	if len(p.Phases) != 18 || p.Start != "2026-10-19" || p.Phases[0].Start != p.Start || p.Phases[17].End != p.End {
		t.Fatalf("phases %d from %s to %s", len(p.Phases), p.Phases[0].Start, p.Phases[len(p.Phases)-1].End)
	}
	for i, ph := range p.Phases {
		if ph.N != i+1 || ph.Slug == "" || ph.Project.Slug == "" || len(ph.Exit) == 0 {
			t.Errorf("phase %d is incomplete", i+1)
		}
		if i > 0 && day(p.Phases[i-1].End).AddDate(0, 0, 1) != day(ph.Start) {
			t.Errorf("phase %d starts %s, the day after %s expected", ph.N, ph.Start, p.Phases[i-1].End)
		}
		for _, d := range ph.Domains {
			if !skills[d] {
				t.Errorf("phase %d names unknown domain %q", ph.N, d)
			}
		}
	}
	if len(p.Habits) != 4 || len(p.Tasks) == 0 {
		t.Errorf("habits %d, tasks %d", len(p.Habits), len(p.Tasks))
	}
}

// The plan keeps its own dates while its first phase runs; after that it
// starts again on the coming Monday.
func TestThePlanStartsOnItsDatesOrTheComingMonday(t *testing.T) {
	p := *Library()[0].Plan
	at := func(s string) time.Time { return day(s) }
	for _, c := range []struct {
		today string
		start string
	}{
		{"2026-10-17", "2026-10-19"}, // before the start
		{"2026-11-04", "2026-10-19"}, // during phase 1
		{"2027-01-10", "2026-10-19"}, // its last day
		{"2027-01-13", "2027-01-18"}, // a Wednesday: the coming Monday
		{"2027-01-18", "2027-01-18"}, // a Monday: that day
		{"2027-01-17", "2027-01-18"}, // a Sunday: the next day
	} {
		if got := at(p.Start).AddDate(0, 0, p.Shift(at(c.today))).Format(time.DateOnly); got != c.start {
			t.Errorf("started on %s: plan from %s, want %s", c.today, got, c.start)
		}
	}
}

// Every certification is taken in a phase that works on its domain, so
// each domain's exam comes with the phase that prepares it, and every
// domain has one (EP-141).
func TestEachCertificationBelongsToItsPhase(t *testing.T) {
	p := *Library()[0].Plan
	certified := map[string]bool{}
	for _, c := range p.Certs {
		if c.Phase < 1 || c.Phase > len(p.Phases) || !slices.Contains(p.Phases[c.Phase-1].Domains, c.Skill) {
			t.Errorf("%s: phase %d does not work on %q", c.Code, c.Phase, c.Skill)
		}
		certified[c.Skill] = true
	}
	for _, s := range Library()[0].Skills {
		if s.Kind != "soft" && !certified[s.Title] {
			t.Errorf("%q has no certification", s.Title)
		}
	}
}
