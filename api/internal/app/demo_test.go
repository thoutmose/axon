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
		"/v1/habits": 8, "/v1/roadmaps": 1, "/v1/trash": 2, "/v1/agent-memories": 4, "/v1/decisions?status=reviewed": 12, "/v1/decisions?status=due": 1, "/v1/proposals?status=pending": 2, "/v1/assessments": 1,
	} {
		if n := count(t, visitor, path); n < min {
			t.Errorf("%s: %d items, want at least %d", path, n, min)
		}
	}
	// The one roadmap, under way (EP-95).
	if r := visitor.mustDo(200, "GET", "/v1/roadmaps", nil); !bytes.Contains(r, []byte("Top-5% in 26 engineering domains")) {
		t.Errorf("roadmaps = %s", r)
	}
	// A tech watch on the roadmap's domains, in the morning digest (EP-110).
	for path, min := range map[string]int{"/v1/watch/topics": 4, "/v1/watch/sources": 4, "/v1/watch/items": 30} {
		if n := count(t, visitor, path); n < min {
			t.Errorf("%s: %d items, want at least %d", path, n, min)
		}
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
