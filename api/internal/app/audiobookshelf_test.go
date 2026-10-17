package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Podcasts from the person's Audiobookshelf (EP-124): connected with its
// address and API token (never shown again), each show a tech watch
// source, each episode an item; an episode's words come from the person's
// own copy through Whisper; disconnecting keeps what was read.
func TestAudiobookshelf(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	whispered := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/audio/transcriptions" { // the operator's Whisper
			f, _, _ := r.FormFile("file")
			b, _ := io.ReadAll(f)
			whispered = string(b)
			_, _ = w.Write([]byte(`{"text":"Raft elects a leader, and the leader replicates the log to a quorum.","language":"english"}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/libraries":
			_, _ = w.Write([]byte(`{"libraries":[{"id":"lib_pod","name":"Podcasts","mediaType":"podcast"}]}`))
		case "/api/libraries/lib_pod/items":
			_, _ = w.Write([]byte(`{"results":[{"id":"li_1","media":{"metadata":{"title":"Software Engineering Radio"}}}],"total":1}`))
		case "/api/items/li_1":
			_, _ = w.Write([]byte(`{"media":{"metadata":{"title":"Software Engineering Radio","author":"IEEE"},"episodes":[
				{"id":"ep_1","title":"Raft in practice","description":"How a cluster agrees.","publishedAt":1760000000000,
				 "enclosure":{"url":"https://example.com/ep1.mp3"},"audioFile":{"ino":"901","metadata":{"filename":"ep1.mp3"}}}]}}`))
		case "/api/items/li_1/file/901":
			_, _ = w.Write([]byte("my-own-copy"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// A server on a private network is refused unless the operator allows it.
	alice.mustDo(422, "PUT", "/v1/audiobookshelf", []byte(`{"url":"`+srv.URL+`","token":"tok"}`))
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	alice.mustDo(422, "PUT", "/v1/audiobookshelf", []byte(`{"url":"`+srv.URL+`","token":"wrong"}`))
	alice.mustDo(422, "PUT", "/v1/audiobookshelf", []byte(`{"url":"ftp://x","token":"tok"}`))
	var st struct {
		Connected bool   `json:"connected"`
		URL       string `json:"url"`
		Shows     int    `json:"shows"`
	}
	out := alice.mustDo(200, "PUT", "/v1/audiobookshelf", []byte(`{"url":"`+srv.URL+`/","token":"tok"}`))
	_ = json.Unmarshal(out, &st)
	if !st.Connected || st.URL != srv.URL || st.Shows != 1 || bytes.Contains(out, []byte("tok")) {
		t.Fatalf("connected = %s", out)
	}
	if bytes.Contains(alice.mustDo(200, "GET", "/v1/audiobookshelf", nil), []byte("tok")) {
		t.Fatal("the token was shown")
	}
	var list struct {
		Items []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Kind  string `json:"kind"`
			Audio string `json:"audio"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/items?status=all", nil), &list)
	if len(list.Items) != 1 || list.Items[0].Title != "Raft in practice" || list.Items[0].Kind != "podcast" {
		t.Fatalf("items = %+v", list.Items)
	}
	// Syncing again finds nothing new.
	var synced struct {
		Shows    int `json:"shows"`
		Episodes int `json:"new_episodes"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/audiobookshelf/sync", nil), &synced)
	if synced.Shows != 1 || synced.Episodes != 0 {
		t.Fatalf("second sync = %+v", synced)
	}

	// With Whisper, the episode's words come from the person's own copy.
	t.Setenv("AXON_WHISPER_URL", srv.URL)
	var it struct {
		TranscriptSource string `json:"transcript_source"`
		Summary          string `json:"summary"` // the words' own sentences (EP-148)
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/watch/items/"+list.Items[0].ID+"/transcript", nil), &it)
	if it.TranscriptSource != "whisper" || whispered != "my-own-copy" || !bytes.Contains([]byte(it.Summary), []byte("quorum")) {
		t.Fatalf("transcribed = %+v, sent %q", it, whispered)
	}

	// Disconnecting stops following the shows and keeps the episodes.
	alice.mustDo(204, "DELETE", "/v1/audiobookshelf", nil)
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/audiobookshelf", nil), &st)
	if st.Connected {
		t.Fatal("still connected")
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/items?status=all", nil), &list)
	if len(list.Items) != 1 {
		t.Fatalf("after disconnecting = %+v", list.Items)
	}
}
