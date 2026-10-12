package knowledge

import (
	"testing"

	"github.com/google/uuid"
)

// Two triangles joined by one weak edge are two communities.
func TestLouvainFindsTwoCommunities(t *testing.T) {
	adj := make([]map[int]float64, 6)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	link := func(a, b int, w float64) { adj[a][b], adj[b][a] = w, w }
	link(0, 1, 1)
	link(1, 2, 1)
	link(0, 2, 1)
	link(3, 4, 1)
	link(4, 5, 1)
	link(3, 5, 1)
	link(2, 3, 0.1)
	comm, q := louvain(adj)
	if comm[0] != comm[1] || comm[1] != comm[2] || comm[3] != comm[4] || comm[4] != comm[5] || comm[0] == comm[3] {
		t.Fatalf("communities = %v", comm)
	}
	if q < 0.4 {
		t.Fatalf("modularity = %v", q)
	}
	rank := pageRank(adj)
	if rank[2] <= rank[0] { // node 2 also links across
		t.Fatalf("pagerank = %v", rank)
	}
	sum := 0.0
	for _, r := range rank {
		sum += r
	}
	if sum < 0.99 || sum > 1.01 {
		t.Fatalf("pagerank sums to %v", sum)
	}
}

func TestZettelGraphStructure(t *testing.T) {
	raw := []*doc{
		note("Spaced repetition", "permanent", "Spaced repetition and retrieval practice fight the forgetting curve. Memory needs reviews."),
		note("Retrieval practice", "permanent", "Retrieval practice with spaced repetition flattens the forgetting curve and improves recall."),
		note("Make It Stick", "literature", "Henry Roediger shows retrieval practice and spaced repetition beat rereading the forgetting curve."),
		note("Kubernetes pods", "permanent", "Kubernetes schedules pods on nodes. The control plane watches pods and nodes."),
		note("Kubernetes in Action", "literature", "A Kubernetes service routes traffic to pods on nodes behind the control plane."),
		note("Kubernetes operators", "permanent", "Operators extend the Kubernetes control plane to manage pods on nodes."),
	}
	analyse(raw)
	docs := make([]*analysed, len(raw))
	for i, d := range raw {
		text := Clean(d.content)
		docs[i] = &analysed{doc: d, phrases: KeyPhrases(d.info.Title+".\n"+text, phrasesPerNote), read: Read(text)}
	}
	_ = uuid.Nil
	adj, edges := zettelEdges(docs, nil, zMinScore)
	if len(edges) == 0 {
		t.Fatal("no edges")
	}
	for _, e := range edges {
		if len(e.Reasons) == 0 || e.Score <= 0 {
			t.Fatalf("edge without evidence: %+v", e)
		}
	}
	comm, _ := louvain(adj)
	if comm[0] != comm[1] || comm[0] != comm[2] || comm[3] != comm[4] || comm[0] == comm[3] {
		t.Fatalf("communities = %v", comm)
	}
}
