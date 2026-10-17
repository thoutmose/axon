package app

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/demo"
)

func TestDemoIsOffUnlessEnabled(t *testing.T) {
	e := newEnv(t, false)
	c := e.client()
	if !bytes.Contains(c.mustDo(200, "GET", "/v1/auth/status", nil), []byte(`"demo_available":false`)) {
		t.Fatal("demo should be off")
	}
	c.mustDo(404, "POST", "/v1/auth/demo", nil)
}

func count(t *testing.T, c *client, path string) int {
	t.Helper()
	var out struct {
		Items []json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", path, nil), &out)
	return len(out.Items)
}

func TestDemoAccountHasSampleDataAndCannotTouchAnythingElse(t *testing.T) {
	e := newEnvWith(t, auth.Config{Demo: true})
	visitor := e.client()
	visitor.mustDo(503, "POST", "/v1/auth/demo", nil) // not seeded yet

	ctx := context.Background()
	if _, err := demo.Reset(ctx, pool, time.Now()); err != nil {
		t.Fatal(err)
	}
	// The demo does not count as the first account: the owner can still register.
	owner := e.client()
	owner.register("Owner")
	secret := owner.createTask(`{"title":"Owner's private task"}`)

	if !bytes.Contains(visitor.mustDo(200, "GET", "/v1/auth/status", nil), []byte(`"demo_available":true`)) {
		t.Fatal("demo should be offered")
	}
	visitor.mustDo(204, "POST", "/v1/auth/demo", nil)
	if !bytes.Contains(visitor.mustDo(200, "GET", "/v1/me", nil), []byte(`"demo":true`)) {
		t.Fatal("me should say demo")
	}

	// Sample data in every area.
	for path, min := range map[string]int{
		"/v1/tasks?view=next": 3, "/v1/tasks?view=inbox": 2, "/v1/projects": 1, "/v1/notes": 8, "/v1/goals": 2,
		"/v1/habits": 8, "/v1/roadmaps": 1, "/v1/trash": 2, "/v1/agent-memories": 4, "/v1/decisions?status=reviewed": 12, "/v1/invites": 2, "/v1/decisions?status=due": 1, "/v1/proposals?status=pending": 2, "/v1/assessments": 1,
	} {
		if n := count(t, visitor, path); n < min {
			t.Errorf("%s: %d items, want at least %d", path, n, min)
		}
	}
	// The one roadmap, under way (EP-95).
	if r := visitor.mustDo(200, "GET", "/v1/roadmaps", nil); !bytes.Contains(r, []byte("Top-5% in 37 engineering domains")) {
		t.Errorf("roadmaps = %s", r)
	}
	// A tech watch on the roadmap's domains, in the morning digest (EP-110).
	for path, min := range map[string]int{"/v1/watch/topics": 4, "/v1/watch/sources": 4, "/v1/watch/items": 30} {
		if n := count(t, visitor, path); n < min {
			t.Errorf("%s: %d items, want at least %d", path, n, min)
		}
	}
	// The demo shows its sample invitations, and cannot create one (EP-120); its
	// videos are YouTube talks whose transcript it may read (EP-123).
	visitor.mustDo(403, "POST", "/v1/invites", []byte(`{}`))
	// Nothing the demo drank is in the future: its coffee is from before the reset.
	var future bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM caffeine_entries c JOIN users u ON u.id = c.user_id WHERE u.demo AND c.drunk_at > now())`).Scan(&future); err != nil || future {
		t.Errorf("a demo coffee in the future (%v)", err)
	}
	if b := visitor.mustDo(200, "GET", "/v1/watch/items?status=all&kind=video", nil); !bytes.Contains(b, []byte("Raft - The Understandable Distributed Protocol")) {
		t.Errorf("no YouTube talk in the demo: %.300s", b)
	}
	// The reader's tabs count each kind (EP-127).
	var kinds struct {
		Items struct{ Videos, Podcasts int } `json:"items"`
	}
	_ = json.Unmarshal(visitor.mustDo(200, "GET", "/v1/watch/stats", nil), &kinds)
	if kinds.Items.Videos < 5 || kinds.Items.Podcasts < 2 {
		t.Errorf("demo watch kinds = %+v", kinds.Items)
	}
	if d := visitor.mustDo(200, "GET", "/v1/me/digest?kind=daily", nil); !bytes.Contains(d, []byte(`"key":"Tech watch"`)) {
		t.Errorf("the daily digest has no tech watch: %s", d)
	}
	// Its knowledge graph has linked concepts in several communities, and its podcast an episode (EP-111, EP-113).
	var wg struct {
		Stats struct {
			Concepts    int `json:"concepts"`
			Edges       int `json:"edges"`
			Communities int `json:"communities"`
		} `json:"stats"`
	}
	_ = json.Unmarshal(visitor.mustDo(200, "GET", "/v1/watch/graph", nil), &wg)
	if wg.Stats.Concepts < 20 || wg.Stats.Edges < 20 || wg.Stats.Communities < 3 {
		t.Errorf("demo watch graph = %+v", wg.Stats)
	}
	var ep struct {
		Chapters []json.RawMessage `json:"chapters"`
	}
	_ = json.Unmarshal(visitor.mustDo(200, "GET", "/v1/watch/podcast", nil), &ep)
	if len(ep.Chapters) < 3 {
		t.Errorf("demo podcast has %d chapters", len(ep.Chapters))
	}
	// Two months of reviews, so recall calibration has something to compare (EP-77).
	var cal struct {
		Reviews int               `json:"reviews"`
		Buckets []json.RawMessage `json:"buckets"`
	}
	_ = json.Unmarshal(visitor.mustDo(200, "GET", "/v1/flashcards/calibration", nil), &cal)
	if cal.Reviews < 20 || len(cal.Buckets) < 2 {
		t.Errorf("demo calibration: %d reviews in %d buckets", cal.Reviews, len(cal.Buckets))
	}
	if d := visitor.mustDo(200, "GET", "/v1/decks", nil); !bytes.Contains(d, []byte("MIT 6.042J: logic")) {
		t.Errorf("the 6.042J deck is missing: %s", d)
	}
	var zg struct {
		Stats struct {
			Notes       int `json:"notes"`
			Communities int `json:"communities"`
		} `json:"stats"`
	}
	_ = json.Unmarshal(visitor.mustDo(200, "GET", "/v1/notes/zettelkasten-graph", nil), &zg)
	if zg.Stats.Notes < 10 || zg.Stats.Communities < 2 {
		t.Errorf("demo knowledge graph = %+v", zg.Stats)
	}
	// The journal shows what helps, from 60 seeded days (EP-92).
	var helps struct {
		Days     int `json:"days"`
		Findings []struct {
			Factor, Outcome string
		} `json:"findings"`
	}
	_ = json.Unmarshal(visitor.mustDo(200, "GET", "/v1/journal/helps", nil), &helps)
	if helps.Days != 60 || len(helps.Findings) < 2 {
		t.Errorf("demo what helps = %+v", helps)
	}
	// The seeded proposals can be approved like real ones.
	var props struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	_ = json.Unmarshal(visitor.mustDo(200, "GET", "/v1/proposals?status=pending", nil), &props)
	for _, p := range props.Items {
		visitor.mustDo(200, "POST", "/v1/proposals/"+p.ID+"/approve", nil)
	}

	// Isolation both ways.
	visitor.mustDo(404, "GET", "/v1/tasks/"+secret.ID, nil)
	if count(t, owner, "/v1/projects") != 0 || count(t, owner, "/v1/notes") != 0 {
		t.Fatal("the owner sees demo data")
	}

	// No credentials, tokens, providers, outside reach, or erasure.
	for _, req := range []struct{ method, path, body string }{
		{"POST", "/v1/me/tokens", `{"name":"x","scope":"read","expires_in_days":1}`},
		{"POST", "/v1/me/passkeys/begin", `{}`},
		{"POST", "/v1/me/recovery-codes", ``},
		{"POST", "/v1/me/totp/begin", ``},
		{"DELETE", "/v1/me", `{"confirm":"DELETE"}`},
		{"PUT", "/v1/me/agent-provider", `{"provider":"anthropic","model":"claude-opus-5-5","api_key":"sk-x"}`},
		{"POST", "/v1/repos", `{"owner":"a","repo":"b","token":"t"}`},
	} {
		var body []byte
		if req.body != "" {
			body = []byte(req.body)
		}
		if code, out := visitor.do(req.method, req.path, body); code != 403 || !bytes.Contains(out, []byte("demo account cannot")) {
			t.Errorf("%s %s = %d %s", req.method, req.path, code, out)
		}
	}
	visitor.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))
	if !bytes.Contains(visitor.mustDo(200, "GET", "/v1/agents", nil), []byte(`"reason":"demo"`)) {
		t.Fatal("agents should be off in the demo")
	}
	visitor.mustDo(503, "POST", "/v1/agents/task-manager/runs", []byte(`{"instruction":"Hi"}`))

	// A reset wipes what visitors changed and signs them out.
	visitor.createTask(`{"title":"Visitor graffiti"}`)
	if _, err := demo.Reset(ctx, pool, time.Now()); err != nil {
		t.Fatal(err)
	}
	visitor.mustDo(401, "GET", "/v1/me", nil)
	visitor.mustDo(204, "POST", "/v1/auth/demo", nil)
	if bytes.Contains(visitor.mustDo(200, "GET", "/v1/tasks?view=inbox", nil), []byte("graffiti")) {
		t.Fatal("reset kept visitor changes")
	}
	owner.mustDo(200, "GET", "/v1/tasks/"+secret.ID, nil)
}

// A new account starts clean (EP-139): nothing of anyone else's, but the
// roadmap library and the note templates to start from.
func TestANewAccountStartsClean(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	for _, path := range []string{"/v1/notes", "/v1/tasks", "/v1/habits", "/v1/goals", "/v1/projects", "/v1/roadmaps", "/v1/media", "/v1/watch/sources"} {
		var list struct {
			Items []json.RawMessage `json:"items"`
		}
		_ = json.Unmarshal(alice.mustDo(200, "GET", path, nil), &list)
		if len(list.Items) != 0 {
			t.Errorf("%s: %d items, want none", path, len(list.Items))
		}
	}
	var sources struct {
		Items []json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/sources", nil), &sources)
	if len(sources.Items) == 0 {
		t.Error("the roadmap library offers nothing")
	}
}

// The demo's roadmap and what surrounds it go to a real account on request
// (axon demo starter), without the demo's history, and only once.
func TestDemoStarterGivesARealAccountTheRoadmap(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	added, err := demo.Starter(t.Context(), pool, "Alice", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	// Morning sunlight comes with the roadmap's plan (EP-166), so the protocols add one habit less.
	if len(added) != 3+len(demo.LearningNotes)+len(demo.Funds)+len(demo.Schools)+len(demo.Protocols)-1 {
		t.Fatalf("added = %v", added)
	}
	var list struct {
		Items []struct {
			Title string `json:"title"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps", nil), &list)
	if len(list.Items) != 1 || list.Items[0].Title != "Top-5% in 37 engineering domains" {
		t.Fatalf("roadmaps = %+v", list.Items)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/habits", nil), &list)
	if len(list.Items) < len(demo.Protocols) {
		t.Fatalf("habits = %+v", list.Items)
	}
	var cards int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM cards c JOIN entities e ON e.id = c.id
		WHERE c.deck = 'MIT 6.042J: logic' AND e.user_id = (SELECT id FROM users WHERE display_name = 'Alice')`).Scan(&cards); err != nil || cards == 0 {
		t.Fatalf("the literature note's cards = %d, %v", cards, err)
	}
	again, err := demo.Starter(t.Context(), pool, "Alice", time.Now())
	if err != nil || len(again) != 0 {
		t.Fatalf("second run = %v, %v", again, err)
	}
	if _, err := demo.Starter(t.Context(), pool, "Nobody", time.Now()); err == nil {
		t.Error("an unknown account is refused")
	}
}
