package app

import (
	"bytes"
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

	"github.com/thoutmose/axon/api/internal/watch"
)

// fakeWriter writes the fake sites' fixed pages.
type fakeWriter struct{ http.ResponseWriter }

func (w fakeWriter) print(s string) { _, _ = io.WriteString(w.ResponseWriter, s) } //nolint:gosec // a test server's own pages

func (w fakeWriter) printf(format string, a ...any) {
	_, _ = fmt.Fprintf(w.ResponseWriter, format, a...) //nolint:gosec // a test server's own pages
}

// fakeWeb plays the sites a tech watch follows.
type fakeWeb struct {
	srv     *httptest.Server
	feedHit atomic.Int32
}

const longStory = `PostgreSQL 18 ships a new vacuum strategy that reduces bloat on large tables and shortens maintenance windows.
Benchmarks on busy clusters show lower write amplification, faster index cleanup, and smaller WAL volume on primaries.
The release also improves logical replication failover, JSON_TABLE support, and the planner for partitioned tables.
Operators can upgrade with pg_upgrade in minutes, and extensions keep working thanks to a stable ABI for most hooks.
Cloud providers announced support within the week, and the community published migration guides for common setups.`

// watchDay is yesterday, so the fake feeds' stories stay inside the window
// in which Ingest merges the same story (10 days), whenever the tests run.
var watchDay = time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)

