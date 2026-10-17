package app

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// answerKey reads the correct answers from the bundled bank.
func answerKey(t *testing.T) map[string]int {
	t.Helper()
	b, err := os.ReadFile("../roadmaps/assessment.json")
	if err != nil {
		t.Fatal(err)
	}
	var bank struct {
		Domains []struct {
			Questions []struct {
				ID     string `json:"id"`
				Answer int    `json:"answer"`
			} `json:"questions"`
		} `json:"domains"`
	}
	_ = json.Unmarshal(b, &bank)
	key := map[string]int{}
	for _, d := range bank.Domains {
		for _, q := range d.Questions {
			key[q.ID] = q.Answer
		}
	}
	return key
}

type assessmentJSON struct {
	ID      string `json:"id"`
	Results []struct {
		Domain  string `json:"domain"`
		Correct int    `json:"correct"`
		Level   int    `json:"level"`
		Reason  string `json:"reason"`
	} `json:"results"`
	Suggested []string `json:"suggested"`
}

func TestAssessmentPlacesYouOnRoadmapsAndInformsTheTutor(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")

	qs := alice.mustDo(200, "GET", "/v1/assessment/questions", nil)
	if bytes.Contains(qs, []byte(`"answer"`)) || bytes.Count(qs, []byte(`"question"`)) != 185 { // five per domain, easy to hard
		t.Fatalf("questions leak answers or are incomplete: %d", bytes.Count(qs, []byte(`"question"`)))
	}

	profile := `"profile":{"role":"Support engineer","experience":"3 years of scripting","goals":"Become a backend engineer","hours_per_week":6,"formats":["book","practice"]}`
	for _, bad := range []string{
		`{` + profile + `,"interests":[],"ratings":{},"answers":{}}`,
		`{` + profile + `,"interests":["astrology"],"ratings":{"astrology":3},"answers":{}}`,
		`{` + profile + `,"interests":["python"],"ratings":{},"answers":{}}`,
		`{` + profile + `,"interests":["python"],"ratings":{"python":9},"answers":{}}`,
	} {
		alice.mustDo(422, "POST", "/v1/assessments", []byte(bad))
	}

	key := answerKey(t)
	answers := map[string]int{"rust-1": (key["rust-1"] + 1) % 4} // python all right, rust wrong
	for _, id := range []string{"python-1", "python-2", "python-3", "python-4", "python-5"} {
		answers[id] = key[id]
	}
	ab, _ := json.Marshal(answers)
	var a assessmentJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/assessments", []byte(`{`+profile+`,"interests":["python","rust"],"ratings":{"python":4,"rust":1},"answers":`+string(ab)+`}`)), &a)
	if len(a.Results) != 2 || a.Results[0].Level != 5 || a.Results[0].Correct != 5 || a.Results[1].Level != 1 ||
		!strings.Contains(a.Results[0].Reason, "answered 5 of 5") || len(a.Suggested) != 1 || a.Suggested[0] != "rust" {
		t.Fatalf("assessment = %+v", a)
	}

	// Placement on the roadmap: Python, at level 5, is being learnt (its proof is
	// still to come); Rust, at level 1, starts from the beginning.
	var top5 roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"top5-engineer","place":true}`)), &top5)
	var full struct {
		Description string `json:"description"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+top5.ID, nil), &full)
	if !strings.Contains(full.Description, "Placed by your skills assessment: “Python” is at level 3 of 5 or more") {
		t.Fatalf("placement note: %q", full.Description)
	}
	for _, s := range top5.Skills {
		if want := map[bool]string{true: "learning", false: "todo"}[s.Title == "Python"]; s.Status != want {
			t.Fatalf("skill %s is %s", s.Title, s.Status)
		}
	}
	bob := e.client()
	bob.register("Bob")
	if code, out := bob.do("POST", "/v1/roadmaps/import", []byte(`{"source":"top5-engineer","place":true}`)); code != 422 || !bytes.Contains(out, []byte("Take the skills assessment")) {
		t.Fatalf("unassessed placement = %d %s", code, out)
	}

	// The Tutor reads it instead of asking again.
	alice.enableAgents()
	fake.load(toolUse("get_assessment", `{}`), finalText("I see you want to become a backend engineer."))
	r := alice.runAgent("tutor", "Plan my learning")
	if r.Run.Status != "completed" {
		t.Fatalf("run = %+v", r)
	}
	reqs := fake.lastRequests()
	sent, _ := json.Marshal(reqs[1].Messages)
	if !bytes.Contains(sent, []byte("Support engineer")) || !bytes.Contains(sent, []byte("answered 5 of 5")) {
		t.Fatalf("the tutor did not get the assessment: %s", sent)
	}

	// History, then deletion.
	if count(t, alice, "/v1/assessments") != 1 {
		t.Fatal("assessment not listed")
	}
	alice.mustDo(200, "GET", "/v1/assessments/"+a.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/assessments/"+a.ID, nil)
	alice.mustDo(404, "GET", "/v1/assessments/"+a.ID, nil)
}
