package watch

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseRSS(t *testing.T) {
	b, _ := os.ReadFile("testdata/substack.xml")
	f, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "Platformer & Co" || f.SiteURL != "https://example.substack.com" || len(f.Items) != 2 {
		t.Fatalf("feed = %+v", f)
	}
	e := f.Items[0]
	if e.Title != "Why Postgres keeps winning" || e.Author != "Ada Writer" || e.GUID != "https://example.substack.com/p/why-postgres" ||
		!strings.Contains(e.Content, "<b>keeps</b>") || e.Image != "https://cdn.example/cover.jpg" ||
		!e.Published.Equal(time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("item = %+v", e)
	}
	if s := f.Items[1]; s.GUID != s.URL || len(s.Categories) != 1 || !s.Published.Equal(time.Date(2026, 10, 12, 6, 0, 0, 0, time.UTC)) {
		t.Fatalf("second = %+v", s)
	}
}

func TestParseAtom(t *testing.T) {
	b, _ := os.ReadFile("testdata/youtube.xml")
	f, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "Some Channel" || f.SiteURL != "https://www.youtube.com/channel/UC123" || len(f.Items) != 2 {
		t.Fatalf("feed = %+v", f)
	}
	v := f.Items[0]
	if v.URL != "https://www.youtube.com/watch?v=abc" || v.Author != "Gopher" || v.Summary != "What is new &amp; why." || v.Image == "" ||
		!v.Published.Equal(time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("video = %+v", v)
	}
	r := f.Items[1]
	if r.Title != "v1.2.0 stable" || r.Content != "<p>Fixes &amp; features</p>" || r.Categories[0] != "release" {
		t.Fatalf("release = %+v", r)
	}
}

func TestParseRDFAndJSON(t *testing.T) {
	rdf := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns="http://purl.org/rss/1.0/" xmlns:dc="http://purl.org/dc/elements/1.1/">
<channel rdf:about="https://x.example/"><title>RDF</title><link>https://x.example/</link></channel>
<item rdf:about="https://x.example/1"><title>One</title><link>https://x.example/1</link><dc:date>2026-10-01T10:00:00Z</dc:date></item></rdf:RDF>`
	f, err := Parse([]byte(rdf))
	if err != nil || f.Title != "RDF" || len(f.Items) != 1 || f.Items[0].GUID != "https://x.example/1" || f.Items[0].Published.IsZero() {
		t.Fatalf("rdf = %+v %v", f, err)
	}
	js := `{"version":"https://jsonfeed.org/version/1.1","title":"J","home_page_url":"https://j.example","items":[{"id":"1","url":"https://j.example/1","title":"T","content_text":"a < b","date_published":"2026-10-01T10:00:00Z","authors":[{"name":"N"}],"tags":["go"]}]}`
	f, err = Parse([]byte(js))
	if err != nil || f.Items[0].Content != "<p>a &lt; b</p>" || f.Items[0].Author != "N" || f.Items[0].GUID != "1" {
		t.Fatalf("json = %+v %v", f, err)
	}
	for _, bad := range []string{"<html><body>hi</body></html>", "{}", "", "not xml"} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}
