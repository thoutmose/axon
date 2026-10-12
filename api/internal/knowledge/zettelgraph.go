package knowledge

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
)

// The Zettelkasten knowledge graph (EP-47): literature and permanent notes
// connected by combined NLP evidence, with communities, labels, importance,
// and bridges, all computed locally.
//
//   - Edge score: 0.45 × TF-IDF cosine + 0.30 × IDF-weighted cosine of the
//     notes' key phrases over the concept vocabulary (phrases in two notes or more) + 0.15 × WordNet synonyms between the
//     notes' keywords + 0.10 × shared named entities; a wikilink or
//     connection the person made adds 0.25 and is always kept.
//   - Sparsification: each note keeps its strongest neighbours (k-nearest
//     neighbour graph), so hubs do not drown the structure.
//   - Communities: Louvain modularity optimisation, multi-level, run over
//     several fixed node orders, keeping the partition with the best modularity.
//   - Labels: class-based TF-IDF (c-TF-IDF): words frequent in a community
//     and rare in the others.
//   - Importance: weighted PageRank. Bridges: the participation coefficient,
//     notes whose links spread across communities.

// ZNode is a note in the graph.
type ZNode struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"`
	Community int       `json:"community"`
	PageRank  float64   `json:"pagerank"` // 0 to 1, relative to the most central note
	Degree    int       `json:"degree"`
	Bridge    bool      `json:"bridge"`
	// Participation is 0 when every link stays in the note's community and
	// approaches 1 when links spread evenly across communities.
	Participation float64  `json:"participation"`
	Phrases       []string `json:"phrases"`
}

// ZEdge links two notes with the evidence behind it.
type ZEdge struct {
	From    uuid.UUID `json:"from"`
	To      uuid.UUID `json:"to"`
	Score   float64   `json:"score"` // 0 to about 1.5
	Kinds   []string  `json:"kinds"` // link, phrases, synonyms, entities, wording
	Reasons []string  `json:"reasons"`
}

// ZCommunity is a group of notes found by Louvain.
type ZCommunity struct {
	ID    int         `json:"id"`
	Label string      `json:"label"`
	Terms []string    `json:"terms"`
	Size  int         `json:"size"`
	Core  []NoteBrief `json:"core"` // its most central notes
}

// ZStats summarise the graph.
type ZStats struct {
	Notes       int     `json:"notes"`
	Edges       int     `json:"edges"`
	Communities int     `json:"communities"`
	Modularity  float64 `json:"modularity"` // above about 0.3 means clear communities
	Density     float64 `json:"density"`
	Isolated    int     `json:"isolated"`
}

// ZettelGraph is the whole graph.
type ZettelGraph struct {
	Nodes       []ZNode      `json:"nodes"`
	Edges       []ZEdge      `json:"edges"`
	Communities []ZCommunity `json:"communities"`
	Stats       ZStats       `json:"stats"`
	Truncated   bool         `json:"truncated"`
}

const (
	zMaxNotes   = 600
	zNeighbours = 6
	zMinScore   = 0.08
)

