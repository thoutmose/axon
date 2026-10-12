package images

import (
	"strings"
	"testing"
)

func TestSanitizeSVG(t *testing.T) {
	in := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100" onload="alert(1)">
<script>alert(1)</script>
<style>circle{fill:url(https://evil.example/x)}</style>
<defs><linearGradient id="g"><stop offset="0" stop-color="#f00"/></linearGradient></defs>
<circle cx="50" cy="50" r="40" fill="url(#g)" stroke="url(https://evil.example/)"/>
<a href="javascript:alert(1)"><text x="10" y="20">click</text></a>
<image href="https://evil.example/track.png"/>
<foreignObject><div>html</div></foreignObject>
<text x="50" y="95" text-anchor="middle">Mitochondria &amp; ATP</text>
</svg>`
	out, err := SanitizeSVG(in)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, bad := range []string{"script", "onload", "style", "evil", "javascript", "<a", "image", "foreignObject", "click", "xlink"} {
		if strings.Contains(s, bad) {
			t.Errorf("sanitised SVG keeps %q:\n%s", bad, s)
		}
	}
	for _, good := range []string{`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">`, `fill="url(#g)"`, "Mitochondria &amp; ATP", `<stop offset="0" stop-color="#f00"></stop>`} {
		if !strings.Contains(s, good) {
			t.Errorf("sanitised SVG lost %q:\n%s", good, s)
		}
	}
	for _, bad := range []string{"<html/>", "not xml <", "<svg><g></svg>", strings.Repeat("x", MaxSVG+1)} {
		if _, err := SanitizeSVG(bad); err == nil {
			t.Errorf("accepted %.30q", bad)
		}
	}
}
