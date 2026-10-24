package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

// Topic names that differ only by case, underscores, hyphens, or spaces are
// one topic: creating one again is refused and an import reuses it.
func TestWatchTopicNamesMatchLoosely(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(201, "POST", "/v1/watch/topics", []byte(`{"name":"Data Engineering"}`))
	alice.mustDo(422, "POST", "/v1/watch/topics", []byte(`{"name":"data_engineering"}`))

	opml := `<?xml version="1.0"?><opml version="2.0"><body><outline text="data-engineering"><outline type="rss" text="Pipes" xmlUrl="https://example.com/pipes.xml"/></outline></body></opml>`
	code, out := alice.do("POST", "/v1/watch/import?format=opml", []byte(opml), "Content-Type", "text/x-opml")
	var imp struct {
		Added  int `json:"added"`
		Topics int `json:"topics_created"`
	}
	_ = json.Unmarshal(out, &imp)
	if code != 200 || imp.Added != 1 || imp.Topics != 0 {
		t.Fatalf("opml import = %d %s", code, out)
	}
	var topics struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	list := alice.mustDo(200, "GET", "/v1/watch/topics", nil)
	_ = json.Unmarshal(list, &topics)
	if len(topics.Items) != 1 || !bytes.Contains(list, []byte(`"Data Engineering"`)) {
		t.Fatalf("topics = %s", list)
	}
}
