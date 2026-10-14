package watch

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/knowledge"
)

// The knowledge graph of the news (EP-74). It is built from the tech watch
// only, so it never mixes with the Zettelkasten graph of the person's own
// notes; where a concept also lives in their literature and permanent
// notes, the node says so (a bridge), with the notes.
//
// Concepts are the key phrases (TextRank) of the period's news, keyed by
// their Snowball stems and merged with their thesaurus synonyms; those in at
// least two items are kept. Two concepts are linked by how much more often
// they appear together than by chance (NPMI). Communities come from
// Louvain, centrality from PageRank. Each concept carries a summary drawn
// from its own news: the sentences closest to what those items share
// (TF-IDF), and a trend: how its count in the last week compares with the
// weeks before. Concepts with no strong link, and news set aside as
// outliers, form the fringe layer, off by default.

// GraphFilter selects the news of a graph.
type GraphFilter struct {
	From, To time.Time
	TopicID  string
	SourceID *uuid.UUID
	MinNPMI  float64 // 0.15 by default
	Fringe   bool    // keep concepts with no strong link
}

// GNode is a concept (or a topic, or a source).
type GNode struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"` // concept, topic, source
	Label     string    `json:"label"`
	Count     int       `json:"count"` // news it appears in
	Sources   int       `json:"sources"`
	Community int       `json:"community"`
	Rank      float64   `json:"rank"`  // PageRank, 0 to 1 relative to the top
	Trend     float64   `json:"trend"` // last week against the weeks before: > 1.5 rising, < -1.5 fading
	Fringe    bool      `json:"fringe"`
	Summary   string    `json:"summary"`
	Items     []GItem   `json:"items"`
	Related   []string  `json:"related"`
	InNotes   []GItem   `json:"in_notes"` // the person's literature and permanent notes that share it
	Topics    []string  `json:"topics"`
	FirstSeen time.Time `json:"first_seen"`
}

// GItem names a news item or a note.
type GItem struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

// GEdge links two concepts.
type GEdge struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Together int     `json:"together"`
	NPMI     float64 `json:"npmi"`
	Reason   string  `json:"reason"`
}

// GCommunity is a group of concepts, named by its most central ones.
type GCommunity struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
	Size  int    `json:"size"`
}

// Graph is the knowledge graph of a period's news.
type Graph struct {
	Nodes       []GNode      `json:"nodes"`
	Edges       []GEdge      `json:"edges"`
	Communities []GCommunity `json:"communities"`
	Stats       struct {
		Items       int     `json:"items"`
		Outliers    int     `json:"outliers"`
		Concepts    int     `json:"concepts"`
		Edges       int     `json:"edges"`
		Communities int     `json:"communities"`
		Modularity  float64 `json:"modularity"`
		Bridges     int     `json:"bridges"`
	} `json:"stats"`
}

// MaxGraphItems bounds the news one graph reads.
const MaxGraphItems = 2000

type gdoc struct {
	id                   uuid.UUID
	title, summary, body string
	lang                 string
	source               string
	topics               []string
	published            time.Time
	outlier              bool
	concepts             []string // concept keys
	terms                []string
	sentences            []gsentence // read once, on first summary
}

type gsentence struct {
	text  string
	stems []string
}

// split returns the item's sentences fit for a summary: 6 to 45 words, no links.
func (d *gdoc) split() []gsentence {
	if d.sentences != nil {
		return d.sentences
	}
	d.sentences = []gsentence{}
	text := d.summary + ". " + d.body
	for _, s := range sentenceSplit.Split(strings.ReplaceAll(text, "\n", " "), -1) {
		s = strings.TrimSpace(strings.Trim(s, ". "))
		words := len(strings.Fields(s))
		if words < 6 || words > 45 || strings.Contains(s, "http") {
			continue
		}
		stems, _ := knowledge.TokensIn(s, d.lang)
		d.sentences = append(d.sentences, gsentence{s, stems})
	}
	return d.sentences
}

func phraseKey(p, lang string) string {
	stems, _ := knowledge.TokensIn(p, lang)
	return strings.Join(stems, " ")
}

