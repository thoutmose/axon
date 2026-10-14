package watch

import (
	"html"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/thoutmose/axon/api/internal/clip"
	"github.com/thoutmose/axon/api/internal/knowledge"
)

// Topic is a domain the person watches.
type Topic struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
}

// Analysis is what Axon reads in one news item, all locally.
type Analysis struct {
	Markdown string   // the article (or its excerpt) as Markdown
	Summary  string   // plain text, at most two or three sentences
	Lang     string   // en or fr
	Words    int      // in the text Axon has
	Terms    []string // distinct stems, for near-duplicates and the graph
	Tokens   []string // stems in order, for the fingerprint
	SimHash  uint64
	Keywords []string // key phrases, by TextRank
	Topics   []string // ids of the person's topics it fits
	Score    float64  // how well it fits them
	Labels   []string // topics, key phrases, and the feed's own categories
	Outlier  string   // why it stands apart, or empty
}

var (
	blockTags = regexp.MustCompile(`(?i)</?(p|div|br|li|h[1-6]|blockquote|pre|tr|section|article)[^>]*>`)
	linkRe    = regexp.MustCompile(`https?://`)
	sentence  = regexp.MustCompile(`[^.!?]+[.!?]+`)
)

// PlainText turns a feed's HTML into text, paragraphs kept.
func PlainText(h string) string {
	s := blockTags.ReplaceAllString(h, "\n")
	s = html.UnescapeString(stripTags(s))
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.Join(strings.Fields(l), " "); l != "" {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}

// markdown renders an item's HTML as Markdown, links made absolute.
func markdown(h, pageURL string) string {
	if strings.TrimSpace(h) == "" {
		return ""
	}
	base, err := url.Parse(pageURL)
	if err != nil || base.Host == "" {
		base, _ = url.Parse("https://localhost/")
	}
	if p, err := clip.Read("<html><body><article>"+h+"</article></body></html>", base); err == nil && strings.TrimSpace(p.Markdown) != "" {
		return p.Markdown
	}
	return PlainText(h)
}

// firstSentences is the start of a text, up to about n characters, ending
// on a sentence.
func firstSentences(text string, n int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= n {
		return text
	}
	out := ""
	for _, s := range sentence.FindAllString(text, -1) {
		if len(out)+len(s) > n {
			break
		}
		out += s
	}
	if out == "" {
		cut := n
		for cut > 0 && !utf8Start(text[cut]) {
			cut--
		}
		return strings.TrimSpace(text[:cut]) + "…"
	}
	return strings.TrimSpace(out)
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

// Analyse reads an entry against the person's topics.
func Analyse(e Entry, topics []Topic) Analysis {
	var a Analysis
	body := e.Content
	if PlainText(body) == "" || len(PlainText(e.Summary)) > len(PlainText(body)) {
		body = firstOf(e.Content, e.Summary)
	}
	a.Markdown = markdown(body, e.URL)
	text := PlainText(body)
	if e.Markdown != "" { // the article's page, read by Axon
		a.Markdown = e.Markdown
		text = mdText(e.Markdown)
	}
	a.Summary = firstSentences(firstOf(PlainText(e.Summary), text), 400)
	full := e.Title + ".\n" + text
	a.Lang = knowledge.Language(full)
	stems, _ := knowledge.TokensIn(full, a.Lang)
	a.Tokens = stems
	a.Words = len(strings.Fields(text))
	seen := map[string]bool{}
	for _, s := range stems {
		if !seen[s] {
			seen[s] = true
			a.Terms = append(a.Terms, s)
		}
	}
	if len(a.Terms) > 400 {
		a.Terms = a.Terms[:400]
	}
	a.SimHash = SimHash(stems)
	for _, p := range knowledge.KeyPhrases(full, 6) {
		a.Keywords = append(a.Keywords, p.Text)
	}
	titleStems, _ := knowledge.TokensIn(e.Title, a.Lang)
	for _, t := range topics {
		if score := topicScore(t, titleStems, seen, a.Lang); score > 0 {
			a.Topics = append(a.Topics, t.ID)
			a.Labels = append(a.Labels, strings.ToLower(t.Name))
			a.Score += score
		}
	}
	for _, k := range a.Keywords[:min(3, len(a.Keywords))] {
		a.Labels = append(a.Labels, strings.ToLower(k))
	}
	for _, c := range e.Categories {
		a.Labels = append(a.Labels, strings.ToLower(c))
	}
	a.Labels = uniq(a.Labels, 12)
	a.Outlier = outlier(e, text, a.Words)
	return a
}

// topicScore is how strongly a text is about a topic. Each keyword (or the
// topic's name) is found when every one of its words appears, as itself or
// as a thesaurus synonym, each through a different word of the text. A
// keyword in the title counts three; in the text, one per distinct word of
// evidence, and one word alone is not enough: the same "sql" cannot count
// for both "Databases" and "SQL".
func topicScore(t Topic, title []string, body map[string]bool, lang string) float64 {
	inTitle := 0
	evidence := map[string]bool{}
	for _, kw := range append([]string{t.Name}, t.Keywords...) {
		ks, _ := knowledge.TokensIn(kw, lang)
		if len(ks) == 0 {
			continue
		}
		if ev := find(ks, func(s string) bool { return slices.Contains(title, s) }); ev != nil {
			inTitle++
			continue
		}
		for _, s := range find(ks, func(s string) bool { return body[s] }) {
			evidence[s] = true
		}
	}
	if inTitle == 0 && len(evidence) < 2 {
		return 0
	}
	return float64(3*inTitle + len(evidence))
}

// find returns, for each stem of a keyword, a different word of the text
// that is it or its synonym; nil when one is missing.
func find(stems []string, has func(string) bool) []string {
	var used []string
	for _, k := range stems {
		pick := ""
		if has(k) && !slices.Contains(used, k) {
			pick = k
		} else {
			for _, s := range knowledge.SynonymsOf(k) {
				if has(s) && !slices.Contains(used, s) {
					pick = s
					break
				}
			}
		}
		if pick == "" {
			return nil
		}
		used = append(used, pick)
	}
	return used
}

// outlier says why an item stands apart from ordinary news.
func outlier(e Entry, text string, words int) string {
	links := len(linkRe.FindAllString(e.Content+e.Summary, -1))
	switch {
	case words < 25 && e.Audio == "" && !strings.Contains(e.URL, "youtube.com"):
		return "too short"
	case words > 0 && links*8 > words:
		return "mostly links"
	case strings.Count(e.Title, "!") >= 2 || (len(e.Title) > 12 && strings.ToUpper(e.Title) == e.Title):
		return "shouting title"
	}
	return ""
}

func uniq(list []string, n int) []string {
	var out []string
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s != "" && len(s) <= 60 && !slices.Contains(out, s) {
			out = append(out, s)
		}
		if len(out) == n {
			break
		}
	}
	return out
}

var mdSyntax = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)|[#>*_` + "`" + `]+`)

// mdText is the words of a Markdown text, links reduced to their labels.
func mdText(md string) string {
	return PlainText(mdSyntax.ReplaceAllString(md, "$1"))
}
