package knowledge

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
)

// Advanced note analysis (EP-46) for literature and permanent notes, all
// local: key phrases by TextRank, readability, topics by label propagation
// over the similarity graph, and a concept graph scored by normalised
// pointwise mutual information (NPMI), with WordNet synonyms between concepts.

// Phrase is a key phrase of a note.
type Phrase struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"` // 0 to 1, relative to the note's best phrase
	key   string  // stems, to match the phrase across notes
}

var segmentRe = regexp.MustCompile(`[.!?;:,()\[\]"“”«»\n]+`)

// KeyPhrases ranks a text's candidate phrases (runs of up to three content
// words between stopwords and punctuation) with TextRank over a word
// co-occurrence graph, and returns the best k.
func KeyPhrases(text string, k int) []Phrase {
	lang := Language(text)
	type cand struct{ stems, surface []string }
	var cands []cand
	var seq [][]string // stems per segment, for co-occurrence
	for _, seg := range segmentRe.Split(strings.ToLower(text), -1) {
		var run cand
		var segStems []string
		flush := func() {
			for len(run.stems) > 0 {
				n := min(3, len(run.stems))
				cands = append(cands, cand{run.stems[:n], run.surface[:n]})
				run = cand{run.stems[n:], run.surface[n:]}
			}
		}
		for _, f := range strings.FieldsFunc(seg, func(r rune) bool { return !unicode.IsLetter(r) && r != '\'' && r != '-' }) {
			f = strings.Trim(f, "'-")
			if i := strings.LastIndex(f, "'"); i >= 0 {
				f = f[i+1:]
			}
			if len([]rune(f)) < 3 || stopwords[f] {
				flush()
				continue
			}
			s := StemIn(f, lang)
			run.stems, run.surface = append(run.stems, s), append(run.surface, f)
			segStems = append(segStems, s)
		}
		flush()
		seq = append(seq, segStems)
	}
	if len(cands) == 0 {
		return []Phrase{}
	}
	// Co-occurrence within a window of two words, then PageRank.
	nbrs := map[string]map[string]bool{}
	link := func(a, b string) {
		if a == b {
			return
		}
		if nbrs[a] == nil {
			nbrs[a] = map[string]bool{}
		}
		if nbrs[b] == nil {
			nbrs[b] = map[string]bool{}
		}
		nbrs[a][b], nbrs[b][a] = true, true
	}
	for _, s := range seq {
		for i := range s {
			if nbrs[s[i]] == nil {
				nbrs[s[i]] = map[string]bool{}
			}
			for j := i + 1; j < len(s) && j <= i+2; j++ {
				link(s[i], s[j])
			}
		}
	}
	rank := map[string]float64{}
	for w := range nbrs {
		rank[w] = 1
	}
	for range 30 {
		next := map[string]float64{}
		for w := range nbrs {
			sum := 0.0
			for v := range nbrs[w] {
				sum += rank[v] / float64(len(nbrs[v]))
			}
			next[w] = 0.15 + 0.85*sum
		}
		rank = next
	}
	// Every n-gram of one to three words within a run is a candidate, scored by
	// how often it occurs, how central its words are, and its length; a
	// phrase inside a better one is dropped ("spaced" under "spaced repetition").
	type gram struct {
		surface string
		stems   []string
		freq    int
	}
	grams := map[string]*gram{}
	for _, c := range cands {
		for i := range c.stems {
			for n := 1; n <= 3 && i+n <= len(c.stems); n++ {
				key := strings.Join(c.stems[i:i+n], " ")
				if g := grams[key]; g != nil {
					g.freq++
				} else {
					grams[key] = &gram{strings.Join(c.surface[i:i+n], " "), c.stems[i : i+n], 1}
				}
			}
		}
	}
	all := make([]Phrase, 0, len(grams))
	for key, g := range grams {
		if len(g.stems) > 2 && g.freq < 2 {
			continue // a three-word phrase must recur to count
		}
		sum := 0.0
		for _, st := range g.stems {
			sum += rank[st]
		}
		all = append(all, Phrase{Text: g.surface, Score: float64(g.freq) * sum, key: key})
	}
	slices.SortFunc(all, func(a, b Phrase) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.key, b.key)
	})
	var out []Phrase
	inside := func(small, big string) bool { return strings.Contains(" "+big+" ", " "+small+" ") }
