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
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	qs := alice.mustDo(200, "GET", "/v1/assessment/questions", nil)
	if bytes.Contains(qs, []byte(`"answer"`)) || bytes.Count(qs, []byte(`"question"`)) != 72 {
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
	for _, id := range []string{"python-1", "python-2", "python-3"} {
		answers[id] = key[id]
	}
	ab, _ := json.Marshal(answers)
	var a assessmentJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/assessments", []byte(`{`+profile+`,"interests":["python","rust"],"ratings":{"python":4,"rust":1},"answers":`+string(ab)+`}`)), &a)
	if len(a.Results) != 2 || a.Results[0].Level != 5 || a.Results[0].Correct != 3 || a.Results[1].Level != 1 ||
		!strings.Contains(a.Results[0].Reason, "answered 3 of 3") || len(a.Suggested) != 1 || a.Suggested[0] != "rust" {
		t.Fatalf("assessment = %+v", a)
	}

	// Placement: Python's first two sections are done, with the reason; Rust starts from zero.
	var py roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"python","place":true}`)), &py)
	var full struct {
		Description string `json:"description"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+py.ID, nil), &full)
	if py.Done != 3 || !strings.Contains(full.Description, "Placed by your skills assessment") || !strings.Contains(full.Description, "“1 · Start” and “2 · Idioms” are marked done") {
		t.Fatalf("python placement: done %d, %q", py.Done, full.Description)
	}
	for _, s := range py.Skills {
		if (s.Section == "3 · Ship") != (s.Status == "todo") {
			t.Fatalf("skill %s in %s is %s", s.Title, s.Section, s.Status)
		}
	}
	var rs roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"rust","place":true}`)), &rs)
	if rs.Done != 0 {
		t.Fatalf("rust placed at %d", rs.Done)
	}
	if code, out := alice.do("POST", "/v1/roadmaps/import", []byte(`{"source":"cloud","place":true}`)); code != 422 || !bytes.Contains(out, []byte("Take the skills assessment")) {
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
	if !bytes.Contains(sent, []byte("Support engineer")) || !bytes.Contains(sent, []byte("answered 3 of 3")) {
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
