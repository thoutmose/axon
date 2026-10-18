package watch

import (
	"cmp"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/knowledge"
)

// The knowledge layer of the news graph (owner request): a knowledge graph
// in the proper sense, not only concepts seen together. Entities come from
// an ontology of the tech world (organizations, technologies, languages,
// databases, cloud services, standards, and concepts, each an instance of a
// class, with who makes it and what it was forked from) and from the CVEs
// the news name; aliases resolve to one entity. Facts are subject–predicate–
// object triples: the ontology's own (is a, developed by, fork of) and those
// read from the news, sentence by sentence, between two entities next to
// each other (acquires, releases, integrates with, uses, alternative to,
// migrates to, partners with, invests in, deprecates, affects, part of).
// Every fact read from the news keeps its evidence: the news and the
// sentence. Two entities in the same sentences with no stated relation are
// mentioned together.

//go:embed ontology.json
var ontologyJSON []byte

// OEntity is an entity of the ontology.
type OEntity struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Type    string   `json:"type"`
	Class   string   `json:"class"`
	By      string   `json:"by"`
	ForkOf  string   `json:"fork_of"`
	Aliases []string `json:"aliases"`
	Case    []string `json:"case"` // aliases matched with their case only ("Go", "Spark")
}

type ontology struct {
	byID    map[string]*OEntity
	foldKey map[string]string // an alias's words, lower-cased → entity
	casKey  map[string]string // an alias's words as written → entity
	longest int               // words in the longest alias
}

var onto = loadOntology()

var cveRE = regexp.MustCompile(`\bCVE-\d{4}-\d{4,7}\b`)

func loadOntology() *ontology {
	var raw struct {
		Entities []OEntity `json:"entities"`
	}
	if err := json.Unmarshal(ontologyJSON, &raw); err != nil {
		panic("watch: ontology: " + err.Error())
	}
	o := &ontology{byID: map[string]*OEntity{}, foldKey: map[string]string{}, casKey: map[string]string{}}
	key := func(a string) string {
		var ws []string
		for _, t := range nameWords(a) {
			ws = append(ws, a[t[0]:t[1]])
		}
		o.longest = max(o.longest, len(ws))
		return strings.Join(ws, " ")
	}
	for i := range raw.Entities {
		en := &raw.Entities[i]
		o.byID[en.ID] = en
		for _, a := range append([]string{en.Label}, en.Aliases...) {
			if !slices.Contains(en.Case, a) {
				o.foldKey[strings.ToLower(key(a))] = en.ID
			}
		}
		for _, a := range en.Case {
			o.casKey[key(a)] = en.ID
		}
	}
	return o
}

// nameWords splits text into words, where a word keeps the dots inside it
// ("Node.js") and a trailing + or # ("C++", "C#"); slashes and hyphens part
// words, so "Kafka/Flink" and "Kafka-based" name Kafka, and the aliases
// split the same way ("Pub/Sub" is "Pub Sub").
func nameWords(s string) [][2]int {
	var out [][2]int
	start := -1
	inWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
	for i, r := range s {
		switch {
		case inWord(r):
			if start < 0 {
				start = i
			}
		case start >= 0 && r == '.' && i+1 < len(s) && inWord(rune(s[i+1])) && inWord(rune(s[i-1])):
			// a dot between letters stays in the word
		case start >= 0 && (r == '+' || r == '#'):
			// "C++", "C#": kept, and the word ends after them
		default:
			if start >= 0 {
				out = append(out, [2]int{start, i})
				start = -1
			}
		}
	}
	if start >= 0 {
		out = append(out, [2]int{start, len(s)})
	}
	return out
}

// mention is an entity named in a sentence, where.
type mention struct {
	id         string
	start, end int
}

