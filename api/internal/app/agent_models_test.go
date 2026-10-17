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

// AI usage (EP-152): every run's tokens and cost, this month, by month, and
// by agent, and the tool results' characters before and after compaction.
func TestAIUsageCountsTokensAndCompaction(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))
	for _, title := range []string{"Plan the garden", "Book the train", "Call Ana", "Fix the bike"} {
		alice.mustDo(201, "POST", "/v1/tasks", []byte(`{"title":"`+title+`","status":"next"}`))
	}
	fake.load(toolUse("list_tasks", `{"status":"next"}`), finalText("Four tasks."))
	if r := alice.runAgent("task-manager", "What is next?"); r.Run.Status != "completed" {
		t.Fatalf("run = %+v", r.Run)
	}
	var u struct {
		ThisMonth struct {
			Runs, InputTokens, OutputTokens int64
		} `json:"this_month"`
		AllTime struct {
			Runs int64 `json:"runs"`
		} `json:"all_time"`
		ByAgent []struct {
			Agent    string `json:"agent"`
			Input    int64  `json:"input_tokens"`
			ToolRaw  int64  `json:"tool_chars_raw"`
			ToolSent int64  `json:"tool_chars_sent"`
		} `json:"by_agent"`
		ByMonth []struct {
			Month string `json:"month"`
			Runs  int64  `json:"runs"`
		} `json:"by_month"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/ai-usage", nil), &u)
	if u.AllTime.Runs != 1 || len(u.ByAgent) != 1 || u.ByAgent[0].Agent != "task-manager" || u.ByAgent[0].Input != 2500 {
		t.Fatalf("usage = %+v", u)
	}
	if a := u.ByAgent[0]; a.ToolRaw == 0 || a.ToolSent == 0 || a.ToolSent >= a.ToolRaw {
		t.Fatalf("compaction = %d → %d characters", a.ToolRaw, a.ToolSent)
	}
	if len(u.ByMonth) != 12 || u.ByMonth[11].Runs != 1 || u.ByMonth[0].Runs != 0 {
		t.Fatalf("by month = %+v", u.ByMonth)
	}
	// The model was told how compact results read, and got rows.
	reqs := fake.lastRequests()
	if !strings.Contains(reqs[0].System[0].Text, `{"cols"`) {
		t.Error("the system prompt does not explain compact results")
	}
}
