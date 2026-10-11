package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/graph"
)

type taskJSON struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	ScheduledOn *string `json:"scheduled_on"`
	DueOn       *string `json:"due_on"`
	Context     *string `json:"context"`
}

func (c *client) createTask(body string) taskJSON {
	c.e.t.Helper()
	var t taskJSON
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/tasks", []byte(body)), &t)
	return t
}

func (c *client) patchTask(id, body string, headers ...string) (int, []byte) {
	c.e.t.Helper()
	etag := c.etag("/v1/tasks/" + id)
	return c.do("PATCH", "/v1/tasks/"+id, []byte(body), append([]string{"If-Match", etag}, headers...)...)
}

func (c *client) view(query string) []taskJSON {
	c.e.t.Helper()
	var res struct {
		Items []taskJSON `json:"items"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/tasks?"+query, nil), &res)
	return res.Items
}

func titles(ts []taskJSON) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Title
	}
	return out
}

func day(offset int) string { return time.Now().UTC().AddDate(0, 0, offset).Format(time.DateOnly) }

func TestTaskCaptureClarifyAndViews(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	inbox := alice.createTask(`{"title":"Call the dentist"}`)
	if inbox.Status != "inbox" {
		t.Fatalf("captured status = %s", inbox.Status)
	}
	alice.createTask(`{"title":"Overdue report","status":"next","due_on":"` + day(-2) + `"}`)
	alice.createTask(`{"title":"Today's run","status":"next","scheduled_on":"` + day(0) + `","context":"home"}`)
	alice.createTask(`{"title":"Friday review","status":"next","scheduled_on":"` + day(4) + `"}`)
	alice.createTask(`{"title":"Plan trip","status":"next","due_on":"` + day(20) + `"}`)
	alice.createTask(`{"title":"Learn the cello","status":"someday","scheduled_on":"` + day(0) + `"}`)
	alice.createTask(`{"title":"Write the report","status":"doing"}`)
	alice.mustDo(422, "POST", "/v1/tasks", []byte(`{"title":"  "}`))
	alice.mustDo(422, "POST", "/v1/tasks", []byte(`{"title":"x","context":"@not ok!"}`))
	alice.mustDo(422, "POST", "/v1/tasks", []byte(`{"title":"x","bogus":1}`))

	check := func(view string, want ...string) {
		t.Helper()
		got := titles(alice.view("view=" + view))
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s = %v, want %v", view, got, want)
		}
	}
	check("inbox", "Call the dentist")
	check("today", "Overdue report", "Today's run")
	check("week", "Today's run", "Friday review")
	check("month", "Today's run", "Friday review", "Plan trip")
	check("someday", "Learn the cello")
	check("doing", "Write the report")
	if got := titles(alice.view("context=@home")); fmt.Sprint(got) != "[Today's run]" {
		t.Errorf("context filter = %v", got)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/contexts", nil), []byte(`"@home"`)) {
		t.Error("contexts missing @home")
	}

	// Clarify the inbox item: next action with a context; then complete it.
	if code, body := alice.patchTask(inbox.ID, `{"status":"next","context":"@phone"}`); code != 200 {
		t.Fatalf("clarify = %d %s", code, body)
	}
	if code, body := alice.patchTask(inbox.ID, `{"status":"done"}`); code != 200 || !bytes.Contains(body, []byte(`"completed_at":"`)) {
		t.Fatalf("complete = %d %s", code, body)
	}
	check("done", "Call the dentist")
	if code, _ := alice.do("PATCH", "/v1/tasks/"+inbox.ID, []byte(`{"title":"x"}`), "If-Match", `"stale"`); code != 412 {
		t.Fatalf("stale PATCH = %d, want 412", code)
	}
	if code, _ := alice.patchTask(inbox.ID, `{"context":null,"scheduled_on":null}`); code != 200 {
		t.Fatal("clearing fields failed")
	}
	alice.mustDo(200, "GET", "/v1/tasks/"+inbox.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/tasks/"+inbox.ID, nil)
	alice.mustDo(404, "GET", "/v1/tasks/"+inbox.ID, nil)
}

func TestCompletingARecurringTaskCreatesTheNextOccurrenceOnce(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	monday := "2026-10-12"
	weekly := alice.createTask(`{"title":"Water plants","status":"next","scheduled_on":"` + monday + `","due_on":"2026-10-13","recurrence":"FREQ=WEEKLY;BYDAY=MO,TH"}`)
	alice.mustDo(422, "POST", "/v1/tasks", []byte(`{"title":"No anchor","recurrence":"FREQ=DAILY"}`))
	alice.mustDo(422, "POST", "/v1/tasks", []byte(`{"title":"Bad rule","scheduled_on":"2026-10-12","recurrence":"FREQ=HOURLY"}`))

	// Two tabs complete the same occurrence; a retry replays it.
	etag := alice.etag("/v1/tasks/" + weekly.ID)
	for range 2 {
		code, _ := alice.do("PATCH", "/v1/tasks/"+weekly.ID, []byte(`{"status":"done"}`), "If-Match", etag)
		if code != 200 && code != 412 {
			t.Fatalf("complete = %d", code)
		}
	}
	alice.patchTask(weekly.ID, `{"status":"next"}`)
	alice.patchTask(weekly.ID, `{"status":"done"}`) // reopen and complete again: still one successor

	var count int
	var scheduled, due time.Time
	err := pool.QueryRow(context.Background(), `SELECT count(*), min(scheduled_on), min(due_on) FROM tasks
		WHERE series_id = $1 AND occurrence_on = '2026-10-15'`, uuid.MustParse(weekly.ID)).Scan(&count, &scheduled, &due)
	if err != nil || count != 1 {
		t.Fatalf("next occurrences = %d (%v), want exactly 1", count, err)
	}
	if scheduled.Format(time.DateOnly) != "2026-10-15" || due.Format(time.DateOnly) != "2026-10-16" {
		t.Fatalf("next occurrence scheduled %s due %s, want Thursday and Friday (due date keeps its offset)", scheduled, due)
	}
}

func TestTodayFollowsTheAccountTimeZone(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	etag := alice.etag("/v1/me")
	if code, body := alice.do("PATCH", "/v1/me", []byte(`{"timezone":"Mars/Olympus"}`), "If-Match", etag); code != 422 {
		t.Fatalf("bogus zone = %d %s", code, body)
	}
	// Kiritimati (UTC+14) is always a day ahead of Pago Pago (UTC−11).
	for zone, want := range map[string]string{"Pacific/Kiritimati": time.Now().In(mustZone("Pacific/Kiritimati")).Format(time.DateOnly),
		"Pacific/Pago_Pago": time.Now().In(mustZone("Pacific/Pago_Pago")).Format(time.DateOnly)} {
		if code, _ := alice.do("PATCH", "/v1/me", []byte(`{"timezone":"`+zone+`"}`), "If-Match", alice.etag("/v1/me")); code != 204 {
			t.Fatalf("set %s = %d", zone, code)
		}
		var res struct {
			Today string `json:"today"`
		}
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks?view=today", nil), &res)
		if res.Today != want {
			t.Errorf("today in %s = %s, want %s", zone, res.Today, want)
		}
	}
}

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func TestCheckIns(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(200, "PUT", "/v1/checkins/daily/"+day(0), []byte(`{"note":"Good focus"}`))
	alice.mustDo(200, "PUT", "/v1/checkins/daily/"+day(0), []byte(`{"note":"Updated"}`))
	var weekly struct {
		Day string `json:"day"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "PUT", "/v1/checkins/weekly/2026-10-15", []byte(`{}`)), &weekly)
	if weekly.Day != "2026-10-12" {
		t.Fatalf("weekly check-in stored on %s, want the Monday 2026-10-12", weekly.Day)
	}
	alice.mustDo(404, "PUT", "/v1/checkins/hourly/"+day(0), []byte(`{}`))
	list := alice.mustDo(200, "GET", "/v1/checkins?from=2026-10-01&to="+day(1), nil)
	if !bytes.Contains(list, []byte(`"Updated"`)) || bytes.Contains(list, []byte(`"Good focus"`)) {
		t.Fatalf("check-ins = %s", list)
	}
}

// Views must stay fast and bounded with a heavy task list.
func TestViewsStayFastWithTenThousandTasks(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	u := userID(t, alice)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		WITH ids AS (SELECT gen_random_uuid() AS id, n FROM generate_series(1, 10000) n),
		     ent AS (INSERT INTO entities (id, user_id, type, title) SELECT id, $1, 'task', 'Task ' || n FROM ids RETURNING id)
		INSERT INTO tasks (id, status, scheduled_on)
		SELECT ids.id, (ARRAY['inbox','next','waiting','someday','done'])[1 + n % 5], current_date + (n % 60) - 30
		FROM ids`, u)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `ANALYZE tasks; ANALYZE entities`)
	for _, view := range []string{"today", "week", "month", "inbox", "done", "all"} {
		start := time.Now()
		var res struct {
			Items      []taskJSON `json:"items"`
			NextCursor *string    `json:"next_cursor"`
		}
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks?view="+view, nil), &res)
		elapsed := time.Since(start)
		if len(res.Items) > 100 {
			t.Errorf("%s returned %d items; pages are capped at 100", view, len(res.Items))
		}
		if elapsed > 300*time.Millisecond {
			t.Errorf("%s took %s with 10,000 tasks", view, elapsed)
		}
		if res.NextCursor != nil {
			next := alice.view("view=" + view + "&after=" + *res.NextCursor)
			if len(next) == 0 || next[0].ID == res.Items[len(res.Items)-1].ID {
				t.Errorf("%s: second page is empty or overlaps", view)
			}
		}
	}
	if problems, _ := integrity(); problems != 0 {
		t.Errorf("integrity problems: %d", problems)
	}
}

func integrity() (int, error) {
	problems, err := graph.CheckIntegrity(context.Background(), pool)
	return len(problems), err
}
