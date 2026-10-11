// Package knowledge explores notes as a knowledge graph (EP-37): filters and
// statistics following PARA and the Zettelkasten, and edges built from
// wikilinks plus local NLP (TF-IDF keywords, named entities, a thesaurus,
// and cosine similarity). Every edge says why it exists, and nothing needs
// an AI provider.
package knowledge

import (
	_ "embed"
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var (
	frontMatter = regexp.MustCompile(`(?s)\A---\n.*?\n---\n`)
	codeBlock   = regexp.MustCompile("(?s)```.*?```")
	inlineCode  = regexp.MustCompile("`[^`]*`")
	link        = regexp.MustCompile(`https?://\S+`)
	markup      = regexp.MustCompile(`[#>*_\[\]()|!~=]+|(?m)^\s*[-+]\s|-{2,}`)
)

// Clean strips front matter, code, links, and Markdown syntax.
func Clean(md string) string {
	s := frontMatter.ReplaceAllString(md, "")
	s = codeBlock.ReplaceAllString(s, " ")
	s = inlineCode.ReplaceAllString(s, " ")
	s = link.ReplaceAllString(s, " ")
	return markup.ReplaceAllString(s, " ")
}

var stopwords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`a about above after again against all also am an and any are as at be because been
		before being below between both but by can could did do does doing down during each few for from further had has
		have having he her here hers herself him himself his how i if in into is it its itself just let me more most my
		myself no nor not now of off on once only or other our ours ourselves out over own same she should so some such
		than that the their theirs them themselves then there these they this those through to too under until up very
		was we were what when where which while who whom why will with would you your yours yourself yourselves one two
		get got make made use used using like also well much many may might must new way thing things see e.g i.e etc
		au aux avec ce ces cet cette dans de des du elle elles en est et eux il ils je la le les leur leurs lui ma mais
		me même mes moi mon ne nos notre nous on ou par pas pour qu que qui sa se ses son sur ta te tes toi ton tu un une
		vos votre vous été être avoir fait faire plus très tout tous toute toutes comme aussi donc car ni si`) {
		stopwords[w] = true
	}
	loadThesaurus()
}

// Stem reduces English and French inflections with a few safe suffix rules,
// enough to match "notes" with "note" and "learning" with "learn".
func Stem(w string) string {
	r := []rune(w)
	n := len(r)
	cut := func(suffix string, keep int) bool {
		if strings.HasSuffix(w, suffix) && n-len([]rune(suffix)) >= keep {
			w = string(r[:n-len([]rune(suffix))])
			return true
		}
		return false
	}
	switch {
	case cut("ies", 3):
		w += "y"
	case cut("ing", 4), cut("edly", 4), cut("ed", 4), cut("ly", 4), cut("ement", 4), cut("ements", 4), cut("ations", 4),
		cut("ation", 4), cut("es", 4), cut("s", 3), cut("x", 4):
	}
	return w
}

// Tokens are the content words of a text, lower-cased and stemmed.
func Tokens(text string) []string {
	stems, _ := TokensWithWords(text)
	return stems
}

// TokensWithWords also returns each token's word as written (lower-cased),
// so stems can be shown as real words.
func TokensWithWords(text string) (stems, surface []string) {
	for _, f := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && r != '\'' }) {
		f = strings.Trim(f, "'")
		if i := strings.LastIndex(f, "'"); i >= 0 { // l'idée → idée
			f = f[i+1:]
		}
		if len([]rune(f)) < 3 || stopwords[f] {
			continue
		}
		stems = append(stems, Stem(f))
		surface = append(surface, f)
	}
	return stems, surface
}

//go:embed thesaurus.txt
var thesaurusText string

// group maps a stem to its synonym group, and words keeps a readable word per stem.
var (
	group = map[string]int{}
	words = map[string]string{}
)

func loadThesaurus() {
	n := 0
	for _, line := range strings.Split(thesaurusText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		n++
		for _, w := range strings.Split(line, ",") {
			w = strings.ToLower(strings.TrimSpace(w))
			if w == "" {
				continue
			}
			s := Stem(w)
			if _, taken := group[s]; !taken {
				group[s] = n
				words[s] = w
			}
		}
	}
}

// Synonyms reports whether two stems are in the same thesaurus group.
func Synonyms(a, b string) bool {
	ga, okA := group[a]
	gb, okB := group[b]
	return okA && okB && ga == gb && a != b
}