next:
	for _, p := range all {
		for i, q := range out {
			switch {
			case inside(p.key, q.key):
				continue next // already covered by a longer phrase
			case inside(q.key, p.key):
				// A longer phrase takes the place of a word it covers at least half the time.
				if 2*grams[p.key].freq >= grams[q.key].freq {
					out[i] = p
				}
				continue next
			}
		}
		if len(out) < k {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b Phrase) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.key, b.key)
	})
	for i := range out {
		out[i].Score = math.Round(out[i].Score/out[0].Score*100) / 100
	}
	return out
}

// Readability describes how easy a text is to read.
type Readability struct {
	Language       string `json:"language"` // en or fr
	Words          int    `json:"words"`
	Sentences      int    `json:"sentences"`
	ReadingMinutes int    `json:"reading_minutes"`
	// Ease is Flesch reading ease in English and Kandel–Moles in French:
	// about 90 very easy, 60 plain, 30 difficult.
	Ease float64 `json:"ease"`
}

var sentenceEnd = regexp.MustCompile(`[.!?]+(\s|$)`)

// Read measures a cleaned text.
func Read(text string) Readability {
	r := Readability{Language: Language(text)}
	syllables := 0
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) }) {
		r.Words++
		syllables += countSyllables(w, r.Language)
	}
	r.Sentences = max(1, len(sentenceEnd.FindAllStringIndex(strings.TrimSpace(text), -1)))
	if r.Words == 0 {
		return r
	}
	r.ReadingMinutes = max(1, int(math.Round(float64(r.Words)/220)))
	wps, spw := float64(r.Words)/float64(r.Sentences), float64(syllables)/float64(r.Words)
	if r.Language == "fr" {
		r.Ease = 207 - 1.015*wps - 73.6*spw
	} else {
		r.Ease = 206.835 - 1.015*wps - 84.6*spw
	}
	r.Ease = math.Round(math.Max(0, math.Min(100, r.Ease)))
	return r
}

// countSyllables counts vowel groups, dropping a silent final e.
func countSyllables(w, lang string) int {
	n, prev := 0, false
	for _, r := range w {
		v := strings.ContainsRune("aeiouyàâäéèêëîïôöùûüœæ", r)
		if v && !prev {
			n++
		}
		prev = v
	}
	if n > 1 && strings.HasSuffix(w, "e") && !strings.HasSuffix(w, "le") && lang == "en" {
		n--
	}
	if n > 1 && lang == "fr" && (strings.HasSuffix(w, "e") || strings.HasSuffix(w, "es")) {
		n--
	}
	return max(1, n)
}

// NoteBrief names a note in an analysis.
type NoteBrief struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Kind  string    `json:"kind"`
}

// Topic is a group of notes about the same thing.
type Topic struct {
	ID    int         `json:"id"`
	Label string      `json:"label"` // its three most characteristic words
	Terms []string    `json:"terms"`
	Notes []NoteBrief `json:"notes"`
}

// Concept is a key phrase found in several notes.
type Concept struct {
	ID    string      `json:"id"`
	Text  string      `json:"text"`
	Notes int         `json:"notes"`
	Of    []NoteBrief `json:"of"`
}

// ConceptEdge links two concepts: they appear together more than chance
// (relation co-occurs, with the NPMI) or are WordNet synonyms.
type ConceptEdge struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Relation string  `json:"relation"` // co-occurs or synonym
	Together int     `json:"together"` // notes with both
	NPMI     float64 `json:"npmi"`
	Reason   string  `json:"reason"`
}

// Analysis is the corpus-level picture of literature and permanent notes.
type Analysis struct {
	Notes     int            `json:"notes"`
	Words     int            `json:"words"`
	Languages map[string]int `json:"languages"`
	Ease      float64        `json:"ease"` // average readability
	Topics    []Topic        `json:"topics"`
	Concepts  []Concept      `json:"concepts"`
	Edges     []ConceptEdge  `json:"edges"`
	Truncated bool           `json:"truncated"`
}

var errNoNote = errors.New("no such note")

// AnalysedKinds are literature and permanent notes (Zettelkasten).
var AnalysedKinds = []string{"literature", "permanent"}

const (
	maxConcepts     = 80
	maxConceptEdges = 200
	phrasesPerNote  = 10
	topicSimilarity = 0.12
)

type analysed struct {
	*doc
	phrases []Phrase
	read    Readability
}

