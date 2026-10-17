package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	// Its plan can be read before it is started: the phases at the dates
	// starting it today gives, each with its project and exams (EP-142).
	var preview struct {
		Started bool   `json:"started"`
		Start   string `json:"start"`
		Phases  []struct {
			Milestone *struct{} `json:"milestone"`
			Project   *struct {
				ID    *string `json:"id"`
				Title string  `json:"title"`
			} `json:"project"`
			Exit  []struct{ Title string } `json:"exit"`
			Certs []struct{ Code string }  `json:"certifications"`
		} `json:"phases"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/sources/top5-engineer/plan", nil), &preview)
	if preview.Started || len(preview.Phases) != 18 || preview.Phases[0].Milestone != nil || preview.Phases[0].Project == nil ||
		preview.Phases[0].Project.ID != nil || !strings.HasPrefix(preview.Phases[0].Project.Title, "pipekit") || len(preview.Phases[0].Certs) != 1 {
		t.Fatalf("preview = %+v", preview)
	}
	if last := preview.Phases[0].Exit[len(preview.Phases[0].Exit)-1].Title; !strings.HasPrefix(last, "Skills assessment retaken") {
		t.Errorf("phase 1 ends with %q", last)
	}
	alice.mustDo(404, "GET", "/v1/roadmaps/sources/nope/plan", nil)
	if out := alice.mustDo(200, "GET", "/v1/roadmaps/today", nil); string(out) != `{"plan":null}`+"\n" && string(out) != `{"plan":null}` {
		t.Fatalf("today before a start = %s", out)
	}

	var rm roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"top5-engineer"}`)), &rm)
	if rm.Source != "library" || len(rm.Skills) != 43 || len(rm.Edges) < 8 {
		t.Fatalf("roadmap = %d skills, %d edges", len(rm.Skills), len(rm.Edges))
	}
	var radar struct {
		Hard, Soft struct{ Spokes []struct{ Label string } }
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/radar", nil), &radar)
	if len(radar.Hard.Spokes) != 8 || len(radar.Soft.Spokes) != 3 {
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
	if len(goals.Items) != 1 || goals.Items[0].Measure == nil || goals.Items[0].Measure.Target != 37 || len(goals.Items[0].Milestones) != 18 {
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
	if len(later.Items) != 17 || active != 2 || pipekit == "" || side == "" {
		t.Fatalf("projects = %+v", projects.Items)
	}
	board := alice.mustDo(200, "GET", "/v1/projects/"+pipekit+"/board", nil)
	for _, want := range []string{`"Phase 1 exit check"`, `"kind":"epic"`, `"OverTheWire Bandit 0–15"`, `"kind":"subtask"`,
		`"Scheduler with retries"`, `"pipekit: create repo (uv, pytest, mypy, ruff, GitHub Actions CI)"`} {
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

	// The weekly rhythm is on the calendar, with the first phase's tracks (EP-97).
	start := first.AddDate(0, 0, -83)
	var cal struct {
		Items []struct {
			Date      string `json:"date"`
			Type      string `json:"type"`
			Title     string `json:"title"`
			RoadmapID string `json:"roadmap_id"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/calendar?types=study&from="+start.Format(time.DateOnly)+"&to="+start.AddDate(0, 0, 6).Format(time.DateOnly), nil), &cal)
	if len(cal.Items) != 7 || cal.Items[0].Title != "Theory: MIT 6.042J discrete maths · 2–3 h" || cal.Items[0].RoadmapID != rm.ID {
		t.Fatalf("study blocks = %+v", cal.Items)
	}
	if n := alice.mustDo(200, "GET", "/v1/notes?q=mini+Airflow", nil); !bytes.Contains(n, []byte("1-Projects/pipekit/pipekit.md")) {
		t.Errorf("pipekit's project note is missing: %.300s", n)
	}

	// The plan's 45 certifications, planned for the end of their phases (EP-104, EP-141).
	var certs struct {
		Items []certJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/certifications", nil), &certs)
	if len(certs.Items) != 45 || certs.Items[0].TargetOn == nil || certs.Items[0].Status != "planned" {
		t.Fatalf("certifications = %+v", certs.Items)
	}

	// The roadmap's page shows the plan: each phase as its milestone, with
	// its project's board and the exams at its end (EP-142).
	var plan struct {
		Started bool    `json:"started"`
		GoalID  *string `json:"goal_id"`
		Phases  []struct {
			Title     string `json:"title"`
			State     string `json:"state"`
			Milestone *struct {
				ID   string `json:"id"`
				Done bool   `json:"done"`
			} `json:"milestone"`
			Project *struct {
				ID          string `json:"id"`
				Done, Total int
			} `json:"project"`
			Certs []struct {
				Code   string  `json:"code"`
				ID     *string `json:"id"`
				Status string  `json:"status"`
			} `json:"certifications"`
		} `json:"phases"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+rm.ID+"/plan", nil), &plan)
	if !plan.Started || plan.GoalID == nil || *plan.GoalID != g.ID || len(plan.Phases) != 18 {
		t.Fatalf("plan = %+v", plan)
	}
	p1 := plan.Phases[0]
	if p1.Milestone == nil || p1.Milestone.Done || p1.Project == nil || p1.Project.ID != pipekit || p1.Project.Total == 0 ||
		len(p1.Certs) != 1 || p1.Certs[0].Code != "PCPP1" || p1.Certs[0].ID == nil || p1.Certs[0].Status != "planned" {
		t.Fatalf("phase 1 = %+v", p1)
	}
	for _, ph := range plan.Phases {
		if ph.Milestone == nil || ph.Project == nil {
			t.Errorf("%s lacks its milestone or project", ph.Title)
		}
	}
	// Today's part of it (EP-145): phase 1, its milestone ahead, PCPP1 next.
	var today struct {
		Plan *struct {
			RoadmapID string `json:"roadmap_id"`
			Phases    int    `json:"phases"`
			Phase     *struct {
				N int `json:"n"`
			} `json:"phase"`
			Weeks    int `json:"weeks"`
			DaysLeft int `json:"days_left"`
			NextCert *struct {
				Code string `json:"code"`
			} `json:"next_certification"`
		} `json:"plan"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/today", nil), &today)
	if today.Plan == nil || today.Plan.RoadmapID != rm.ID || today.Plan.Phases != 18 || today.Plan.Phase == nil ||
		today.Plan.Weeks != 12 || today.Plan.DaysLeft <= 0 || today.Plan.NextCert == nil {
		t.Fatalf("today = %+v", today.Plan)
	}
	if out := alice.mustDo(200, "GET", "/v1/roadmaps/"+mine(t, alice)+"/plan", nil); !bytes.Contains(out, []byte(`"started":false`)) {
		t.Errorf("a roadmap without a plan = %s", out)
	}

	// Once only.
	if code, out := alice.do("POST", "/v1/roadmaps/import", []byte(`{"source":"top5-engineer"}`)); code != 422 || !bytes.Contains(out, []byte("already started")) {
		t.Fatalf("second start = %d %s", code, out)
	}

	// A roadmap started from an earlier version of the plan says so;
	// starting the new version keeps the exams and habits it already
	// planned, and the calendar shows the new one only.
	ctx := context.Background()
	// Version 1 shared phase titles with version 2, so only the stored
	// version tells them apart (EP-146).
	if _, err := pool.Exec(ctx, `UPDATE roadmaps SET plan_version = 1 WHERE id = $1`, rm.ID); err != nil {
		t.Fatal(err)
	}
	if l := alice.mustDo(200, "GET", "/v1/roadmaps", nil); !bytes.Contains(l, []byte(`"plan_id":"top5-engineer","plan_current":false`)) {
		t.Fatalf("the earlier roadmap in the list = %.400s", l)
	}
	var old struct {
		Outdated bool    `json:"outdated"`
		Newer    *string `json:"newer"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+rm.ID+"/plan", nil), &old)
	if !old.Outdated || old.Newer != nil {
		t.Fatalf("old plan = %+v", old)
	}
	var again roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"top5-engineer"}`)), &again)
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+rm.ID+"/plan", nil), &old)
	if old.Newer == nil || *old.Newer != again.ID {
		t.Fatalf("old plan after the new start = %+v", old)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/certifications", nil), &certs)
	if len(certs.Items) != 45 {
		t.Errorf("certifications after the new start = %d", len(certs.Items))
	}
	if h := alice.mustDo(200, "GET", "/v1/habits?status=active", nil); bytes.Count(h, []byte(`"title":`)) != 4 {
		t.Errorf("habits after the new start = %s", h)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/calendar?types=study&from="+start.Format(time.DateOnly)+"&to="+start.AddDate(0, 0, 6).Format(time.DateOnly), nil), &cal)
	if len(cal.Items) != 7 || cal.Items[0].RoadmapID != again.ID {
		t.Errorf("study blocks after the new start = %+v", cal.Items)
	}
}

// Study hours against the plan (EP-105): focus sessions on the roadmap, its
// skills, or its plan's projects and tasks count; others do not.
func TestStudyHoursAgainstThePlan(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	var rm roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"top5-engineer"}`)), &rm)
	var dsa string
	for _, s := range rm.Skills {
		if s.Title == "Data structures and algorithms" {
			dsa = s.ID
		}
	}
	var projects struct {
		Items []struct{ ID, Title string } `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/projects", nil), &projects)
	pipekit := ""
	for _, p := range projects.Items {
		if strings.HasPrefix(p.Title, "pipekit") {
			pipekit = p.ID
		}
	}
	focus := func(target string, minutes int) {
		t.Helper()
		body := `{"planned_minutes":25}`
		if target != "" {
			body = `{"planned_minutes":25,"target_id":"` + target + `"}`
		}
		var s struct{ ID string }
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/focus-sessions", []byte(body)), &s)
		// Started long enough ago to have worked that long.
		if _, err := pool.Exec(context.Background(), `UPDATE focus_sessions SET started_at = now() - make_interval(mins => $2) WHERE id = $1`, s.ID, minutes+1); err != nil {
			t.Fatal(err)
		}
		alice.mustDo(200, "POST", "/v1/focus-sessions/"+s.ID+"/finish", []byte(fmt.Sprintf(`{"worked_minutes":%d,"rounds":1,"status":"completed"}`, minutes)))
	}
	focus(dsa, 50)
	focus(rm.ID, 30)
	focus(pipekit, 40)
	focus("", 25) // a session for nothing in particular: not counted

	var h struct {
		TargetHours float64 `json:"target_hours"`
		Weeks       []struct {
			Week    string `json:"week"`
			Minutes int    `json:"minutes"`
		} `json:"weeks"`
		Days []struct {
			Date    string `json:"date"`
			Block   string `json:"block"`
			Minutes int    `json:"minutes"`
		} `json:"days"`
		Skills []struct {
			Title   string `json:"title"`
			Minutes int    `json:"minutes"`
		} `json:"skills"`
		Projects []struct {
			Title   string `json:"title"`
			Minutes int    `json:"minutes"`
		} `json:"projects"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+rm.ID+"/hours", nil), &h)
	if h.TargetHours != 22 || len(h.Weeks) != 12 || h.Weeks[11].Minutes != 120 || len(h.Days) != 7 {
		t.Fatalf("hours = %+v", h)
	}
	if len(h.Skills) != 1 || h.Skills[0].Minutes != 50 || len(h.Projects) != 1 || h.Projects[0].Minutes != 40 {
		t.Fatalf("by skill %+v, by project %+v", h.Skills, h.Projects)
	}
	var mine roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Mine"}`)), &mine)
	if out := alice.mustDo(200, "GET", "/v1/roadmaps/"+mine.ID+"/hours", nil); !bytes.Contains(out, []byte(`"target_hours":null`)) {
		t.Fatalf("a roadmap without a plan = %s", out)
	}
}

// mine creates a roadmap by hand and returns its id.
func mine(t *testing.T, c *client) string {
	t.Helper()
	var rm roadmapJSON
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"By hand"}`)), &rm)
	return rm.ID
}