// mentions finds the entities a sentence names, in order: at each word,
// the longest run of words that is an alias (as written for the aliases
// that need their case, else in any case).
func (o *ontology) mentions(s string) []mention {
	var out []mention
	ws := nameWords(s)
	for i := 0; i < len(ws); {
		found := 0
		for n := min(o.longest, len(ws)-i); n > 0 && found == 0; n-- {
			parts := make([]string, n)
			for k := range n {
				parts[k] = s[ws[i+k][0]:ws[i+k][1]]
			}
			k := strings.Join(parts, " ")
			id, ok := o.casKey[k]
			if !ok {
				id, ok = o.foldKey[strings.ToLower(k)]
			}
			if ok {
				out = append(out, mention{id, ws[i][0], ws[i+n-1][1]})
				found = n
			}
		}
		i += max(found, 1)
	}
	for _, m := range cveRE.FindAllStringIndex(s, -1) {
		out = append(out, mention{"cve:" + s[m[0]:m[1]], m[0], m[1]})
	}
	slices.SortFunc(out, func(a, b mention) int { return cmp.Or(cmp.Compare(a.start, b.start), cmp.Compare(b.end, a.end)) })
	// Overlaps keep the longer, earlier name; repeats of an entity collapse.
	kept := out[:0]
	for _, m := range out {
		if len(kept) > 0 && m.start < kept[len(kept)-1].end {
			continue
		}
		kept = append(kept, m)
	}
	return kept
}

// A predicate read between two entities: the words between them.
type pattern struct {
	predicate string
	re        *regexp.Regexp
	from      string // the subject's type it needs, if any
	reverse   bool   // the object comes first ("X was acquired by Y")
}

var patterns = []pattern{
	{"acquires", regexp.MustCompile(`(?i)\b(?:was|is being|has been) (?:acquired|bought) by\b`), "", true},
	{"acquires", regexp.MustCompile(`(?i)\b(?:acquires?|acquired|acquiring|to acquire|buys|bought|takeover of)\b`), "organization", false},
	{"fork_of", regexp.MustCompile(`(?i)\b(?:a )?fork(?:ed)? (?:of|from)\b`), "", false},
	{"deprecates", regexp.MustCompile(`(?i)\b(?:deprecat\w*|drops? support for|dropping support for|sunsets?|sunsetting|retires?|retiring|end[- ]of[- ]life for)\b`), "", false},
	{"migrates_to", regexp.MustCompile(`(?i)\b(?:migrat\w* (?:\w+ )?to|mov(?:es|ed|ing) (?:\w+ )?to|switch(?:es|ed|ing)? to|replac\w* (?:\w+ )?with)\b`), "", false},
	{"integrates_with", regexp.MustCompile(`(?i)\b(?:integrat\w* with|integration (?:with|for)|connects? (?:to|with)|adds? (?:native )?support for|now supports?|support for|plugs? into|works with)\b`), "", false},
	{"partners_with", regexp.MustCompile(`(?i)\b(?:partners? with|partnership with|teams? up with|collaborat\w* with|alliance with)\b`), "", false},
	{"invests_in", regexp.MustCompile(`(?i)\b(?:invests? in|investment in|backs|backed|funding (?:for|of))\b`), "organization", false},
	{"part_of", regexp.MustCompile(`(?i)\b(?:joins?|joined|donat\w* to|graduat\w* (?:at|in|from)|incubat\w* (?:at|in)|contributed to)\b`), "", false},
	{"releases", regexp.MustCompile(`(?i)\b(?:releases?|released|launch(?:es|ed)?|unveil(?:s|ed)?|announce[sd]?|introduce[sd]?|ships?|shipped|debuts?|rolls? out|rolled out|open[- ]sourc\w*|previews?)\b`), "organization", false},
	{"alternative_to", regexp.MustCompile(`(?i)(?:\bvs\.?|\bversus\b|\balternative to\b|\breplacement for\b|\bcompetes? with\b|\brivals?\b|\binstead of\b)`), "", false},
	{"uses", regexp.MustCompile(`(?i)\b(?:built (?:on|with)|powered by|runs? on|running on|uses|using|based on|on top of|leverag\w*|adopts?|adopted|written in|rewritten in|rewrite in)\b`), "", false},
}