func analyseKinds(ctx context.Context, q db.Querier, userID uuid.UUID, kinds []string) ([]*analysed, Words, bool, error) {
	var docs []*doc
	truncated := false
	for _, k := range kinds {
		d, t, err := load(ctx, q, userID, Filter{Kind: k})
		if err != nil {
			return nil, nil, false, err
		}
		docs, truncated = append(docs, d...), truncated || t
	}
	words := analyse(docs)
	out := make([]*analysed, len(docs))
	df := map[string]int{}
	for i, d := range docs {
		text := Clean(d.content)
		out[i] = &analysed{doc: d, phrases: cachedKeyPhrases(d.info.Title+".\n"+text, phrasesPerNote*2), read: Read(text)}
		for _, p := range out[i].phrases {
			df[p.key]++
		}
	}
	// A phrase that recurs across notes is more likely a concept than an
	// accident of wording ("spaced repetition" rather than "recall feels"):
	// weigh each note's phrases by 1 + ln(document frequency) in the corpus.
	for _, a := range out {
		for j := range a.phrases {
			a.phrases[j].Score *= 1 + math.Log(float64(df[a.phrases[j].key]))
		}
		slices.SortStableFunc(a.phrases, func(x, y Phrase) int { return cmp.Compare(y.Score, x.Score) })
		a.phrases = a.phrases[:min(phrasesPerNote, len(a.phrases))]
		if len(a.phrases) > 0 {
			top := a.phrases[0].Score
			for j := range a.phrases {
				a.phrases[j].Score = math.Round(a.phrases[j].Score/top*100) / 100
			}
		}
	}
	return out, words, truncated, nil
}

func brief(d *analysed) NoteBrief {
	return NoteBrief{ID: d.info.ID, Title: d.info.Title, Kind: d.info.Kind}
}

// Analyse builds topics and the concept graph of the person's literature
// and permanent notes (or one of those kinds).
func Analyse(ctx context.Context, q db.Querier, userID uuid.UUID, kinds []string) (Analysis, error) {
	docs, _, truncated, err := analyseKinds(ctx, q, userID, kinds)
	if err != nil {
		return Analysis{}, err
	}
	a := Analysis{Notes: len(docs), Languages: map[string]int{}, Topics: []Topic{}, Concepts: []Concept{}, Edges: []ConceptEdge{}, Truncated: truncated}
	ease := 0.0
	for _, d := range docs {
		a.Words += d.read.Words
		a.Languages[d.read.Language]++
		ease += d.read.Ease
	}
	if len(docs) > 0 {
		a.Ease = math.Round(ease / float64(len(docs)))
	}
	a.Topics = topics(docs)
	a.Concepts, a.Edges = concepts(docs)
	return a, nil
}

// neighbour is a similar note and how similar it is.
type neighbour struct {
	j int
	w float64
}

// similarPairs lists, for each note, the notes whose cosine similarity with
// it is at least min, in ascending order. Only notes that share a term can
// score above zero, so an inverted index (term → notes) compares those
// alone: linear in the shared terms rather than quadratic in the notes
// (EP-78; 3,000 notes took 4.4 s comparing every pair).
func similarPairs(docs []*analysed, min float64) [][]neighbour {
	type posting struct {
		doc int
		w   float64
	}
	index := map[string][]posting{}
	for i, d := range docs {
		for t, w := range d.vec {
			index[t] = append(index[t], posting{i, w})
		}
	}
	adj := make([][]neighbour, len(docs))
	dot := make([]float64, len(docs)) // reused: only the touched entries are reset
	var touched []int
	for i, d := range docs {
		for t, w := range d.vec {
			for _, p := range index[t] {
				if p.doc <= i {
					continue
				}
				if dot[p.doc] == 0 {
					touched = append(touched, p.doc)
				}
				dot[p.doc] += w * p.w
			}
		}
		slices.Sort(touched)
		for _, j := range touched {
			if dot[j] >= min {
				adj[i] = append(adj[i], neighbour{j, dot[j]})
				adj[j] = append(adj[j], neighbour{i, dot[j]})
			}
			dot[j] = 0
		}
		touched = touched[:0]
	}
	return adj
}

