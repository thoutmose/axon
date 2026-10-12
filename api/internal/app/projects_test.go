package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

type cardJSON struct {
	TaskID   string  `json:"task_id"`
	Title    string  `json:"title"`
	Status   string  `json:"status"`
	Column   string  `json:"column"`
	Kind     string  `json:"kind"`
	ParentID *string `json:"parent_id"`
	Done     int     `json:"children_done"`
	Total    int     `json:"children"`
}

type boardJSON struct {
	Project struct {
		ID         string `json:"id"`
		DocsFolder string `json:"docs_folder"`
		WIPLimit   *int   `json:"wip_limit"`
		Done       int    `json:"done"`
		Total      int    `json:"total"`
	} `json:"project"`
	Columns []struct {
		ID    string     `json:"id"`
		Limit *int       `json:"limit"`
		Cards []cardJSON `json:"cards"`
	} `json:"columns"`
}

func (c *client) board(id string) boardJSON {
	c.e.t.Helper()
	var b boardJSON
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/projects/"+id+"/board", nil), &b)
	return b
}

func (b boardJSON) column(id string) []string {
	for _, c := range b.Columns {
		if c.ID == id {
			var out []string
			for _, card := range c.Cards {
				out = append(out, card.Title)
			}
			return out
		}
	}
	return nil
}

func (c *client) card(project, body string) cardJSON {
	c.e.t.Helper()
	var card cardJSON
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/projects/"+project+"/cards", []byte(body)), &card)
	return card
}

func TestProjectBoardHierarchyAndWIP(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	var p struct {
		ID         string `json:"id"`
		DocsFolder string `json:"docs_folder"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/projects", []byte(`{"title":"Axon mobile app","wip_limit":1}`)), &p)
	if p.DocsFolder != "1-Projects/axon-mobile-app" {
		t.Fatalf("docs folder = %s", p.DocsFolder)
	}
	epic := alice.card(p.ID, `{"title":"Offline mode","kind":"epic","column":"backlog"}`)
	issue := alice.card(p.ID, `{"title":"Cache notes","kind":"issue","parent_id":"`+epic.TaskID+`"}`)
	sub := alice.card(p.ID, `{"title":"Pick storage","kind":"subtask","parent_id":"`+issue.TaskID+`"}`)
	other := alice.card(p.ID, `{"title":"App icon"}`)
	alice.mustDo(422, "POST", "/v1/projects/"+p.ID+"/cards", []byte(`{"title":"x","kind":"subtask"}`))
	alice.mustDo(422, "POST", "/v1/projects/"+p.ID+"/cards", []byte(`{"title":"x","kind":"issue","parent_id":"`+issue.TaskID+`"}`))
	alice.mustDo(422, "POST", "/v1/projects/"+p.ID+"/cards", []byte(`{"title":"x","kind":"epic","parent_id":"`+epic.TaskID+`"}`))
	alice.mustDo(422, "POST", "/v1/projects/"+p.ID+"/cards", []byte(`{"title":"x","column":"someday"}`))

	b := alice.board(p.ID)
	if got := b.column("todo"); len(got) != 3 || b.column("backlog")[0] != "Offline mode" {
		t.Fatalf("board = %+v", b)
	}
	if *b.Columns[2].Limit != 1 {
		t.Fatalf("doing limit = %+v", b.Columns[2].Limit)
	}

	// WIP: the second card into Doing needs confirmation, never a silent block.
	move := func(task, body string) (int, []byte) {
		return alice.do("PUT", "/v1/projects/"+p.ID+"/cards/"+task, []byte(body))
	}
	if code, out := move(issue.TaskID, `{"column":"doing"}`); code != 200 {
		t.Fatalf("first move = %d %s", code, out)
	}
	if code, out := move(other.TaskID, `{"column":"doing"}`); code != 409 || !bytes.Contains(out, []byte("WIP limit")) {
		t.Fatalf("over limit = %d %s", code, out)
	}
	if code, _ := move(other.TaskID, `{"column":"doing","over_limit":true}`); code != 200 {
		t.Fatal("confirmed move refused")
	}

	// Done on the board and in the task list stay the same thing, both ways.
	move(sub.TaskID, `{"column":"done"}`)
	var task taskJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks/"+sub.TaskID, nil), &task)
	if task.Status != "done" {
		t.Fatalf("sub-task status = %s", task.Status)
	}
	alice.patchTask(other.TaskID, `{"status":"done"}`)
	b = alice.board(p.ID)
	if got := b.column("done"); len(got) != 2 {
		t.Fatalf("done column = %v", got)
	}
	for _, c := range b.Columns[2].Cards {
		if c.TaskID == issue.TaskID && (c.Done != 1 || c.Total != 1) {
			t.Fatalf("issue children = %+v", c)
		}
	}
	move(other.TaskID, `{"column":"todo"}`)
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks/"+other.TaskID, nil), &task)
	if task.Status != "next" {
		t.Fatalf("reopened status = %s", task.Status)
	}

	// Reordering, attaching an existing task, detaching it.
	first := alice.card(p.ID, `{"title":"Splash screen"}`)
	move(first.TaskID, `{"before":"`+other.TaskID+`"}`)
	if got := alice.board(p.ID).column("todo"); len(got) != 2 || got[0] != "Splash screen" {
		t.Fatalf("todo order = %v", got)
	}
	loose := alice.createTask(`{"title":"Write privacy policy","status":"next"}`)
	if code, _ := move(loose.ID, `{"column":"backlog"}`); code != 200 {
		t.Fatal("attach refused")
	}
	alice.mustDo(204, "DELETE", "/v1/cards/"+loose.ID, nil)
	alice.mustDo(200, "GET", "/v1/tasks/"+loose.ID, nil)

	// Docs are the notes in the project's folder.
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"1-Projects/axon-mobile-app/spec.md","content":"# Spec"}`))
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"1-Projects/other/spec.md","content":"# Other"}`))
	docs := alice.mustDo(200, "GET", "/v1/projects/"+p.ID+"/docs", nil)
	if !bytes.Contains(docs, []byte("axon-mobile-app/spec.md")) || bytes.Contains(docs, []byte("other/spec.md")) {
		t.Fatalf("docs = %s", docs)
	}

	// A project can support a goal; settings use If-Match.
	var g goalJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Ship v1"}`)), &g)
	alice.link(p.ID, "supports", g.ID)
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/projects/"+p.ID, nil), []byte(`"status":"active"`)) {
		t.Fatal("project not readable")
	}
	etag := alice.etag("/v1/projects/" + p.ID)
	if code, out := alice.do("PATCH", "/v1/projects/"+p.ID, []byte(`{"wip_limit":null,"status":"on_hold"}`), "If-Match", etag); code != 200 {
		t.Fatalf("patch = %d %s", code, out)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/projects?status=on_hold", nil), []byte(p.ID)) {
		t.Fatal("on hold project not listed")
	}
	alice.mustDo(400, "GET", "/v1/projects?status=gone", nil)

	// Deleting the project keeps its tasks.
	alice.mustDo(204, "DELETE", "/v1/projects/"+p.ID, nil)
	alice.mustDo(404, "GET", "/v1/projects/"+p.ID+"/board", nil)
	alice.mustDo(200, "GET", "/v1/tasks/"+epic.TaskID, nil)
	alice.mustDo(404, "DELETE", "/v1/cards/"+epic.TaskID, nil)
	e.client().mustDo(401, "GET", "/v1/projects", nil)
}