// Predicates, in the words people read: the ontology's own, those the
// person's model may state (KGPredicates), and co-occurrences.
var predicates = func() map[string]string {
	m := map[string]string{"is_a": "is a", "developed_by": "developed by", "mentioned_with": "mentioned with", "related_to": "goes with"}
	for k, v := range KGPredicates {
		if m[k] == "" {
			m[k] = v
		}
	}
	return m
}()

// GEvidence is where a fact was read.
type GEvidence struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Sentence string    `json:"sentence"`
}

type triple struct{ s, p, o string }

type fact struct {
	n        int // articles that state it, each counted once
	last     uuid.UUID
	evidence []GEvidence
}

// kgRead reads the entities and facts of the news: those the person's model
// read in an item when it has, else the ontology's names and the words
// between them. Entities the model found and the ontology lacks come back
// in extra, by id.
func kgRead(docs []*gdoc) (count map[string]int, items map[string][]*gdoc, facts map[triple]*fact, extra map[string]*OEntity) {
	count, items, facts, extra = map[string]int{}, map[string][]*gdoc{}, map[triple]*fact{}, map[string]*OEntity{}
	add := func(t triple, d *gdoc, sentence string) {
		f := facts[t]
		if f == nil {
			f = &fact{}
			facts[t] = f
		}
		if f.last != d.id { // docs come one after the other
			f.n, f.last = f.n+1, d.id
		}
		if len(f.evidence) < 3 && !slices.ContainsFunc(f.evidence, func(e GEvidence) bool { return e.ID == d.id }) {
			f.evidence = append(f.evidence, GEvidence{d.id, d.title, cut(sentence, 280)})
		}
	}
	for _, d := range docs {
		seen := map[string]bool{}
		if d.kg != nil {
			ids := map[string]string{} // the item's names → entity id
			for _, e := range d.kg.Entities {
				id := resolve(e, extra)
				for _, n := range append([]string{e.Name}, e.Aliases...) {
					ids[n] = id
				}
				if !seen[id] {
					seen[id] = true
					count[id]++
					items[id] = append(items[id], d)
				}
			}
			for _, f := range d.kg.Facts {
				if s, o := ids[f.Subject], ids[f.Object]; s != "" && o != "" && s != o {
					if symmetric[f.Predicate] && s > o {
						s, o = o, s
					}
					add(triple{s, f.Predicate, o}, d, f.Sentence)
				}
			}
			continue
		}
		text := d.title + ". " + d.summary + ". " + d.body
		for _, s := range sentenceSplit.Split(strings.ReplaceAll(text, "\n", " "), -1) {
			// Links name hosts, not subjects; a feed's footer names its publisher.
			s = strings.TrimSpace(linkRE.ReplaceAllString(s, " "))
			if len(s) < 8 || len(s) > 600 || strings.Contains(s, "appeared first on") {
				continue
			}
			ms := onto.mentions(s)
			for _, m := range ms {
				if !seen[m.id] {
					seen[m.id] = true
					count[m.id]++
					items[m.id] = append(items[m.id], d)
				}
			}
			stated := map[[2]string]bool{}
			for i := 0; i+1 < len(ms); i++ {
				a, b := ms[i], ms[i+1]
				if a.id == b.id {
					continue
				}
				gap := s[a.end:b.start]
				if p, rev, ok := relate(a.id, b.id, gap, s[:a.start]); ok {
					t := triple{a.id, p, b.id}
					if rev {
						t = triple{b.id, p, a.id}
					}
					add(t, d, s)
					stated[[2]string{a.id, b.id}], stated[[2]string{b.id, a.id}] = true, true
				}
			}
			for i := range ms {
				for j := i + 1; j < len(ms); j++ {
					a, b := ms[i].id, ms[j].id
					if a == b || stated[[2]string{a, b}] {
						continue
					}
					if a > b {
						a, b = b, a
					}
					add(triple{a, "mentioned_with", b}, d, s)
				}
			}
		}
	}
	return count, items, facts, extra
}