// BuildZettelGraph builds the graph of the person's literature and permanent
// notes (or one kind). minScore raises the threshold for NLP edges.
func BuildZettelGraph(ctx context.Context, q db.Querier, userID uuid.UUID, kinds []string, minScore float64) (ZettelGraph, error) {
	docs, words, truncated, err := analyseKinds(ctx, q, userID, kinds)
	if err != nil {
		return ZettelGraph{}, err
	}
	if len(docs) > zMaxNotes {
		docs, truncated = docs[:zMaxNotes], true
	}
	g := ZettelGraph{Nodes: []ZNode{}, Edges: []ZEdge{}, Communities: []ZCommunity{}, Truncated: truncated}
	n := len(docs)
	if n == 0 {
		return g, nil
	}
	minScore = max(minScore, zMinScore)
	index := map[uuid.UUID]int{}
	ids := make([]uuid.UUID, n)
	for i, d := range docs {
		index[d.info.ID] = i
		ids[i] = d.info.ID
	}

	// Explicit links.
	links := map[zpair]string{}
	rows, err := q.Query(ctx, `SELECT source_id, target_id FROM links WHERE user_id = $1 AND source_id = ANY($2) AND target_id = ANY($2)`, userID, ids)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var s, t uuid.UUID
		if err := rows.Scan(&s, &t); err != nil {
			rows.Close()
			return g, err
		}
		if index[s] != index[t] {
			links[zkey(index[s], index[t])] = "Linked with [[" + docs[index[t]].info.Title + "]]"
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return g, err
	}

	adj, edges := zettelEdges(docs, links, minScore)
	g.Edges = edges

	community, q2 := louvain(adj)
	rank := pageRank(adj)
	maxRank := 0.0
	for _, r := range rank {
		maxRank = math.Max(maxRank, r)
	}

	// Renumber communities by size, largest first.
	sizes := map[int]int{}
	for _, c := range community {
		sizes[c]++
	}
	order := make([]int, 0, len(sizes))
	for c := range sizes {
		order = append(order, c)
	}
	slices.SortFunc(order, func(x, y int) int {
		if s := cmp.Compare(sizes[y], sizes[x]); s != 0 {
			return s
		}
		return cmp.Compare(x, y)
	})
	renum := map[int]int{}
	for i, c := range order {
		renum[c] = i + 1
	}

	isolated := 0
	for i, d := range docs {
		deg := len(adj[i])
		if deg == 0 {
			isolated++
		}
		// Participation coefficient over neighbours' communities.
		part := 0.0
		if deg > 0 {
			k, per := 0.0, map[int]float64{}
			for j, w := range adj[i] {
				per[community[j]] += w
				k += w
			}
			sum := 0.0
			for _, w := range per {
				sum += (w / k) * (w / k)
			}
			part = math.Round((1-sum)*100) / 100
		}
		var phrases []string
		for _, p := range d.phrases[:min(3, len(d.phrases))] {
			phrases = append(phrases, p.Text)
		}
		if phrases == nil {
			phrases = []string{}
		}
		pr := 0.0
		if maxRank > 0 {
			pr = math.Round(rank[i]/maxRank*100) / 100
		}
		g.Nodes = append(g.Nodes, ZNode{ID: d.info.ID, Title: d.info.Title, Kind: d.info.Kind, Community: renum[community[i]],
			PageRank: pr, Degree: deg, Participation: part, Bridge: part >= 0.5 && deg >= 2, Phrases: phrases})
	}

	// c-TF-IDF labels for communities with at least two notes.
	g.Communities = labelCommunities(docs, words, community, renum, g.Nodes)
	nEdges := len(g.Edges)
	density := 0.0
	if n > 1 {
		density = math.Round(float64(2*nEdges)/float64(n*(n-1))*1000) / 1000
	}
	g.Stats = ZStats{Notes: n, Edges: nEdges, Communities: len(g.Communities), Modularity: math.Round(q2*100) / 100, Density: density, Isolated: isolated}
	return g, nil
}