// BuildGraph builds the knowledge graph of the news matching f.
func BuildGraph(ctx context.Context, q db.Querier, userID uuid.UUID, f GraphFilter) (Graph, error) {
	g := Graph{Nodes: []GNode{}, Edges: []GEdge{}, Communities: []GCommunity{}}
	if f.MinNPMI == 0 {
		f.MinNPMI = 0.15
	}
	args := []any{userID, f.From, f.To}
	where := "i.user_id = $1 AND i.published_at >= $2 AND i.published_at < $3"
	if f.TopicID != "" {
		args = append(args, f.TopicID)
		where += fmt.Sprintf(" AND $%d::uuid = ANY (i.topic_ids)", len(args))
	}
	if f.SourceID != nil {
		args = append(args, *f.SourceID)
		where += fmt.Sprintf(" AND i.source_id = $%d", len(args))
	}
	rows, err := q.Query(ctx, `SELECT i.id, i.title, i.summary, left(i.content, 20000), i.lang, i.source_title, i.topic_ids::text[], i.published_at,
		i.outlier <> '', i.keywords, i.terms FROM watch_items i WHERE `+where+fmt.Sprintf(` ORDER BY i.published_at DESC LIMIT %d`, MaxGraphItems), args...)
	if err != nil {
		return g, err
	}
	var keywords [][]string
	docs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*gdoc, error) {
		d := &gdoc{}
		var kw []string
		err := r.Scan(&d.id, &d.title, &d.summary, &d.body, &d.lang, &d.source, &d.topics, &d.published, &d.outlier, &kw, &d.terms)
		keywords = append(keywords, kw)
		return d, err
	})
	if err != nil {
		return g, err
	}
	topicNames := map[string]string{}
	if topics, err := ListTopics(ctx, q, userID); err == nil {
		for _, t := range topics {
			topicNames[t.ID] = t.Name
		}
	}
	// Concepts: key phrases by stems; single words merged with their synonyms.
	surface := map[string]map[string]int{}
	alias := map[string]string{} // key → canonical key
	canon := func(k string) string {
		for alias[k] != "" && alias[k] != k {
			k = alias[k]
		}
		return k
	}
	for i, d := range docs {
		if d.outlier {
			g.Stats.Outliers++
			if !f.Fringe {
				continue
			}
		}
		for _, p := range keywords[i] {
			k := phraseKey(p, d.lang)
			if k == "" {
				continue
			}
			if surface[k] == nil {
				surface[k] = map[string]int{}
				if !strings.Contains(k, " ") {
					for _, syn := range knowledge.SynonymsOf(k) {
						if surface[syn] != nil && !strings.Contains(syn, " ") {
							alias[k] = canon(syn)
							break
						}
					}
				}
			}
			surface[k][strings.ToLower(p)]++
		}
	}
	longest := 0
	for k := range surface {
		longest = max(longest, strings.Count(k, " ")+1)
	}
	// A news item carries a concept when its words do, not only when the
	// phrase made its own top key phrases.
	for _, d := range docs {
		if d.outlier && !f.Fringe {
			continue
		}
		stems, _ := knowledge.TokensIn(d.title+". "+d.summary+". "+d.body, d.lang)
		seen := map[string]bool{}
		for i := range stems {
			for n := 1; n <= longest && i+n <= len(stems); n++ {
				k := strings.Join(stems[i:i+n], " ")
				if _, ok := surface[k]; !ok {
					continue
				}
				if c := canon(k); !seen[c] {
					seen[c] = true
					d.concepts = append(d.concepts, c)
				}
			}
		}
		slices.Sort(d.concepts)
	}
	used := docs[:0:0]
	for _, d := range docs {
		if !d.outlier || f.Fringe {
			used = append(used, d)
		}
	}
	g.Stats.Items = len(used)
	df := map[string]int{}
	for _, d := range used {
		for _, c := range d.concepts {
			df[c]++
		}
	}
	var keys []string
	for k, n := range df {
		if n >= 2 {
			keys = append(keys, k)
		}
	}
	slices.SortFunc(keys, func(a, b string) int { return cmp.Or(cmp.Compare(df[b], df[a]), strings.Compare(a, b)) })
	keys = keys[:min(len(keys), 240)]
	keys = slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return subsumed(k, keys, df, used) })
	if len(keys) > 120 {
		keys = keys[:120]
	}
	index := map[string]int{}
	for i, k := range keys {
		index[k] = i
	}
	n := len(keys)
	// Co-occurrence and NPMI.
	together := map[[2]int]int{}
	for _, d := range used {
		var in []int
		for _, c := range d.concepts {
			if i, ok := index[c]; ok {
				in = append(in, i)
			}
		}
		slices.Sort(in)
		for a := range in {
			for b := a + 1; b < len(in); b++ {
				together[[2]int{in[a], in[b]}]++
			}
		}
	}
	adj := make([]map[int]float64, n)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	N := float64(max(1, len(used)))
	for pair, both := range together {
		if both < 2 {
			continue
		}
		pa, pb, pab := float64(df[keys[pair[0]]])/N, float64(df[keys[pair[1]]])/N, float64(both)/N
		npmi := 1.0
		if pab < 1 {
			npmi = math.Log(pab/(pa*pb)) / -math.Log(pab)
		}
		if npmi < f.MinNPMI {
			continue
		}
		adj[pair[0]][pair[1]], adj[pair[1]][pair[0]] = npmi*float64(both), npmi*float64(both)
		g.Edges = append(g.Edges, GEdge{From: "c:" + keys[pair[0]], To: "c:" + keys[pair[1]], Together: both, NPMI: round2(npmi),
			Reason: fmt.Sprintf("Together in %d news (NPMI %.2f)", both, npmi)})
	}
	slices.SortFunc(g.Edges, func(a, b GEdge) int {
		return cmp.Or(cmp.Compare(b.NPMI*float64(b.Together), a.NPMI*float64(a.Together)), strings.Compare(a.From+a.To, b.From+b.To))
	})
	if len(g.Edges) > 400 {
		g.Edges = g.Edges[:400]
	}
	var comm []int
	var ranks []float64
	if n > 0 {
		comm, g.Stats.Modularity = knowledge.Louvain(adj)
		ranks = knowledge.PageRank(adj)
	}
	top := 0.0
	for _, r := range ranks {
		top = math.Max(top, r)
	}
	// Renumber communities by size, biggest first; isolated concepts get 0.
	sizes := map[int]int{}
	for i, c := range comm {
		if len(adj[i]) > 0 {
			sizes[c]++
		}
	}
	order := make([]int, 0, len(sizes))
	for c := range sizes {
		order = append(order, c)
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Or(cmp.Compare(sizes[b], sizes[a]), cmp.Compare(a, b)) })
	renum := map[int]int{}
	for i, c := range order {
		renum[c] = i + 1
	}
	// Per concept: its news, sources, topics, trend, summary.
	notes, err := noteConcepts(ctx, q, userID)
	if err != nil {
		return g, err
	}
	weekAgo := f.To.Add(-7 * 24 * time.Hour)
	weeks := math.Max(1, f.To.Sub(f.From).Hours()/24/7-1)
	// A trend needs weeks to compare with: with no news older than a week,
	// nothing is rising, it is all new.
	history := slices.ContainsFunc(used, func(d *gdoc) bool { return !d.published.After(weekAgo) })
	corpus := knowledge.NewCorpus(termsOf(used))
	for i, k := range keys {
		node := GNode{ID: "c:" + k, Kind: "concept", Label: best(surface, k, alias), Count: df[k], Items: []GItem{}, Related: []string{}, InNotes: []GItem{}, Topics: []string{}}
		if len(adj[i]) > 0 {
			node.Community = renum[comm[i]]
		} else {
			node.Fringe = true
		}
		if top > 0 {
			node.Rank = round2(ranks[i] / top)
		}
		srcs, tps := map[string]bool{}, map[string]bool{}
		var mine []*gdoc
		recent, before := 0, 0
		for _, d := range used {
			if !slices.Contains(d.concepts, k) {
				continue
			}
			mine = append(mine, d)
			srcs[d.source] = true
			for _, t := range d.topics {
				if name, ok := topicNames[t]; ok {
					tps[name] = true
				}
			}
			if d.published.After(weekAgo) {
				recent++
			} else {
				before++
			}
			if node.FirstSeen.IsZero() || d.published.Before(node.FirstSeen) {
				node.FirstSeen = d.published
			}
		}
		node.Sources = len(srcs)
		for t := range tps {
			node.Topics = append(node.Topics, t)
		}
		slices.Sort(node.Topics)
		if history {
			expected := float64(before) / weeks
			node.Trend = round2((float64(recent) - expected) / math.Sqrt(expected+1))
		}
		for _, d := range mine[:min(3, len(mine))] {
			node.Items = append(node.Items, GItem{d.id, d.title})
		}
		node.Summary = summarise(k, mine[:min(20, len(mine))], corpus) // the newest news suffice
		for _, nb := range neighbours(adj[i], 4) {
			node.Related = append(node.Related, best(surface, keys[nb], alias))
		}
		if ns, ok := notes[k]; ok {
			node.InNotes = ns[:min(3, len(ns))]
			g.Stats.Bridges++
		}
		if node.Fringe && !f.Fringe {
			continue
		}
		g.Nodes = append(g.Nodes, node)
	}
	for _, c := range order {
		members := []GNode{}
		for _, nd := range g.Nodes {
			if nd.Community == renum[c] {
				members = append(members, nd)
			}
		}
		slices.SortFunc(members, func(a, b GNode) int { return cmp.Compare(b.Rank, a.Rank) })
		var names []string
		for _, m := range members[:min(3, len(members))] {
			names = append(names, m.Label)
		}
		g.Communities = append(g.Communities, GCommunity{ID: renum[c], Label: strings.Join(names, " · "), Size: sizes[c]})
	}
	g.Stats.Concepts, g.Stats.Edges, g.Stats.Communities = len(g.Nodes), len(g.Edges), len(g.Communities)
	g.Stats.Modularity = round2(g.Stats.Modularity)
	return g, nil
}

