package roadmaps

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRadarHasOneSpokePerRoadmapFromThree(t *testing.T) {
	counts := []skillCount{
		{"hard", "r2", "Rust", "1 · Start", "done", 2, "", ""},
		{"hard", "r1", "Python", "1 · Start", "done", 1, "", ""},
		{"hard", "r1", "Python", "2 · Idioms", "learning", 1, "", ""},
		{"hard", "r1", "Python", "3 · Ship", "todo", 2, "", ""},
		{"hard", "r2", "Rust", "2 · Deepen", "todo", 2, "", ""},
		{"hard", "r3", "SQL", "", "todo", 3, "", ""},
	}
	got := radar(counts, nil).Hard
	want := RadarKind{By: "roadmap", Spokes: []Spoke{
		{Label: "Python", Roadmap: "Python", Done: 1, Learning: 1, Total: 4},
		{Label: "Rust", Roadmap: "Rust", Done: 2, Learning: 0, Total: 4},
		{Label: "SQL", Roadmap: "SQL", Done: 0, Learning: 0, Total: 3},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hard = %+v, want %+v", got, want)
	}
}

func TestRadarFallsBackToSectionsUnderThreeRoadmaps(t *testing.T) {
	counts := []skillCount{
		{"soft", "r9", "Soft skills", "2 · Communicate", "done", 2, "", ""},
		{"soft", "r9", "Soft skills", "1 · Learn", "done", 1, "", ""},
		{"soft", "r9", "Soft skills", "1 · Learn", "learning", 1, "", ""},
		{"soft", "r9", "Soft skills", "3 · Craft", "todo", 1, "", ""},
		{"hard", "r1", "Python", "1 · Start", "done", 1, "", ""},
		{"hard", "r2", "Go", "", "todo", 1, "", ""},
	}
	r := radar(counts, nil)
	soft := RadarKind{By: "section", Spokes: []Spoke{
		{Label: "Learn", Roadmap: "Soft skills", Done: 1, Learning: 1, Total: 2},
		{Label: "Communicate", Roadmap: "Soft skills", Done: 2, Total: 2},
		{Label: "Craft", Roadmap: "Soft skills", Total: 1},
	}}
	if !reflect.DeepEqual(r.Soft, soft) {
		t.Fatalf("soft = %+v, want %+v", r.Soft, soft)
	}
	// A skill outside any section stands for its roadmap.
	hard := RadarKind{By: "section", Spokes: []Spoke{
		{Label: "Go", Roadmap: "Go", Total: 1},
		{Label: "Start", Roadmap: "Python", Done: 1, Total: 1},
	}}
	if !reflect.DeepEqual(r.Hard, hard) {
		t.Fatalf("hard = %+v, want %+v", r.Hard, hard)
	}
}

func TestRadarFoldsSmallSpokesIntoOther(t *testing.T) {
	var counts []skillCount
	for i := range maxSpokes + 3 {
		// Roadmap i has i+1 skills, one of them done.
		counts = append(counts, skillCount{"hard", fmt.Sprint(i), fmt.Sprintf("R%02d", i), "", "done", 1, "", ""})
		if i > 0 {
			counts = append(counts, skillCount{"hard", fmt.Sprint(i), fmt.Sprintf("R%02d", i), "", "todo", i, "", ""})
		}
	}
	spokes := radar(counts, nil).Hard.Spokes
	if len(spokes) != maxSpokes {
		t.Fatalf("%d spokes, want %d", len(spokes), maxSpokes)
	}
	// The four smallest roadmaps (1 to 4 skills) make up Other, shown last.
	if other := spokes[len(spokes)-1]; other != (Spoke{Label: "Other", Done: 4, Total: 10}) {
		t.Fatalf("other = %+v", other)
	}
	if spokes[0].Label != "R04" {
		t.Fatalf("first spoke = %+v, want R04 (order kept)", spokes[0])
	}
}

func TestRadarIsEmptyWithoutSkills(t *testing.T) {
	r := radar(nil, nil)
	if r.Hard.Spokes == nil || r.Soft.Spokes == nil || len(r.Hard.Spokes)+len(r.Soft.Spokes) != 0 {
		t.Fatalf("radar = %+v, want empty lists", r)
	}
}

func TestRadarHasOneSpokePerDomainOfAPlan(t *testing.T) {
	plan := lib.Roadmaps[0]
	var counts []skillCount
	for i, sk := range plan.Skills[:5] {
		status := "todo"
		if i == 1 {
			status = "done"
		}
		counts = append(counts, skillCount{"hard", "r1", plan.Title, sk.Section, status, 1, sk.Title, plan.ID})
	}
	levels := map[string]int{plan.Skills[0].Assess: 4}
	got := radar(counts, levels).Hard
	if got.By != "domain" || len(got.Spokes) != 5 {
		t.Fatalf("hard = %+v, want five domain spokes", got)
	}
	for _, s := range got.Spokes {
		want := s.Label == plan.Skills[0].Title
		if (s.Level != nil) != want || (want && *s.Level != 4) {
			t.Fatalf("%s level = %v, want 4 only on %s", s.Label, s.Level, plan.Skills[0].Title)
		}
		if s.Label == plan.Skills[1].Title && s.Done != 1 {
			t.Fatalf("%s = %+v, want done", s.Label, s)
		}
	}
	// With a roadmap of sections beside it, the radar goes by section.
	counts = append(counts, skillCount{"hard", "r2", "Go", "1 · Start", "todo", 1, "", ""})
	if by := radar(counts, levels).Hard.By; by != "section" {
		t.Fatalf("by = %s, want section", by)
	}
}
