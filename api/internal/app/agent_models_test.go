package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thoutmose/axon/api/internal/auth"
)

// A model for all agents, one for an agent, and video summaries in prose
// (EP-151): the choice wins over the agent's default, an agent's own over
// all agents', and a summary comes back as checked lines and keywords.
func TestAgentModelsAndVideoSummaries(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))

	var m struct {
		All    string `json:"all"`
		Agents []struct {
			Agent, Title, Default, Model string
		} `json:"agents"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/agent-models", nil), &m)
	found := map[string]bool{}
	for _, a := range m.Agents {
		found[a.Agent] = a.Default != "" && a.Model == ""
	}
	if m.All != "" || !found["tutor"] || !found["task-manager"] || !found["jarvis"] || !found["video-summary"] {
		t.Fatalf("models = %+v", m)
	}
	alice.mustDo(422, "PUT", "/v1/me/agent-models", []byte(`{"models":{"nobody":"claude-sonnet-5-5"}}`))
	alice.mustDo(422, "PUT", "/v1/me/agent-models", []byte(`{"models":{"*":"not a model!"}}`))
	alice.mustDo(400, "PUT", "/v1/me/agent-models", []byte(`{}`))
	alice.mustDo(200, "PUT", "/v1/me/agent-models", []byte(`{"models":{"*":"claude-sonnet-5-5","tutor":"claude-opus-5-5"}}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/agent-models", nil), &m)
	if m.All != "claude-sonnet-5-5" {
		t.Fatalf("all = %q", m.All)
	}

	fake.load(finalText("Done."))
	if r := alice.runAgent("task-manager", "Tidy my inbox."); r.Run.Status != "completed" {
		t.Fatalf("run = %+v", r.Run)
	}
	if got := fake.lastRequests()[0].Model; got != "claude-sonnet-5-5" {
		t.Fatalf("task manager used %q, want all agents' model", got)
	}
	fake.load(finalText("Here is a plan."))
	alice.runAgent("tutor", "Plan me something.")
	if got := fake.lastRequests()[0].Model; got != "claude-opus-5-5" {
		t.Fatalf("tutor used %q, want its own model", got)
	}

	// Video summaries: off until asked for, then once per item.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<rss version="2.0"><channel><title>Talks</title><item><title>Raft in practice</title>
			<link>https://example.com/raft</link><guid>r1</guid><pubDate>Mon, 12 Oct 2026 08:00:00 GMT</pubDate>
			<description>A talk.</description></item></channel></rss>`))
	}))
	defer srv.Close()
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	alice.follow(srv.URL+"/feed.xml", "")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE watch_items SET transcript = 'so today raft elects a leader and the leader replicates its log'`); err != nil {
		t.Fatal(err)
	}
	runner, err := VideoSummaries(auth.Config{EncryptionKey: strings.Repeat("ab", 32)}, pool, slog.New(slog.NewTextHandler(io.Discard, nil)), fake)
	if err != nil {
		t.Fatal(err)
	}
	fake.load()
	if err := runner.SummariseVideos(ctx, 5); err != nil || len(fake.lastRequests()) != 0 {
		t.Fatalf("summarised without being asked: %v, %d calls", err, len(fake.lastRequests()))
	}
	alice.mustDo(200, "PUT", "/v1/me/agent-settings", []byte(`{"monthly_budget_cents":200,"video_summaries":true}`))
	fake.load(finalText(`Here: {"lines":["Raft elects a leader among the servers.","The leader appends client requests to its log.",` +
		`"It replicates the entries to the followers.","An entry is committed once a majority stores it."],` +
		`"keywords":["raft","leader election","log replication","majority","Raft","` + strings.Repeat("x", 80) + `"]}`))
	if err := runner.SummariseVideos(ctx, 5); err != nil {
		t.Fatal(err)
	}
	reqs := fake.lastRequests()
	if len(reqs) != 1 || reqs[0].Model != "claude-sonnet-5-5" || len(reqs[0].Tools) != 0 {
		t.Fatalf("summary request = %d, model %q, %d tools", len(reqs), reqs[0].Model, len(reqs[0].Tools))
	}
	var items struct {
		Items []struct {
			Summary  string   `json:"summary"`
			Keywords []string `json:"keywords"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/items?status=all&outliers=1", nil), &items)
	if len(items.Items) != 1 || !strings.HasPrefix(items.Items[0].Summary, "Raft elects a leader among the servers.\nThe leader") ||
		strings.Join(items.Items[0].Keywords, ",") != "raft,leader election,log replication,majority" {
		t.Fatalf("summarised item = %+v", items.Items)
	}
	// Once only, and recorded like a run.
	fake.load()
	_ = runner.SummariseVideos(ctx, 5)
	if len(fake.lastRequests()) != 0 {
		t.Fatal("summarised twice")
	}
	if runs := alice.mustDo(200, "GET", "/v1/agent-runs", nil); !bytes.Contains(runs, []byte(`"video-summary"`)) {
		t.Fatalf("runs = %s", runs)
	}
}
