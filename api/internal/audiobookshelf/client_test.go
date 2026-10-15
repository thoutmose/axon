package audiobookshelf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeABS answers like an Audiobookshelf server with one podcast library,
// one show, and two episodes, for the token "tok".
func fakeABS(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/libraries":
			_, _ = w.Write([]byte(`{"libraries":[{"id":"lib_books","name":"Books","mediaType":"book"},{"id":"lib_pod","name":"Podcasts","mediaType":"podcast"}]}`))
		case "/api/libraries/lib_pod/items":
			_, _ = w.Write([]byte(`{"results":[{"id":"li_1","mediaType":"podcast","media":{"metadata":{"title":"Software Engineering Radio","author":"IEEE","imageUrl":"https://example.com/se.png","language":"en"}}}],"total":1}`))
		case "/api/items/li_1":
			_, _ = w.Write([]byte(`{"id":"li_1","media":{"metadata":{"title":"Software Engineering Radio","author":"IEEE","imageUrl":"https://example.com/se.png"},"episodes":[
				{"id":"ep_1","title":"Raft in practice","description":"<p>How <b>Raft</b> elects a leader.</p>","publishedAt":1760000000000,
				 "enclosure":{"url":"https://example.com/ep1.mp3","type":"audio/mpeg"},"audioFile":{"ino":"901","metadata":{"filename":"ep1.mp3"}}},
				{"id":"ep_2","title":"Local only","description":"Recorded at home.","pubDate":"Mon, 12 Oct 2026 08:00:00 GMT","audioFile":{"ino":"902","metadata":{"filename":"ep2.m4a"}}}]}}`))
		case "/api/items/li_1/file/901":
			_, _ = w.Write([]byte("mp3-bytes"))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestClient(t *testing.T) {
	srv := fakeABS(t)
	defer srv.Close()
	ctx := context.Background()
	if _, err := (&Client{Base: srv.URL, Token: "wrong"}).Libraries(ctx); err == nil {
		t.Fatal("a wrong token was accepted")
	}
	c := &Client{Base: srv.URL + "/", Token: "tok"}
	libs, err := c.Libraries(ctx)
	if err != nil || len(libs) != 1 || libs[0].ID != "lib_pod" {
		t.Fatalf("podcast libraries = %+v, %v", libs, err)
	}
	shows, err := c.Shows(ctx, "lib_pod")
	if err != nil || len(shows) != 1 || shows[0].Title != "Software Engineering Radio" {
		t.Fatalf("shows = %+v, %v", shows, err)
	}
	feed, err := c.Feed(ctx, "li_1")
	if err != nil || feed.Title != "Software Engineering Radio" || len(feed.Items) != 2 {
		t.Fatalf("feed = %+v, %v", feed, err)
	}
	ep := feed.Items[0]
	if ep.GUID != "abs:li_1:ep_1:901" || ep.URL != srv.URL+"/item/li_1?episode=ep_1" || ep.Audio != "https://example.com/ep1.mp3" ||
		ep.Published.Year() != 2025 || ep.Author != "IEEE" || ep.Image != "https://example.com/se.png" {
		t.Fatalf("episode = %+v", ep)
	}
	// An episode with no public file still counts as one, through its page.
	if feed.Items[1].Audio != srv.URL+"/item/li_1?episode=ep_2" || feed.Items[1].Published.Day() != 12 {
		t.Fatalf("local episode = %+v", feed.Items[1])
	}
	audio, name, err := c.Audio(ctx, "abs:li_1:ep_1:901")
	if err != nil || string(audio) != "mp3-bytes" || name != "ep1.mp3" {
		t.Fatalf("audio = %q %q %v", audio, name, err)
	}
	if _, _, err := c.Audio(ctx, "abs:../../x"); err == nil {
		t.Fatal("a crafted guid was followed")
	}
}
