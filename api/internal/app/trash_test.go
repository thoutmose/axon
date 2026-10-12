package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type trashJSON struct {
	Items []struct {
		ID       string `json:"id"`
		EntityID string `json:"entity_id"`
		Type     string `json:"type"`
		Title    string `json:"title"`
		With     int    `json:"with"`
	} `json:"items"`
}

func (c *client) trash() trashJSON {
	c.e.t.Helper()
	var tr trashJSON
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/trash", nil), &tr)
	return tr
}

// Deleting keeps what was removed for 30 days; Undo and the trash put it
// back with what came with it.
func TestTrash(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	// A note: undo brings it back, and the vault sync sees it again.
	var note struct{ ID string }
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/Spacing.md","content":"# Spacing\n\nSee [[Retrieval]]."}`)), &note)
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/Retrieval.md","content":"# Retrieval\n"}`))
	alice.mustDo(204, "DELETE", "/v1/notes/"+note.ID, nil)
	if tr := alice.trash(); len(tr.Items) != 1 || tr.Items[0].Title != "Spacing" || tr.Items[0].Type != "note" {
		t.Fatalf("trash = %+v", tr)
	}
	bob.mustDo(404, "POST", "/v1/trash/undo", []byte(`{"entity_id":"`+note.ID+`"}`))
	alice.mustDo(200, "POST", "/v1/trash/undo", []byte(`{"entity_id":"`+note.ID+`"}`))
	if out := alice.mustDo(200, "GET", "/v1/notes/"+note.ID, nil); !strings.Contains(string(out), "See [[Retrieval]]") {
		t.Fatalf("restored note = %s", out)
	}
	if out := alice.mustDo(200, "GET", "/v1/entities/"+note.ID+"/links", nil); !strings.Contains(string(out), "references") {
		t.Fatalf("restored links = %s", out)
	}
	if out := alice.mustDo(200, "GET", "/v1/notes/sync", nil); strings.Contains(string(out), `"tombstones":[{"path":"Zettel/Spacing.md"`) || !strings.Contains(string(out), "Zettel/Spacing.md") {
		t.Fatalf("sync after restore = %s", out)
	}
	if len(alice.trash().Items) != 0 {
		t.Fatal("restored note still in the trash")
	}

	// A goal and its milestones are one deletion, and come back together.
	var goal struct{ ID string }
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Run 10 km"}`)), &goal)
	alice.mustDo(201, "POST", "/v1/goals/"+goal.ID+"/milestones", []byte(`{"title":"Run 3 km"}`))
	alice.mustDo(201, "POST", "/v1/goals/"+goal.ID+"/milestones", []byte(`{"title":"Run 5 km"}`))
	alice.mustDo(204, "DELETE", "/v1/goals/"+goal.ID, nil)
	tr := alice.trash()
	if len(tr.Items) != 1 || tr.Items[0].Title != "Run 10 km" || tr.Items[0].With != 2 {
		t.Fatalf("goal in trash = %+v", tr)
	}
	alice.mustDo(200, "POST", "/v1/trash/"+tr.Items[0].ID+"/restore", nil)
	if out := alice.mustDo(200, "GET", "/v1/goals/"+goal.ID, nil); !strings.Contains(string(out), "Run 3 km") || !strings.Contains(string(out), "Run 5 km") {
		t.Fatalf("restored goal = %s", out)
	}

	// A habit's check-ins and a card's reviews come back with them.
	var habit struct{ ID string }
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read one page"}`)), &habit)
	today := time.Now().UTC().Format(time.DateOnly)
	alice.mustDo(200, "PUT", "/v1/habits/"+habit.ID+"/checkins/"+today, []byte(`{"status":"done"}`))
	alice.mustDo(204, "DELETE", "/v1/habits/"+habit.ID, nil)
	alice.mustDo(200, "POST", "/v1/trash/undo", []byte(`{"entity_id":"`+habit.ID+`"}`))
	if out := alice.mustDo(200, "GET", "/v1/habits/"+habit.ID+"/checkins?from="+today+"&to="+today, nil); !strings.Contains(string(out), `"done"`) {
		t.Fatalf("restored check-ins = %s", out)
	}
	var card struct{ ID string }
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/flashcards", []byte(`{"front":"Why space?","back":"Effort"}`)), &card)
	alice.mustDo(200, "POST", "/v1/flashcards/"+card.ID+"/review", []byte(`{"rating":3}`))
	alice.mustDo(204, "DELETE", "/v1/flashcards/"+card.ID, nil)
	alice.mustDo(200, "POST", "/v1/trash/undo", []byte(`{"entity_id":"`+card.ID+`"}`))
	if out := alice.mustDo(200, "GET", "/v1/flashcards/"+card.ID, nil); !strings.Contains(string(out), `"reps":1`) {
		t.Fatalf("restored card = %s", out)
	}

	// A path taken meanwhile: said, nothing half-restored.
	alice.mustDo(204, "DELETE", "/v1/notes/"+note.ID, nil)
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/Spacing.md","content":"# Spacing, again\n"}`))
	alice.mustDo(409, "POST", "/v1/trash/undo", []byte(`{"entity_id":"`+note.ID+`"}`))
	alice.mustDo(404, "GET", "/v1/notes/"+note.ID, nil)

	// Bulk deletions are one line; forget one, empty the rest.
	a := alice.createTask(`{"title":"Task A"}`).ID
	b := alice.createTask(`{"title":"Task B"}`).ID
	alice.mustDo(200, "POST", "/v1/tasks/bulk", []byte(`{"ids":["`+a+`","`+b+`"],"delete":true}`))
	tr = alice.trash()
	if tr.Items[0].With != 1 {
		t.Fatalf("bulk deletion = %+v", tr.Items[0])
	}
	bob.mustDo(404, "DELETE", "/v1/trash/"+tr.Items[0].ID, nil)
	alice.mustDo(204, "DELETE", "/v1/trash/"+tr.Items[0].ID, nil)
	alice.mustDo(204, "DELETE", "/v1/trash", nil)
	if len(alice.trash().Items) != 0 {
		t.Fatal("trash not empty")
	}
}