// topics groups notes by label propagation over the similarity graph
// (cosine of TF-IDF vectors), in a fixed order so results are stable.
func topics(docs []*analysed) []Topic {
	n := len(docs)
	adj := similarPairs(docs, topicSimilarity)
	label := make([]int, n)
	for i := range label {
		label[i] = i
	}
	for range 20 {
		changed := false
		for i := range n {
			if len(adj[i]) == 0 {
				continue
			}
			weight := map[int]float64{}
			for _, e := range adj[i] {
				weight[label[e.j]] += e.w
			}
			best, bw := label[i], weight[label[i]]
			for l, w := range weight {
				if w > bw || (w == bw && l < best) {
					best, bw = l, w
				}
			}
			if best != label[i] {
				label[i], changed = best, true
			}
		}
		if !changed {
			break
		}
	}
	members := map[int][]*analysed{}
	for i, l := range label {
		members[l] = append(members[l], docs[i])
	}
	var out []Topic
	for _, ms := range members {
		if len(ms) < 2 {
			continue
		}
		sum := Vector{}
		for _, d := range ms {
			for t, w := range d.vec {
				sum[t] += w
			}
		}
		var terms []string
		for _, t := range Keywords(sum, 5) {
			terms = append(terms, readable(t))
		}
		t := Topic{Terms: terms, Label: strings.Join(terms[:min(3, len(terms))], " · ")}
		for _, d := range ms {
			t.Notes = append(t.Notes, brief(d))
		}
		slices.SortFunc(t.Notes, func(a, b NoteBrief) int { return cmp.Compare(a.Title, b.Title) })
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b Topic) int {
		if c := cmp.Compare(len(b.Notes), len(a.Notes)); c != 0 {
			return c
		}
		return cmp.Compare(a.Label, b.Label)
	})
	for i := range out {
		out[i].ID = i + 1
	}
	if out == nil {
		out = []Topic{}
	}
	return out
}

// readable shows a stem as a word, from the thesaurus or the notes.
func readable(stem string) string {
	if w, ok := words[stem]; ok {
		return w
	}
	return stem
}

// concepts keeps key phrases found in at least two notes and links those
// that appear together more often than chance, or are synonyms.
func concepts(docs []*analysed) ([]Concept, []ConceptEdge) {
	type agg struct {
		text  string
		score float64
		docs  []*analysed
	}
	byKey := map[string]*agg{}
	for _, d := range docs {
		for _, p := range d.phrases {
			a := byKey[p.key]
			if a == nil {
				a = &agg{text: p.Text}
				byKey[p.key] = a
			}
			a.score += p.Score
			a.docs = append(a.docs, d)
		}
	}
	var keys []string
	for k, a := range byKey {
		if len(a.docs) >= 2 {
			keys = append(keys, k)
		}
	}
	slices.SortFunc(keys, func(x, y string) int {
		if c := cmp.Compare(len(byKey[y].docs), len(byKey[x].docs)); c != 0 {
			return c
		}
		if c := cmp.Compare(byKey[y].score, byKey[x].score); c != 0 {
			return c
		}
		return cmp.Compare(x, y)
	})
	keys = keys[:min(maxConcepts, len(keys))]
	nodes := make([]Concept, len(keys))
	sets := make([]map[uuid.UUID]bool, len(keys))
	for i, k := range keys {
		a := byKey[k]
		nodes[i] = Concept{ID: k, Text: a.text, Notes: len(a.docs)}
		sets[i] = map[uuid.UUID]bool{}
		for _, d := range a.docs {
			sets[i][d.info.ID] = true
			if len(nodes[i].Of) < 8 {
				nodes[i].Of = append(nodes[i].Of, brief(d))
			}
		}
	}
	total := float64(len(docs))
	var edges []ConceptEdge
	for i := range keys {
		for j := i + 1; j < len(keys); j++ {
			both := 0
			for id := range sets[i] {
				if sets[j][id] {
					both++
				}
			}
			if both >= 2 {
				pxy := float64(both) / total
				px, py := float64(len(sets[i]))/total, float64(len(sets[j]))/total
				npmi := 1.0
				if pxy < 1 {
					npmi = math.Log(pxy/(px*py)) / -math.Log(pxy)
				}
				if npmi >= 0.1 {
					npmi = math.Round(npmi*100) / 100
					edges = append(edges, ConceptEdge{From: keys[i], To: keys[j], Relation: "co-occurs", Together: both, NPMI: npmi,
						Reason: fmt.Sprintf("Together in %d notes (NPMI %.2f)", both, npmi)})
				}
			}
			ki, kj := strings.Fields(keys[i]), strings.Fields(keys[j])
			if len(ki) == 1 && len(kj) == 1 && Synonyms(ki[0], kj[0]) {
				edges = append(edges, ConceptEdge{From: keys[i], To: keys[j], Relation: "synonym",
					Reason: fmt.Sprintf("“%s” and “%s” are synonyms (thesaurus)", nodes[i].Text, nodes[j].Text)})
			}
		}
	}
	slices.SortStableFunc(edges, func(a, b ConceptEdge) int { return cmp.Compare(b.NPMI, a.NPMI) })
	edges = edges[:min(maxConceptEdges, len(edges))]
	if edges == nil {
		edges = []ConceptEdge{}
	}
	return nodes, edges
}

