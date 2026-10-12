package clip

import (
	"net/url"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	base, _ := url.Parse("https://blog.example.org/posts/spacing")
	doc := `<!doctype html><html><head><title>Ignored</title><meta property="og:title" content="Why spacing works">
<meta property="og:site_name" content="Learning Blog"><style>p{color:red}</style><script>track()</script></head>
<body><nav><a href="/">Home</a> <a href="/about">About</a></nav><header>Site header</header>
<article><h1>Why spacing works</h1><p>Forgetting a <strong>little</strong> makes retrieval <em>effortful</em>.</p>
<p>See <a href="/posts/retrieval">retrieval practice</a> and <code>FSRS</code>.</p>
<ul><li>Space reviews</li><li>Mix topics</li></ul><blockquote>Effort is the point.</blockquote>
<pre>review(card)
schedule()</pre><form><input name="email"><button>Subscribe</button></form></article>
<footer>© 2026</footer></body></html>`
	p, err := Read(doc, base)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Why spacing works" || p.Site != "Learning Blog" {
		t.Errorf("page = %+v", p)
	}
	for _, want := range []string{"## Why spacing works", "Forgetting a **little** makes retrieval *effortful*.",
		"[retrieval practice](https://blog.example.org/posts/retrieval)", "`FSRS`", "- Space reviews\n- Mix topics",
		"> Effort is the point.", "```\nreview(card)\nschedule()\n```"} {
		if !strings.Contains(p.Markdown, want) {
			t.Errorf("markdown lacks %q:\n%s", want, p.Markdown)
		}
	}
	for _, bad := range []string{"Home", "Site header", "track()", "color:red", "Subscribe", "© 2026"} {
		if strings.Contains(p.Markdown, bad) {
			t.Errorf("markdown keeps %q:\n%s", bad, p.Markdown)
		}
	}
	if _, err := Read(`<html><body><script>x()</script></body></html>`, base); err == nil {
		t.Error("an empty page was clipped")
	}
}
