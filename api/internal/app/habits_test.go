package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

type habitJSON struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Days  []int  `json:"days"`
	State struct {
		Today          string  `json:"today"`
		ScheduledToday bool    `json:"scheduled_today"`
		CheckedIn      *string `json:"checked_in"`
		Streak         int     `json:"streak"`
		Recovery       *struct {
			Missed  int    `json:"missed"`
			Message string `json:"message"`
		} `json:"recovery"`
	} `json:"state"`
}

func (c *client) habit(id string) habitJSON {
	c.e.t.Helper()
	var h habitJSON
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/habits/"+id, nil), &h)
	return h
}

func TestHabitCheckInsRecoveryAndGoals(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	var h habitJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read one page","stack_after":"After I pour my coffee","two_minute":"Open the book"}`)), &h)
	if len(h.Days) != 7 || !h.State.ScheduledToday || h.State.Recovery != nil || h.State.Streak != 0 {
		t.Fatalf("new habit = %+v", h)
	}
	today := h.State.Today

	// Pretend the habit is two weeks old, then miss yesterday.
	if _, err := pool.Exec(t.Context(), `UPDATE entities SET created_at = now() - interval '14 days' WHERE id = $1`, h.ID); err != nil {
		t.Fatal(err)
	}
	alice.mustDo(200, "PUT", "/v1/habits/"+h.ID+"/checkins/"+day(-2), []byte(`{"status":"done"}`))
	h = alice.habit(h.ID)
	if h.State.Recovery == nil || h.State.Recovery.Missed != 1 {
		t.Fatalf("expected a recovery prompt: %+v", h.State)
	}
	// The two-minute version counts and clears the prompt.
	alice.mustDo(200, "PUT", "/v1/habits/"+h.ID+"/checkins/"+today, []byte(`{"status":"two_minute","note":"Just one paragraph."}`))
	h = alice.habit(h.ID)
	if h.State.Recovery != nil || h.State.CheckedIn == nil || *h.State.CheckedIn != "two_minute" || h.State.Streak != 1 {
		t.Fatalf("after two-minute check-in: %+v", h.State)
	}
	// Backfill yesterday within the week; the future and old days are refused.
	alice.mustDo(200, "PUT", "/v1/habits/"+h.ID+"/checkins/"+day(-1), []byte(`{"status":"done"}`))
	if h = alice.habit(h.ID); h.State.Streak != 3 {
		t.Fatalf("streak after backfill = %d", h.State.Streak)
	}
	alice.mustDo(422, "PUT", "/v1/habits/"+h.ID+"/checkins/"+day(2), []byte(`{"status":"done"}`))
	alice.mustDo(422, "PUT", "/v1/habits/"+h.ID+"/checkins/"+day(-20), []byte(`{"status":"done"}`))
	alice.mustDo(422, "PUT", "/v1/habits/"+h.ID+"/checkins/"+today, []byte(`{"status":"perfect"}`))
	alice.mustDo(422, "PUT", "/v1/habits/"+h.ID+"/checkins/not-a-day", []byte(`{"status":"done"}`))
	alice.mustDo(204, "DELETE", "/v1/habits/"+h.ID+"/checkins/"+day(-1), nil)
	hist := alice.mustDo(200, "GET", "/v1/habits/"+h.ID+"/checkins", nil)
	if !bytes.Contains(hist, []byte("Just one paragraph.")) || bytes.Contains(hist, []byte(day(-1))) {
		t.Fatalf("history = %s", hist)
	}
	alice.mustDo(400, "GET", "/v1/habits/"+h.ID+"/checkins?from="+day(0)+"&to="+day(-5), nil)

	// Schedule, status, and links to a goal.
	etag := alice.etag("/v1/habits/" + h.ID)
	if code, out := alice.do("PATCH", "/v1/habits/"+h.ID, []byte(`{"days":[5,1,3],"cue":"07:00, kitchen"}`), "If-Match", etag); code != 200 {
		t.Fatalf("patch = %d %s", code, out)
	}
	if h = alice.habit(h.ID); len(h.Days) != 3 || h.Days[0] != 1 {
		t.Fatalf("days = %v", h.Days)
	}
	alice.mustDo(422, "POST", "/v1/habits", []byte(`{"title":"x","days":[0]}`))
	alice.mustDo(422, "POST", "/v1/habits", []byte(`{"title":" "}`))
	var g goalJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Read 12 books"}`)), &g)
	alice.link(h.ID, "supports", g.ID)
	if code, _ := alice.do("PATCH", "/v1/habits/"+h.ID, []byte(`{"status":"paused"}`), "If-Match", `"stale"`); code != 412 {
		t.Fatal("stale patch accepted")
	}
	etag = alice.etag("/v1/habits/" + h.ID)
	alice.do("PATCH", "/v1/habits/"+h.ID, []byte(`{"status":"paused"}`), "If-Match", etag)
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/habits?status=paused", nil), []byte(h.ID)) {
		t.Fatal("paused habit not listed")
	}
	alice.mustDo(400, "GET", "/v1/habits?status=gone", nil)

	// Signed-out access is refused, then deletion.
	anon := e.client() // signed out
	anon.mustDo(401, "GET", "/v1/habits/"+h.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/habits/"+h.ID, nil)
	alice.mustDo(404, "GET", "/v1/habits/"+h.ID, nil)
}
