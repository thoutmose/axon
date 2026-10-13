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

	if code, out := alice.do("POST", "/v1/roadmaps/import", []byte(`{"source":"roadmap.sh"}`)); code != 422 || !bytes.Contains(out, []byte("may not be redistributed")) {
		t.Fatalf("roadmap.sh import = %d %s", code, out)
	}
	var other roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Other"}`)), &other)
	var elsewhere struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/"+other.ID+"/skills", []byte(`{"title":"Elsewhere"}`)), &elsewhere)

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
	alice.mustDo(422, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"x","requires":["`+elsewhere.ID+`"]}`))
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

// The skills radar counts acquired skills by area, for each kind, and never
// another person's.
func TestSkillsRadar(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	var soft roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Soft skills for engineers"}`)), &soft)
	for i, sk := range []string{`"1 · Learn","status":"done"`, `"1 · Learn","status":"learning"`, `"2 · Communicate"`, `"2 · Communicate"`,
		`"3 · Craft"`, `"3 · Craft","status":"skipped"`} {
		alice.mustDo(201, "POST", "/v1/roadmaps/"+soft.ID+"/skills", []byte(fmt.Sprintf(`{"title":"Skill %d","kind":"soft","section":%s}`, i, sk)))
	}
	for _, title := range []string{"Go", "SQL", "Linux"} {
		var rm roadmapJSON
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"`+title+`"}`)), &rm)
		alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"Basics","status":"done"}`))
		alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"More"}`))
	}

	type spoke struct {
		Label, Roadmap        string
		Done, Learning, Total int
	}
	var r struct {
		Hard, Soft struct {
			By     string
			Spokes []spoke
		}
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/radar", nil), &r)
	if r.Hard.By != "roadmap" || fmt.Sprint(r.Hard.Spokes) != "[{Go Go 1 0 2} {Linux Linux 1 0 2} {SQL SQL 1 0 2}]" {
		t.Fatalf("hard = %+v", r.Hard)
	}
	// One soft roadmap: a spoke per section; the skipped skill is left out.
	if r.Soft.By != "section" || fmt.Sprint(r.Soft.Spokes) != "[{Learn Soft skills for engineers 1 1 2} {Communicate Soft skills for engineers 0 0 2} {Craft Soft skills for engineers 0 0 1}]" {
		t.Fatalf("soft = %+v", r.Soft)
	}
	if out := bob.mustDo(200, "GET", "/v1/roadmaps/radar", nil); string(out) != `{"hard":{"by":"section","spokes":[]},"soft":{"by":"section","spokes":[]}}`+"\n" {
		t.Fatalf("bob = %s", out)
	}
}

// Starting the Top-5% roadmap (EP-95) starts the whole plan: the goal and
// its measure, a milestone and a project per phase with the exit check on
// its board, the 42 side project, the habits, the first tasks, areas,
// resources, and the roadmap's note, all connected; once only.
func TestStartingTheTop5RoadmapStartsThePlan(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	src := alice.mustDo(200, "GET", "/v1/roadmaps/sources", nil)
	if bytes.Count(src, []byte(`"id":`)) != 1 || !bytes.Contains(src, []byte(`"id":"top5-engineer"`)) {
		t.Fatalf("sources = %s", src)
	}
	var rm roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"top5-engineer"}`)), &rm)
	if rm.Source != "library" || len(rm.Skills) != 32 || len(rm.Edges) < 8 {
		t.Fatalf("roadmap = %d skills, %d edges", len(rm.Skills), len(rm.Edges))
	}
	var radar struct {
		Hard, Soft struct{ Spokes []struct{ Label string } }
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/radar", nil), &radar)
	if len(radar.Hard.Spokes) != 7 || len(radar.Soft.Spokes) != 3 {
		t.Fatalf("radar = %+v", radar)
	}

	var goals struct {
		Items []struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			TargetDate string `json:"target_date"`
			Measure    *struct {
				Unit   string  `json:"unit"`
				Target float64 `json:"target"`
			} `json:"measure"`
			Milestones []struct {
				Title string `json:"title"`
				DueOn string `json:"due_on"`
			} `json:"milestones"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/goals", nil), &goals)
	if len(goals.Items) != 1 || goals.Items[0].Measure == nil || goals.Items[0].Measure.Target != 26 || len(goals.Items[0].Milestones) != 14 {
		t.Fatalf("goals = %+v", goals)
	}
	g := goals.Items[0]
	first, _ := time.Parse(time.DateOnly, g.Milestones[0].DueOn)
	second, _ := time.Parse(time.DateOnly, g.Milestones[1].DueOn)
	if !bytes.HasPrefix([]byte(g.Milestones[0].Title), []byte("Phase 1:")) || second.Sub(first) != 84*24*time.Hour {
		t.Fatalf("milestones = %+v", g.Milestones[:2])
	}

	var projects struct {
		Items []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/projects", nil), &projects)
	var later struct{ Items []struct{} }
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/projects?status=on_hold", nil), &later)
	active, pipekit, side := 0, "", ""
	for _, p := range projects.Items {
		if p.Status == "active" {
			active++
		}
		switch {
		case bytes.HasPrefix([]byte(p.Title), []byte("pipekit")):
			pipekit = p.ID
		case p.Title == "42 cursus (self-taught)":
			side = p.ID
		}
	}
	if len(later.Items) != 13 || active != 2 || pipekit == "" || side == "" {
		t.Fatalf("projects = %+v", projects.Items)
	}
	board := alice.mustDo(200, "GET", "/v1/projects/"+pipekit+"/board", nil)
	for _, want := range []string{`"Phase 1 exit check"`, `"kind":"epic"`, `"OverTheWire Bandit 0–15"`, `"kind":"subtask"`,
		`"DAG model and topological sort with hypothesis tests"`, `"pipekit: create repo (uv, pytest, mypy, ruff, GitHub Actions CI)"`} {
		if !bytes.Contains(board, []byte(want)) {
			t.Errorf("pipekit board lacks %s", want)
		}
	}
	if b := alice.mustDo(200, "GET", "/v1/projects/"+side+"/board", nil); !bytes.Contains(b, []byte(`"libft (~25 h)"`)) {
		t.Errorf("42 board = %s", b)
	}

	habits := alice.mustDo(200, "GET", "/v1/habits?status=active", nil)
	if bytes.Count(habits, []byte(`"title":`)) != 4 || !bytes.Contains(habits, []byte("Weekly reflection and roadmap check")) {
		t.Errorf("habits = %s", habits)
	}
	if next := alice.mustDo(200, "GET", "/v1/tasks?status=next&limit=100", nil); !bytes.Contains(next, []byte("6.042J lectures 1–3")) {
		t.Errorf("next tasks lack the first week")
	}
	if a := alice.mustDo(200, "GET", "/v1/areas", nil); !bytes.Contains(a, []byte("Engineering mastery")) || !bytes.Contains(a, []byte(`"Maths"`)) {
		t.Errorf("areas = %s", a)
	}
	if r := alice.mustDo(200, "GET", "/v1/resources", nil); !bytes.Contains(r, []byte("42 subjects and testers")) {
		t.Errorf("resources = %s", r)
	}
	if n := alice.mustDo(200, "GET", "/v1/notes?q=Domain+coverage+matrix", nil); !bytes.Contains(n, []byte("Roadmaps/top5-engineer.md")) {
		t.Errorf("the roadmap's note is missing: %s", n)
	}
	if links := alice.mustDo(200, "GET", "/v1/entities/"+g.ID+"/links?direction=incoming&limit=100", nil); bytes.Count(links, []byte(`"supports"`)) < 20 {
		t.Errorf("the goal is supported by too little: %s", links)
	}

	// Once only.
	if code, out := alice.do("POST", "/v1/roadmaps/import", []byte(`{"source":"top5-engineer"}`)); code != 422 || !bytes.Contains(out, []byte("already started")) {
		t.Fatalf("second start = %d %s", code, out)
	}
}
