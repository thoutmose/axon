package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/thoutmose/axon/api/internal/media"
)

// fakeCatalog stands in for Open Library, its covers, TMDB, and TMDB images.
type fakeCatalog struct {
	mu    sync.Mutex
	calls []string
}

var jpeg = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{1}, 64)...)

func (f *fakeCatalog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.calls = append(f.calls, r.URL.Path+"?"+r.URL.RawQuery)
	f.mu.Unlock()
	switch {
	case r.URL.Path == "/ol/search.json" && strings.HasPrefix(r.URL.Query().Get("q"), "key:"):
		_, _ = w.Write([]byte(`{"docs":[{"title":"Dune","author_name":["Frank Herbert"],"first_publish_year":1965,"cover_i":123,"number_of_pages_median":612,"subject":["Science fiction","Deserts"]}]}`))
	case r.URL.Path == "/ol/search.json" && strings.Contains(r.URL.Query().Get("q"), "Herbert"):
		_, _ = w.Write([]byte(`{"docs":[]}`)) // as Open Library: a title with its author often finds nothing
	case r.URL.Path == "/ol/search.json":
		_, _ = w.Write([]byte(`{"docs":[{"key":"/works/OL1W","title":"Dune","author_name":["Frank Herbert"],"first_publish_year":1965,"cover_i":123},{"key":"/books/OL9M","title":"Not a work"}]}`))
	case r.URL.Path == "/olc/b/id/123-M.jpg", r.URL.Path == "/img/poster.jpg":
		_, _ = w.Write(jpeg)
	case r.URL.Path == "/tmdb/search/movie":
		_, _ = w.Write([]byte(`{"results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30","overview":"A hacker learns the truth.","poster_path":"/poster.jpg"}]}`))
	case r.URL.Path == "/tmdb/movie/603":
		_, _ = w.Write([]byte(`{"id":603,"title":"The Matrix","release_date":"1999-03-30","overview":"A hacker learns the truth.","poster_path":"/poster.jpg",
			"genres":[{"name":"Action"},{"name":"Science Fiction"}],"credits":{"crew":[{"name":"Lana Wachowski","job":"Director"},{"name":"Someone","job":"Editor"}]}}`))
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeCatalog) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func useFakeCatalog(t *testing.T) *fakeCatalog {
	t.Helper()
	fake := &fakeCatalog{}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	saved := media.DefaultEndpoints
	media.DefaultEndpoints = media.Endpoints{OpenLibrary: srv.URL + "/ol", OpenLibraryCovers: srv.URL + "/olc", TMDB: srv.URL + "/tmdb", TMDBImages: srv.URL + "/img"}
	t.Cleanup(func() { media.DefaultEndpoints = saved })
	for _, table := range []string{"media_lookups", "media_covers"} {
		if _, err := pool.Exec(context.Background(), "DELETE FROM "+table); err != nil {
			t.Fatal(err)
		}
	}
	return fake
}

type mediaJSON struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Creators   string   `json:"creators"`
	Genres     []string `json:"genres"`
	Progress   int      `json:"progress"`
	Total      *int     `json:"total"`
	StartedOn  *string  `json:"started_on"`
	FinishedOn *string  `json:"finished_on"`
	Source     string   `json:"source"`
	Overview   string   `json:"overview"`
	CoverID    *string  `json:"cover_id"`
}

func (c *client) addMedia(body string) mediaJSON {
	var m mediaJSON
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/media", []byte(body)), &m)
	return m
}

