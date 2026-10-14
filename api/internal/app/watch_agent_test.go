package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The Tech watch agent reads the person's watch as data and proposes
// sources and topics; an address it cannot vouch for is refused.
func TestTechWatchAgent(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.enableAgents()
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	alice.mustDo(201, "POST", "/v1/watch/topics", []byte(`{"name":"Databases","keywords":["Postgres","pgvector"]}`))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString(`<rss version="2.0"><channel><title>Tech</title>`)
		for i, s := range graphStories {
			at := time.Now().Add(-time.Duration(i+1) * time.Hour).UTC().Format(time.RFC1123Z)
			b.WriteString(`<item><title>` + s[0] + `</title><link>http://` + r.Host + `/s/` + string(rune('a'+i)) + `</link><guid>` + string(rune('a'+i)) +
				`</guid><pubDate>` + at + `</pubDate><description>` + s[1] + `</description></item>`)
		}
		b.WriteString(`</channel></rss>`)
		fakeWriter{w}.print(b.String())
	}))
	defer srv.Close()
	alice.follow(srv.URL+"/feed.xml", "")

	fake.load(
		toolUse("list_watch_topics", `{}`, "list_watch_sources", `{}`, "list_watch_news", `{"days":3,"topic":"databases"}`, "watch_trends", `{}`),
		toolUse("suggest_watch_source", `{"url":"http://insecure.example/feed","reason":"More news."}`,
			"suggest_watch_topic", `{"name":"Kernel","keywords":["Linux kernel","Rust"],"reason":"Three of your stories are about Rust in the kernel."}`),
		finalText("Proposed a Kernel topic."),
	)
	r := alice.runAgent("tech-watch", "What am I missing?")
	if r.Run.Status != "completed" || len(r.Proposals) != 1 || !strings.Contains(r.Proposals[0].Summary, "Kernel") {
		t.Fatalf("run = %+v", r)
	}
	var sent []byte
	for _, req := range fake.lastRequests() {
		b, _ := json.Marshal(req.Messages)
		sent = append(sent, b...)
	}
	for _, want := range []string{"Vector search lands in Postgres", `\"news_30_days\":3`, `\"health\":\"ok\"`, "linux kernel", "https address"} {
		if !bytes.Contains(sent, []byte(want)) {
			t.Fatalf("the model did not get %q: %s", want, sent)
		}
	}
	if bytes.Contains(sent, []byte("Rust drivers reach the Linux kernel")) {
		t.Fatal("news outside the topic reached the model")
	}
	if out := alice.mustDo(200, "GET", "/v1/watch/topics", nil); bytes.Contains(out, []byte(`"Kernel"`)) {
		t.Fatal("added before approval")
	}
	alice.mustDo(200, "POST", "/v1/proposals/"+r.Proposals[0].ID+"/approve", nil)
	if out := alice.mustDo(200, "GET", "/v1/watch/topics", nil); !bytes.Contains(out, []byte(`"name":"Kernel"`)) {
		t.Fatalf("after approval: %s", out)
	}
	// Approving a topic labels the news it gathers.
	var stories []watchItemJSON
	for _, it := range alice.watchItems("?status=all") {
		if len(it.TopicIDs) > 0 {
			stories = append(stories, it)
		}
	}
	if len(stories) < 5 {
		t.Fatalf("topics label %d stories", len(stories))
	}
}
