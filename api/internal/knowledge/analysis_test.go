package knowledge

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestKeyPhrasesFavourConnectedTerms(t *testing.T) {
	text := `Spaced repetition schedules reviews just before forgetting. Spaced repetition beats cramming.
	The forgetting curve explains why spaced repetition and retrieval practice work. Retrieval practice strengthens memory.`
	got := KeyPhrases(text, 5)
	if len(got) == 0 || got[0].Score != 1 {
		t.Fatalf("phrases = %+v", got)
	}
	var texts []string
	for _, p := range got {
		texts = append(texts, p.Text)
	}
	if !slices.Contains(texts, "spaced repetition") || !slices.Contains(texts, "retrieval practice") {
		t.Fatalf("phrases = %v", texts)
	}
	if len(KeyPhrases("", 5)) != 0 || len(KeyPhrases("the and of", 5)) != 0 {
		t.Fatal("phrases from nothing")
	}
}

func TestReadability(t *testing.T) {
	easy := Read("The cat sat. The dog ran. We had fun.")
	hard := Read("Notwithstanding considerable methodological heterogeneity, metacognitive interventions demonstrated statistically significant improvements in comprehension.")
	if easy.Language != "en" || easy.Sentences != 3 || easy.Words != 9 || easy.Ease <= hard.Ease || easy.ReadingMinutes != 1 {
		t.Fatalf("easy %+v, hard %+v", easy, hard)
	}
	if fr := Read("Le chat dort. Nous lisons des livres."); fr.Language != "fr" || fr.Sentences != 2 {
		t.Fatalf("fr = %+v", fr)
	}
}

func note(title, kind, content string) *doc {
	return &doc{info: NoteInfo{ID: uuid.New(), Title: title, Kind: kind}, content: content}
}

func TestTopicsAndConcepts(t *testing.T) {
	raw := []*doc{
		note("Spaced repetition", "permanent", "Spaced repetition and retrieval practice fight the forgetting curve."),
		note("Retrieval practice", "permanent", "Retrieval practice with spaced repetition beats rereading and the forgetting curve."),
		note("Make It Stick", "literature", "The book shows retrieval practice and spaced repetition beat cramming."),
		note("Kubernetes pods", "permanent", "Kubernetes schedules pods on nodes; the control plane watches pods and nodes."),
		note("Kubernetes services", "literature", "A Kubernetes service routes traffic to pods on nodes behind the control plane."),
		note("Sourdough", "permanent", "Sourdough needs a starter, flour, water, and patience."),
	}
	analyse(raw)
	docs := make([]*analysed, len(raw))
	for i, d := range raw {
		text := Clean(d.content)
		docs[i] = &analysed{doc: d, phrases: KeyPhrases(d.info.Title+".\n"+text, phrasesPerNote), read: Read(text)}
	}
	tops := topics(docs)
	if len(tops) != 2 || len(tops[0].Notes) != 3 || len(tops[1].Notes) != 2 {
		t.Fatalf("topics = %+v", tops)
	}
	if !strings.Contains(tops[0].Label, "repetit") && !strings.Contains(tops[0].Label, "retriev") {
		t.Fatalf("label = %q", tops[0].Label)
	}
	nodes, edges := concepts(docs)
	have := map[string]bool{}
	for _, n := range nodes {
		have[n.Text] = true
		if n.Notes < 2 {
			t.Fatalf("concept %q is in %d note", n.Text, n.Notes)
		}
	}
	if !have["spaced repetition"] || !have["retrieval practice"] || have["sourdough"] {
		t.Fatalf("concepts = %v", have)
	}
	found := false
	for _, e := range edges {
		if e.Relation == "co-occurs" && e.Together >= 2 && e.NPMI > 0 && strings.HasPrefix(e.Reason, "Together in") {
			found = true
		}
	}
	if !found {
		t.Fatalf("edges = %+v", edges)
	}
}

func TestKeyPhrasesPreferRecurringPhrasesToTheirWords(t *testing.T) {
	got := KeyPhrases("Interleaving practice mixes problem types. Interleaving practice beats blocked practice.", 5)
	if got[0].Text != "interleaving practice" {
		t.Fatalf("phrases = %+v", got)
	}
	for _, p := range got[1:] {
		if p.Text == "practice" || p.Text == "interleaving" {
			t.Fatalf("a word inside the best phrase is listed: %+v", got)
		}
	}
}
