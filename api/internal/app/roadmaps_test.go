package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type roadmapJSON struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Attribution string `json:"attribution"`
	Done        int    `json:"done"`
	Total       int    `json:"total"`
	Skills      []struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Section   string `json:"section"`
		Status    string `json:"status"`
		Resources []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"resources"`
	} `json:"skills"`
	Edges []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"edges"`
}

func TestRoadmapsImportManualEditingAndProgress(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	src := alice.mustDo(200, "GET", "/v1/roadmaps/sources", nil)
	if !bytes.Contains(src, []byte(`"license_url":"https://github.com/ossu/computer-science/blob/master/LICENSE"`)) {
		t.Fatalf("sources = %s", src)
	}
	if code, out := alice.do("POST", "/v1/roadmaps/import", []byte(`{"source":"roadmap.sh"}`)); code != 422 || !bytes.Contains(out, []byte("may not be redistributed")) {
		t.Fatalf("roadmap.sh import = %d %s", code, out)
	}
	var ossu roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"ossu"}`)), &ossu)
	// 63 courses grouped into 42 skills: the three calculus courses are one skill.
	if ossu.Source != "ossu" || len(ossu.Skills) != 42 || len(ossu.Edges) != 13 || ossu.Total != 42 ||
		!bytes.Contains([]byte(ossu.Attribution), []byte("MIT License")) {
		t.Fatalf("ossu = %d skills, %d edges, %+v", len(ossu.Skills), len(ossu.Edges), ossu.Attribution)
	}
	for _, s := range ossu.Skills {
		if s.Title == "Calculus" && len(s.Resources) != 3 {
			t.Fatalf("calculus resources = %+v", s.Resources)
		}
	}

	// A manual roadmap with prerequisites.
	var rm roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Japanese N4"}`)), &rm)
	var kana, kanji, grammar struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"Kana","section":"Basics","url":"https://example.org/kana"}`)), &kana)
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"300 kanji","section":"Vocabulary","requires":["`+kana.ID+`"]}`)), &kanji)
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"Te-form","section":"Grammar"}`)), &grammar)
	alice.link(grammar.ID, "requires", kana.ID)
	alice.mustDo(422, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"x","requires":["`+ossu.Skills[0].ID+`"]}`))
	alice.mustDo(422, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"x","url":"javascript:alert(1)"}`))
	alice.mustDo(422, "PATCH", "/v1/skills/"+kana.ID, []byte(`{"status":"mastered"}`))

	alice.mustDo(200, "PATCH", "/v1/skills/"+kana.ID, []byte(`{"status":"done"}`))
	alice.mustDo(200, "PATCH", "/v1/skills/"+grammar.ID, []byte(`{"status":"skipped"}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+rm.ID, nil), &rm)
	if rm.Done != 1 || rm.Total != 2 || len(rm.Edges) != 2 {
		t.Fatalf("progress = %d/%d, edges %d", rm.Done, rm.Total, len(rm.Edges))
	}
	alice.mustDo(200, "PATCH", "/v1/roadmaps/"+rm.ID, []byte(`{"description":"Pass JLPT N4 in July."}`))

	// Roadmaps connect to goals like everything else.
	var g goalJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Pass JLPT N4"}`)), &g)
	alice.link(rm.ID, "supports", g.ID)
	alice.link(kanji.ID, "supports", g.ID)

	alice.mustDo(204, "DELETE", "/v1/skills/"+grammar.ID, nil)
	alice.mustDo(404, "PATCH", "/v1/skills/"+grammar.ID, []byte(`{"status":"done"}`))
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/roadmaps", nil), []byte("Japanese N4")) {
		t.Fatal("roadmap not listed")
	}
	alice.mustDo(204, "DELETE", "/v1/roadmaps/"+rm.ID, nil)
	alice.mustDo(404, "GET", "/v1/entities/"+kanji.ID, nil)
	alice.mustDo(404, "GET", "/v1/roadmaps/"+rm.ID, nil)
}

func TestLargeRoadmapStaysFast(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	var rm roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Everything"}`)), &rm)
	u := userID(t, alice)
	_, err := pool.Exec(context.Background(), `
		WITH ids AS (SELECT gen_random_uuid() AS id, n FROM generate_series(1, 500) n),
		     ent AS (INSERT INTO entities (id, user_id, type, title) SELECT id, $1, 'skill', 'Skill ' || n FROM ids RETURNING id),
		     sk AS (INSERT INTO skills (id, roadmap_id, position, section) SELECT id, $2, n, 'Part ' || (n / 50) FROM ids RETURNING id)
		INSERT INTO links (id, user_id, source_id, relation, target_id)
		SELECT gen_random_uuid(), $1, a.id, 'requires', b.id FROM ids a JOIN ids b ON b.n = a.n - 1`, u, rm.ID)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+rm.ID, nil), &rm)
	if len(rm.Skills) != 500 || len(rm.Edges) != 499 {
		t.Fatalf("skills %d edges %d", len(rm.Skills), len(rm.Edges))
	}
	if d := time.Since(start); d > 300*time.Millisecond {
		t.Errorf("a 500-skill roadmap took %s", d)
	}
}

// Roadmaps imported before skills were competencies (one skill per book or
// course) can be regrouped: progress and connections survive, and skills
// the person added by hand stay.
func TestRegroupTurnsAnOlderImportIntoSkills(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	var rm roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Python"}`)), &rm)
	if _, err := pool.Exec(context.Background(), `UPDATE roadmaps SET source = 'library' WHERE id = $1`, rm.ID); err != nil {
		t.Fatal(err)
	}
	add := func(title, status string) string {
		var s struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"`+title+`","status":"`+status+`"}`)), &s)
		return s.ID
	}
	tutorial := add("The Python Tutorial", "done")
	add("Python for Everybody", "learning")
	fluent := add("Fluent Python, 2nd edition", "todo")
	alice.link(fluent, "requires", tutorial)
	add("Write a CLI tool", "learning")
	var goal struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Ship a Python package"}`)), &goal)
	alice.link(fluent, "supports", goal.ID)

	alice.mustDo(200, "POST", "/v1/roadmaps/"+rm.ID+"/regroup", nil)
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/roadmaps/"+rm.ID+"/regroup", nil), &rm) // twice changes nothing
	got := map[string]string{}
	var idiomatic string
	for _, s := range rm.Skills {
		got[s.Title] = s.Status
		if s.Title == "Idiomatic Python" {
			idiomatic = s.ID
		}
	}
	want := map[string]string{"Python fundamentals": "learning", "Idiomatic Python": "todo", "Write a CLI tool": "learning",
		"Automation with Python": "todo", "Packaging and distribution": "todo", "Application architecture in Python": "todo"}
	if fmt.Sprint(got) != fmt.Sprint(want) || rm.Skills[len(rm.Skills)-1].Title != "Write a CLI tool" {
		t.Fatalf("regrouped = %v", got)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/entities/"+idiomatic+"/links", nil), []byte(goal.ID)) {
		t.Error("the goal connection did not move to the skill")
	}
	if len(rm.Edges) < 3 {
		t.Errorf("edges = %+v", rm.Edges)
	}
	var manual roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Mine"}`)), &manual)
	alice.mustDo(422, "POST", "/v1/roadmaps/"+manual.ID+"/regroup", nil)
}
