package watch

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCanonical(t *testing.T) {
	for in, want := range map[string]string{
		"http://WWW.Example.com/a/b/?utm_source=rss&b=2&a=1#top": "https://example.com/a/b?a=1&b=2",
		"https://example.com/":                              "https://example.com",
		"https://medium.com/@ada/post-1?source=rss----1":    "https://medium.com/@ada/post-1",
		"https://example.substack.com/p/x?utm_medium=email": "https://example.substack.com/p/x",
		"https://example.com:443/x?fbclid=1":                "https://example.com/x",
		// YouTube in all its forms is one video, without its share tracking (EP-123).
		"https://youtu.be/dQw4w9WgXcQ?si=Ab12":                                 "https://youtube.com/watch?v=dQw4w9WgXcQ",
		"https://m.youtube.com/watch?v=dQw4w9WgXcQ&feature=share&pp=yg":        "https://youtube.com/watch?v=dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/abcDEF12345?feature=share":             "https://youtube.com/watch?v=abcDEF12345",
		"https://youtu.be/dQw4w9WgXcQ?t=42":                                    "https://youtube.com/watch?t=42&v=dQw4w9WgXcQ",
		"https://www.linkedin.com/posts/x?trk=public_post&rcm=ACoAA":           "https://linkedin.com/posts/x",
		"https://example.com/x?mibextid=Zb&__twitter_impression=true&gs_lcp=1": "https://example.com/x",
	} {
		if got := Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
	if string(URLHash("https://example.com/x?utm_source=a")) != string(URLHash("http://www.example.com/x/")) {
		t.Fatal("the same page hashes differently")
	}
}

func TestCleanLinksAndVideoID(t *testing.T) {
	md := "See [the talk](https://youtu.be/dQw4w9WgXcQ?si=x) and https://example.com/a?utm_source=rss&id=3."
	if got := CleanLinks(md); got != "See [the talk](https://youtube.com/watch?v=dQw4w9WgXcQ) and https://example.com/a?id=3." {
		t.Fatalf("CleanLinks = %q", got)
	}
	for in, want := range map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL1": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                         "dQw4w9WgXcQ",
		"https://youtube.com/shorts/abcDEF12345":               "abcDEF12345",
		"https://www.youtube.com/embed/abcDEF12345":            "abcDEF12345",
		"https://example.com/watch?v=dQw4w9WgXcQ":              "",
		"https://youtube.com/watch?v=short":                    "",
	} {
		if got := VideoID(in); got != want {
			t.Errorf("VideoID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSimHash(t *testing.T) {
	words := func(s string) []string { return strings.Fields(s) }
	article := strings.Repeat("postgres seventeen adds incremental backup and faster vacuum for large tables in production clusters ", 4) +
		"the release also improves logical replication failover json table support and the planner for partitioned tables " +
		"benchmarks show lower write amplification under heavy update workloads and smaller wal volume on busy primaries"
	a := SimHash(words(article))
	b := SimHash(words(article + " read more on our blog"))
	c := SimHash(words(strings.Repeat("apple announces a new laptop with a faster chip and a brighter screen for creative professionals this autumn ", 4)))
	if Distance(a, b) > NearDuplicate {
		t.Fatalf("near duplicates differ by %d bits", Distance(a, b))
	}
	if Distance(a, c) <= NearDuplicate*4 {
		t.Fatalf("different stories differ by only %d bits", Distance(a, c))
	}
	// Short texts: the overlap of their words decides.
	if j := Jaccard(words("postgres 17 adds incremental backup"), words("postgres 17 adds incremental backup support")); j < 0.8 {
		t.Fatalf("short near duplicates: jaccard %v", j)
	}
	if j := Jaccard(words("a b c"), words("b c d")); j != 0.5 {
		t.Fatalf("jaccard = %v", j)
	}
}

func TestSchedule(t *testing.T) {
	paris, _ := time.LoadLocation("Europe/Paris")
	at := time.Date(2026, 10, 14, 22, 30, 0, 0, time.UTC) // 00:30 in Paris
	for expr, want := range map[string]string{
		"0 0 * * *":    "2026-10-16 00:00", // midnight, local: today's has passed
		"@daily":       "2026-10-16 00:00",
		"30 6 * * 1-5": "2026-10-15 06:30",
		"*/30 * * * *": "2026-10-15 01:00",
		"0 9 1 * *":    "2026-11-01 09:00",
		"0 8 * * 0":    "2026-10-18 08:00", // Sunday
		"0 8 * * 7":    "2026-10-18 08:00",
		"0 12 13 * 5":  "2026-10-16 12:00", // the 13th or a Friday
	} {
		s, err := ParseSchedule(expr)
		if err != nil {
			t.Fatalf("%q: %v", expr, err)
		}
		if got := s.Next(at, paris).Format("2006-01-02 15:04"); got != want {
			t.Errorf("%q next = %s, want %s", expr, got, want)
		}
	}
	for _, bad := range []string{"* * * * *", "*/5 * * * *", "0 0 * *", "61 0 * * *", "0 0 32 * *", "a b c d e", "0 0 * * 8", "5-1 * * * *"} {
		if _, err := ParseSchedule(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestKnownFeeds(t *testing.T) {
	for in, want := range map[string]string{
		"https://medium.com/@ada":                                  "https://medium.com/feed/@ada",
		"https://medium.com/@ada/some-post-123":                    "https://medium.com/feed/@ada",
		"https://medium.com/better-programming":                    "https://medium.com/feed/better-programming",
		"https://ada.medium.com/":                                  "https://ada.medium.com/feed",
		"https://www.platformer.substack.com/p/x":                  "https://platformer.substack.com/feed",
		"https://www.youtube.com/channel/UCabcdefghijklmnopqrstuv": "https://www.youtube.com/feeds/videos.xml?channel_id=UCabcdefghijklmnopqrstuv",
		"https://www.youtube.com/playlist?list=PL123":              "https://www.youtube.com/feeds/videos.xml?playlist_id=PL123",
		"https://github.com/golang/go":                             "https://github.com/golang/go/releases.atom",
		"https://github.com/torvalds":                              "https://github.com/torvalds.atom",
		"https://old.reddit.com/r/golang/":                         "https://www.reddit.com/r/golang/.rss",
		"https://dev.to/ben":                                       "https://dev.to/feed/ben",
	} {
		if got, ok := Known(in); !ok || got != want {
			t.Errorf("Known(%q) = %q %v, want %q", in, got, ok, want)
		}
	}
	if _, ok := Known("https://example.com/blog"); ok {
		t.Fatal("an unknown site has a known feed")
	}
}

func TestAlternates(t *testing.T) {
	page := `<html><head><link rel="stylesheet" href="/a.css"><link rel="alternate" type="application/rss+xml" title="RSS" href="/feed.xml">
<link rel="alternate" type="application/atom+xml" href="https://cdn.example/atom"><link rel="alternate" hreflang="fr" href="/fr"></head><body><link rel="alternate" type="application/rss+xml" href="/late"></body></html>`
	base, _ := url.Parse("https://blog.example/post/1")
	got := Alternates([]byte(page), base)
	if strings.Join(got, " ") != "https://blog.example/feed.xml https://cdn.example/atom" {
		t.Fatalf("alternates = %v", got)
	}
}
