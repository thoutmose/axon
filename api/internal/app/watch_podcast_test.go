package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// putSetting changes one section of a person's settings.
func (c *client) putSetting(key string, value any) {
	c.e.t.Helper()
	var doc map[string]any
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	doc[key] = value
	body, _ := json.Marshal(doc)
	c.mustDo(200, "PUT", "/v1/me/settings", body)
}

// The podcast is off until turned on; an episode reads the period's news
// in the length, format, and language asked for, and says who made it.
func TestWatchPodcast(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	alice.mustDo(201, "POST", "/v1/watch/topics", []byte(`{"name":"Databases","keywords":["Postgres","pgvector"]}`))
	alice.mustDo(201, "POST", "/v1/watch/topics", []byte(`{"name":"Kernel","keywords":["Linux kernel","Rust"]}`))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString(`<rss version="2.0"><channel><title>Tech</title>`)
		for i, s := range graphStories {
			at := time.Now().Add(-time.Duration(i+1) * time.Hour).UTC().Format(time.RFC1123Z)
			b.WriteString(`<item><title>` + s[0] + `</title><link>http://` + r.Host + `/s/` + string(rune('a'+i)) + `</link><guid>` + string(rune('a'+i)) +
				`</guid><pubDate>` + at + `</pubDate><description>` + s[1] + ` More at https://example.org/x.</description></item>`)
		}
		b.WriteString(`</channel></rss>`)
		fakeWriter{w}.print(b.String())
	}))
	defer srv.Close()
	if _, st := alice.follow(srv.URL+"/feed.xml", ""); st["new"] != 6 {
		t.Fatalf("stats = %v", st)
	}

	alice.mustDo(409, "GET", "/v1/watch/podcast", nil)
	alice.putSetting("podcast", map[string]any{"enabled": true, "format": "digest", "lang": "en", "minutes": 10, "days": 1, "voice": "", "rate": 1})
	type episode struct {
		Title, Intro, Outro string
		Words               int
		Chapters            []struct{ Title, Text, Topic, URL string }
	}
	var ep episode
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/podcast", nil), &ep)
	if len(ep.Chapters) != 6 || !strings.HasPrefix(ep.Intro, "Hello. Here is your tech watch for ") || !strings.Contains(ep.Intro, "6 stories") ||
		!strings.Contains(ep.Outro, "Axon Project") || !strings.HasPrefix(ep.Title, "Tech watch, ") {
		t.Fatalf("episode = %+v", ep)
	}
	for _, c := range ep.Chapters {
		if !strings.HasPrefix(c.Text, "From Tech: ") || strings.Contains(c.Text, "http") || c.URL == "" {
			t.Fatalf("chapter = %+v", c)
		}
	}

	// A three-minute briefing keeps to its length; a deep dive goes by topic; French links.
	var brief episode
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/podcast?format=briefing&minutes=3", nil), &brief)
	if brief.Words > 3*150 || brief.Words >= ep.Words {
		t.Fatalf("briefing: %d words, digest %d", brief.Words, ep.Words)
	}
	var deep episode
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/podcast?format=deep&minutes=30&lang=fr", nil), &deep)
	if !strings.HasPrefix(deep.Intro, "Bonjour.") || !strings.HasPrefix(deep.Chapters[0].Text, "Passons à ") || !strings.HasPrefix(deep.Title, "Veille technologique, ") {
		t.Fatalf("deep = %+v", deep)
	}
	for i := 1; i < len(deep.Chapters); i++ {
		announced := strings.HasPrefix(deep.Chapters[i].Text, "Passons à ")
		if changed := deep.Chapters[i].Topic != deep.Chapters[i-1].Topic && deep.Chapters[i].Topic != ""; announced != changed {
			t.Fatalf("topic heading at %d: %+v", i, deep.Chapters)
		}
	}

	for _, bad := range []string{"format=radio", "minutes=60", "days=3", "lang=de", "topic=x", "minutes=x"} {
		alice.mustDo(422, "GET", "/v1/watch/podcast?"+bad, nil)
	}
	alice.mustDo(422, "PUT", "/v1/me/settings", []byte(`{"podcast":{"enabled":true,"format":"digest","lang":"en","minutes":10,"days":2,"rate":1}}`))
	bob.mustDo(409, "GET", "/v1/watch/podcast", nil)
	bob.putSetting("podcast", map[string]any{"enabled": true, "format": "briefing", "lang": "en", "minutes": 5, "days": 7, "voice": "", "rate": 1.2})
	var none episode
	_ = json.Unmarshal(bob.mustDo(200, "GET", "/v1/watch/podcast", nil), &none)
	if len(none.Chapters) != 0 || !strings.Contains(none.Intro, "no news") {
		t.Fatalf("bob = %+v", none)
	}
}

// The digests carry the period's top news, and follow the person's
// template: order, sections left out, no quote, no picture.
func TestDigestNewsAndTemplate(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
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
	alice.mustDo(201, "POST", "/v1/tasks", []byte(`{"title":"Write the report","due_on":"`+time.Now().Format(time.DateOnly)+`"}`))

	type digestJSON struct {
		Sections []struct {
			Key   string
			Items []struct {
				N         int
				Type, URL string
				Text      string
			}
		}
		Quote   struct{ Text string }
		Picture bool
	}
	var d digestJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/digest?kind=daily", nil), &d)
	last := d.Sections[len(d.Sections)-1]
	if d.Sections[0].Key != "To do today" || last.Key != "Tech watch" || len(last.Items) != 5 || last.Items[0].N != 0 ||
		!strings.HasPrefix(last.Items[0].URL, "/watch?item=") || d.Quote.Text == "" || !d.Picture {
		t.Fatalf("digest = %+v", d)
	}
	var weekly digestJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/digest?kind=weekly", nil), &weekly)
	if k := weekly.Sections[len(weekly.Sections)-1].Key; k != "Tech watch" {
		t.Fatalf("weekly ends with %q", k)
	}

	alice.putSetting("digest", map[string]any{"daily": []string{"watch", "tasks"}, "weekly": []string{"week"}, "monthly": []string{"goals", "month", "watch"}, "quote": false, "picture": false})
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/digest?kind=daily", nil), &d)
	if len(d.Sections) != 2 || d.Sections[0].Key != "Tech watch" || d.Sections[1].Key != "To do today" || d.Sections[1].Items[0].N != 1 || d.Quote.Text != "" || d.Picture {
		t.Fatalf("templated digest = %+v", d)
	}
	alice.mustDo(422, "PUT", "/v1/me/settings", []byte(`{"digest":{"daily":["watch","watch"]}}`))
	alice.mustDo(422, "PUT", "/v1/me/settings", []byte(`{"digest":{"daily":["goals"]}}`))
}