// relate says what links a to b, read in the words between them.
func relate(a, b, gap, before string) (string, bool, bool) {
	if len(gap) > 90 {
		return "", false, false
	}
	ta, tb := entityType(a), entityType(b)
	// Concepts are too broad for a stated fact: they are only mentioned with.
	if ta == "concept" || tb == "concept" {
		return "", false, false
	}
	if ta == "vulnerability" && tb != "vulnerability" {
		return "affects", false, true
	}
	if tb == "vulnerability" && ta != "vulnerability" {
		return "affects", true, true
	}
	// "from X to Y" in a sentence about moving: X gives way to Y.
	if strings.TrimSpace(strings.ToLower(gap)) == "to" && endsFrom.MatchString(before) {
		return "migrates_to", false, true
	}
	for _, p := range patterns {
		if !p.re.MatchString(gap) {
			continue
		}
		subject := ta
		if p.reverse {
			subject = tb
		}
		if p.from != "" && subject != p.from {
			continue
		}
		if p.predicate == "part_of" && tb != "organization" || p.predicate == "releases" && tb == "organization" ||
			p.predicate == "alternative_to" && len(gap) > 30 {
			continue
		}
		return p.predicate, p.reverse, true
	}
	return "", false, false
}

// resolve links an entity the model named to the ontology's, by any of its
// names, or makes it an entity of its own, typed and classed as the model
// said, the first time it is seen.
func resolve(e KGEntity, extra map[string]*OEntity) string {
	for _, n := range append([]string{e.Name}, e.Aliases...) {
		if strings.HasPrefix(strings.ToUpper(n), "CVE-") && cveRE.MatchString(strings.ToUpper(n)) {
			return "cve:" + strings.ToUpper(n)
		}
		k := strings.Join(strings.Fields(n), " ")
		if id, ok := onto.casKey[k]; ok {
			return id
		}
		if id, ok := onto.foldKey[strings.ToLower(k)]; ok {
			return id
		}
	}
	id := "x:" + strings.ToLower(strings.Join(strings.Fields(e.Name), " "))
	if extra[id] == nil {
		class := e.Class
		if class == "" {
			class = strings.ToUpper(e.Type[:1]) + strings.ReplaceAll(e.Type[1:], "_", " ")
		}
		extra[id] = &OEntity{ID: id, Label: e.Name, Type: e.Type, Class: class}
	}
	return id
}

var (
	endsFrom = regexp.MustCompile(`(?i)\bfrom\s*$`)
	linkRE   = regexp.MustCompile(`\]\([^)]*\)|https?://\S+`)
)

func entityType(id string) string {
	if strings.HasPrefix(id, "cve:") {
		return "vulnerability"
	}
	if e := onto.byID[id]; e != nil {
		return e.Type
	}
	return ""
}

func entityLabel(id string) string {
	if strings.HasPrefix(id, "cve:") {
		return strings.TrimPrefix(id, "cve:")
	}
	return onto.byID[id].Label
}

func entityClass(id string) string {
	if strings.HasPrefix(id, "cve:") {
		return "Vulnerability"
	}
	return onto.byID[id].Class
}

// Bounds of the knowledge layer, so the graph stays readable.
const (
	maxEntities  = 150
	maxConcepts  = 60
	maxMentioned = 300 // co-mentions kept, strongest first
)

// kgContext is what the concept graph already worked out that entities
// need too: the corpus for summaries, the notes for bridges, the period for
// trends, and the link threshold.
type kgContext struct {
	corpus  knowledge.Corpus
	notes   map[string][]GItem
	weekAgo time.Time
	weeks   float64
	history bool
	minNPMI float64
	fringe  bool
}

