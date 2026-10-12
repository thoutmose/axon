package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// The knowledge explorer filters by PARA and type, counts for charts, and
// connects notes from links and local NLP, every edge with its reason.
func TestKnowledgeExplorerAndNLPGraph(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	note := func(path, content string) string {
		var n noteJSON
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"`+path+`","content":`+quote(content)+`}`)), &n)
		return n.ID
	}
	spaced := note("Zettel/permanent/spaced.md", "---\ntype: permanent\ntags: [memory]\n---\n# Spaced repetition\n\nSpaced repetition schedules memory reviews at growing intervals. Retrieval practice and spaced repetition beat rereading.\n")
	retrieval := note("Zettel/permanent/retrieval.md", "---\ntype: permanent\n---\n# Retrieval practice\n\nRetrieval practice strengthens memory: testing yourself with spaced repetition intervals beats rereading.\n")
	ddia := note("Zettel/literature/ddia.md", "---\ntype: literature\n---\n# Reading notes\n\nI read Designing Data-Intensive Applications by Martin Kleppmann.\n")
	raft := note("3-Resources/raft.md", "# Consensus\n\nRaft is explained in Designing Data-Intensive Applications too.\n")
	recall := note("0-Inbox/recall.md", "# Recall\n\nRecall drills every evening.\n")
	budget := note("2-Areas/money.md", "# Money\n\nBudget, savings, and the monthly ledger. [[Spaced repetition]]\n")

	var ex struct {
		Notes []struct {
			ID       string   `json:"id"`
			Keywords []string `json:"keywords"`
			Links    int      `json:"links"`
		} `json:"notes"`
		Stats struct {
			Total   int `json:"total"`
			Orphans int `json:"orphans"`
			ByKind  []struct {
				Label string `json:"label"`
				Count int    `json:"count"`
			} `json:"by_kind"`
		} `json:"stats"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/explore", nil), &ex)
	kinds := map[string]int{}
	for _, k := range ex.Stats.ByKind {
		kinds[k.Label] = k.Count
	}
	if ex.Stats.Total != 6 || kinds["permanent"] != 2 || kinds["literature"] != 1 || kinds["note"] != 3 || ex.Stats.Orphans != 4 {
		t.Fatalf("stats = %+v", ex.Stats)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/explore?para=resources", nil), &ex)
	if ex.Stats.Total != 1 || ex.Notes[0].ID != raft {
		t.Fatalf("para filter = %+v", ex.Notes)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/explore?kind=permanent&tag=memory", nil), &ex)
	if ex.Stats.Total != 1 || ex.Notes[0].ID != spaced {
		t.Fatalf("kind and tag filter = %+v", ex.Notes)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/explore?orphans=true", nil), &ex)
	for _, n := range ex.Notes {
		if n.ID == budget || n.ID == spaced { // the wikilink connects them
			t.Errorf("linked note %s listed as an orphan", n.ID)
		}
	}
	alice.mustDo(400, "GET", "/v1/notes/explore?para=nowhere", nil)
	alice.mustDo(400, "GET", "/v1/notes/explore?from=yesterday", nil)
	alice.mustDo(200, "GET", "/v1/notes/explore?q=Kleppmann&from=2020-01-01&to=2099-01-01", nil)

	var g struct {
		Nodes []struct{ ID string } `json:"nodes"`
		Edges []struct {
			From, To, Kind, Why string
		} `json:"edges"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/graph", nil), &g)
	why := func(a, b string) (string, string) {
		for _, e := range g.Edges {
			if (e.From == a && e.To == b) || (e.From == b && e.To == a) {
				return e.Kind, e.Why
			}
		}
		return "", ""
	}
	if k, w := why(budget, spaced); k != "link" || !strings.Contains(w, "Spaced repetition") {
		t.Errorf("wikilink edge = %s %q", k, w)
	}
	if k, w := why(ddia, raft); k != "entity" || !strings.Contains(w, "Designing Data-Intensive Applications") {
		t.Errorf("entity edge = %s %q", k, w)
	}
	if k, w := why(spaced, retrieval); k != "keyword" && k != "similar" || w == "" {
		t.Errorf("keyword edge = %s %q", k, w)
	}
	if k, w := why(recall, budget); k != "" {
		t.Errorf("unrelated notes connected: %s %q", k, w)
	}
	for _, e := range g.Edges {
		if e.Why == "" {
			t.Errorf("edge %s–%s has no reason", e.From, e.To)
		}
	}

	// Bob sees none of it.
	_ = json.Unmarshal(bob.mustDo(200, "GET", "/v1/notes/graph", nil), &g)
	if len(g.Nodes) != 0 {
		t.Fatalf("bob sees %d notes", len(g.Nodes))
	}
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