// subsumed tells whether a concept appears only inside a longer one in the
// same news ("kernel" when every item says "linux kernel").
func subsumed(k string, keys []string, df map[string]int, docs []*gdoc) bool {
	for _, long := range keys {
		if long == k || df[long] != df[k] || !strings.Contains(" "+long+" ", " "+k+" ") {
			continue
		}
		same := true
		for _, d := range docs {
			if slices.Contains(d.concepts, k) != slices.Contains(d.concepts, long) {
				same = false
				break
			}
		}
		if same {
			return true
		}
	}
	return false
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

func termsOf(docs []*gdoc) [][]string {
	out := make([][]string, len(docs))
	for i, d := range docs {
		out[i] = d.terms
	}
	return out
}

// best is the most frequent way a concept was written.
func best(surface map[string]map[string]int, k string, alias map[string]string) string {
	counts := map[string]int{}
	for key, forms := range surface {
		c := key
		for alias[c] != "" && alias[c] != c {
			c = alias[c]
		}
		if c != k {
			continue
		}
		for s, n := range forms {
			counts[s] += n
		}
	}
	label, top := k, 0
	for s, n := range counts {
		if n > top || n == top && s < label {
			label, top = s, n
		}
	}
	return label
}

func neighbours(row map[int]float64, n int) []int {
	out := make([]int, 0, len(row))
	for j := range row {
		out = append(out, j)
	}
	slices.SortFunc(out, func(a, b int) int { return cmp.Or(cmp.Compare(row[b], row[a]), cmp.Compare(a, b)) })
	return out[:min(n, len(out))]
}

var sentenceSplit = regexp.MustCompile(`(?:[.!?])\s+`)

// summarise is an extractive summary of a concept: of the sentences in its
// news that mention it, the two closest to what those news share (the
// centroid of their TF-IDF vectors), not repeating each other.
func summarise(key string, docs []*gdoc, corpus knowledge.Corpus) string {
	if len(docs) == 0 {
		return ""
	}
	centroid := knowledge.Vector{}
	for _, d := range docs {
		for t, w := range corpus.TFIDF(d.terms) {
			centroid[t] += w / float64(len(docs))
		}
	}
	centroid = normalise(centroid)
	want := strings.Fields(key)
	type cand struct {
		text  string
		score float64
		stems []string
	}
	var cands []cand
	for _, d := range docs {
		for _, sn := range d.split() {
			hit := 0
			for _, w := range want {
				if slices.Contains(sn.stems, w) {
					hit++
				}
			}
			if hit < len(want) {
				continue
			}
			cands = append(cands, cand{sn.text, knowledge.Cosine(corpus.TFIDF(sn.stems), centroid), sn.stems})
		}
	}
	slices.SortStableFunc(cands, func(a, b cand) int { return cmp.Compare(b.score, a.score) })
	var picked []cand
	for _, c := range cands {
		if len(picked) == 2 {
			break
		}
		dup := false
		for _, p := range picked {
			dup = dup || Jaccard(p.stems, c.stems) > 0.5
		}
		if !dup {
			picked = append(picked, c)
		}
	}
	if len(picked) == 0 {
		return firstSentences(docs[0].summary, 240)
	}
	var parts []string
	for _, p := range picked {
		parts = append(parts, p.text+".")
	}
	return cut(strings.Join(parts, " "), 400)
}

func normalise(v knowledge.Vector) knowledge.Vector {
	var norm float64
	for _, w := range v {
		norm += w * w
	}
	norm = math.Sqrt(norm)
	out := knowledge.Vector{}
	for t, w := range v {
		if norm > 0 {
			out[t] = w / norm
		}
	}
	return out
}

// noteConcepts maps the key phrases of the person's literature and permanent
// notes (by stems) to those notes, to find the concepts the news share with
// their Zettelkasten.
func noteConcepts(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string][]GItem, error) {
	rows, err := q.Query(ctx, `SELECT n.id, e.title, left(n.content, 20000) FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND n.kind IN ('literature', 'permanent') ORDER BY e.updated_at DESC LIMIT 500`, userID)
	if err != nil {
		return nil, err
	}
	type note struct {
		id             uuid.UUID
		title, content string
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (note, error) {
		var n note
		err := r.Scan(&n.id, &n.title, &n.content)
		return n, err
	})
	if err != nil {
		return nil, err
	}
	out := map[string][]GItem{}
	for _, n := range list {
		text := knowledge.Clean(n.content)
		lang := knowledge.Language(text)
		for _, p := range knowledge.KeyPhrases(text, 12) {
			if k := phraseKey(p.Text, lang); k != "" && !slices.ContainsFunc(out[k], func(g GItem) bool { return g.ID == n.id }) {
				out[k] = append(out[k], GItem{n.id, n.title})
			}
		}
	}
	return out, nil
}
