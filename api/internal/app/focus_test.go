package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

type focusJSON struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	WorkedMinutes int    `json:"worked_minutes"`
	Target        *struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"target"`
}

func TestFocusSessionsLinkToWhatTheyWereFor(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	task := alice.createTask(`{"title":"Write the report"}`)

	var s focusJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/focus-sessions", []byte(`{"target_id":"`+task.ID+`","planned_minutes":75}`)), &s)
	if s.Title != "Write the report" || s.Status != "running" || s.Target == nil || s.Target.Type != "task" {
		t.Fatalf("session = %+v", s)
	}
	// Linked like everything else: the task's connections show it.
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/entities/"+task.ID+"/links?direction=incoming", nil), []byte("focused_on")) {
		t.Fatal("the task does not show its focus session")
	}
	// Worked minutes cannot exceed the time the session has run.
	alice.mustDo(422, "POST", "/v1/focus-sessions/"+s.ID+"/finish", []byte(`{"worked_minutes":50,"rounds":2,"status":"completed"}`))
	alice.mustDo(422, "POST", "/v1/focus-sessions/"+s.ID+"/finish", []byte(`{"worked_minutes":0,"rounds":0,"status":"paused"}`))
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/focus-sessions/"+s.ID+"/finish", []byte(`{"worked_minutes":0,"rounds":0,"status":"stopped"}`)), &s)
	if s.Status != "stopped" {
		t.Fatalf("finished = %+v", s)
	}
	alice.mustDo(422, "POST", "/v1/focus-sessions/"+s.ID+"/finish", []byte(`{"worked_minutes":0,"rounds":0,"status":"completed"}`))

	// A session with time worked counts today and starts a streak.
	var free focusJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/focus-sessions", []byte(`{"title":"Deep work","planned_minutes":25}`)), &free)
	_, err := pool.Exec(t.Context(), `UPDATE focus_sessions SET started_at = now() - interval '30 minutes' WHERE id = $1`, free.ID)
	if err != nil {
		t.Fatal(err)
	}
	alice.mustDo(200, "POST", "/v1/focus-sessions/"+free.ID+"/finish", []byte(`{"worked_minutes":25,"rounds":1,"status":"completed"}`))
	var list struct {
		Items        []focusJSON `json:"items"`
		TodayMinutes int         `json:"today_minutes"`
		Streak       int         `json:"streak"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/focus-sessions", nil), &list)
	if len(list.Items) != 2 || list.TodayMinutes != 25 || list.Streak != 1 {
		t.Fatalf("list = %+v", list)
	}
	var o overviewJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/dashboard/overview?days=7", nil), &o)
	for _, k := range o.KPIs {
		if k.ID == "focus_minutes" && k.Value != 25 {
			t.Fatalf("focus KPI = %d", k.Value)
		}
	}

	alice.mustDo(422, "POST", "/v1/focus-sessions", []byte(`{"planned_minutes":0}`))
	alice.mustDo(422, "POST", "/v1/focus-sessions", []byte(`{"planned_minutes":25,"target_id":"00000000-0000-7000-8000-000000000000"}`))
	alice.mustDo(200, "GET", "/v1/focus-sessions/"+s.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/focus-sessions/"+s.ID, nil)
	alice.mustDo(404, "GET", "/v1/focus-sessions/"+s.ID, nil)

	// Pomodoro settings: defaults and validation.
	set := alice.mustDo(200, "GET", "/v1/me/settings", nil)
	if !bytes.Contains(set, []byte(`"work_minutes":25,"short_break_minutes":5,"long_break_minutes":15,"rounds":4,"sessions":1`)) {
		t.Fatalf("focus defaults = %s", set)
	}
	if code, out := alice.putYAML("version: 1\nfocus:\n  work_minutes: 500\n"); code != 422 || !bytes.Contains(out, []byte("line 3: focus.work_minutes")) {
		t.Fatalf("bad focus yaml = %d %s", code, out)
	}
	// Music composed in the browser is a focus sound like the noises (EP-83).
	for _, sound := range []string{"music", "lofi", "ambient", "piano", "rain"} {
		if code, out := alice.putYAML("version: 1\nfocus:\n  sound: " + sound + "\n"); code != 200 {
			t.Fatalf("sound %s = %d %s", sound, code, out)
		}
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me/settings", nil), []byte(`"sound":"rain"`)) {
		t.Fatal("focus sound not saved")
	}
	if code, out := alice.putYAML("version: 1\nfocus:\n  sound: jazz\n"); code != 422 || !bytes.Contains(out, []byte("focus.sound")) {
		t.Fatalf("unknown sound = %d %s", code, out)
	}
}