// addKnowledge adds the entities, their classes, and the facts of the news
// to the concept graph, links concepts to the entities they go with, and
// works out rank and communities over it all.
func addKnowledge(g *Graph, docs []*gdoc, kc kgContext) {
	count, items, facts, extra := kgRead(docs)
	// Entities the model found and the ontology lacks have their own label,
	// type, and class.
	entityType := func(id string) string {
		if e := extra[id]; e != nil {
			return e.Type
		}
		return entityType(id)
	}
	entityLabel := func(id string) string {
		if e := extra[id]; e != nil {
			return e.Label
		}
		return entityLabel(id)
	}
	entityClass := func(id string) string {
		if e := extra[id]; e != nil {
			return e.Class
		}
		return entityClass(id)
	}
	ids := make([]string, 0, len(count))
	for id := range count {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int { return cmp.Or(cmp.Compare(count[b], count[a]), strings.Compare(a, b)) })
	// Like a concept, an entity needs two news: one alone is noise.
	ids = slices.DeleteFunc(ids, func(id string) bool { return count[id] < 2 })
	shown := map[string]bool{}
	for _, id := range ids[:min(len(ids), maxEntities)] {
		shown[id] = true
	}
	// A maker of two shown entities joins them, even unnamed in the news.
	makers := map[string]int{}
	for id := range shown {
		if e := onto.byID[id]; e != nil && e.By != "" {
			makers[e.By]++
		}
	}
	for by, n := range makers {
		if n >= 2 && !shown[by] {
			shown[by] = true
			ids = append(ids, by) // after the named ones, as count is 0
		}
	}

	// Concepts that are entities give way to them; the rest are capped.
	names := map[string]bool{}
	for id := range shown {
		if e := onto.byID[id]; e != nil {
			for _, a := range append([]string{e.Label}, e.Aliases...) {
				names[strings.ToLower(a)] = true
			}
		}
	}
	concepts := g.Nodes[:0:0]
	for _, n := range g.Nodes {
		if !names[strings.ToLower(n.Label)] {
			n.Type = "concept"
			concepts = append(concepts, n)
		}
	}
	slices.SortStableFunc(concepts, func(a, b GNode) int { return cmp.Compare(b.Count, a.Count) })
	if len(shown) > 0 && len(concepts) > maxConcepts {
		concepts = concepts[:maxConcepts]
	}
	keep := map[string]bool{}
	for _, n := range concepts {
		keep[n.ID] = true
	}
	g.Nodes = concepts
	g.Edges = slices.DeleteFunc(g.Edges, func(e GEdge) bool { return !keep[e.From] || !keep[e.To] })
	for i := range g.Edges {
		g.Edges[i].Predicate = "related_to"
	}

	slices.SortStableFunc(ids, func(a, b string) int { return cmp.Or(cmp.Compare(count[b], count[a]), strings.Compare(a, b)) })
	classes := map[string]bool{}
	var classOrder []string
	for _, id := range ids {
		if !shown[id] {
			continue
		}
		node := GNode{ID: "e:" + id, Kind: "entity", Type: entityType(id), Class: entityClass(id), Label: entityLabel(id), Count: count[id],
			Items: []GItem{}, Related: []string{}, InNotes: []GItem{}, Topics: []string{}}
		srcs := map[string]bool{}
		for _, d := range items[id] {
			srcs[d.source] = true
			if node.FirstSeen.IsZero() || d.published.Before(node.FirstSeen) {
				node.FirstSeen = d.published
			}
		}
		node.Sources = len(srcs)
		for _, d := range items[id][:min(3, len(items[id]))] {
			node.Items = append(node.Items, GItem{d.id, d.title})
		}
		if len(items[id]) > 0 {
			node.Summary = summarise(phraseKey(node.Label, "en"), items[id][:min(20, len(items[id]))], kc.corpus)
			recent := 0
			for _, d := range items[id] {
				if d.published.After(kc.weekAgo) {
					recent++
				}
			}
			if kc.history {
				expected := float64(len(items[id])-recent) / kc.weeks
				node.Trend = round2((float64(recent) - expected) / math.Sqrt(expected+1))
			}
		}
		// A bridge: the person's notes that name it, by any of its names.
		if e := onto.byID[id]; e != nil {
			for _, name := range append([]string{e.Label}, e.Aliases...) {
				for _, n := range kc.notes[phraseKey(name, "en")] {
					if len(node.InNotes) < 3 && !slices.ContainsFunc(node.InNotes, func(x GItem) bool { return x.ID == n.ID }) {
						node.InNotes = append(node.InNotes, n)
					}
				}
			}
		}
		g.Nodes = append(g.Nodes, node)
		if !classes[node.Class] {
			classes[node.Class] = true
			classOrder = append(classOrder, node.Class)
		}
		g.Edges = append(g.Edges, GEdge{From: node.ID, To: "k:" + node.Class, Predicate: "is_a", Directed: true, Reason: node.Label + " is a " + node.Class})
	}
	for _, c := range classOrder {
		g.Nodes = append(g.Nodes, GNode{ID: "k:" + c, Kind: "class", Type: "class", Label: c,
			Items: []GItem{}, Related: []string{}, InNotes: []GItem{}, Topics: []string{}})
	}
	// The ontology's facts between shown entities; the news repeating one adds no second edge.
	known := map[triple]bool{}
	for _, id := range ids {
		e := onto.byID[id]
		if e == nil || !shown[id] {
			continue
		}
		if e.By != "" && shown[e.By] {
			known[triple{id, "developed_by", e.By}] = true
			g.Edges = append(g.Edges, GEdge{From: "e:" + id, To: "e:" + e.By, Predicate: "developed_by", Directed: true,
				Reason: e.Label + " is developed by " + onto.byID[e.By].Label})
		}
		if e.ForkOf != "" && shown[e.ForkOf] {
			known[triple{id, "fork_of", e.ForkOf}] = true
			g.Edges = append(g.Edges, GEdge{From: "e:" + id, To: "e:" + e.ForkOf, Predicate: "fork_of", Directed: true,
				Reason: e.Label + " is a fork of " + onto.byID[e.ForkOf].Label})
		}
	}
	// The facts read in the news, with their evidence; co-mentions only when strong.
	var read, together []GEdge
	typed := map[[2]string]bool{}
	for t := range facts {
		if t.p != "mentioned_with" {
			typed[[2]string{t.s, t.o}], typed[[2]string{t.o, t.s}] = true, true
		}
	}
	for t, f := range facts {
		if !shown[t.s] || !shown[t.o] || known[t] {
			continue
		}
		e := GEdge{From: "e:" + t.s, To: "e:" + t.o, Predicate: t.p, Directed: !symmetric[t.p],
			Together: f.n, Evidence: f.evidence,
			Reason: fmt.Sprintf("%s %s %s (%d news)", entityLabel(t.s), predicates[t.p], entityLabel(t.o), f.n)}
		if t.p == "mentioned_with" {
			if f.n >= 2 && !typed[[2]string{t.s, t.o}] {
				together = append(together, e)
			}
			continue
		}
		read = append(read, e)
	}
	byStrength := func(a, b GEdge) int {
		return cmp.Or(cmp.Compare(b.Together, a.Together), strings.Compare(a.From+a.To, b.From+b.To))
	}
	slices.SortFunc(read, byStrength)
	slices.SortFunc(together, byStrength)
	g.Edges = append(g.Edges, read...)
	g.Edges = append(g.Edges, together[:min(len(together), maxMentioned)]...)

	// Concepts go with the entities their news share, by NPMI as concepts
	// go with each other.
	inDoc := map[*gdoc]map[string]bool{}
	for _, d := range docs {
		inDoc[d] = map[string]bool{}
		for _, c := range d.concepts {
			inDoc[d]["c:"+c] = true
		}
	}
	for id, ds := range items {
		for _, d := range ds {
			inDoc[d]["e:"+id] = true
		}
	}
	df := map[string]int{}
	for _, d := range docs {
		for _, c := range d.concepts {
			df["c:"+c]++
		}
	}
	N := float64(max(1, len(docs)))
	for _, c := range concepts {
		var best []GEdge
		for _, id := range ids {
			if !shown[id] || count[id] == 0 {
				continue
			}
			both := 0
			for _, d := range items[id] {
				if inDoc[d][c.ID] {
					both++
				}
			}
			if both < 2 {
				continue
			}
			pa, pb, pab := float64(df[c.ID])/N, float64(count[id])/N, float64(both)/N
			npmi := 1.0
			if pab < 1 {
				npmi = math.Log(pab/(pa*pb)) / -math.Log(pab)
			}
			if npmi < kc.minNPMI {
				continue
			}
			best = append(best, GEdge{From: c.ID, To: "e:" + id, Predicate: "related_to", Together: both, NPMI: round2(npmi),
				Reason: fmt.Sprintf("Together in %d news (NPMI %.2f)", both, npmi)})
		}
		// Its three strongest, so concepts hang on to entities without a hairball.
		slices.SortFunc(best, func(a, b GEdge) int {
			return cmp.Or(cmp.Compare(b.NPMI*float64(b.Together), a.NPMI*float64(a.Together)), strings.Compare(a.To, b.To))
		})
		g.Edges = append(g.Edges, best[:min(3, len(best))]...)
	}
	// A concept left with no link is fringe, shown only when asked.
	linked := map[string]bool{}
	for _, e := range g.Edges {
		linked[e.From], linked[e.To] = true, true
	}
	g.Nodes = slices.DeleteFunc(g.Nodes, func(n GNode) bool { return n.Kind == "concept" && !linked[n.ID] && !kc.fringe })
	g.Stats.Bridges = 0
	for i := range g.Nodes {
		if g.Nodes[i].Kind == "concept" && !linked[g.Nodes[i].ID] {
			g.Nodes[i].Fringe = true
		}
		if len(g.Nodes[i].InNotes) > 0 {
			g.Stats.Bridges++
		}
	}

	rankAndGroup(g)
	g.Stats.Entities, g.Stats.Facts, g.Stats.Classes = len(shown), len(read), len(classes)
}