// labelCommunities names each community by class-based TF-IDF: a term's
// frequency in the community, times log(1 + average words per community /
// the term's frequency across all communities).
func labelCommunities(docs []*analysed, words Words, community []int, renum map[int]int, nodes []ZNode) []ZCommunity {
	tf := map[int]map[string]float64{}
	count := map[int]float64{}
	total := map[string]float64{}
	for i, d := range docs {
		c := community[i]
		if tf[c] == nil {
			tf[c] = map[string]float64{}
		}
		for _, s := range d.stems {
			tf[c][s]++
			count[c]++
			total[s]++
		}
	}
	avg := 0.0
	for _, w := range count {
		avg += w
	}
	avg /= math.Max(1, float64(len(count)))
	out := []ZCommunity{}
	for c, terms := range tf {
		var members []int
		for i := range docs {
			if community[i] == c {
				members = append(members, i)
			}
		}
		if len(members) < 2 {
			continue
		}
		type tw struct {
			t string
			w float64
		}
		var ws []tw
		for t, f := range terms {
			ws = append(ws, tw{t, f / count[c] * math.Log(1+avg/total[t])})
		}
		slices.SortFunc(ws, func(x, y tw) int {
			if d := cmp.Compare(y.w, x.w); d != 0 {
				return d
			}
			return cmp.Compare(x.t, y.t)
		})
		var top []string
		for _, w := range ws[:min(5, len(ws))] {
			top = append(top, words.Of(w.t))
		}
		slices.SortFunc(members, func(x, y int) int { return cmp.Compare(nodes[y].PageRank, nodes[x].PageRank) })
		var core []NoteBrief
		for _, m := range members[:min(5, len(members))] {
			core = append(core, brief(docs[m]))
		}
		out = append(out, ZCommunity{ID: renum[c], Label: strings.Join(top[:min(3, len(top))], " · "), Terms: top, Size: len(members), Core: core})
	}
	slices.SortFunc(out, func(x, y ZCommunity) int { return cmp.Compare(x.ID, y.ID) })
	return out
}

// louvain finds communities by greedily moving nodes to the neighbouring
// community with the best modularity gain, then merging communities into
// single nodes and repeating. Greedy moves depend on the order nodes are
// visited, so it tries several fixed orders (natural, reverse, by weighted
// degree, and seeded shuffles) and keeps the best modularity: results are
// stable from run to run.
func louvain(adj []map[int]float64) ([]int, float64) {
	n := len(adj)
	var orders [][]int
	natural := make([]int, n)
	for i := range natural {
		natural[i] = i
	}
	orders = append(orders, natural)
	reverse := slices.Clone(natural)
	slices.Reverse(reverse)
	orders = append(orders, reverse)
	byDegree := slices.Clone(natural)
	strength := func(i int) float64 {
		s := 0.0
		for _, w := range adj[i] {
			s += w
		}
		return s
	}
	slices.SortStableFunc(byDegree, func(a, b int) int { return cmp.Compare(strength(b), strength(a)) })
	orders = append(orders, byDegree)
	seed := uint64(2463534242)
	for range 5 {
		o := slices.Clone(natural)
		for i := n - 1; i > 0; i-- { // xorshift Fisher–Yates, deterministic
			seed ^= seed << 13
			seed ^= seed >> 7
			seed ^= seed << 17
			j := int(seed % uint64(i+1)) //nolint:gosec // the modulus is below i+1, a small int
			o[i], o[j] = o[j], o[i]
		}
		orders = append(orders, o)
	}
	var best []int
	bestQ := math.Inf(-1)
	for _, o := range orders {
		comm := louvainFrom(adj, o)
		if q := modularity(adj, comm); q > bestQ+1e-12 {
			best, bestQ = comm, q
		}
	}
	return best, bestQ
}

// louvainFrom runs the multi-level Louvain method visiting nodes in order
// on the first level.
func louvainFrom(adj []map[int]float64, order []int) []int {
	member := make([]int, len(adj))
	for i := range member {
		member[i] = i
	}
	cur := adj
	for level := range 10 {
		o := order
		if level > 0 {
			o = make([]int, len(cur))
			for i := range o {
				o[i] = i
			}
		}
		comm, moved := louvainLevel(cur, o)
		if !moved {
			break
		}
		ids := map[int]int{}
		for _, i := range o {
			if _, ok := ids[comm[i]]; !ok {
				ids[comm[i]] = len(ids)
			}
		}
		for i := range member {
			member[i] = ids[comm[member[i]]]
		}
		next := make([]map[int]float64, len(ids))
		for i := range next {
			next[i] = map[int]float64{}
		}
		for i, nb := range cur {
			for j, w := range nb {
				next[ids[comm[i]]][ids[comm[j]]] += w
			}
		}
		cur = next
	}
	return member
}

