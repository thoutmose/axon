package app

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type dashboardJSON struct {
	Today string `json:"today"`
	KPIs  []struct {
		ID         string `json:"id"`
		Value      int    `json:"value"`
		Definition string `json:"definition"`
	} `json:"kpis"`
	Tasks      []taskJSON       `json:"tasks"`
	Goals      []goalJSON       `json:"goals"`
	Milestones []map[string]any `json:"milestones"`
	Notes      []map[string]any `json:"notes"`
	CheckedIn  bool             `json:"checked_in"`
	Totals     map[string]int   `json:"totals"`
}

func (c *client) dashboard() dashboardJSON {
	c.e.t.Helper()
	var d dashboardJSON
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/dashboard", nil), &d)
	return d
}

func (d dashboardJSON) kpi(id string) int {
	for _, k := range d.KPIs {
		if k.ID == id {
			return k.Value
		}
	}
	return -1
}

func TestDashboardNumbersMatchTheItemsBehindThem(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	d := alice.dashboard()
	if d.Totals["tasks"] != 0 || d.Totals["notes"] != 0 || d.Totals["goals"] != 0 || len(d.Tasks) != 0 || d.kpi("today") != 0 {
		t.Fatalf("new account dashboard = %+v", d)
	}
	for _, k := range d.KPIs {
		if k.Definition == "" {
			t.Errorf("KPI %s has no definition", k.ID)
		}
	}

	alice.createTask(`{"title":"Late report","status":"next","due_on":"` + day(-2) + `"}`)
	alice.createTask(`{"title":"Call dentist","status":"next","scheduled_on":"` + day(0) + `"}`)
	alice.createTask(`{"title":"Unsorted thought"}`)
	alice.createTask(`{"title":"Maybe learn sailing","status":"someday","due_on":"` + day(-1) + `"}`)
	done := alice.createTask(`{"title":"Water plants","status":"next"}`)
	alice.patchTask(done.ID, `{"status":"done"}`)
	var g goalJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Run 10 km","target_date":"`+day(20)+`"}`)), &g)
	alice.mustDo(201, "POST", "/v1/goals/"+g.ID+"/milestones", []byte(`{"title":"Run 5 km","due_on":"`+day(3)+`"}`))
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"content":"# Training log"}`))

	d = alice.dashboard()
	// Every number equals the size of the list it links to.
	for kpi, view := range map[string]string{"overdue": "overdue", "today": "today", "inbox": "inbox"} {
		if got, want := d.kpi(kpi), len(alice.view("view="+view)); got != want {
			t.Errorf("KPI %s = %d, but view %s lists %d", kpi, got, view, want)
		}
	}
	if d.kpi("overdue") != 1 || d.kpi("today") != 2 || d.kpi("inbox") != 1 {
		t.Fatalf("kpis = %+v", d.KPIs)
	}
	if len(d.Tasks) != 2 || d.Tasks[0].Title != "Late report" {
		t.Fatalf("today's tasks = %v", titles(d.Tasks))
	}
	if len(d.Goals) != 1 || len(d.Milestones) != 1 || len(d.Notes) != 1 || d.Totals["tasks"] != 5 {
		t.Fatalf("dashboard = %+v", d)
	}

	cal := alice.mustDo(200, "GET", "/v1/calendar?from="+day(-7)+"&to="+day(30), nil)
	for _, want := range []string{`"Late report"`, `"Call dentist"`, `"kind":"target"`, `"Run 5 km"`} {
		if !bytes.Contains(cal, []byte(want)) {
			t.Errorf("calendar lacks %s: %s", want, cal)
		}
	}
	if bytes.Contains(alice.mustDo(200, "GET", "/v1/calendar?from="+day(-7)+"&to="+day(30)+"&types=goal", nil), []byte(`"type":"task"`)) {
		t.Error("types=goal returned tasks")
	}
	alice.mustDo(400, "GET", "/v1/calendar?from="+day(0)+"&to="+day(200), nil)
	alice.mustDo(400, "GET", "/v1/calendar?from="+day(0)+"&to="+day(1)+"&types=habit", nil)
}

func TestCalendarFeedsContainOnlyWhatWasChosenAndCanBeRevoked(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.createTask(`{"title":"Dentist","status":"next","due_on":"` + day(2) + `"}`)
	alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Secret goal","target_date":"`+day(10)+`"}`))

	var created struct {
		Feed struct {
			ID string `json:"id"`
		} `json:"feed"`
		Path string `json:"path"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/me/calendar-feeds", []byte(`{"name":"Phone","types":["task"]}`)), &created)
	alice.mustDo(422, "POST", "/v1/me/calendar-feeds", []byte(`{"name":"Phone","types":[]}`))
	alice.mustDo(422, "POST", "/v1/me/calendar-feeds", []byte(`{"name":"Phone","types":["habit"]}`))
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me/calendar-feeds", nil), []byte(`"Phone"`)) {
		t.Fatal("feed not listed")
	}

	// The calendar app has no session: the URL is the credential.
	app := e.client()
	path := strings.TrimPrefix(created.Path, "/api")
	ics := string(app.mustDo(200, "GET", path, nil))
	if !strings.Contains(ics, "SUMMARY:Due: Dentist") || strings.Contains(ics, "Secret goal") {
		t.Fatalf("feed = %s", ics)
	}
	app.mustDo(404, "GET", "/v1/calendar-feeds/not-a-real-token.ics", nil)

	// Scripts cannot manage feeds: they are account settings.
	var tok struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/me/tokens", []byte(`{"name":"script","scope":"write","expires_in_days":1}`)), &tok)
	if code, _ := app.do("GET", "/v1/me/calendar-feeds", nil, "Authorization", "Bearer "+tok.Token); code != 403 {
		t.Fatalf("token listing feeds = %d", code)
	}

	alice.mustDo(204, "DELETE", "/v1/me/calendar-feeds/"+created.Feed.ID, nil)
	alice.mustDo(404, "DELETE", "/v1/me/calendar-feeds/"+created.Feed.ID, nil)
	app.mustDo(404, "GET", path, nil)
	audit := alice.mustDo(200, "GET", "/v1/me/audit", nil)
	if !bytes.Contains(audit, []byte("calendar_feed.created")) || !bytes.Contains(audit, []byte("calendar_feed.revoked")) {
		t.Fatalf("feed changes not audited: %s", audit)
	}
}

func TestDashboardStaysFastWithTenThousandTasks(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	u := userID(t, alice)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		WITH ids AS (SELECT gen_random_uuid() AS id, n FROM generate_series(1, 10000) n),
		     ent AS (INSERT INTO entities (id, user_id, type, title) SELECT id, $1, 'task', 'Task ' || n FROM ids RETURNING id)
		INSERT INTO tasks (id, status, scheduled_on, due_on, completed_at)
		SELECT ids.id, (ARRAY['inbox','next','waiting','someday','done'])[1 + n % 5], current_date + (n % 60) - 30,
		       CASE WHEN n % 3 = 0 THEN current_date + (n % 40) - 20 END, CASE WHEN n % 5 = 4 THEN now() - (n % 10) * interval '1 day' END
		FROM ids`, u)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `ANALYZE tasks; ANALYZE entities`)
	alice.dashboard() // warm up
	var worst time.Duration
	for range 20 {
		start := time.Now()
		alice.dashboard()
		worst = max(worst, time.Since(start))
	}
	if worst > 300*time.Millisecond {
		t.Errorf("slowest of 20 dashboard loads took %s with 10,000 tasks", worst)
	}
}
