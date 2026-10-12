package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAllowlistRobotsAndText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /private\n")
		case "/page":
			_, _ = fmt.Fprint(w, `<html><head><title>Big &amp; small</title><script>steal()</script></head><body><nav>menu</nav><p>Hello <b>learner</b>.</p></body></html>`)
		case "/away":
			http.Redirect(w, r, "https://evil.example/collect", http.StatusFound)
		}
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	oldAllow, oldHTTP, oldPriv := Allow, AllowHTTP, AllowPrivate
	Allow, AllowHTTP, AllowPrivate = []string{u.Hostname()}, true, true
	defer func() { Allow, AllowHTTP, AllowPrivate = oldAllow, oldHTTP, oldPriv }()

	p, err := Fetch(context.Background(), srv.URL+"/page")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Big & small" || p.Text != "Hello learner." || strings.Contains(p.Text, "steal") || strings.Contains(p.Text, "menu") {
		t.Fatalf("page = %+v", p)
	}
	if _, err := Fetch(context.Background(), srv.URL+"/private/notes"); !errors.Is(err, ErrRobots) {
		t.Fatalf("robots ignored: %v", err)
	}
	if _, err := Fetch(context.Background(), srv.URL+"/away"); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("redirect off the allowlist followed: %v", err)
	}
	if _, err := Fetch(context.Background(), "https://evil.example/?data=secret"); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("unlisted host fetched: %v", err)
	}
	AllowHTTP = false
	if _, err := Allowed("http://" + u.Hostname() + "/page"); !errors.Is(err, ErrNotAllowed) {
		t.Fatal("plain http allowed")
	}
}

func TestPrivateAddressesRefused(t *testing.T) {
	old := Allow
	Allow = []string{"localhost"}
	defer func() { Allow = old }()
	if _, err := Fetch(context.Background(), "https://localhost/x"); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("loopback fetched: %v", err)
	}
}

func TestAllowedSubdomains(t *testing.T) {
	if _, err := Allowed("https://en.wikipedia.org/wiki/Spaced_repetition"); err != nil {
		t.Fatal(err)
	}
	if _, err := Allowed("https://wikipedia.org.evil.example/"); err == nil {
		t.Fatal("suffix trick allowed")
	}
}