// louvainLevel runs the local moving phase on one level, visiting nodes in
// order. Self-loops (from aggregation) count inside the node.
func louvainLevel(adj []map[int]float64, order []int) ([]int, bool) {
	n := len(adj)
	comm := make([]int, n)
	k := make([]float64, n)
	m2 := 0.0 // twice the total edge weight
	for i, nb := range adj {
		comm[i] = i
		for _, w := range nb {
			k[i] += w
		}
		m2 += k[i]
	}
	if m2 == 0 {
		return comm, false
	}
	tot := slices.Clone(k)
	movedAny := false
	for range 50 {
		moved := false
		for _, i := range order {
			old := comm[i]
			tot[old] -= k[i]
			toComm := map[int]float64{}
			for j, w := range adj[i] {
				if j != i {
					toComm[comm[j]] += w
				}
			}
			best, bestGain := old, toComm[old]-tot[old]*k[i]/m2
			cands := make([]int, 0, len(toComm))
			for c := range toComm {
				cands = append(cands, c)
			}
			slices.Sort(cands)
			for _, c := range cands {
				if gain := toComm[c] - tot[c]*k[i]/m2; gain > bestGain+1e-12 {
					best, bestGain = c, gain
				}
			}
			comm[i] = best
			tot[best] += k[i]
			if best != old {
				moved, movedAny = true, true
			}
		}
		if !moved {
			break
		}
	}
	return comm, movedAny
}

// modularity is Newman's Q of a partition of a weighted graph.
func modularity(adj []map[int]float64, comm []int) float64 {
	m2, in := 0.0, map[int]float64{}
	tot := map[int]float64{}
	for i, nb := range adj {
		for j, w := range nb {
			m2 += w
			tot[comm[i]] += w
			if comm[i] == comm[j] {
				in[comm[i]] += w
			}
		}
	}
	if m2 == 0 {
		return 0
	}
	q := 0.0
	for c, t := range tot {
		q += in[c]/m2 - (t/m2)*(t/m2)
	}
	return q
}

// pageRank is weighted PageRank with damping 0.85; isolated notes keep the
// teleport share.
func pageRank(adj []map[int]float64) []float64 {
	n := len(adj)
	rank := make([]float64, n)
	out := make([]float64, n)
	for i := range n {
		rank[i] = 1 / float64(n)
		for _, w := range adj[i] {
			out[i] += w
		}
	}
	for range 50 {
		next := make([]float64, n)
		dangling := 0.0
		for i := range n {
			if out[i] == 0 {
				dangling += rank[i]
			}
		}
		for i := range n {
			next[i] = (1-0.85)/float64(n) + 0.85*dangling/float64(n)
			for j, w := range adj[i] {
				next[i] += 0.85 * rank[j] * w / out[j]
			}
		}
		rank = next
	}
	return rank
}

type zpair struct{ a, b int }

func zkey(a, b int) zpair {
	if a > b {
		a, b = b, a
	}
	return zpair{a, b}
}

