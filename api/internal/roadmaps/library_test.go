package roadmaps

import (
	"encoding/json"
	"strings"
	"testing"
)

// Every curated roadmap groups its resources under skills: competencies,
// not course titles. A skill needs at least one resource, and prerequisites
// point to earlier skills, so imports never drop an edge.
func TestLibrarySkillsAreCompetenciesWithResources(t *testing.T) {
	var c curriculum
	if err := json.Unmarshal(ossuJSON, &c); err != nil {
		t.Fatal(err)
	}
	ossu := LibraryRoadmap{ID: "ossu", Skills: c.Skills}
	for _, course := range c.Courses {
		ossu.Resources = append(ossu.Resources, Resource{Title: course.Title, URL: course.URL, Skill: course.Skill})
	}
	for _, d := range append(Library(), ossu) {
		if len(d.Skills) == 0 {
			t.Errorf("%s has no skills", d.ID)
		}
		seen := map[string]bool{}
		resources := map[string]int{}
		for _, r := range d.Resources {
			resources[r.Skill]++
			if !strings.HasPrefix(r.URL, "https://") && !strings.HasPrefix(r.URL, "http://") {
				t.Errorf("%s: %q has no link", d.ID, r.Title)
			}
		}
		for _, s := range d.Skills {
			for _, req := range s.Requires {
				if !seen[req] {
					t.Errorf("%s: %q requires %q, which is not an earlier skill", d.ID, s.Title, req)
				}
			}
			if seen[s.Title] {
				t.Errorf("%s: skill %q appears twice", d.ID, s.Title)
			}
			seen[s.Title] = true
			if resources[s.Title] == 0 {
				t.Errorf("%s: skill %q has no resource", d.ID, s.Title)
			}
			if s.Section == "" {
				t.Errorf("%s: skill %q has no section", d.ID, s.Title)
			}
		}
		for skill := range resources {
			if !seen[skill] {
				t.Errorf("%s: resources name unknown skill %q", d.ID, skill)
			}
		}
	}
}

func TestLibraryCoversTheOwnersDomains(t *testing.T) {
	want := []string{"cs", "dsa", "os", "api-design", "system-design", "databases", "linux-bash", "data-engineering",
		"data-analysis", "data-science", "machine-learning", "maths", "c-cpp", "java", "rust", "python", "networking",
		"devops", "dataops", "gitops", "mlops", "web", "devsecops", "security", "containers", "cloud", "soft-skills"}
	have := map[string]bool{}
	for _, d := range Library() {
		have[d.ID] = true
	}
	for _, id := range want {
		if !have[id] {
			t.Errorf("library has no %s roadmap", id)
		}
	}
}