func newFakeWeb(t *testing.T) *fakeWeb {
	t.Setenv("AXON_CALENDAR_PRIVATE", "1") // the fake sites are on 127.0.0.1
	f := &fakeWeb{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		base := "http://" + r.Host
		w := fakeWriter{rw}
		switch r.URL.Path {
		case "/blog":
			w.Header().Set("Content-Type", "text/html")
			w.print(`<html><head><title>Blog</title><link rel="alternate" type="application/rss+xml" href="/feed.xml"></head><body>Hi</body></html>`)
		case "/feed.xml":
			f.feedHit.Add(1)
			if r.Header.Get("If-None-Match") == `"v1"` {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", `"v1"`)
			w.printf(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel><title>Data Weekly</title><link>%[1]s</link>
<item><title>Postgres 18 makes vacuum faster</title><link>%[1]s/pg18?utm_source=rss</link><guid>pg18</guid><pubDate>%[4]s</pubDate>
<content:encoded><![CDATA[<p>%[2]s</p>]]></content:encoded><category>Release</category></item>
<item><title>A short note</title><link>%[1]s/short</link><guid>short</guid><description>Short.</description></item>
<item><title>Indexing strategies for SQL databases explained</title><link>%[1]s/index</link><guid>idx</guid><description>%[3]s</description></item>
</channel></rss>`, base, longStory, strings.Repeat("B-tree and hash indexes in SQL databases trade write cost for read speed in Postgres. ", 6),
				watchDay.Add(8*time.Hour).Format(http.TimeFormat))
		case "/other.xml":
			w.printf(`<feed xmlns="http://www.w3.org/2005/Atom"><title>Other</title>
<entry><id>o1</id><title>Same link, other feed</title><link href="http://www.%[3]s/pg18/"/><updated>%[5]s</updated><summary>Same page.</summary></entry>
<entry><id>o2</id><title>Postgres 18: a faster vacuum</title><link href="%[1]s/mirror/pg18"/><updated>%[6]s</updated><content type="html">&lt;p&gt;%[2]s Read more on our site.&lt;/p&gt;</content></entry>
<entry><id>o3</id><title>Rust in the kernel, a year on</title><link href="%[1]s/rust"/><updated>%[7]s</updated><content type="html">&lt;p&gt;%[4]s&lt;/p&gt;</content></entry>
</feed>`, base, longStory, r.Host, strings.Repeat("The kernel now carries more drivers written in Rust, with fewer memory safety bugs reported. ", 5),
				watchDay.Add(9*time.Hour).Format(time.RFC3339), watchDay.Add(10*time.Hour).Format(time.RFC3339), watchDay.Add(-14*time.Hour).Format(time.RFC3339))
		case "/teaser.xml":
			w.printf(`<rss version="2.0"><channel><title>Teasers</title><item><title>Full text please</title><link>%[1]s/article/1</link><guid>t1</guid><description>A teaser.</description></item></channel></rss>`, base)
		case "/article/1":
			w.Header().Set("Content-Type", "text/html")
			w.printf(`<html><body><nav>menu</nav><article><h1>Full text please</h1><p>%s</p></article></body></html>`, strings.Repeat("The whole article is here, read from its page by Axon. ", 20))
		case "/broken.xml":
			w.WriteHeader(http.StatusInternalServerError)
		case "/api/greader.php/accounts/ClientLogin":
			_ = r.ParseForm()
			if r.Form.Get("Passwd") != "api-pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.print("SID=x\nLSID=x\nAuth=tok123\n")
		case "/api/greader.php/reader/api/0/subscription/list":
			if r.Header.Get("Authorization") != "GoogleLogin auth=tok123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.printf(`{"subscriptions":[{"id":"feed/%[1]s/teaser3.xml","title":"Teasers 3","url":"%[1]s/teaser3.xml","categories":[{"id":"user/-/label/Reading","label":"Reading"}]}]}`, base)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

type watchSourceJSON struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	News      bool   `json:"news"`
	Health    string `json:"health"`
	Failures  int    `json:"failures"`
	LastError string `json:"last_error"`
	Items     int    `json:"items"`
	Merged    int    `json:"merged"`
	Schedule  string `json:"schedule"`
	NextFetch string `json:"next_fetch_at"`
}

type watchItemJSON struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	Labels   []string `json:"labels"`
	TopicIDs []string `json:"topic_ids"`
	AlsoIn   []string `json:"also_in"`
	Outlier  string   `json:"outlier"`
	Status   string   `json:"status"`
	Minutes  int      `json:"reading_minutes"`
}

func (c *client) watchItems(query string) []watchItemJSON {
	c.e.t.Helper()
	var res struct {
		Items []watchItemJSON `json:"items"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/watch/items"+query, nil), &res)
	return res.Items
}

func (c *client) follow(url, extra string) (watchSourceJSON, map[string]int) {
	c.e.t.Helper()
	var res struct {
		Source watchSourceJSON `json:"source"`
		Stats  map[string]int  `json:"stats"`
	}
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/watch/sources", []byte(`{"url":"`+url+`"`+extra+`}`)), &res)
	return res.Source, res.Stats
}

// The tech watch follows sources, keeps each story once, labels it, and
// says how every source is doing.
func TestTechWatch(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	web := newFakeWeb(t)
	base := web.srv.URL

	var topic struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/watch/topics", []byte(`{"name":"Databases","keywords":["Postgres","SQL","index"]}`)), &topic)
	alice.mustDo(422, "POST", "/v1/watch/topics", []byte(`{"name":"Databases"}`))
	alice.mustDo(422, "POST", "/v1/watch/topics", []byte(`{"name":""}`))
	alice.mustDo(200, "PUT", "/v1/watch/topics/"+topic.ID, []byte(`{"name":"Databases","keywords":["Postgres","SQL","index"]}`))
	bob.mustDo(404, "PUT", "/v1/watch/topics/"+topic.ID, []byte(`{"name":"Mine"}`))

	// Discover the feed behind a blog page, then follow it.
	var preview struct {
		URL    string   `json:"url"`
		Title  string   `json:"title"`
		Recent []string `json:"recent"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/watch/discover", []byte(`{"url":"`+base+`/blog"}`)), &preview)
	if preview.URL != base+"/feed.xml" || preview.Title != "Data Weekly" || len(preview.Recent) != 3 {
		t.Fatalf("preview = %+v", preview)
	}
	alice.mustDo(422, "POST", "/v1/watch/discover", []byte(`{"url":"`+base+`/nothing"}`))
	src, stats := alice.follow(base+"/blog", `,"topic_ids":["`+topic.ID+`"]`)
	if src.URL != base+"/feed.xml" || src.Schedule != "0 0 * * *" || stats["new"] != 3 || src.Health != "ok" {
		t.Fatalf("followed %+v %v", src, stats)
	}
	alice.mustDo(409, "POST", "/v1/watch/sources", []byte(`{"url":"`+base+`/feed.xml"}`))

	// Another source carries the same link (with tracking) and the same story
	// under another link: both are stored once.
	other, stats := alice.follow(base+"/other.xml", "")
	if stats["new"] != 1 || stats["merged"] != 2 {
		t.Fatalf("other feed stats = %v", stats)
	}
	items := alice.watchItems("?outliers=1")
	if len(items) != 4 {
		t.Fatalf("%d items stored, want 4", len(items))
	}
	var pg watchItemJSON
	for _, it := range items {
		if it.Title == "Postgres 18 makes vacuum faster" {
			pg = it
		}
	}
	if len(pg.AlsoIn) != 1 || pg.AlsoIn[0] != "Other" || pg.Labels[0] != "databases" || len(pg.TopicIDs) != 1 || pg.Minutes < 1 {
		t.Fatalf("the merged story = %+v", pg)
	}
	// The short note is an outlier: hidden unless asked for.
	if n := len(alice.watchItems("")); n != 3 {
		t.Fatalf("without outliers: %d items", n)
	}
	if got := alice.watchItems("?label=databases"); len(got) != 2 {
		t.Fatalf("by label: %+v", got)
	}
	if got := alice.watchItems("?q=kernel&source=" + other.ID); len(got) != 1 {
		t.Fatalf("by words and source: %+v", got)
	}
	if got := alice.watchItems("?topic=" + topic.ID + "&from=" + watchDay.Format(time.DateOnly) + "&to=" + watchDay.AddDate(0, 0, 1).Format(time.DateOnly)); len(got) != 1 {
		t.Fatalf("by topic and period: %+v", got)
	}
	alice.mustDo(422, "GET", "/v1/watch/items?status=gone", nil)
	if len(bob.watchItems("?outliers=1")) != 0 {
		t.Fatal("Bob sees Alice's news")
	}

	// Fetching again: the feed answers 304 (it sent an ETag).
	hits := web.feedHit.Load()
	var fetched struct {
		Stats map[string]int `json:"stats"`
		Error string         `json:"error"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/watch/sources/"+src.ID+"/fetch", nil), &fetched)
	if fetched.Error != "" || fetched.Stats["new"] != 0 || web.feedHit.Load() != hits+1 {
		t.Fatalf("fetch again = %+v", fetched)
	}
	bob.mustDo(404, "POST", "/v1/watch/sources/"+src.ID+"/fetch", nil)

	// A broken source is recorded, and retried sooner than its schedule.
	broken, _ := alice.follow(base+"/teaser.xml", `,"full_text":true,"schedule":"0 6 * * 1"`)
	if _, err := pool.Exec(context.Background(), `UPDATE watch_sources SET url = $2 WHERE id = $1`, broken.ID, base+"/broken.xml"); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/watch/sources/"+broken.ID+"/fetch", nil), &fetched)
	var srcs struct {
		Items []watchSourceJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/sources", nil), &srcs)
	for _, s := range srcs.Items {
		if s.ID == broken.ID {
			next, _ := time.Parse(time.RFC3339Nano, s.NextFetch)
			if s.Health != "failing" || s.Failures != 1 || !strings.Contains(s.LastError, "500") || time.Until(next) > 20*time.Minute {
				t.Fatalf("broken source = %+v", s)
			}
		}
		if s.ID == src.ID && s.Merged != 0 || s.ID == other.ID && s.Merged != 1 { // two entries, one story
			t.Fatalf("merged counts = %+v", s)
		}
	}
	// The teaser's article was read from its page (full text).
	var full struct {
		Item struct {
			Content string `json:"content"`
		} `json:"item"`
		Note string `json:"note"`
	}
	for _, it := range alice.watchItems("?q=Full") {
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/items/"+it.ID, nil), &full)
	}
	if !strings.Contains(full.Item.Content, "The whole article is here") || strings.Contains(full.Item.Content, "menu") {
		t.Fatalf("full text = %.200q", full.Item.Content)
	}

	// The worker fetches what is due.
	if _, err := pool.Exec(context.Background(), `UPDATE watch_sources SET next_fetch_at = now() - interval '1 minute', url = $2 WHERE id = $1`, broken.ID, base+"/teaser.xml"); err != nil {
		t.Fatal(err)
	}
	w := &watch.Worker{DB: pool, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := w.Step(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/sources", nil), &srcs)
	for _, s := range srcs.Items {
		if s.ID == broken.ID && s.Health != "ok" {
			t.Fatalf("after the worker: %+v", s)
		}
	}

	// A feed note, kept as a note in Feeds/, linked to the item.
	var one struct {
		Item watchItemJSON `json:"item"`
		Note string        `json:"note"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/items/"+pg.ID, nil), &one)
	for _, want := range []string{"type: feed", `source: "Data Weekly"`, `topics: ["Databases"]`, "## Key points", "## Why it matters", "## My notes", "also in Other"} {
		if !strings.Contains(one.Note, want) {
			t.Fatalf("feed note lacks %q:\n%s", want, one.Note)
		}
	}
	alice.mustDo(204, "PATCH", "/v1/watch/items/"+pg.ID, []byte(`{"status":"read","labels":["databases","to share"]}`))
	alice.mustDo(422, "PATCH", "/v1/watch/items/"+pg.ID, []byte(`{"status":"lost"}`))
	bob.mustDo(404, "PATCH", "/v1/watch/items/"+pg.ID, []byte(`{"status":"read"}`))
	var kept struct {
		NoteID string `json:"note_id"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/watch/items/"+pg.ID+"/keep", nil), &kept)
	notes := alice.notes()
	found := false
	for p, n := range notes {
		if strings.HasPrefix(p, "Feeds/2026/10/") && n.ID == kept.NoteID && n.Kind == "feed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no feed note: %v", notes)
	}
	if got := alice.watchItems("?status=saved"); len(got) != 1 || !containsString(got[0].Labels, "to share") {
		t.Fatalf("saved = %+v", got)
	}
	var marked struct {
		Marked int `json:"marked"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/watch/items/read?source="+other.ID, nil), &marked)
	if marked.Marked != 1 {
		t.Fatalf("marked = %d", marked.Marked)
	}

	// Imports: CSV (a bad line, a source already followed, a new topic), OPML, FreshRSS.
	csv := "url,title,topics,schedule,full_text\nnot a url,,,,\n" + base + "/feed.xml,Again,,,\n" + base + "/teaser2.xml,Teasers 2,AI|Databases,@daily,yes\n" + base + "/x.xml,Bad,,* * * * *,\n"
	var imp struct {
		Added    int `json:"added"`
		Existing int `json:"existing"`
		Topics   int `json:"topics_created"`
		Failed   []struct {
			Line   int    `json:"line"`
			Reason string `json:"reason"`
		} `json:"failed"`
	}
	if code, out := alice.do("POST", "/v1/watch/import?format=csv", []byte(csv), "Content-Type", "text/csv"); code != 200 {
		t.Fatalf("csv import = %d %s", code, out)
	} else {
		_ = json.Unmarshal(out, &imp)
	}
	if imp.Added != 1 || imp.Existing != 1 || imp.Topics != 1 || len(imp.Failed) != 2 || imp.Failed[0].Line != 2 {
		t.Fatalf("csv import = %+v", imp)
	}
	opml := `<?xml version="1.0"?><opml version="2.0"><body><outline text="Systems"><outline type="rss" text="Kernel" xmlUrl="` + base + `/kernel.xml"/></outline></body></opml>`
	code, out := alice.do("POST", "/v1/watch/import?format=opml", []byte(opml), "Content-Type", "text/x-opml")
	if code != 200 || !strings.Contains(string(out), `"added":1`) {
		t.Fatalf("opml import = %d %s", code, out)
	}
	alice.mustDo(422, "POST", "/v1/watch/import/freshrss", []byte(`{"url":"`+base+`","user":"a","password":"wrong"}`))
	if out := alice.mustDo(200, "POST", "/v1/watch/import/freshrss", []byte(`{"url":"`+base+`","user":"a","password":"api-pass"}`)); !strings.Contains(string(out), `"added":1`) {
		t.Fatalf("freshrss import = %s", out)
	}
	var topics struct {
		Items []struct{ Name string } `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/topics", nil), &topics)
	if len(topics.Items) != 4 { // Databases, AI, Systems, Reading
		t.Fatalf("topics = %+v", topics)
	}
	if out := alice.mustDo(200, "GET", "/v1/watch/sources.opml", nil); !strings.Contains(string(out), "Exported from Axon Project") || !strings.Contains(string(out), `xmlUrl="`+base+`/kernel.xml"`) {
		t.Fatalf("opml = %s", out)
	}

	// Monitoring at a glance.
	var st struct {
		Sources struct{ Total, OK, Failing, Waiting int }                                `json:"sources"`
		Items   struct{ Total, Merged, Outliers, Saved, Articles, Videos, Podcasts int } `json:"items"`
		Daily   []struct{ Count int }                                                    `json:"daily"`
		Labels  []struct{ Label string }                                                 `json:"labels"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/stats", nil), &st)
	if st.Sources.Total != 6 || st.Sources.Waiting != 3 || st.Items.Total != 5 || st.Items.Merged != 1 || st.Items.Outliers != 1 || st.Items.Saved != 1 || len(st.Daily) != 30 || len(st.Labels) == 0 ||
		st.Items.Articles+st.Items.Videos+st.Items.Podcasts != st.Items.Total-st.Items.Outliers {
		t.Fatalf("stats = %+v", st)
	}

	// Schedules are checked; a topic and a source can be removed.
	alice.mustDo(422, "PATCH", "/v1/watch/sources/"+src.ID, []byte(`{"schedule":"* * * * *"}`))
	var changed watchSourceJSON
	_ = json.Unmarshal(alice.mustDo(200, "PATCH", "/v1/watch/sources/"+src.ID, []byte(`{"schedule":"30 7 * * *","title":"Data weekly"}`)), &changed)
	if changed.Schedule != "30 7 * * *" || changed.Title != "Data weekly" {
		t.Fatalf("changed = %+v", changed)
	}
	alice.mustDo(204, "DELETE", "/v1/watch/topics/"+topic.ID, nil)
	alice.mustDo(404, "DELETE", "/v1/watch/topics/"+topic.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/watch/sources/"+other.ID+"?items=delete", nil)
	if n := len(alice.watchItems("?outliers=1&status=all")); n != 4 {
		t.Fatalf("after deleting a source with its news: %d", n)
	}

	// Retention: unsaved, unlinked news go; kept ones stay.
	if _, err := watch.Purge(context.Background(), pool, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if left := alice.watchItems("?outliers=1&status=all"); len(left) != 1 || left[0].ID != pg.ID {
		t.Fatalf("after the purge: %+v", left)
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// The podcast's natural voices (EP-132): the operator's speech server lists
// its voices and speaks a part; without one, the device's voices are used.
func TestPodcastNaturalVoices(t *testing.T) {
	tts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/audio/voices":
			_, _ = w.Write([]byte(`{"voices":["af_heart","ff_siwis"]}`))
		case "/v1/audio/speech":
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte("ID3 mp3"))
		}
	}))
	defer tts.Close()

	t.Setenv("AXON_TTS_URL", "")
	alice := newEnv(t, false).client()
	alice.register("Alice")
	if out := alice.mustDo(200, "GET", "/v1/watch/podcast/voices", nil); !bytes.Contains(out, []byte(`"available":false`)) {
		t.Fatalf("without a server = %s", out)
	}
	alice.mustDo(409, "POST", "/v1/watch/podcast/speech", []byte(`{"text":"Hello","voice":"af_heart"}`))

	t.Setenv("AXON_TTS_URL", tts.URL)
	bob := newEnv(t, false).client()
	bob.register("Bob")
	if out := bob.mustDo(200, "GET", "/v1/watch/podcast/voices", nil); !bytes.Contains(out, []byte(`{"id":"ff_siwis","lang":"fr"}`)) {
		t.Fatalf("voices = %s", out)
	}
	if out := bob.mustDo(200, "POST", "/v1/watch/podcast/speech", []byte(`{"text":"Hello","voice":"af_heart","speed":1.2}`)); string(out) != "ID3 mp3" {
		t.Fatalf("speech = %q", out)
	}
	bob.mustDo(422, "POST", "/v1/watch/podcast/speech", []byte(`{"text":"","voice":"af_heart"}`))
	bob.mustDo(422, "POST", "/v1/watch/podcast/speech", []byte(`{"text":"Hi","voice":"bad voice!"}`))
	bob.mustDo(422, "POST", "/v1/watch/podcast/speech", []byte(`{"text":"Hi","voice":"af_heart","speed":5}`))
}

// General news (EP-155): a news source's items stay out of the tech watch,
// have their own tab, and make the news digest, a story many outlets carry
// first.
func TestNewsApartFromTheTechWatch(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	now := time.Now().UTC()
	feed := func(outlet string, stories ...string) string {
		var b strings.Builder
		b.WriteString(`<rss version="2.0"><channel><title>` + outlet + `</title>`)
		for i, s := range stories {
			fmt.Fprintf(&b, `<item><title>%s</title><link>https://%s.example/%d</link><guid>%s-%d</guid><pubDate>%s</pubDate>
				<description>%s %s %s</description></item>`, s, strings.ToLower(outlet), i, outlet, i,
				now.Add(-time.Duration(i+1)*time.Hour).Format(time.RFC1123Z), s, s, s)
		}
		return b.String() + `</channel></rss>`
	}
	feeds := map[string]string{
		"/bbc.xml":     feed("BBC", "Climate summit ends with a global methane deal agreed by ministers", "Local football club wins the cup final"),
		"/reuters.xml": feed("Reuters", "Climate summit ends with a global methane deal agreed by ministers"),
		"/tech.xml":    feed("Tech", "Postgres eighteen ships asynchronous input output for faster scans"),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(feeds[r.URL.Path]))
	}))
	defer srv.Close()
	bbc, _ := alice.follow(srv.URL+"/bbc.xml", `,"news":true`)
	alice.follow(srv.URL+"/reuters.xml", `,"news":true`)
	alice.follow(srv.URL+"/tech.xml", "")
	if !bbc.News {
		t.Fatalf("source = %+v", bbc)
	}
	titles := func(q string) string {
		var l struct {
			Items []struct {
				Title string `json:"title"`
			} `json:"items"`
		}
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/items?status=all&outliers=1"+q, nil), &l)
		var out []string
		for _, it := range l.Items {
			out = append(out, it.Title)
		}
		return strings.Join(out, " | ")
	}
	if got := titles(""); strings.Contains(got, "summit") || !strings.Contains(got, "Postgres") {
		t.Fatalf("the tech watch = %s", got)
	}
	if got := titles("&kind=news"); !strings.Contains(got, "summit") || !strings.Contains(got, "football") || strings.Contains(got, "Postgres") {
		t.Fatalf("the news = %s", got)
	}
	var d struct {
		Title    string `json:"title"`
		Sections []struct {
			Items []struct {
				Text   string `json:"text"`
				Detail string `json:"detail"`
			} `json:"items"`
		} `json:"sections"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/digest?kind=news", nil), &d)
	if d.Title != "The news" || len(d.Sections) != 1 || len(d.Sections[0].Items) < 2 {
		t.Fatalf("news digest = %+v", d)
	}
	top := d.Sections[0].Items[0]
	if !strings.Contains(top.Text, "summit") || !strings.Contains(top.Detail, "BBC") || !strings.Contains(top.Detail, "Reuters") {
		t.Fatalf("top story = %+v", top)
	}
	// The daily digest's tech watch has no general news.
	if daily := alice.mustDo(200, "GET", "/v1/me/digest?kind=daily", nil); bytes.Contains(daily, []byte("summit")) {
		t.Fatalf("the daily digest shows the news: %s", daily)
	}
}
