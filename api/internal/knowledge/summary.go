package knowledge

import (
	"regexp"
	"slices"
	"strings"
)

// Summaries of long spoken texts (EP-148): a video's or an episode's words
// become a few of its own sentences, the most central ones, by TextRank
// over sentence similarity, kept in the order they were said. Nothing is
// invented: each line is something the speaker said.

var (
	spokenSentence = regexp.MustCompile(`[^.!?…]+[.!?…]+["”»)]*`)
	// Words of speech that carry no idea, in English and French: they never
	// make a keyword, and leave the excerpts of unpunctuated captions.
	fillers = regexp.MustCompile(`(?i)\b(um+|uh+|er+|ah+|hmm+|okay|ok|yeah|yep|yes|alright|all right|you know|i mean|kind of|sort of|` +
		`actually|basically|literally|really|gonna|wanna|gotta|anyway|euh|bah|ben|voilà|en fait|du coup|genre|quoi)\b`)
	// Verbs and words of speech that are frequent in a talk whatever it is about.
	spoken = regexp.MustCompile(`(?i)\b(know|think|thing|things|going|want|say|said|saying|get|got|getting|lot|lots|look|see|mean|make|go|goes|` +
		`come|put|take|let|let's|right|well|just|like|so|now|here|there|guys|people|stuff|way|really|okay|kind)\b`)
)

// sentences splits a text into sentences. Automatic captions often have no
// punctuation, so a text with too few sentence ends is cut into runs of
// about 25 words instead.
func sentences(text string) []string {
	text = strings.Join(strings.Fields(text), " ")
	words := strings.Fields(text)
	var out []string
	for _, s := range spokenSentence.FindAllString(text, -1) {
		if s = strings.TrimSpace(s); len(strings.Fields(s)) >= 5 {
			out = append(out, s)
		}
	}
	if len(words) > 0 && len(out) >= len(words)/60 {
		return out
	}
	out = out[:0]
	for i := 0; i < len(words); i += 25 {
		if run := excerpt(strings.Join(words[i:min(i+25, len(words))], " ")); len(strings.Fields(run)) >= 5 {
			out = append(out, run)
		}
	}
	return out
}

// excerpt tidies a run of unpunctuated speech: fillers and stutters out, a
// capital first, and an ellipsis to say the line is cut from a longer one.
func excerpt(run string) string {
	var kept []string
	for _, w := range strings.Fields(fillers.ReplaceAllString(run, "")) {
		if n := len(kept); n == 0 || !strings.EqualFold(kept[n-1], w) { // "I I", "a a"
			kept = append(kept, w)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	run = strings.Join(kept, " ")
	r := []rune(run)
	return strings.ToUpper(string(r[0])) + string(r[1:]) + "…"
}

// Summarise returns between lo and hi of a text's own sentences, about one
// in twelve, the most central first chosen and then put back in order.
func Summarise(text string, lo, hi int) []string {
	ss := sentences(text)
	if len(ss) <= lo {
		return ss
	}
	n := min(max(lo, (len(ss)+6)/12), hi)
	lang := Language(text)
	docs := make([][]string, len(ss))
	for i, s := range ss {
		docs[i], _ = TokensIn(s, lang)
	}
	c := NewCorpus(docs)
	vecs := make([]Vector, len(ss))
	for i, d := range docs {
		vecs[i] = c.TFIDF(d)
	}
	// Long texts compare each sentence with its neighbours only, so a
	// two-hour talk stays cheap; ideas that come back still connect.
	const reach = 80
	adj := make([]map[int]float64, len(ss))
	for i := range ss {
		adj[i] = map[int]float64{}
	}
	for i := range ss {
		for j := i + 1; j < min(len(ss), i+reach); j++ {
			if w := Cosine(vecs[i], vecs[j]); w > 0.05 {
				adj[i][j], adj[j][i] = w, w
			}
		}
	}
	rank := pageRank(adj)
	order := make([]int, len(ss))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		switch {
		case rank[a] > rank[b]:
			return -1
		case rank[a] < rank[b]:
			return 1
		}
		return a - b
	})
	// The best, skipping one that says the same as a line already kept.
	var keep []int
	for _, i := range order {
		if len(keep) == n {
			break
		}
		if !slices.ContainsFunc(keep, func(k int) bool { return Cosine(vecs[i], vecs[k]) > 0.6 }) {
			keep = append(keep, i)
		}
	}
	slices.Sort(keep)
	out := make([]string, len(keep))
	for i, k := range keep {
		out[i] = ss[k]
	}
	return out
}

// KeyPhrasesBetween returns between lo and hi key phrases of a text: its
// TextRank phrases, then its heaviest single terms when phrases run short.
func KeyPhrasesBetween(text string, lo, hi int) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		if k := strings.ToLower(s); !seen[k] && len(out) < hi {
			seen[k] = true
			out = append(out, s)
		}
	}
	// Fillers and the words of speech break phrases, so none is a keyword.
	text = spoken.ReplaceAllString(fillers.ReplaceAllString(text, ","), ",")
	for _, p := range KeyPhrases(text, hi) {
		add(p.Text)
	}
	if len(out) < lo {
		stems, surface := TokensWithWords(text)
		v := NewCorpus([][]string{stems}).TFIDF(stems)
		count := map[string]int{}
		for _, s := range stems {
			count[s]++
		}
		word := map[string]string{}
		for i, s := range stems {
			if _, ok := word[s]; !ok && i < len(surface) {
				word[s] = surface[i]
			}
		}
		terms := Keywords(v, len(v))
		slices.SortStableFunc(terms, func(a, b string) int { return count[b] - count[a] })
		for _, t := range terms {
			if len(out) >= lo {
				break
			}
			add(word[t])
		}
	}
	return out
}