// Related is a note close to another, with why.
type Related struct {
	NoteBrief
	Similarity float64  `json:"similarity"`
	Reasons    []string `json:"reasons"`
}

// NoteAnalysis is the analysis of one note against the person's literature
// and permanent notes.
type NoteAnalysis struct {
	Readability Readability `json:"readability"`
	Phrases     []Phrase    `json:"phrases"`
	Topic       *Topic      `json:"topic"`
	Related     []Related   `json:"related"`
}

// AnalyseNote analyses one note: its key phrases, readability, topic, and
// the five closest literature or permanent notes, each with reasons.
func AnalyseNote(ctx context.Context, q db.Querier, userID, noteID uuid.UUID) (NoteAnalysis, error) {
	docs, _, _, err := analyseKinds(ctx, q, userID, AnalysedKinds)
	if err != nil {
		return NoteAnalysis{}, err
	}
	var self *analysed
	for _, d := range docs {
		if d.info.ID == noteID {
			self = d
		}
	}
	if self == nil { // a fleeting or project note: analyse it against the corpus
		one, _, err := load(ctx, q, userID, Filter{})
		if err != nil {
			return NoteAnalysis{}, err
		}
		for _, d := range one {
			if d.info.ID == noteID {
				all := append([]*doc{d}, docsOf(docs)...)
				analyse(all)
				text := Clean(d.content)
				self = &analysed{doc: d, phrases: cachedKeyPhrases(d.info.Title+".\n"+text, phrasesPerNote), read: Read(text)}
			}
		}
		if self == nil {
			return NoteAnalysis{}, errNoNote
		}
	}
	out := NoteAnalysis{Readability: self.read, Phrases: self.phrases, Related: []Related{}}
	for _, t := range topics(docs) {
		if slices.ContainsFunc(t.Notes, func(n NoteBrief) bool { return n.ID == noteID }) {
			out.Topic = &t
		}
	}
	mine := map[string]string{}
	for _, p := range self.phrases {
		mine[p.key] = p.Text
	}
	selfKeys := Keywords(self.vec, 10)
	var rel []Related
	for _, d := range docs {
		if d.info.ID == noteID {
			continue
		}
		sim := Cosine(self.vec, d.vec)
		var reasons []string
		var shared []string
		for _, p := range d.phrases {
			if t, ok := mine[p.key]; ok {
				shared = append(shared, "“"+t+"”")
			}
		}
		if len(shared) > 0 {
			reasons = append(reasons, "Shares "+strings.Join(shared[:min(3, len(shared))], ", "))
		}
	syn:
		for _, a := range selfKeys {
			for _, b := range Keywords(d.vec, 10) {
				if Synonyms(a, b) {
					reasons = append(reasons, fmt.Sprintf("“%s” here, “%s” there (synonyms)", readable(a), readable(b)))
					break syn
				}
			}
		}
		if sim < 0.1 && len(reasons) == 0 {
			continue
		}
		reasons = append(reasons, fmt.Sprintf("Similar wording (%.0f %%)", sim*100))
		rel = append(rel, Related{NoteBrief: brief(d), Similarity: math.Round(sim*100) / 100, Reasons: reasons})
	}
	slices.SortFunc(rel, func(a, b Related) int {
		if c := cmp.Compare(len(b.Reasons), len(a.Reasons)); c != 0 {
			return c
		}
		return cmp.Compare(b.Similarity, a.Similarity)
	})
	if len(rel) > 0 {
		out.Related = rel[:min(5, len(rel))]
	}
	return out, nil
}

func docsOf(a []*analysed) []*doc {
	out := make([]*doc, len(a))
	for i, d := range a {
		out[i] = d.doc
	}
	return out
}
