package app

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

type progressJSON struct {
	Enabled bool `json:"enabled"`
	XP      int  `json:"xp"`
	Level   int  `json:"level"`
	Points  int  `json:"points"`
	Streak  struct {
		State   string `json:"state"`
		Days    int    `json:"days"`
		Freezes int    `json:"freezes"`
	} `json:"streak"`
	Badges []struct {
		ID       string  `json:"id"`
		EarnedAt *string `json:"earned_at"`
	} `json:"badges"`
}

func TestProgressIsOptInTransparentAndFedByEveryArea(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	// Off by default: no numbers, no ledger, no freezes.
	if out := alice.mustDo(200, "GET", "/v1/progress", nil); string(bytes.TrimSpace(out)) != `{"enabled":false}` {
		t.Fatalf("progress while off = %s", out)
	}
	alice.mustDo(409, "GET", "/v1/progress/ledger", nil)
	alice.mustDo(409, "POST", "/v1/progress/freezes", nil)
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/progress/rules", nil), []byte(`"freeze_cost":100`)) {
		t.Fatal("rules not listed")
	}

	if code, out := alice.putYAML("version: 1\ngamification:\n  enabled: true\n"); code != 200 {
		t.Fatalf("turn on = %d %s", code, out)
	}

	// Points from several areas.
	task := alice.createTask(`{"title":"Ship it","due_on":"` + time.Now().AddDate(0, 0, 1).Format(time.DateOnly) + `"}`)
	alice.patchTask(task.ID, `{"status":"done"}`, "If-Match", alice.etag("/v1/tasks/"+task.ID))
	var h habitJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read"}`)), &h)
	alice.mustDo(200, "PUT", "/v1/habits/"+h.ID+"/checkins/"+h.State.Today, []byte(`{"status":"done"}`))
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/permanent/x.md","content":"---\ntype: permanent\n---\n# X"}`))
	var c struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/flashcards", []byte(`{"front":"Q?","back":"A"}`)), &c)
	alice.mustDo(200, "POST", "/v1/flashcards/"+c.ID+"/review", []byte(`{"rating":3}`))

	var p progressJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/progress", nil), &p)
	// task 10 + on time 5 + habit 5 + note 3 + permanent 2 + review 1 = 26
	if !p.Enabled || p.XP != 26 || p.Level != 1 || p.Streak.State != "active" || p.Streak.Days != 1 {
		t.Fatalf("progress = %+v", p)
	}
	earned := map[string]bool{}
	for _, b := range p.Badges {
		earned[b.ID] = b.EarnedAt != nil
	}
	if !earned["first_task"] || earned["tasks_100"] {
		t.Fatalf("badges = %v", earned)
	}

	// Every point is explained, and the ledger adds up to the total.
	var l struct {
		Items []struct {
			Rule   string `json:"rule"`
			Points int    `json:"points"`
			Title  string `json:"title"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/progress/ledger?days=7", nil), &l)
	sum, rules := 0, map[string]bool{}
	for _, it := range l.Items {
		sum += it.Points
		rules[it.Rule] = true
	}
	for _, r := range []string{"task_done", "task_on_time", "habit_kept", "note_written", "permanent_note", "card_review"} {
		if !rules[r] {
			t.Errorf("ledger lacks %s", r)
		}
	}
	if sum != p.XP {
		t.Fatalf("ledger sums to %d, total is %d", sum, p.XP)
	}

	// Freezes cost earned points only.
	if code, out := alice.do("POST", "/v1/progress/freezes", nil); code != 409 || !bytes.Contains(out, []byte("never from money")) {
		t.Fatalf("freeze without points = %d %s", code, out)
	}
	for i := 0; i < 8; i++ {
		tk := alice.createTask(`{"title":"Chore"}`)
		alice.patchTask(tk.ID, `{"status":"done"}`, "If-Match", alice.etag("/v1/tasks/"+tk.ID))
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/progress/freezes", nil), &p)
	if p.Streak.Freezes != 1 || p.Points != p.XP-100 {
		t.Fatalf("after buying = %+v", p)
	}
}