// Books come from Open Library without a key, covers are served by Axon,
// lookups are cached and capped, tracking works by hand, and every
// recommendation says why.
func TestMediaTrackingCatalogueAndRecommendations(t *testing.T) {
	fake := useFakeCatalog(t)
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	if out := alice.mustDo(200, "GET", "/v1/media/catalogs", nil); !bytes.Contains(out, []byte(`"films_and_series":false`)) || !bytes.Contains(out, []byte(`"consented":false`)) {
		t.Fatalf("catalogs = %s", out)
	}
	// Lookups send the search text away, so they wait for consent.
	alice.mustDo(403, "GET", "/v1/media/search?kind=book&q=dune", nil)
	alice.mustDo(403, "POST", "/v1/media", []byte(`{"kind":"book","catalog":{"source":"openlibrary","source_id":"OL1W"}}`))
	if n := fake.count("/"); n != 0 {
		t.Fatalf("%d catalogue calls without consent", n)
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/external_metadata", []byte(`{"granted":true}`))
	alice.mustDo(503, "GET", "/v1/media/search?kind=film&q=matrix", nil) // no TMDB key: say so, add by hand instead
	out := alice.mustDo(200, "GET", "/v1/media/search?kind=book&q=dune", nil)
	if !bytes.Contains(out, []byte(`"source_id":"OL1W"`)) || bytes.Contains(out, []byte("Not a work")) || !bytes.Contains(out, []byte(`"has_cover":true`)) {
		t.Fatalf("search = %s", out)
	}
	alice.mustDo(200, "GET", "/v1/media/search?kind=book&q=Dune", nil)
	if n := fake.count("/ol/search.json"); n != 1 {
		t.Fatalf("Open Library asked %d times, want 1 (the second search is cached)", n)
	}

	dune := alice.addMedia(`{"kind":"book","catalog":{"source":"openlibrary","source_id":"OL1W"}}`)
	if dune.Title != "Dune" || dune.Creators != "Frank Herbert" || dune.Total == nil || *dune.Total != 612 || dune.CoverID == nil || dune.Status != "want" {
		t.Fatalf("from the catalogue: %+v", dune)
	}
	code, img := alice.do("GET", "/v1/media/covers/"+*dune.CoverID, nil)
	if code != 200 || !bytes.Equal(img, jpeg) {
		t.Fatalf("cover = %d, %d bytes", code, len(img))
	}
	alice.mustDo(422, "POST", "/v1/media", []byte(`{"kind":"book","catalog":{"source":"openlibrary","source_id":"../etc"}}`))

	// By hand, with ratings and progress.
	arrival := alice.addMedia(`{"title":"Arrival","kind":"film","creators":"Denis Villeneuve","genres":["Science fiction"]}`)
	alice.mustDo(422, "PATCH", "/v1/media/"+arrival.ID, []byte(`{"rating":11}`))
	alice.mustDo(200, "PATCH", "/v1/media/"+arrival.ID, []byte(`{"status":"done","rating":9}`))
	alice.addMedia(`{"title":"Dune: Part One","kind":"film","creators":"Denis Villeneuve, Eric Roth"}`)
	alice.addMedia(`{"title":"Something else","kind":"film"}`)
	alice.mustDo(422, "POST", "/v1/media", []byte(`{"title":"No kind"}`))
	var book mediaJSON
	_ = json.Unmarshal(alice.mustDo(200, "PATCH", "/v1/media/"+dune.ID, []byte(`{"status":"doing","progress":200}`)), &book)
	if book.StartedOn == nil || book.Progress != 200 {
		t.Fatalf("doing: %+v", book)
	}
	alice.mustDo(422, "PATCH", "/v1/media/"+dune.ID, []byte(`{"progress":700}`)) // more than the 612 pages
	alice.mustDo(422, "PATCH", "/v1/media/"+dune.ID, []byte(`{"started_on":"2026-10-10","finished_on":"2026-10-01"}`))

	var recs struct {
		Items []struct {
			Item    mediaJSON `json:"item"`
			Reasons []string  `json:"reasons"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/media/recommendations?kind=film", nil), &recs)
	if len(recs.Items) != 2 || recs.Items[0].Item.Title != "Dune: Part One" ||
		!strings.Contains(strings.Join(recs.Items[0].Reasons, " "), "By Denis Villeneuve, like Arrival, which you rated 9/10.") ||
		!strings.HasPrefix(recs.Items[1].Reasons[0], "On your list since") {
		t.Fatalf("recommendations = %+v", recs.Items)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/media/recommendations", nil), &recs)
	for _, r := range recs.Items {
		if len(r.Reasons) == 0 {
			t.Fatalf("%s has no reason", r.Item.Title)
		}
	}

	var done mediaJSON
	_ = json.Unmarshal(alice.mustDo(200, "PATCH", "/v1/media/"+dune.ID, []byte(`{"status":"done"}`)), &done)
	if done.Progress != 612 || done.FinishedOn == nil {
		t.Fatalf("done: %+v", done)
	}
	if out := alice.mustDo(200, "GET", "/v1/media?kind=book&status=done", nil); !bytes.Contains(out, []byte(dune.ID)) || bytes.Contains(out, []byte(arrival.ID)) {
		t.Fatalf("filtered list = %s", out)
	}
	alice.mustDo(400, "GET", "/v1/media?kind=game", nil)

	// Private to its owner.
	bob.mustDo(404, "GET", "/v1/media/"+dune.ID, nil)
	bob.mustDo(404, "PATCH", "/v1/media/"+dune.ID, []byte(`{"rating":1}`))
	bob.mustDo(404, "DELETE", "/v1/media/"+dune.ID, nil)
	if out := bob.mustDo(200, "GET", "/v1/media", nil); bytes.Contains(out, []byte("Dune")) {
		t.Fatalf("bob sees %s", out)
	}
	alice.mustDo(200, "GET", "/v1/media/"+dune.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/media/"+dune.ID, nil)
	alice.mustDo(404, "GET", "/v1/media/"+dune.ID, nil)

	// Lookups are capped per person, cached or not.
	limited := false
	for range 40 {
		if code, _ := alice.do("GET", "/v1/media/search?kind=book&q=dune", nil); code == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("no lookup limit")
	}
}

// With a TMDB key, films come from TMDB with genres and director, and TMDB
// text and covers are dropped once older than TMDB's six months.
func TestMediaTMDBAndRetention(t *testing.T) {
	fake := useFakeCatalog(t)
	t.Setenv("TMDB_API_KEY", "test-key")
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(204, "PUT", "/v1/me/consents/external_metadata", []byte(`{"granted":true}`))

	if out := alice.mustDo(200, "GET", "/v1/media/search?kind=film&q=matrix", nil); !bytes.Contains(out, []byte(`"title":"The Matrix"`)) {
		t.Fatalf("search = %s", out)
	}
	if fake.count("/tmdb/search/movie?api_key=test-key") != 1 {
		t.Fatalf("calls = %v", fake.calls)
	}
	m := alice.addMedia(`{"kind":"film","status":"done","rating":8,"catalog":{"source":"tmdb","source_id":"603"}}`)
	if m.Creators != "Lana Wachowski" || len(m.Genres) != 2 || m.Overview == "" || m.CoverID == nil || m.Source != "tmdb" {
		t.Fatalf("from TMDB: %+v", m)
	}

	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE media_items SET fetched_at = now() - interval '181 days'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE media_covers SET fetched_at = now() - interval '181 days'`); err != nil {
		t.Fatal(err)
	}
	if err := media.Prune(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var after mediaJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/media/"+m.ID, nil), &after)
	if after.Overview != "" || after.CoverID != nil || after.Title != "The Matrix" || after.Creators != "Lana Wachowski" {
		t.Fatalf("after six months: %+v", after)
	}
}

// The Media agent reads the list without reviews or catalogue text, and
// its suggestions wait for approval.
func TestMediaAgentReadsTheListAndProposes(t *testing.T) {
	useFakeCatalog(t)
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.enableAgents()
	arrival := alice.addMedia(`{"title":"Arrival","kind":"film","status":"done","rating":9,"creators":"Denis Villeneuve","review":"private thoughts"}`)
	if arrival.ID == "" {
		t.Fatal("not created")
	}
	fake.load(
		toolUse("list_media", `{}`, "media_recommendations", `{"kind":"film"}`),
		toolUse("suggest_media", `{"kind":"film","title":"Sicario","creators":"Denis Villeneuve","year":2015,"reason":"Same director as Arrival, which you rated 9."}`),
		finalText("Proposed Sicario."),
	)
	r := alice.runAgent("media", "What next?")
	if r.Run.Status != "completed" || len(r.Proposals) != 1 || !strings.Contains(r.Proposals[0].Summary, "Sicario") {
		t.Fatalf("run = %+v", r)
	}
	for _, req := range fake.lastRequests() {
		b, _ := json.Marshal(req.Messages)
		if bytes.Contains(b, []byte("private thoughts")) {
			t.Fatal("the review reached the model")
		}
	}
	if out := alice.mustDo(200, "GET", "/v1/media", nil); bytes.Contains(out, []byte("Sicario")) {
		t.Fatal("added before approval")
	}
	alice.mustDo(200, "POST", "/v1/proposals/"+r.Proposals[0].ID+"/approve", nil)
	if out := alice.mustDo(200, "GET", "/v1/media?status=want", nil); !bytes.Contains(out, []byte(`"title":"Sicario"`)) {
		t.Fatalf("after approval: %s", out)
	}
}

// Items added by hand, imported, or from the demo have no cover: "Find
// missing covers" looks each one up (with the person's consent), takes a
// cover only from the same work, and remembers the lookups.
func TestFindMissingCovers(t *testing.T) {
	fake := useFakeCatalog(t)
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	dune := alice.addMedia(`{"title":"Dune","kind":"book","creators":"Frank Herbert"}`)
	other := alice.addMedia(`{"title":"Dune","kind":"book","creators":"Someone Else"}`)
	film := alice.addMedia(`{"title":"The Matrix","kind":"film"}`)
	if dune.CoverID != nil {
		t.Fatal("a hand-made item came with a cover")
	}
	alice.mustDo(403, "POST", "/v1/media/covers/find", nil)

	alice.mustDo(204, "PUT", "/v1/me/consents/external_metadata", []byte(`{"granted":true}`))
	var res struct {
		Checked        int  `json:"checked"`
		Found          int  `json:"found"`
		FilmsAndSeries bool `json:"films_and_series"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/media/covers/find", nil), &res)
	if res.Checked != 2 || res.Found != 1 || res.FilmsAndSeries {
		t.Fatalf("find covers = %+v; want 2 books checked, 1 found, films need TMDB", res)
	}
	var got mediaJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/media/"+dune.ID, nil), &got)
	if got.CoverID == nil {
		t.Fatal("Dune by Frank Herbert got no cover")
	}
	alice.mustDo(200, "GET", "/v1/media/covers/"+*got.CoverID, nil)
	for _, id := range []string{other.ID, film.ID} {
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/media/"+id, nil), &got)
		if got.CoverID != nil {
			t.Fatalf("%s got a cover that is not its own", got.Title)
		}
	}
	// Again: the one without a match is checked from the cache.
	searches := fake.count("/ol/search.json")
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/media/covers/find", nil), &res)
	if res.Checked != 1 || res.Found != 0 || fake.count("/ol/search.json") != searches {
		t.Fatalf("again = %+v, %d new searches", res, fake.count("/ol/search.json")-searches)
	}
}
