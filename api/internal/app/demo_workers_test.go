package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/calendars"
	"github.com/thoutmose/axon/api/internal/notify"
	"github.com/thoutmose/axon/api/internal/watch"
)

// makeDemo turns the signed-in user into a demo account, as the shared one is.
func (c *client) makeDemo() {
	c.e.t.Helper()
	var me struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/me", nil), &me)
	if _, err := pool.Exec(context.Background(), `UPDATE users SET demo = true WHERE id = $1`, me.ID); err != nil {
		c.e.t.Fatal(err)
	}
}

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// The reminder worker sends a normal account its digest and task reminder,
// and a demo account neither.
func TestRemindersSkipDemo(t *testing.T) {
	e := newEnv(t, true)
	alice, visitor := e.client(), e.client()
	alice.register("Alice")
	visitor.register("Visitor")
	push := newFakePushService(t)
	w, err := Reminders(auth.Config{BaseURL: origin, EncryptionKey: strings.Repeat("ab", 32)}, pool, quietLog())
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().UTC().Format(time.DateOnly)
	remind := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	for name, c := range map[string]*client{"alice": alice, "visitor": visitor} {
		c.createTask(`{"title":"Task of ` + name + `","status":"next","due_on":"` + today + `","remind_at":"` + remind + `"}`)
		c.mustDo(201, "POST", "/v1/me/push-subscriptions", push.subscription("/send/"+name))
		var doc map[string]any
		_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
		delete(doc, "ai")
		doc["reminders"] = map[string]any{"enabled": true, "morning": "00:00", "evening": "", "email": "", "weekly_at": "", "monthly_at": ""}
		body, _ := json.Marshal(doc)
		c.mustDo(200, "PUT", "/v1/me/settings", body)
	}
	visitor.makeDemo()

	at, _ := time.Parse("2006-01-02 15:04", today+" 00:10")
	if err := w.Step(context.Background(), at); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	sent := map[string]int{}
	for _, p := range push.received() {
		sent[p.path]++
	}
	if sent["/send/alice"] == 0 {
		t.Fatalf("the normal account got nothing: %v", sent)
	}
	if sent["/send/visitor"] != 0 {
		t.Fatalf("the demo account was served: %v", sent)
	}
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM tasks t JOIN entities e ON e.id = t.id JOIN users u ON u.id = e.user_id
		WHERE u.demo AND t.reminded_at IS NOT NULL`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("a demo task was marked reminded: %d, %v", n, err)
	}
}

// The calendar worker refreshes a normal account's sources, never a demo one's.
func TestCalendarsSkipDemo(t *testing.T) {
	e := newEnv(t, true)
	alice, visitor := e.client(), e.client()
	alice.register("Alice")
	visitor.register("Visitor")
	byPath := map[string]*atomic.Int32{"/alice.ics": {}, "/visitor.ics": {}}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := byPath[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		c.Add(1)
		w.Header().Set("Content-Type", "text/calendar")
		_, _ = fmt.Fprint(w, "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:a\r\nDTSTART:20300101T090000Z\r\nDTEND:20300101T100000Z\r\nSUMMARY:Event\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n")
	}))
	t.Cleanup(srv.Close)
	client := calendars.Client
	calendars.Client = srv.Client()
	t.Cleanup(func() { calendars.Client = client })
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	alice.mustDo(201, "POST", "/v1/calendar/sources", []byte(`{"name":"A","url":"`+srv.URL+`/alice.ics"}`))
	visitor.mustDo(201, "POST", "/v1/calendar/sources", []byte(`{"name":"V","url":"`+srv.URL+`/visitor.ics"}`))
	visitor.makeDemo()
	if _, err := pool.Exec(context.Background(), `UPDATE calendar_sources SET fetched_at = now() - interval '2 days'`); err != nil {
		t.Fatal(err)
	}
	a0, v0 := byPath["/alice.ics"].Load(), byPath["/visitor.ics"].Load()

	store, err := Calendars(auth.Config{BaseURL: origin, EncryptionKey: strings.Repeat("ab", 32)}, pool, quietLog())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() { store.Run(ctx); close(stopped) }()
	eventually(t, "the normal account's calendar is refreshed", func() bool { return byPath["/alice.ics"].Load() > a0 })
	cancel()
	<-stopped
	if byPath["/visitor.ics"].Load() != v0 {
		t.Fatal("the demo account's calendar was fetched")
	}
}

// The index worker embeds a normal account's notes, never a demo account's.
func TestSemanticIndexSkipsDemo(t *testing.T) {
	e := newEnv(t, true)
	alice, visitor := e.client(), e.client()
	alice.register("Alice")
	visitor.register("Visitor")
	srv, _ := fakeEmbeddings(t)
	for _, c := range []*client{alice, visitor} {
		c.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/Vehicles.md","content":"# Vehicles\n\nAn automobile needs an engine."}`))
		c.mustDo(200, "PUT", "/v1/me/embeddings", []byte(`{"provider":"local","base_url":"`+srv.URL+`","model":"test-embed","api_key":"sk-test"}`))
		c.mustDo(204, "PUT", "/v1/me/consents/ai_embeddings", []byte(`{"granted":true}`))
	}
	visitor.makeDemo()
	ix, err := Semantic(auth.Config{EncryptionKey: strings.Repeat("ab", 32)}, pool, quietLog())
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	count := func(demo bool) (n int) {
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM note_embeddings v JOIN users u ON u.id = v.user_id WHERE u.demo = $1`, demo).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if count(false) != 1 {
		t.Fatalf("the normal account's note was not indexed: %d", count(false))
	}
	if count(true) != 0 {
		t.Fatal("a demo note was indexed")
	}
}

// demoFeeds has a normal and a demo account each follow their own feed of one
// item published an hour ago (the same item: each account keeps its own), and
// counts how often each feed is read.
func demoFeeds(t *testing.T, e *env) (alice, visitor *client, reads map[string]*atomic.Int32) {
	t.Helper()
	alice, visitor = e.client(), e.client()
	alice.register("Alice")
	visitor.register("Visitor")
	reads = map[string]*atomic.Int32{"/alice.xml": {}, "/visitor.xml": {}}
	published := time.Now().Add(-time.Hour).UTC().Format(time.RFC1123Z)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := reads[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		c.Add(1)
		_, _ = fmt.Fprintf(w, `<rss version="2.0"><channel><title>Talks</title><item><title>Raft in practice</title>
			<link>https://example.com/raft</link><guid>r1</guid><pubDate>%s</pubDate>
			<description>A talk.</description></item></channel></rss>`, published)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	alice.follow(srv.URL+"/alice.xml", "")
	visitor.follow(srv.URL+"/visitor.xml", "")
	return alice, visitor, reads
}

// demoItemsDone counts the tech watch items of normal (demo false) or demo
// accounts that a worker has handled, by the column it sets.
func demoItemsDone(t *testing.T, column string, demo bool) (n int) {
	t.Helper()
	q := `SELECT count(*) FROM watch_items i JOIN users u ON u.id = i.user_id WHERE u.demo = $1 AND i.` + column + ` IS NOT NULL`
	if err := pool.QueryRow(context.Background(), q, demo).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// The tech watch worker fetches a normal account's due sources, never a demo one's.
func TestTechWatchSkipsDemo(t *testing.T) {
	_, visitor, reads := demoFeeds(t, newEnv(t, true))
	visitor.makeDemo()
	if _, err := pool.Exec(context.Background(), `UPDATE watch_sources SET next_fetch_at = now() - interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	a0, v0 := reads["/alice.xml"].Load(), reads["/visitor.xml"].Load()
	w := &watch.Worker{DB: pool, Log: quietLog()}
	if err := w.Step(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if reads["/alice.xml"].Load() == a0 {
		t.Fatal("the normal account's source was not fetched")
	}
	if reads["/visitor.xml"].Load() != v0 {
		t.Fatal("the demo account's source was fetched")
	}
}

// The age of the most overdue source is 0 until one waits over an hour, and
// ignores the demo account, whose sources are never fetched.
func TestWatchOverdueMetric(t *testing.T) {
	_, visitor, _ := demoFeeds(t, newEnv(t, true))
	visitor.makeDemo()
	ctx := context.Background()
	set := func(name, when string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE watch_sources SET next_fetch_at = `+when+` WHERE user_id = (SELECT id FROM users WHERE display_name = $1)`, name); err != nil {
			t.Fatal(err)
		}
	}
	age := func() float64 {
		t.Helper()
		reg := prometheus.NewRegistry()
		watch.RegisterSourceAge(reg, pool)
		families, err := reg.Gather()
		if err != nil || len(families) != 1 {
			t.Fatal(families, err)
		}
		return families[0].GetMetric()[0].GetGauge().GetValue()
	}
	if _, err := pool.Exec(ctx, `UPDATE watch_sources SET next_fetch_at = now() + interval '1 day'`); err != nil {
		t.Fatal(err)
	}
	set("Visitor", `now() - interval '5 hours'`)
	if got := age(); got != 0 {
		t.Fatalf("a demo source counted: %v", got)
	}
	set("Alice", `now() - interval '30 minutes'`)
	if got := age(); got != 0 {
		t.Fatalf("a source late by 30 minutes counted: %v", got)
	}
	set("Alice", `now() - interval '3 hours'`)
	if got := age(); got < 3*3600 || got > 3*3600+60 {
		t.Fatalf("age = %v, want about 3 hours", got)
	}
}

// The video summary worker summarises a normal account's videos, never a demo one's.
func TestVideoSummariesSkipDemo(t *testing.T) {
	alice, visitor, _ := demoFeeds(t, newEnv(t, true))
	for _, c := range []*client{alice, visitor} {
		c.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))
		c.mustDo(200, "PUT", "/v1/me/agent-settings", []byte(`{"monthly_budget_cents":200,"video_summaries":true}`))
	}
	visitor.makeDemo()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE watch_items SET transcript = 'so today raft elects a leader and the leader replicates its log'`); err != nil {
		t.Fatal(err)
	}
	runner, err := VideoSummaries(auth.Config{EncryptionKey: strings.Repeat("ab", 32)}, pool, quietLog(), fake)
	if err != nil {
		t.Fatal(err)
	}
	summary := finalText(`{"lines":["Raft elects a leader among the servers.","The leader appends client requests to its log.",` +
		`"It replicates the entries to the followers."],"keywords":["raft","leader election","log replication"]}`)
	fake.load(summary, summary) // enough for both, should the demo one be asked
	if err := runner.SummariseVideos(ctx, 5); err != nil {
		t.Fatal(err)
	}
	if n := len(fake.lastRequests()); n != 1 {
		t.Fatalf("the model was asked %d times, want once", n)
	}
	if demoItemsDone(t, "ai_summary_at", false) != 1 {
		t.Fatal("the normal account's video was not summarised")
	}
	if demoItemsDone(t, "ai_summary_at", true) != 0 {
		t.Fatal("a demo video was summarised")
	}
}

// The watch knowledge worker reads a normal account's items, never a demo one's.
func TestWatchKnowledgeSkipsDemo(t *testing.T) {
	alice, visitor, _ := demoFeeds(t, newEnv(t, true))
	for _, c := range []*client{alice, visitor} {
		c.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))
		c.mustDo(200, "PUT", "/v1/me/agent-settings", []byte(`{"monthly_budget_cents":200,"watch_knowledge":true}`))
	}
	visitor.makeDemo()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE watch_items SET outlier = '', content = 'Raft elects a leader. The etcd store is built on Raft for consensus.'`); err != nil {
		t.Fatal(err)
	}
	runner, err := VideoSummaries(auth.Config{EncryptionKey: strings.Repeat("ab", 32)}, pool, quietLog(), fake)
	if err != nil {
		t.Fatal(err)
	}
	kg := finalText(`{"entities":[{"name":"etcd","type":"database","class":"Key-value store","aliases":[]},` +
		`{"name":"Raft","type":"concept","class":"Consensus algorithm","aliases":[]}],` +
		`"facts":[{"subject":"etcd","predicate":"built_on","object":"Raft","sentence":"The etcd store is built on Raft for consensus."}]}`)
	fake.load(kg, kg) // enough for both, should the demo one be read
	if err := runner.ExtractKnowledge(ctx, 5); err != nil {
		t.Fatal(err)
	}
	if n := len(fake.lastRequests()); n != 1 {
		t.Fatalf("the model was asked %d times, want once", n)
	}
	if demoItemsDone(t, "kg_at", false) != 1 {
		t.Fatal("the normal account's item was not read")
	}
	if demoItemsDone(t, "kg_at", true) != 0 {
		t.Fatal("a demo item was read")
	}
}

// The chat bots of a normal account are listened to, a demo account's never.
func TestChatBotsSkipDemo(t *testing.T) {
	e := newEnv(t, true)
	alice, visitor := e.client(), e.client()
	alice.register("Alice")
	visitor.register("Visitor")
	body := `{"kind":"telegram","name":"Telegram","config":{"bot_token":"123456789:` + strings.Repeat("A", 35) + `"}}`
	normal, _ := alice.addChannel(body)
	demo, _ := visitor.addChannel(body)
	visitor.makeDemo()
	bots, err := notify.NewStore(pool, mustSealer(t)).Bots(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var seenNormal bool
	for _, b := range bots {
		if b.ID.String() == demo.ID {
			t.Fatal("a demo bot is listened to")
		}
		seenNormal = seenNormal || b.ID.String() == normal.ID
	}
	if !seenNormal {
		t.Fatalf("the normal account's bot is missing: %+v", bots)
	}
}
