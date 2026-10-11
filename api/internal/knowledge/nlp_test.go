package knowledge

import (
	"slices"
	"testing"
)

func TestTokensStemAndDropStopwords(t *testing.T) {
	got := Tokens("The notes on Learning: l'idée des révisions, and studying!")
	want := []string{"note", "learn", "idée", "révision", "study"}
	if !slices.Equal(got, want) {
		t.Fatalf("Tokens = %v, want %v", got, want)
	}
}

func TestCleanStripsMarkdown(t *testing.T) {
	got := Tokens(Clean("---\ntags: [x]\n---\n# Title\n\n```go\nfunc main()\n```\nSee https://example.org and `code` **bold**"))
	if !slices.Equal(got, []string{"title", "bold"}) {
		t.Fatalf("Clean = %v", got)
	}
}

func TestSynonymsComeFromTheThesaurus(t *testing.T) {
	if !Synonyms(Stem("memory"), Stem("recall")) || !Synonyms(Stem("goal"), Stem("objectif")) {
		t.Error("expected synonyms")
	}
	if Synonyms(Stem("memory"), Stem("memory")) || Synonyms(Stem("memory"), Stem("budget")) {
		t.Error("unexpected synonyms")
	}
}

func TestEntities(t *testing.T) {
	got := Entities("I read Designing Data-Intensive Applications by Martin Kleppmann.\nThe book uses Python.\nPython is mentioned.")
	for _, want := range []string{"Designing Data-Intensive Applications", "Martin Kleppmann", "Python"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %q in %v", want, got)
		}
	}
	if slices.Contains(got, "The") || slices.Contains(got, "I") {
		t.Errorf("sentence starts taken as names: %v", got)
	}
}

func TestTFIDFAndCosine(t *testing.T) {
	docs := [][]string{Tokens("spaced repetition memory review"), Tokens("memory review schedule repetition"), Tokens("budget savings money")}
	c := NewCorpus(docs)
	a, b, z := c.TFIDF(docs[0]), c.TFIDF(docs[1]), c.TFIDF(docs[2])
	if Cosine(a, b) <= 0.3 || Cosine(a, z) != 0 {
		t.Fatalf("cosine a·b = %.2f, a·z = %.2f", Cosine(a, b), Cosine(a, z))
	}
	if k := Keywords(a, 1); k[0] != Stem("spaced") {
		t.Fatalf("top keyword = %v (rarest term should win)", k)
	}
}

func TestWordsShowTheWrittenForm(t *testing.T) {
	w := Words{}
	w.Add(TokensWithWords("spaced repetition, spaced practice"))
	if got := w.Of(Stem("spaced")); got != "spaced" {
		t.Fatalf("Of = %q", got)
	}
}