// zettelEdges scores every pair of notes, keeps those above minScore (and
// every explicit link), and sparsifies to each note's strongest neighbours.
func zettelEdges(docs []*analysed, links map[zpair]string, minScore float64) ([]map[int]float64, []ZEdge) {
	n := len(docs)
	edges := []ZEdge{}
	// Inverse document frequency of key phrases.
	df := map[string]int{}
	for _, d := range docs {
		for _, p := range d.phrases {
			df[p.key]++
		}
	}
	idf := func(k string) float64 { return math.Log(float64(n+1) / float64(df[k])) }
	phraseNorm := make([]float64, n)
	keywords := make([][]string, n)
	for i, d := range docs {
		// The cosine runs over the concept vocabulary only (phrases in at least
		// two notes): a phrase no other note has can never match, so it must
		// not dilute the score.
		for _, p := range d.phrases {
			if df[p.key] >= 2 {
				phraseNorm[i] += idf(p.key) * idf(p.key)
			}
		}
		phraseNorm[i] = math.Sqrt(phraseNorm[i])
		keywords[i] = Keywords(d.vec, 10)
	}

	// Score every pair.
	type cand struct {
		a, b    int
		score   float64
		kinds   []string
		reasons []string
	}
	var cands []cand
	for a := range n {
		for b := a + 1; b < n; b++ {
			c := cand{a: a, b: b}
			if why, ok := links[zkey(a, b)]; ok {
				c.score += 0.25
				c.kinds, c.reasons = append(c.kinds, "link"), append(c.reasons, why)
			}
			// Key phrases, weighted by how rare they are.
			var shared []string
			dot := 0.0
			for _, p := range docs[a].phrases {
				for _, r := range docs[b].phrases {
					if p.key == r.key {
						dot += idf(p.key) * idf(p.key)
						shared = append(shared, "“"+p.Text+"”")
					}
				}
			}
			if dot > 0 && phraseNorm[a] > 0 && phraseNorm[b] > 0 {
				s := dot / (phraseNorm[a] * phraseNorm[b])
				c.score += 0.30 * s
				c.kinds = append(c.kinds, "phrases")
				c.reasons = append(c.reasons, "Shares "+strings.Join(shared[:min(3, len(shared))], ", "))
			}
			// WordNet synonyms between keywords.
			var syn []string
			for _, x := range keywords[a] {
				for _, y := range keywords[b] {
					if Synonyms(x, y) {
						syn = append(syn, readable(x)+" ≈ "+readable(y))
					}
				}
			}
			if len(syn) > 0 {
				c.score += 0.15 * math.Min(1, float64(len(syn))/3)
				c.kinds = append(c.kinds, "synonyms")
				c.reasons = append(c.reasons, "Synonyms: "+strings.Join(syn[:min(3, len(syn))], ", "))
			}
			// Named entities.
			var ents []string
			for _, x := range docs[a].entities {
				if slices.Contains(docs[b].entities, x) {
					ents = append(ents, x)
				}
			}
			if len(ents) > 0 {
				c.score += 0.10 * math.Min(1, float64(len(ents))/2)
				c.kinds = append(c.kinds, "entities")
				c.reasons = append(c.reasons, "Both mention "+strings.Join(ents[:min(3, len(ents))], ", "))
			}
			if cos := Cosine(docs[a].vec, docs[b].vec); cos >= 0.05 {
				c.score += 0.45 * cos
				c.kinds = append(c.kinds, "wording")
				c.reasons = append(c.reasons, fmt.Sprintf("Similar wording (%.0f %%)", cos*100))
			}
			if c.score >= minScore || slices.Contains(c.kinds, "link") {
				cands = append(cands, c)
			}
		}
	}
	// k-nearest-neighbour sparsification: an NLP edge survives when it is
	// among the strongest few of either end; links always survive.
	slices.SortFunc(cands, func(x, y cand) int { return cmp.Compare(y.score, x.score) })
	kept := make([]int, n)
	adj := make([]map[int]float64, n)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	for _, c := range cands {
		link := slices.Contains(c.kinds, "link")
		if !link && kept[c.a] >= zNeighbours && kept[c.b] >= zNeighbours {
			continue
		}
		kept[c.a]++
		kept[c.b]++
		adj[c.a][c.b], adj[c.b][c.a] = c.score, c.score
		edges = append(edges, ZEdge{From: docs[c.a].info.ID, To: docs[c.b].info.ID, Score: math.Round(c.score*100) / 100, Kinds: c.kinds, Reasons: c.reasons})
	}

	return adj, edges
}