var entityRe = regexp.MustCompile(`\b([A-Z][\p{L}'’-]+(?:\s+(?:of|de|du|des|la|le|and|&)?\s*[A-Z][\p{L}'’-]+){0,3})`)

// sentenceStart words are capitalised only because a sentence starts with them.
var sentenceStart = map[string]bool{"The": true, "A": true, "An": true, "In": true, "On": true, "This": true, "That": true,
	"It": true, "We": true, "I": true, "You": true, "If": true, "When": true, "What": true, "How": true, "Why": true,
	"Le": true, "La": true, "Les": true, "Un": true, "Une": true, "Dans": true, "Pour": true, "Ce": true, "Il": true,
	"Elle": true, "Je": true, "Nous": true, "Vous": true, "Quand": true, "See": true, "Also": true, "Note": true}

// Entities finds names: runs of capitalised words that are not merely the
// first word of a sentence ("Martin Kleppmann", "Designing Data-Intensive
// Applications", "Python").
func Entities(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		for _, m := range entityRe.FindAllStringIndex(line, -1) {
			e := strings.TrimSpace(line[m[0]:m[1]])
			first, _, _ := strings.Cut(e, " ")
			if sentenceStart[first] {
				rest := strings.TrimSpace(strings.TrimPrefix(e, first))
				if rest == "" {
					continue
				}
				e = rest
			}
			if len([]rune(e)) < 3 || stopwords[strings.ToLower(e)] || seen[e] {
				continue
			}
			// A single word at the start of a line is probably a sentence start.
			if !strings.Contains(e, " ") && strings.HasPrefix(strings.TrimLeft(line, " \t"), e) {
				continue
			}
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}

// Vector is a sparse TF-IDF vector, normalised to length 1.
type Vector map[string]float64

// Cosine is the similarity of two normalised vectors.
func Cosine(a, b Vector) float64 {
	if len(a) > len(b) {
		a, b = b, a
	}
	var dot float64
	for t, x := range a {
		dot += x * b[t]
	}
	return dot
}

// Corpus holds term statistics for a set of documents.
type Corpus struct {
	DF   map[string]int
	N    int
	Docs [][]string
}

// NewCorpus counts document frequencies.
func NewCorpus(docs [][]string) Corpus {
	c := Corpus{DF: map[string]int{}, N: len(docs), Docs: docs}
	for _, d := range docs {
		seen := map[string]bool{}
		for _, t := range d {
			if !seen[t] {
				seen[t] = true
				c.DF[t]++
			}
		}
	}
	return c
}

// TFIDF weights a document's terms: (1 + log tf) × log(1 + N / df), normalised.
func (c Corpus) TFIDF(doc []string) Vector {
	tf := map[string]int{}
	for _, t := range doc {
		tf[t]++
	}
	v := Vector{}
	var norm float64
	for t, n := range tf {
		w := (1 + math.Log(float64(n))) * math.Log(1+float64(c.N)/float64(c.DF[t]))
		v[t] = w
		norm += w * w
	}
	norm = math.Sqrt(norm)
	for t := range v {
		v[t] /= norm
	}
	return v
}

// Keywords are the k highest-weighted terms.
func Keywords(v Vector, k int) []string {
	terms := make([]string, 0, len(v))
	for t := range v {
		terms = append(terms, t)
	}
	slices.SortFunc(terms, func(a, b string) int {
		if v[a] != v[b] {
			if v[a] > v[b] {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	})
	if len(terms) > k {
		terms = terms[:k]
	}
	return terms
}

// Words remembers, per stem, the word people wrote most often.
type Words map[string]map[string]int

// Add counts the words behind a document's stems.
func (w Words) Add(stems, surface []string) {
	for i, s := range stems {
		if w[s] == nil {
			w[s] = map[string]int{}
		}
		w[s][surface[i]]++
	}
}

// Of is the most written word for a stem (ties: the shortest, then alphabetical).
func (w Words) Of(stem string) string {
	best, n := "", 0
	for word, c := range w[stem] {
		if c > n || c == n && (len(word) < len(best) || len(word) == len(best) && word < best) {
			best, n = word, c
		}
	}
	if best == "" {
		if t, ok := words[stem]; ok {
			return t
		}
		return stem
	}
	return best
}
