package knowledge

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
)

// MaxNotes bounds one exploration; the newest notes come first.
const MaxNotes = 1500

// Filter narrows the notes explored. Empty fields mean "any".
type Filter struct {
	Para, Kind, Tag, Query string
	From, To               *time.Time // created between, inclusive
	Orphans                bool       // only notes with no connection at all
}

// NoteInfo is one note as the explorer shows it.
type NoteInfo struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Path      string    `json:"path"`
	Para      string    `json:"para"`
	Kind      string    `json:"kind"`
	Tags      []string  `json:"tags"`
	Words     int       `json:"words"`
	Links     int       `json:"links"` // connections in and out, any relation
	Keywords  []string  `json:"keywords"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Count is a label and how many notes have it.
type Count struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// Stats summarise the filtered notes for charts.
type Stats struct {
	Total   int     `json:"total"`
	Orphans int     `json:"orphans"`
	Words   int     `json:"words"`
	ByMonth []Count `json:"by_month"` // notes created per month, oldest first
	ByPara  []Count `json:"by_para"`
	ByKind  []Count `json:"by_kind"`
	Tags    []Count `json:"tags"`     // the 40 most used
	Terms   []Count `json:"keywords"` // the 40 keywords most notes share
}

// Exploration is the explorer's answer.
type Exploration struct {
	Notes     []NoteInfo `json:"notes"`
	Stats     Stats      `json:"stats"`
	Truncated bool       `json:"truncated"`
}

type doc struct {
	info     NoteInfo
	content  string
	stems    []string
	vec      Vector
	entities []string
}

func load(ctx context.Context, q db.Querier, userID uuid.UUID, f Filter) ([]*doc, bool, error) {
	rows, err := q.Query(ctx, `SELECT n.id, e.title, n.path, n.para, n.kind, coalesce(n.tags, '{}'), n.content, e.created_at, n.updated_at,
			(SELECT count(*) FROM links l WHERE l.source_id = n.id OR l.target_id = n.id)
		FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND ($2 = '' OR n.para = $2) AND ($3 = '' OR n.kind = $3) AND ($4 = '' OR $4 = ANY(n.tags))
			AND ($5::timestamptz IS NULL OR e.created_at >= $5) AND ($6::timestamptz IS NULL OR e.created_at < $6::timestamptz + interval '1 day')
			AND ($7 = '' OR e.title ILIKE '%' || $7 || '%' OR n.content ILIKE '%' || $7 || '%')
		ORDER BY e.created_at DESC, n.id LIMIT $8`,
		userID, f.Para, f.Kind, f.Tag, f.From, f.To, escapeLike(f.Query), MaxNotes+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var out []*doc
	for rows.Next() {
		d := &doc{}
		if err := rows.Scan(&d.info.ID, &d.info.Title, &d.info.Path, &d.info.Para, &d.info.Kind, &d.info.Tags, &d.content,
			&d.info.CreatedAt, &d.info.UpdatedAt, &d.info.Links); err != nil {
			return nil, false, err
		}
		if f.Orphans && d.info.Links > 0 {
			continue
		}
		out = append(out, d)
	}
	truncated := len(out) > MaxNotes
	if truncated {
		out = out[:MaxNotes]
	}
	return out, truncated, rows.Err()
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// analyse tokenises every note, builds TF-IDF vectors and keywords.
func analyse(docs []*doc) Words {
	words := Words{}
	all := make([][]string, len(docs))
	for i, d := range docs {
		text := Clean(d.content)
		stems, surface := TokensWithWords(d.info.Title + "\n" + text)
		words.Add(stems, surface)
		d.stems = stems
		d.info.Words = len(strings.Fields(text))
		d.entities = Entities(text)
		all[i] = stems
	}
	c := NewCorpus(all)
	for _, d := range docs {
		d.vec = c.TFIDF(d.stems)
		d.info.Keywords = []string{}
		for _, k := range Keywords(d.vec, 8) {
			if c.DF[k] >= 1 {
				d.info.Keywords = append(d.info.Keywords, words.Of(k))
			}
		}
	}
	return words
}

// Explore filters notes and computes statistics for charts.
func Explore(ctx context.Context, q db.Querier, userID uuid.UUID, f Filter) (Exploration, error) {
	docs, truncated, err := load(ctx, q, userID, f)
	if err != nil {
		return Exploration{}, err
	}
	analyse(docs)
	ex := Exploration{Notes: make([]NoteInfo, 0, len(docs)), Truncated: truncated}
	months, paras, kinds, tags, terms := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	for _, d := range docs {
		ex.Notes = append(ex.Notes, d.info)
		ex.Stats.Total++
		ex.Stats.Words += d.info.Words
		if d.info.Links == 0 {
			ex.Stats.Orphans++
		}
		months[d.info.CreatedAt.UTC().Format("2006-01")]++
		paras[d.info.Para]++
		kinds[d.info.Kind]++
		for _, t := range d.info.Tags {
			tags[t]++
		}
		for _, k := range d.info.Keywords {
			terms[k]++
		}
	}
	ex.Stats.ByMonth = sorted(months, func(a, b Count) int { return strings.Compare(a.Label, b.Label) })
	byCount := func(a, b Count) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Label, b.Label)
	}
	ex.Stats.ByPara = sorted(paras, byCount)
	ex.Stats.ByKind = sorted(kinds, byCount)
	ex.Stats.Tags = top(sorted(tags, byCount), 40)
	ex.Stats.Terms = top(sorted(terms, byCount), 40)
	return ex, nil
}

func sorted(m map[string]int, cmp func(a, b Count) int) []Count {
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{k, v})
	}
	slices.SortFunc(out, cmp)
	return out
}

func top(c []Count, n int) []Count {
	if len(c) > n {
		return c[:n]
	}
	return c
}

// Edge kinds, strongest evidence first.
const (
	EdgeLink    = "link"    // a [[wikilink]] or a connection someone made
	EdgeEntity  = "entity"  // both name the same person, work, or thing
	EdgeKeyword = "keyword" // both are about the same distinctive terms
	EdgeSynonym = "synonym" // their keywords are synonyms in the thesaurus
	EdgeSimilar = "similar" // their wording is similar overall (cosine)
)

// Edge connects two notes, with the reason.
type Edge struct {
	From   uuid.UUID `json:"from"`
	To     uuid.UUID `json:"to"`
	Kind   string    `json:"kind"`
	Why    string    `json:"why"`
	Weight float64   `json:"weight"` // 0–1
}

// Node is a note in the graph.
type Node struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Para     string    `json:"para"`
	Kind     string    `json:"kind"`
	Keywords []string  `json:"keywords"`
}

// KnowledgeGraph is notes and why they connect.
type KnowledgeGraph struct {
	Nodes     []Node `json:"nodes"`
	Edges     []Edge `json:"edges"`
	Truncated bool   `json:"truncated"`
}

// Graph limits.
const (
	maxGraphNotes  = 600
	maxEdgesEach   = 6 // NLP edges kept per note, strongest first; links are always kept
	similarAtLeast = 0.22
)

// BuildGraph connects the filtered notes. Wikilinks and connections people
// made are always shown; NLP edges are kept only when no stronger kind
// already links the pair, and only the strongest few per note.
func BuildGraph(ctx context.Context, q db.Querier, userID uuid.UUID, f Filter) (KnowledgeGraph, error) {
	docs, truncated, err := load(ctx, q, userID, f)
	if err != nil {
		return KnowledgeGraph{}, err
	}
	if len(docs) > maxGraphNotes {
		docs, truncated = docs[:maxGraphNotes], true
	}
	words := analyse(docs)
	g := KnowledgeGraph{Nodes: make([]Node, 0, len(docs)), Edges: []Edge{}, Truncated: truncated}
	index := map[uuid.UUID]int{}
	ids := make([]uuid.UUID, len(docs))
	for i, d := range docs {
		index[d.info.ID] = i
		ids[i] = d.info.ID
		g.Nodes = append(g.Nodes, Node{ID: d.info.ID, Title: d.info.Title, Para: d.info.Para, Kind: d.info.Kind, Keywords: d.info.Keywords})
	}

	type pair struct{ a, b int }
	key := func(a, b int) pair {
		if a > b {
			a, b = b, a
		}
		return pair{a, b}
	}
	linked := map[pair]bool{}
	rows, err := q.Query(ctx, `SELECT source_id, target_id, relation FROM links WHERE user_id = $1 AND source_id = ANY($2) AND target_id = ANY($2)`, userID, ids)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var s, t uuid.UUID
		var rel string
		if err := rows.Scan(&s, &t, &rel); err != nil {
			rows.Close()
			return g, err
		}
		p := key(index[s], index[t])
		if linked[p] {
			continue
		}
		linked[p] = true
		why := "Linked with [[" + docs[index[t]].info.Title + "]]"
		if rel != "references" {
			why = "Connected: " + strings.ReplaceAll(rel, "_", " ")
		}
		g.Edges = append(g.Edges, Edge{From: s, To: t, Kind: EdgeLink, Why: why, Weight: 1})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return g, err
	}

	// NLP candidates for every unlinked pair, the strongest reason per pair.
	type cand struct {
		pair
		Edge
	}
	var cands []cand
	entityNotes := map[string][]int{}
	for i, d := range docs {
		for _, e := range d.entities {
			entityNotes[e] = append(entityNotes[e], i)
		}
	}
	best := map[pair]cand{}
	consider := func(a, b int, kind, why string, w float64) {
		p := key(a, b)
		if a == b || linked[p] {
			return
		}
		if cur, ok := best[p]; ok && cur.Weight >= w {
			return
		}
		best[p] = cand{p, Edge{From: docs[p.a].info.ID, To: docs[p.b].info.ID, Kind: kind, Why: why, Weight: w}}
	}
	for e, list := range entityNotes {
		if len(list) < 2 || len(list) > 25 { // a name in everything says nothing
			continue
		}
		for x := 0; x < len(list); x++ {
			for y := x + 1; y < len(list); y++ {
				consider(list[x], list[y], EdgeEntity, "Both mention "+e, 0.9)
			}
		}
	}
	top := make([]map[string]bool, len(docs))
	for i, d := range docs {
		top[i] = map[string]bool{}
		for _, k := range Keywords(d.vec, 8) {
			top[i][k] = true
		}
	}
	for a := range docs {
		for b := a + 1; b < len(docs); b++ {
			var shared []string
			for k := range top[a] {
				if top[b][k] {
					shared = append(shared, words.Of(k))
				}
			}
			slices.Sort(shared)
			if len(shared) >= 2 {
				consider(a, b, EdgeKeyword, "Shared keywords: "+strings.Join(shared, ", "), 0.5+0.1*float64(min(len(shared), 4)))
			}
			if len(shared) == 0 {
				for ka := range top[a] {
					for kb := range top[b] {
						if Synonyms(ka, kb) {
							consider(a, b, EdgeSynonym, fmt.Sprintf("Synonyms: %s ~ %s", words.Of(ka), words.Of(kb)), 0.45)
						}
					}
				}
			}
			if s := Cosine(docs[a].vec, docs[b].vec); s >= similarAtLeast {
				consider(a, b, EdgeSimilar, fmt.Sprintf("Similar wording (cosine %.2f)", s), s)
			}
		}
	}
	for _, c := range best {
		cands = append(cands, c)
	}
	slices.SortFunc(cands, func(x, y cand) int {
		if x.Weight != y.Weight {
			if x.Weight > y.Weight {
				return -1
			}
			return 1
		}
		if x.a != y.a {
			return x.a - y.a
		}
		return x.b - y.b
	})
	degree := make([]int, len(docs))
	for _, c := range cands {
		if degree[c.a] >= maxEdgesEach && degree[c.b] >= maxEdgesEach {
			continue
		}
		degree[c.a]++
		degree[c.b]++
		g.Edges = append(g.Edges, c.Edge)
	}
	return g, nil
}
