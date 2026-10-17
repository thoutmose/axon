package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thoutmose/axon/api/internal/watch"
)

// A YouTube video in the tech watch (EP-123): its link without the share
// tracking, then its words from the captions, which feed its analysis and
// its topics and become its summary and keywords; a second time asks
// nothing more.
func TestWatchVideoTranscript(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	players := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/feed.xml":
			_, _ = w.Write([]byte(`<rss version="2.0"><channel><title>Talks</title><link>https://www.youtube.com/channel/UC_talks</link><item><title>A talk on consensus</title>
				<link>https://youtu.be/oV9rvDllKEg?si=share123</link><guid>v1</guid><pubDate>Mon, 12 Oct 2026 08:00:00 GMT</pubDate>
				<description>A conference talk.</description></item></channel></rss>`))
		case "/youtubei/v1/player":
			players++
			//nolint:gosec // a test server pointing at itself
			_, _ = w.Write([]byte(`{"captions":{"playerCaptionsTracklistRenderer":{"captionTracks":[
				{"baseUrl":"http://` + r.Host + `/api/timedtext?v=oV9rvDllKEg","languageCode":"en","kind":"asr"}]}}}`))
		case "/api/timedtext":
			_, _ = w.Write([]byte(`<timedtext format="3"><body><p t="0" d="1">Raft elects a leader and replicates its log.</p>
				<p t="1" d="1">The leader election and the log replication keep the cluster consistent; consensus needs a quorum.</p>
				<p t="2" d="1">This is how consensus works in practice, with leader election and log replication.</p></body></timedtext>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := watch.YouTubeBase
	watch.YouTubeBase = srv.URL
	defer func() { watch.YouTubeBase = old }()

	alice.mustDo(201, "POST", "/v1/watch/topics", []byte(`{"name":"Distributed systems","keywords":["raft","consensus","leader election","log replication"]}`))
	if _, st := alice.follow(srv.URL+"/feed.xml", ""); st["new"] != 1 {
		t.Fatalf("follow stats = %v", st)
	}
	var list struct {
		Items []struct {
			ID        string   `json:"id"`
			URL       string   `json:"url"`
			Kind      string   `json:"kind"`
			Labels    []string `json:"labels"`
			SourceURL string   `json:"source_url"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/items?status=all", nil), &list)
	if len(list.Items) != 1 {
		t.Fatalf("raw = %s", alice.mustDo(200, "GET", "/v1/watch/items?status=all&outliers=1", nil))
	}
	// The original and its source are a click away (EP-126).
	if list.Items[0].URL != "https://youtube.com/watch?v=oV9rvDllKEg" || list.Items[0].Kind != "video" ||
		list.Items[0].SourceURL != "https://www.youtube.com/channel/UC_talks" {
		t.Fatalf("items = %+v", list.Items)
	}
	id := list.Items[0].ID

	var it struct {
		Content          string   `json:"content"`
		Summary          string   `json:"summary"`
		Keywords         []string `json:"keywords"`
		Labels           []string `json:"labels"`
		TranscriptSource string   `json:"transcript_source"`
		TranscriptLang   string   `json:"transcript_lang"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/watch/items/"+id+"/transcript", nil), &it)
	if it.TranscriptSource != "auto_captions" || it.TranscriptLang != "en" {
		t.Fatalf("transcribed = %+v", it)
	}
	// Not the subtitles: the video's own sentences as its summary, and five
	// to ten keywords (EP-148).
	if strings.Contains(it.Content, "Transcript") || strings.Contains(it.Content, "replicates its log") ||
		!strings.HasPrefix(it.Summary, "Raft elects a leader") || len(it.Keywords) < 5 || len(it.Keywords) > 10 {
		t.Fatalf("shown = %q, %q, %q", it.Content, it.Summary, it.Keywords)
	}
	found := false
	for _, l := range it.Labels {
		found = found || l == "distributed systems"
	}
	if !found {
		t.Fatalf("the transcript did not bring the topic: %v", it.Labels)
	}
	// Already read: YouTube is not asked again.
	alice.mustDo(200, "POST", "/v1/watch/items/"+id+"/transcript", nil)
	if players != 1 {
		t.Fatalf("asked YouTube %d times", players)
	}
}
