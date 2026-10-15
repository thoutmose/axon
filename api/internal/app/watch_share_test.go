package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Exports carry the news a filter selects and say where they come from;
// share links show a selection to anyone with the link, nothing more.
func TestWatchExportsAndShares(t *testing.T) {
	e := newEnv(t, true)
	alice, bob, anyone := e.client(), e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := "http://" + r.Host
		fakeWriter{w}.print(`<rss version="2.0"><channel><title>Mixed</title>
<item><title>Vector search lands in Postgres</title><link>` + base + `/a</link><guid>a</guid><pubDate>Mon, 05 Oct 2026 08:00:00 GMT</pubDate><description>` + graphStories[0][1] + `</description></item>
<item><title>Rust drivers reach the Linux kernel</title><link>` + base + `/b</link><guid>b</guid><pubDate>Tue, 06 Oct 2026 08:00:00 GMT</pubDate><description>` + graphStories[3][1] + `</description></item>
<item><title>Why kernel developers argue about Rust</title><link>` + base + `/c</link><guid>c</guid><pubDate>Wed, 07 Oct 2026 08:00:00 GMT</pubDate><description>` + graphStories[4][1] + `</description></item>
<item><title>A talk on query planners</title><link>https://www.youtube.com/watch?v=planner</link><guid>v</guid><pubDate>Thu, 08 Oct 2026 08:00:00 GMT</pubDate><description>` + graphStories[1][1] + `</description></item>
<item><title>Episode 12: memory safety</title><link>` + base + `/ep12</link><guid>p</guid><pubDate>Fri, 09 Oct 2026 08:00:00 GMT</pubDate><description>` + graphStories[5][1] + `</description><enclosure url="` + base + `/ep12.mp3" type="audio/mpeg" length="1"/></item>
</channel></rss>`)
	}))
	defer srv.Close()
	if _, st := alice.follow(srv.URL+"/feed.xml", ""); st["new"] != 5 {
		t.Fatalf("stats = %v", st)
	}

	// By kind.
	for kind, want := range map[string]int{"article": 3, "video": 1, "podcast": 1} {
		if items := alice.watchItems("?kind=" + kind); len(items) != want {
			t.Fatalf("%s: %d", kind, len(items))
		}
	}
	alice.mustDo(422, "GET", "/v1/watch/items?kind=tweet", nil)
	items := alice.watchItems("?kind=article")
	alice.mustDo(204, "PATCH", "/v1/watch/items/"+items[0].ID, []byte(`{"status":"saved"}`))

	// Exports, each saying where it comes from.
	csv := string(alice.mustDo(200, "GET", "/v1/watch/export?format=csv&kind=article&from=2026-10-01&to=2026-11-01", nil))
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if !strings.HasPrefix(lines[0], "Title,Link,Source,Kind") || len(lines) != 5 || !strings.Contains(lines[4], "Exported from Axon Project") ||
		!strings.Contains(lines[4], "Articles · 1 Oct 2026 to 31 Oct 2026") || !strings.Contains(csv, ",saved,") {
		t.Fatalf("csv = %s", csv)
	}
	for format, prefix := range map[string]string{"pdf": "%PDF-1.4", "ods": "PK", "zip": "PK", "md": "# Tech watch", "json": "{"} {
		if out := string(alice.mustDo(200, "GET", "/v1/watch/export?format="+format, nil)); !strings.HasPrefix(out, prefix) {
			t.Fatalf("%s = %.40q", format, out)
		}
	}
	alice.mustDo(422, "GET", "/v1/watch/export?format=docx", nil)
	if out := string(bob.mustDo(200, "GET", "/v1/watch/export?format=md", nil)); !strings.Contains(out, "0 news") {
		t.Fatalf("bob's export = %s", out)
	}

	// A share link keeps only the filter, lives a while, and is shown once.
	alice.mustDo(422, "POST", "/v1/watch/shares", []byte(`{"name":"","query":"kind=article"}`))
	alice.mustDo(422, "POST", "/v1/watch/shares", []byte(`{"name":"Team","query":"kind=article","days":400}`))
	alice.mustDo(422, "POST", "/v1/watch/shares", []byte(`{"name":"Team","query":"kind=tweet"}`))
	var created struct {
		Share struct {
			ID, Query, Description string
			Views                  int
		}
		Path string
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/watch/shares", []byte(`{"name":"Team","query":"?kind=article&outliers=1&foo=bar","days":7}`)), &created)
	if created.Share.Query != "kind=article" || created.Share.Description != "Articles" || !strings.HasPrefix(created.Path, "/shared/watch/") {
		t.Fatalf("created = %+v", created)
	}
	token := strings.TrimPrefix(created.Path, "/shared/watch/")
	for query, want := range map[string]string{"status=unread&label=ai": "label=ai", "status=saved": "status=saved"} {
		var other struct{ Share struct{ ID, Query string } }
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/watch/shares", []byte(`{"name":"Other","query":"`+query+`"}`)), &other)
		if other.Share.Query != want {
			t.Fatalf("%s kept as %q", query, other.Share.Query)
		}
		alice.mustDo(204, "DELETE", "/v1/watch/shares/"+other.Share.ID, nil)
	}
	var tok struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/me/tokens", []byte(`{"name":"script","scope":"write","expires_in_days":1}`)), &tok)
	if code, _ := anyone.do("POST", "/v1/watch/shares", []byte(`{"name":"Script"}`), "Authorization", "Bearer "+tok.Token); code != 403 {
		t.Fatalf("a token made a share link: %d", code)
	}

	// Anyone with the link sees what the sources published, nothing of Alice's.
	var page struct {
		Name, Attribution string
		Count             int
		Items             []map[string]any
	}
	out := anyone.mustDo(200, "GET", "/v1/shared/watch/"+token, nil)
	_ = json.Unmarshal(out, &page)
	if page.Name != "Team" || page.Count != 3 || len(page.Items) != 3 || !strings.HasPrefix(page.Attribution, "Exported from Axon Project") {
		t.Fatalf("page = %s", out)
	}
	if _, ok := page.Items[0]["status"]; ok || strings.Contains(string(out), "Alice") || strings.Contains(string(out), items[0].ID) {
		t.Fatalf("the page shows Alice's own data: %s", out)
	}
	for _, format := range []string{"csv", "json"} {
		if out := string(anyone.mustDo(200, "GET", "/v1/shared/watch/"+token+"?format="+format, nil)); strings.Contains(out, "saved") || strings.Contains(out, "unread") ||
			!strings.Contains(out, "Exported from Axon Project") || strings.Contains(out, "note_id\": \"") {
			t.Fatalf("shared %s = %s", format, out)
		}
	}
	anyone.mustDo(422, "GET", "/v1/shared/watch/"+token+"?format=docx", nil)
	anyone.mustDo(404, "GET", "/v1/shared/watch/not-a-token", nil)
	var list struct{ Items []struct{ Views int } }
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/shares", nil), &list)
	if len(list.Items) != 1 || list.Items[0].Views != 3 {
		t.Fatalf("shares = %+v", list)
	}

	// An expired link and a revoked one show nothing.
	if _, err := pool.Exec(context.Background(), `UPDATE watch_shares SET expires_at = now() - interval '1 minute' WHERE id = $1`, created.Share.ID); err != nil {
		t.Fatal(err)
	}
	anyone.mustDo(404, "GET", "/v1/shared/watch/"+token, nil)
	bob.mustDo(404, "DELETE", "/v1/watch/shares/"+created.Share.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/watch/shares/"+created.Share.ID, nil)
	alice.mustDo(404, "DELETE", "/v1/watch/shares/"+created.Share.ID, nil)
}
