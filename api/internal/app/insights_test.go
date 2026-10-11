package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/insights"
)

type insightsJSON struct {
	Metrics []struct {
		ID      string `json:"id"`
		Formula string `json:"formula"`
		Points  []struct {
			Week  string  `json:"week"`
			Value float64 `json:"value"`
		} `json:"points"`
		Total float64 `json:"total"`
		Trend struct {
			Direction string `json:"direction"`
			Sentence  string `json:"sentence"`
		} `json:"trend"`
	} `json:"metrics"`
}

// Metrics count what really happened, week by week, with their formula and
// the items behind each number; the review compares weeks and carries open
// tasks over; a year of data stays fast; nothing leaks between people.
func TestInsightsAndWeeklyReview(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	ctx := context.Background()
	uid := userID(t, alice)
	now := time.Now().UTC()
	thisWeek := insights.Monday(now)
	lastWeek := thisWeek.AddDate(0, 0, -7)

	// Three tasks done last week, one this week, one open and overdue.
	for i, when := range []time.Time{lastWeek.Add(10 * time.Hour), lastWeek.Add(34 * time.Hour), lastWeek.Add(58 * time.Hour), thisWeek.Add(time.Hour)} {
		task := alice.createTask(`{"title":"Done task ` + string(rune('A'+i)) + `","status":"next"}`)
		if _, err := pool.Exec(ctx, `UPDATE tasks SET status = 'done', completed_at = $2 WHERE id = $1`, task.ID, when); err != nil {
			t.Fatal(err)
		}
	}
	alice.createTask(`{"title":"Overdue thing","status":"next","due_on":"` + lastWeek.Format(time.DateOnly) + `"}`)

	var in insightsJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/insights?weeks=8", nil), &in)
	var done *struct {
		ID      string `json:"id"`
		Formula string `json:"formula"`
		Points  []struct {
			Week  string  `json:"week"`
			Value float64 `json:"value"`
		} `json:"points"`
		Total float64 `json:"total"`
		Trend struct {
			Direction string `json:"direction"`
			Sentence  string `json:"sentence"`
		} `json:"trend"`
	}
	for i := range in.Metrics {
		if in.Metrics[i].ID == "tasks_done" {
			done = &in.Metrics[i]
		}
		if in.Metrics[i].Formula == "" {
			t.Fatalf("%s has no formula", in.Metrics[i].ID)
		}
	}
	if done == nil || len(done.Points) != 8 || done.Points[6].Value != 3 || done.Points[7].Value != 1 || done.Total != 4 ||
		done.Points[7].Week != insights.Label(thisWeek) || done.Trend.Direction != "new" {
		t.Fatalf("tasks_done = %+v", done)
	}
	alice.mustDo(400, "GET", "/v1/insights?weeks=3", nil)
	alice.mustDo(400, "GET", "/v1/insights?weeks=200", nil)

	var rev struct {
		Week    string  `json:"week"`
		Next    *string `json:"next"`
		Metrics []struct {
			ID       string  `json:"id"`
			Value    float64 `json:"value"`
			Previous float64 `json:"previous"`
		} `json:"metrics"`
		Sections []struct {
			Metric string `json:"metric"`
			Items  []struct {
				Title string `json:"title"`
			} `json:"items"`
		} `json:"sections"`
		CarryOver []struct {
			Title string `json:"title"`
		} `json:"carry_over"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/insights/review?week="+insights.Label(lastWeek), nil), &rev)
	if rev.Week != insights.Label(lastWeek) || rev.Next == nil || *rev.Next != insights.Label(thisWeek) {
		t.Fatalf("review weeks = %+v", rev)
	}
	if rev.Metrics[0].ID != "tasks_done" || rev.Metrics[0].Value != 3 || rev.Metrics[0].Previous != 0 {
		t.Fatalf("review metrics = %+v", rev.Metrics[0])
	}
	if len(rev.Sections) == 0 || rev.Sections[0].Metric != "tasks_done" || len(rev.Sections[0].Items) != 3 {
		t.Fatalf("sections = %+v", rev.Sections)
	}
	if len(rev.CarryOver) != 1 || rev.CarryOver[0].Title != "Overdue thing" {
		t.Fatalf("carry over = %+v", rev.CarryOver)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/insights/review", nil), &rev)
	if rev.Next != nil {
		t.Fatal("the current week has a next week")
	}
	alice.mustDo(400, "GET", "/v1/insights/review?week="+insights.Label(thisWeek.AddDate(0, 0, 14)), nil)
	alice.mustDo(400, "GET", "/v1/insights/review?week=nope", nil)

	if out := alice.mustDo(200, "GET", "/v1/insights/metrics/tasks_done/items?week="+insights.Label(lastWeek), nil); bytes.Count(out, []byte("Done task")) != 3 {
		t.Fatalf("items = %s", out)
	}
	alice.mustDo(404, "GET", "/v1/insights/metrics/nope/items", nil)

	code, body := alice.do("GET", "/v1/insights/export.csv?weeks=8", nil)
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if code != 200 || err != nil || len(rows) != 9 || rows[0][0] != "week" || rows[0][2] != "tasks_done" || rows[7][2] != "3" {
		t.Fatalf("csv %d %v %v", code, err, rows)
	}

	// Another person sees none of it.
	_ = json.Unmarshal(bob.mustDo(200, "GET", "/v1/insights?weeks=8", nil), &in)
	for _, m := range in.Metrics {
		if m.Total != 0 {
			t.Fatalf("bob sees %s = %v", m.ID, m.Total)
		}
	}

	// A year of activity: about 20 tasks and 3 notes a day, still quick.
	if _, err := pool.Exec(ctx, `
		WITH ids AS (SELECT gen_random_uuid() AS id, now() - (n || ' hours')::interval AS at FROM generate_series(1, 365 * 20) n),
		ents AS (INSERT INTO entities (id, user_id, type, title, created_at) SELECT id, $1, 'task', 'Bulk task', at FROM ids RETURNING id, created_at)
		INSERT INTO tasks (id, status, completed_at) SELECT id, 'done', created_at FROM ents`, uid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO entities (id, user_id, type, title, created_at)
		SELECT gen_random_uuid(), $1, 'milestone_note', 'x', now() - (n || ' hours')::interval FROM generate_series(1, 365 * 3) n`, uid); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	alice.mustDo(200, "GET", "/v1/insights?weeks=52", nil)
	alice.mustDo(200, "GET", "/v1/insights/review", nil)
	alice.mustDo(200, "GET", "/v1/insights/export.csv?weeks=104", nil)
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("a year of data took %v", d)
	}
}
