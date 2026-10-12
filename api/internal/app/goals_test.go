package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

type goalJSON struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	TargetDate *string `json:"target_date"`
	Progress   struct {
		Percent *int   `json:"percent"`
		Method  string `json:"method"`
		Done    int    `json:"done"`
		Total   int    `json:"total"`
		Summary string `json:"summary"`
	} `json:"progress"`
	Milestones []struct {
		ID    string  `json:"id"`
		DueOn *string `json:"due_on"`
	} `json:"milestones"`
}

type explanationJSON struct {
	Percent *int   `json:"percent"`
	Method  string `json:"method"`
	Items   []struct {
		ID    string  `json:"id"`
		Type  string  `json:"type"`
		Title string  `json:"title"`
		Done  bool    `json:"done"`
		Via   *string `json:"via"`
	} `json:"items"`
}

func (c *client) goal(id string) goalJSON {
	c.e.t.Helper()
	var g goalJSON
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/goals/"+id, nil), &g)
	return g
}

func (c *client) link(source, relation, target string) string {
	c.e.t.Helper()
	var l struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/links", []byte(`{"source_id":"`+source+`","relation":"`+relation+`","target_id":"`+target+`"}`)), &l)
	return l.ID
}

func TestGoalProgressIsExplainedByItsInputs(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	var g goalJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Pass JLPT N4","outcome":"Read NHK Easy without a dictionary"}`)), &g)
	// Nothing is invented: no target date, no milestones, nothing to measure.
	if g.TargetDate != nil || len(g.Milestones) != 0 || g.Progress.Method != "none" || g.Progress.Percent != nil {
		t.Fatalf("new goal = %+v", g)
	}

	var kana, kanji struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals/"+g.ID+"/milestones", []byte(`{"title":"Kana"}`)), &kana)
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals/"+g.ID+"/milestones", []byte(`{"title":"300 kanji","due_on":"2027-03-01"}`)), &kanji)
	drill := alice.createTask(`{"title":"Drill katakana","status":"next"}`)
	deck := alice.createTask(`{"title":"Make kanji deck","status":"next"}`)
	dropped := alice.createTask(`{"title":"Buy textbook","status":"cancelled"}`)
	alice.link(drill.ID, "supports", g.ID)
	alice.link(deck.ID, "supports", kanji.ID)
	alice.link(dropped.ID, "supports", g.ID)
	// A note can support a goal as context; it does not count.
	var n noteJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"content":"# Study plan"}`)), &n)
	alice.link(n.ID, "supports", g.ID)

	alice.patchTask(drill.ID, `{"status":"done"}`)
	etag := alice.etag("/v1/milestones/" + kana.ID)
	if code, out := alice.do("PATCH", "/v1/milestones/"+kana.ID, []byte(`{"done":true}`), "If-Match", etag); code != 200 {
		t.Fatalf("complete milestone = %d %s", code, out)
	}

	g = alice.goal(g.ID)
	if g.Progress.Method != "items" || *g.Progress.Percent != 50 || g.Progress.Done != 2 || g.Progress.Total != 4 {
		t.Fatalf("progress = %+v", g.Progress)
	}
	var ex explanationJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/goals/"+g.ID+"/progress", nil), &ex)
	got := map[string]bool{}
	for _, it := range ex.Items {
		got[it.Title] = it.Done
		if it.Title == "Make kanji deck" && (it.Via == nil || *it.Via != "300 kanji") {
			t.Errorf("task via milestone = %+v", it)
		}
	}
	want := map[string]bool{"Kana": true, "300 kanji": false, "Drill katakana": true, "Make kanji deck": false}
	if len(got) != len(want) {
		t.Fatalf("explanation items = %+v", ex.Items)
	}
	for k, v := range want {
		if d, ok := got[k]; !ok || d != v {
			t.Fatalf("explanation items = %+v", ex.Items)
		}
	}

	// A measure takes over and says so.
	etag = alice.etag("/v1/goals/" + g.ID)
	if code, out := alice.do("PATCH", "/v1/goals/"+g.ID, []byte(`{"measure":{"unit":"kanji","start":0,"target":300,"current":75},"target_date":"2027-07-05"}`), "If-Match", etag); code != 200 {
		t.Fatalf("set measure = %d %s", code, out)
	}
	g = alice.goal(g.ID)
	if g.Progress.Method != "measure" || *g.Progress.Percent != 25 || g.Progress.Summary != "75 of 300 kanji (from 0)" || *g.TargetDate != "2027-07-05" {
		t.Fatalf("measured progress = %+v", g)
	}
	alice.mustDo(200, "GET", "/v1/goals?status=active", nil)
	alice.mustDo(400, "GET", "/v1/goals?status=someday", nil)
}

func TestArchivingAGoalKeepsHistoryAndLinks(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	var g goalJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Run 10 km","target_date":"2027-05-01","measure":{"unit":"km","start":0,"target":10,"current":3}}`)), &g)
	run := alice.createTask(`{"title":"Easy run","status":"next"}`)
	alice.link(run.ID, "supports", g.ID)
	var m struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals/"+g.ID+"/milestones", []byte(`{"title":"5 km"}`)), &m)

	etag := alice.etag("/v1/goals/" + g.ID)
	if code, _ := alice.do("PATCH", "/v1/goals/"+g.ID, []byte(`{"status":"archived"}`), "If-Match", `"stale"`); code != 412 {
		t.Fatalf("stale archive = %d", code)
	}
	if code, out := alice.do("PATCH", "/v1/goals/"+g.ID, []byte(`{"status":"archived"}`), "If-Match", etag); code != 200 {
		t.Fatalf("archive = %d %s", code, out)
	}
	var list struct {
		Items []goalJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/goals", nil), &list)
	if len(list.Items) != 0 {
		t.Fatalf("archived goal still active: %+v", list.Items)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/goals?status=archived", nil), &list)
	if len(list.Items) != 1 || len(list.Items[0].Milestones) != 1 || *list.Items[0].Progress.Percent != 30 {
		t.Fatalf("archived list = %+v", list.Items)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/entities/"+g.ID+"/links?direction=incoming", nil), []byte(run.ID)) {
		t.Fatal("archiving dropped the task link")
	}
	timeline := alice.mustDo(200, "GET", "/v1/entities/"+g.ID+"/timeline", nil)
	if !bytes.Contains(timeline, []byte(`"created"`)) || !bytes.Contains(timeline, []byte(`"status"`)) {
		t.Fatalf("timeline = %s", timeline)
	}

	// Validation, then deletion takes the milestones with it.
	alice.mustDo(422, "POST", "/v1/goals", []byte(`{"title":"x","measure":{"unit":"km","start":5,"target":5,"current":5}}`))
	alice.mustDo(422, "POST", "/v1/goals", []byte(`{"title":" "}`))
	alice.mustDo(422, "POST", "/v1/goals/"+g.ID+"/milestones", []byte(`{"title":""}`))
	alice.mustDo(404, "POST", "/v1/goals/"+run.ID+"/milestones", []byte(`{"title":"not a goal"}`))
	alice.mustDo(200, "GET", "/v1/milestones/"+m.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/milestones/"+m.ID, nil)
	alice.mustDo(404, "GET", "/v1/milestones/"+m.ID, nil)
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals/"+g.ID+"/milestones", []byte(`{"title":"8 km"}`)), &m)
	alice.mustDo(204, "DELETE", "/v1/goals/"+g.ID, nil)
	alice.mustDo(404, "GET", "/v1/goals/"+g.ID, nil)
	alice.mustDo(404, "GET", "/v1/entities/"+m.ID, nil)
	alice.mustDo(200, "GET", "/v1/tasks/"+run.ID, nil)
}

func TestGoalsAreScopedToTheirOwner(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	var g goalJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Private goal"}`)), &g)
	bob.mustDo(404, "GET", "/v1/goals/"+g.ID, nil)
	bob.mustDo(404, "GET", "/v1/goals/"+g.ID+"/progress", nil)
	bob.mustDo(404, "POST", "/v1/goals/"+g.ID+"/milestones", []byte(`{"title":"sneaky"}`))
	bob.mustDo(404, "DELETE", "/v1/goals/"+g.ID, nil)
}
