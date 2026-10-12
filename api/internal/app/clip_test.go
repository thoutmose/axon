package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thoutmose/axon/api/internal/clip"
)

// A shared link becomes the page's text, from public pages only.
func TestClip(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(422, "POST", "/v1/clip", []byte(`{"url":"https://127.0.0.1:1/page"}`)) // a private address
	alice.mustDo(422, "POST", "/v1/clip", []byte(`{"url":"ftp://example.org/x"}`))

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/article":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprint(w, `<html><head><title>Spacing</title></head><body><nav>Menu</nav><article><h1>Spacing</h1><p>Space your reviews.</p></article></body></html>`)
		case "/file.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = fmt.Fprint(w, "%PDF-1.4")
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	client := clip.Client
	clip.Client = srv.Client()
	t.Cleanup(func() { clip.Client = client })

	var p struct{ Title, Markdown string }
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/clip", []byte(`{"url":"`+srv.URL+`/article"}`)), &p)
	if p.Title != "Spacing" || !strings.Contains(p.Markdown, "Space your reviews.") || strings.Contains(p.Markdown, "Menu") {
		t.Fatalf("clip = %+v", p)
	}
	alice.mustDo(422, "POST", "/v1/clip", []byte(`{"url":"`+srv.URL+`/file.pdf"}`))
}
