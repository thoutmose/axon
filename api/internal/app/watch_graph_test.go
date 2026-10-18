package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// graphStories are two threads of news: vector search in Postgres, and Rust
// in the Linux kernel, each told differently by three items.
var graphStories = [][2]string{
	{"Vector search lands in Postgres", "Vector databases are everywhere this year. The pgvector extension brings vector search to Postgres with approximate nearest neighbour indexes. Teams keep embeddings next to their relational data and avoid a second system. Benchmarks show HNSW indexes answer queries in milliseconds on millions of rows."},
	{"Do you need a vector database?", "Many startups sell vector databases for retrieval augmented generation. Yet Postgres with pgvector covers most workloads: embeddings, filters, and joins in one query. The extension added HNSW indexes and parallel builds, which closed the gap for vector search at moderate scale."},
	{"Scaling embeddings with pgvector", "A case study on vector search at scale. The company stored two hundred million embeddings in Postgres using pgvector and partitioned tables. Vector databases were considered, but operations favoured one system. Index builds took hours, and quantisation cut memory by four."},
	{"Rust drivers reach the Linux kernel", "The Linux kernel merged its first Rust drivers for network devices. Kernel maintainers report fewer memory safety bugs in the new code. Rust for Linux now has stable bindings for common subsystems, and more drivers are under review for the next release."},
	{"Why kernel developers argue about Rust", "Not every Linux kernel maintainer welcomes Rust. Some worry about the bindings, toolchain versions, and review load. Supporters point to memory safety and the drivers already shipped. The debate shaped how the kernel accepts Rust code in each subsystem."},
	{"A year of Rust in the Linux kernel", "One year on, Rust in the Linux kernel carries graphics and storage drivers. Memory safety bugs in those drivers dropped compared with C. The kernel team plans better bindings and a minimum Rust toolchain, and more subsystems now accept Rust drivers."},
}

// The knowledge graph of the news: entities of the ontology with their
// classes, the concepts that go with them, grouped, each summed up on
// hover and pointing to the notes that share it.
func TestWatchGraph(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString(`<rss version="2.0"><channel><title>Tech</title>`)
		for i, s := range graphStories {
			b.WriteString(`<item><title>` + s[0] + `</title><link>http://` + r.Host + `/s/` + string(rune('a'+i)) + `</link><guid>` + string(rune('a'+i)) +
				`</guid><pubDate>Mon, 0` + string(rune('5'+i/2)) + ` Oct 2026 08:00:00 GMT</pubDate><description>` + s[1] + `</description></item>`)
		}
		b.WriteString(`</channel></rss>`)
		_, _ = w.Write([]byte(b.String()))
	}))
	defer srv.Close()
	if _, st := alice.follow(srv.URL+"/feed.xml", ""); st["new"] != 6 {
		t.Fatalf("stats = %v", st)
	}
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/permanent/vectors.md","content":"---\ntype: permanent\n---\n# Vector search\n\nVector search finds neighbours of embeddings. Vector search in Postgres needs an index."}`))

	var g struct {
		Nodes []struct {
			ID, Kind, Class  string
			Label, Summary   string
			Community, Count int
			Fringe           bool
			Related          []string
			Items            []struct{ Title string }
			InNotes          []struct{ Title string } `json:"in_notes"`
		}
		Edges       []struct{ From, To, Predicate string }
		Communities []struct {
			ID    int
			Label string
		}
		Stats struct{ Items, Concepts, Communities, Bridges, Entities int }
	}
	out := alice.mustDo(200, "GET", "/v1/watch/graph?from=2026-09-01&to=2026-11-01", nil)
	_ = json.Unmarshal(out, &g)
	if g.Stats.Items != 6 || len(g.Edges) == 0 || len(g.Communities) != 2 || g.Stats.Bridges != 1 || g.Stats.Entities < 4 {
		t.Fatalf("graph = %+v", g)
	}
	byLabel := map[string]int{}
	for _, want := range []string{"Linux", "pgvector", "memory safety", "Vector search"} {
		if !strings.Contains(string(out), `"label":"`+want+`"`) {
			t.Fatalf("no %q in %s", want, out)
		}
	}
	for i, n := range g.Nodes {
		byLabel[n.Label] = i
		if n.Fringe || n.Community == 0 || n.Kind != "class" && (n.Summary == "" || len(n.Items) == 0) {
			t.Fatalf("node = %+v", n)
		}
	}
	kernel, pgvector, safety, search := g.Nodes[byLabel["Linux"]], g.Nodes[byLabel["pgvector"]], g.Nodes[byLabel["memory safety"]], g.Nodes[byLabel["Vector search"]]
	if kernel.Kind != "entity" || kernel.Class != "Operating system" || safety.Kind != "concept" {
		t.Fatalf("kinds: %+v / %+v", kernel, safety)
	}
	isA := false
	for _, e := range g.Edges {
		isA = isA || e.From == kernel.ID && e.Predicate == "is_a" && e.To == "k:Operating system"
	}
	if !isA {
		t.Fatalf("Linux is not an operating system: %+v", g.Edges)
	}
	if kernel.Community == pgvector.Community || kernel.Community != safety.Community {
		t.Fatalf("communities: kernel %d, pgvector %d, memory safety %d", kernel.Community, pgvector.Community, safety.Community)
	}
	if !strings.Contains(strings.ToLower(safety.Summary), "memory safety") || len(search.InNotes) != 1 || search.InNotes[0].Title != "Vector search" {
		t.Fatalf("hover: %+v / %+v", safety, search)
	}
	// With no news older than a week, nothing is rising: it is all new.
	var fresh struct{ Nodes []struct{ Trend float64 } }
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/watch/graph?from=2026-10-01&to=2026-10-09", nil), &fresh)
	for _, n := range fresh.Nodes {
		if n.Trend != 0 {
			t.Fatalf("trend without history: %+v", fresh)
		}
	}
	if len(fresh.Nodes) == 0 {
		t.Fatal("no concepts in the week")
	}
	// Bob sees nothing; bad periods are refused.
	var empty struct{ Nodes []any }
	_ = json.Unmarshal(bob.mustDo(200, "GET", "/v1/watch/graph", nil), &empty)
	if len(empty.Nodes) != 0 {
		t.Fatalf("bob = %+v", empty)
	}
	alice.mustDo(422, "GET", "/v1/watch/graph?from=2026-11-01&to=2026-09-01", nil)
	alice.mustDo(422, "GET", "/v1/watch/graph?min=2", nil)
}
