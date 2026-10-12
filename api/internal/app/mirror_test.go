package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/mirror"
)

type vaultNote struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// vault lists the person's notes under Axon/, with their content.
func (c *client) vault() map[string]vaultNote {
	c.e.t.Helper()
	var list struct {
		Items []vaultNote `json:"items"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/notes?limit=100", nil), &list)
	out := map[string]vaultNote{}
	for _, n := range list.Items {
		if strings.HasPrefix(n.Path, "Axon/") {
			var full vaultNote
			_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/notes/"+n.ID, nil), &full)
			out[n.Path] = full
		}
	}
	return out
}

func (c *client) editNote(n vaultNote, path, content string) {
	c.e.t.Helper()
	body, _ := json.Marshal(map[string]string{"path": path, "content": content})
	if code, out := c.do("PUT", "/v1/notes/"+n.ID, body, "If-Match", c.etag("/v1/notes/"+n.ID)); code != 200 {
		c.e.t.Fatalf("edit note: %d %s", code, out)
	}
}

// Tasks, goals, milestones, projects, areas, habits, and media are notes in
// the vault, kept in step both ways; health logs only when turned on, and
// removed when the health consent is withdrawn.
func TestVaultMirror(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	w := mirror.New(pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	step := func() {
		t.Helper()
		for range 3 { // a change and the rewrite it causes
			if err := w.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
	}

	task := alice.createTask(`{"title":"Call the plumber","status":"next","context":"@phone"}`)
	var goal struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Run a 10 km race","outcome":"Finish without walking."}`)), &goal)
	alice.mustDo(201, "POST", "/v1/goals/"+goal.ID+"/milestones", []byte(`{"title":"Run 5 km"}`))
	var habit struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read one page","two_minute":"Open the book","identity":"I am a reader"}`)), &habit)
	alice.mustDo(201, "POST", "/v1/media", []byte(`{"title":"Arrival","kind":"film","status":"done","rating":9}`))
	var area struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/areas", []byte(`{"title":"Health","description":"Sleep 7 hours."}`)), &area)
	alice.mustDo(201, "POST", "/v1/projects", []byte(`{"title":"Garden shed","description":"A small shed."}`))
	step()

	v := alice.vault()
	for path, want := range map[string][]string{
		"Axon/Tasks/Call the plumber.md": {"type: task", "axon_id: " + task.ID, "status: next", `context: "@phone"`, "# Call the plumber", "## Notes"},
		"Axon/Goals/Run a 10 km race.md": {"type: goal", "## Outcome\nFinish without walking.", "- [ ] [[Run 5 km]]"},
		"Axon/Milestones/Run 5 km.md":    {"type: milestone", `goal: "[[Run a 10 km race]]"`, "done: false"},
		"Axon/Habits/Read one page.md":   {"type: habit", "## Identity\nI am a reader", "## Make it easy (two-minute version)\nOpen the book"},
		"Axon/Media/Films/Arrival.md":    {"type: film", "rating: 9", "## Review"},
		"Axon/Areas/Health.md":           {"type: area", "## Description\nSleep 7 hours."},
		"Axon/Projects/Garden shed.md":   {"type: project", "## Description\nA small shed."},
	} {
		n, ok := v[path]
		if !ok {
			t.Fatalf("no note at %s; have %v", path, keys(v))
		}
		for _, s := range want {
			if !strings.Contains(n.Content, s) {
				t.Fatalf("%s lacks %q:\n%s", path, s, n.Content)
			}
		}
	}
	if len(bob.vault()) != 0 {
		t.Fatal("bob sees alice's mirrored notes")
	}
	// Item notes do not count as notes written, and the Notes list can leave them out.
	if out := alice.mustDo(200, "GET", "/v1/notes?mirrored=false&limit=100", nil); bytes.Contains(out, []byte(`"path":"Axon/`)) {
		t.Fatalf("mirrored notes listed: %s", out)
	}
	if out := alice.mustDo(200, "GET", "/v1/dashboard", nil); !bytes.Contains(out, []byte(`"notes":0`)) {
		t.Fatalf("today counts item notes: %s", out)
	}
	var ins struct {
		Metrics []struct {
			ID    string  `json:"id"`
			Total float64 `json:"total"`
		} `json:"metrics"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/insights?weeks=4", nil), &ins)
	for _, m := range ins.Metrics {
		if m.ID == "notes_written" && m.Total != 0 {
			t.Fatalf("insights count %v item notes as written", m.Total)
		}
	}

	// Axon → vault: a renamed task moves its note.
	patch := func(path, body string) {
		t.Helper()
		if code, out := alice.do("PATCH", path, []byte(body), "If-Match", alice.etag(path)); code != 200 {
			t.Fatalf("PATCH %s = %d %s", path, code, out)
		}
	}
	patch("/v1/tasks/"+task.ID, `{"title":"Call the plumber again"}`)
	step()
	v = alice.vault()
	if _, ok := v["Axon/Tasks/Call the plumber again.md"]; !ok {
		t.Fatalf("renamed task note missing: %v", keys(v))
	}

	// Vault → Axon: front matter and sections update the task.
	n := v["Axon/Tasks/Call the plumber again.md"]
	edited := strings.Replace(n.Content, "status: next", "status: waiting", 1)
	edited = strings.Replace(edited, "## Notes\n", "## Notes\nAfter 5 pm.\n", 1)
	alice.editNote(n, n.Path, edited)
	step()
	var got struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks/"+task.ID, nil), &got)
	if got.Status != "waiting" || got.Notes != "After 5 pm." {
		t.Fatalf("task after note edit = %+v", got)
	}

	// Free notes on other items survive regeneration.
	g := alice.vault()["Axon/Goals/Run a 10 km race.md"]
	alice.editNote(g, g.Path, strings.Replace(g.Content, "## Notes\n", "## Notes\nTrain on Tuesdays.\n", 1))
	step()
	patch("/v1/goals/"+goal.ID, `{"outcome":"Finish in under an hour."}`)
	step()
	if c := alice.vault()["Axon/Goals/Run a 10 km race.md"].Content; !strings.Contains(c, "Finish in under an hour.") || !strings.Contains(c, "Train on Tuesdays.") {
		t.Fatalf("goal note lost something:\n%s", c)
	}

	// A new note with a type creates the item.
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Axon/Tasks/Buy milk.md","content":"---\ntype: task\nstatus: next\ndue: 2026-12-01\n---\n# Buy milk\n"}`))
	step()
	var tasksList struct {
		Items []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks?view=next", nil), &tasksList)
	var milk string
	for _, it := range tasksList.Items {
		if it.Title == "Buy milk" {
			milk = it.ID
		}
	}
	if milk == "" {
		t.Fatalf("no task from the new note: %+v", tasksList.Items)
	}
	if c := alice.vault()["Axon/Tasks/Buy milk.md"].Content; !strings.Contains(c, "axon_id: "+milk) || !strings.Contains(c, "due: 2026-12-01") {
		t.Fatalf("new note not adopted:\n%s", c)
	}

	// Moving an area's note into Archive/ archives it.
	a := alice.vault()["Axon/Areas/Health.md"]
	alice.editNote(a, "Axon/Archive/Areas/Health.md", a.Content)
	step()
	if out := alice.mustDo(200, "GET", "/v1/areas/"+area.ID, nil); !bytes.Contains(out, []byte(`"status":"archived"`)) {
		t.Fatalf("area after move = %s", out)
	}

	// Deleting a note deletes the item; deleting an item deletes the note.
	alice.mustDo(204, "DELETE", "/v1/notes/"+alice.vault()["Axon/Tasks/Buy milk.md"].ID, nil)
	step()
	alice.mustDo(404, "GET", "/v1/tasks/"+milk, nil)
	alice.mustDo(204, "DELETE", "/v1/tasks/"+task.ID, nil)
	step()
	if _, ok := alice.vault()["Axon/Tasks/Call the plumber again.md"]; ok {
		t.Fatal("the deleted task's note is still there")
	}

	// A check-in refreshes the habit's note.
	today := time.Now().UTC().Format(time.DateOnly)
	alice.mustDo(200, "PUT", "/v1/habits/"+habit.ID+"/checkins/"+today, []byte(`{"status":"two_minute"}`))
	step()
	if c := alice.vault()["Axon/Habits/Read one page.md"].Content; !strings.Contains(c, ": two-minute") {
		t.Fatalf("habit note without the check-in:\n%s", c)
	}

	// Health: only when turned on, and gone when the consent is withdrawn.
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	alice.mustDo(201, "POST", "/v1/health/food", []byte(`{"eaten_on":"`+today+`","meal":"lunch","name":"Lentil soup","kcal":320}`))
	step()
	if _, ok := alice.vault()["Axon/Health/Food/Food "+today+".md"]; ok {
		t.Fatal("health mirrored without being turned on")
	}
	setVault := func(mirrorOn, healthOn bool) {
		t.Helper()
		var doc map[string]any
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
		delete(doc, "ai")
		doc["vault"] = map[string]any{"mirror": mirrorOn, "folder": "Axon", "health": healthOn}
		body, _ := json.Marshal(doc)
		alice.mustDo(200, "PUT", "/v1/me/settings", body)
	}
	setVault(true, true)
	step()
	if c := alice.vault()["Axon/Health/Food/Food "+today+".md"].Content; !strings.Contains(c, "| lunch | Lentil soup |") {
		t.Fatalf("food log note = %q", c)
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":false}`))
	step()
	for p := range alice.vault() {
		if strings.HasPrefix(p, "Axon/Health/") {
			t.Fatalf("health note %s kept after withdrawal", p)
		}
	}

	// Turned off: new items get no note.
	setVault(false, false)
	alice.createTask(`{"title":"Not mirrored","status":"next"}`)
	step()
	if _, ok := alice.vault()["Axon/Tasks/Not mirrored.md"]; ok {
		t.Fatal("mirrored while turned off")
	}
}

func keys(m map[string]vaultNote) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
