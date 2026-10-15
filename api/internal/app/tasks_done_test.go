package app

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
)

// A task checked off stays in its list, crossed out, until the next day
// (EP-116): each view returns the tasks done today apart from the open ones,
// with the list they came from, so Undo puts them back there.
func TestTasksDoneTodayStayInTheirList(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	call := alice.createTask(`{"title":"Call the bank","status":"next","scheduled_on":"` + day(0) + `"}`)
	idea := alice.createTask(`{"title":"An idea for the blog"}`)
	old := alice.createTask(`{"title":"Done last week","status":"next"}`)
	for _, id := range []string{call.ID, idea.ID, old.ID} {
		if code, b := alice.patchTask(id, `{"status":"done"}`); code != 200 {
			t.Fatalf("done = %d %s", code, b)
		}
	}
	if _, err := pool.Exec(context.Background(), `UPDATE tasks SET completed_at = now() - interval '3 days' WHERE id = $1`, old.ID); err != nil {
		t.Fatal(err)
	}

	type doneJSON struct {
		Title    string  `json:"title"`
		Status   string  `json:"status"`
		DoneFrom *string `json:"done_from"`
	}
	doneToday := func(query string) []string {
		t.Helper()
		var res struct {
			Items     []taskJSON `json:"items"`
			DoneToday []doneJSON `json:"done_today"`
		}
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks?"+query, nil), &res)
		if len(res.Items) != 0 {
			t.Errorf("%s: open items %v", query, titles(res.Items))
		}
		var out []string
		for _, d := range res.DoneToday {
			if d.Status != "done" || d.DoneFrom == nil {
				t.Errorf("%s: %+v", query, d)
			}
			out = append(out, d.Title)
		}
		slices.Sort(out)
		return out
	}
	for query, want := range map[string][]string{
		"view=next":  {"Call the bank"},
		"view=inbox": {"An idea for the blog"},
		"view=today": {"Call the bank"},
		"view=all":   {"An idea for the blog", "Call the bank"},
		"view=doing": nil,
	} {
		if got := doneToday(query); !slices.Equal(got, want) {
			t.Errorf("%s: done today = %v, want %v", query, got, want)
		}
	}

	// Today's page shows them too.
	var dash struct {
		DoneToday []doneJSON `json:"done_today"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/dashboard", nil), &dash)
	if len(dash.DoneToday) != 1 || dash.DoneToday[0].Title != "Call the bank" || *dash.DoneToday[0].DoneFrom != "next" {
		t.Fatalf("dashboard done today = %+v", dash.DoneToday)
	}

	// Undo puts it back where it was, and forgets where it came from.
	if code, b := alice.patchTask(idea.ID, `{"status":"inbox"}`); code != 200 {
		t.Fatalf("undo = %d %s", code, b)
	}
	var back struct {
		Status   string  `json:"status"`
		DoneFrom *string `json:"done_from"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks/"+idea.ID, nil), &back)
	if back.Status != "inbox" || back.DoneFrom != nil {
		t.Fatalf("after undo = %+v", back)
	}
}