// rankAndGroup works out each node's centrality (PageRank) and community
// (Louvain) over the whole graph, communities named by their most central
// members, biggest first; nodes with no link have none.
func rankAndGroup(g *Graph) {
	index := map[string]int{}
	for i, n := range g.Nodes {
		index[n.ID] = i
	}
	adj := make([]map[int]float64, len(g.Nodes))
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	weight := map[string]float64{"is_a": 0.5, "developed_by": 1, "fork_of": 1, "mentioned_with": 0.5}
	kept := g.Edges[:0]
	for _, e := range g.Edges {
		a, okA := index[e.From]
		b, okB := index[e.To]
		if !okA || !okB {
			continue
		}
		if e.Evidence == nil {
			e.Evidence = []GEvidence{}
		}
		kept = append(kept, e)
		w, ok := weight[e.Predicate]
		switch {
		case e.Predicate == "related_to":
			w = e.NPMI * float64(e.Together)
		case !ok:
			w = 2
		}
		if e.Together > 1 && e.Predicate != "related_to" {
			w *= float64(e.Together)
		}
		adj[a][b] += w
		adj[b][a] += w
	}
	g.Edges = kept
	g.Communities = []GCommunity{}
	if len(g.Nodes) == 0 {
		return
	}
	comm, mod := knowledge.Louvain(adj)
	ranks := knowledge.PageRank(adj)
	top := 0.0
	for _, r := range ranks {
		top = max(top, r)
	}
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
	for i := range g.Nodes {
		g.Nodes[i].Community = 0
		if len(adj[i]) > 0 {
			g.Nodes[i].Community = renum[comm[i]]
		}
		if top > 0 {
			g.Nodes[i].Rank = round2(ranks[i] / top)
		}
	}
	for _, c := range order {
		var members []GNode
		for _, n := range g.Nodes {
			if n.Community == renum[c] && n.Kind != "class" {
				members = append(members, n)
			}
		}
		slices.SortFunc(members, func(a, b GNode) int { return cmp.Compare(b.Rank, a.Rank) })
		var names []string
		for _, m := range members[:min(3, len(members))] {
			names = append(names, m.Label)
		}
		g.Communities = append(g.Communities, GCommunity{ID: renum[c], Label: strings.Join(names, " · "), Size: sizes[c]})
	}
	g.Stats.Modularity = round2(mod)
}
