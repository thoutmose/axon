package knowledge

import (
	"slices"
	"strings"
	"testing"
)

func stem(w string) string { return StemIn(w, "en") }

func tokens(text string) []string {
	stems, _ := TokensWithWords(text)
	return stems
}

func TestTokensStemAndDropStopwords(t *testing.T) {
	got := tokens("The notes on Learning: l'idée des révisions, and studying!")
	want := []string{"note", "learn", "idé", "révision", "studi"} // English text: the English Snowball stemmer
	if !slices.Equal(got, want) {
		t.Fatalf("Tokens = %v, want %v", got, want)
	}
}

func TestCleanStripsMarkdown(t *testing.T) {
	got := tokens(Clean("---\ntags: [x]\n---\n# Title\n\n```go\nfunc main()\n```\nSee https://example.org and `code` **bold**"))
	if !slices.Equal(got, []string{"titl", "bold"}) {
		t.Fatalf("Clean = %v", got)
	}
}

func TestSynonymsComeFromTheThesaurus(t *testing.T) {
	if !Synonyms(stem("memory"), stem("recall")) || !Synonyms(stem("goal"), stem("objectif")) {
		t.Error("expected synonyms")
	}
	if Synonyms(stem("memory"), stem("memory")) || Synonyms(stem("memory"), stem("budget")) {
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
	docs := [][]string{tokens("spaced repetition memory review"), tokens("memory review schedule repetition"), tokens("budget savings money")}
	c := NewCorpus(docs)
	a, b, z := c.TFIDF(docs[0]), c.TFIDF(docs[1]), c.TFIDF(docs[2])
	if Cosine(a, b) <= 0.3 || Cosine(a, z) != 0 {
		t.Fatalf("cosine a·b = %.2f, a·z = %.2f", Cosine(a, b), Cosine(a, z))
	}
	if k := Keywords(a, 1); k[0] != stem("spaced") {
		t.Fatalf("top keyword = %v (rarest term should win)", k)
	}
}

func TestWordsShowTheWrittenForm(t *testing.T) {
	w := Words{}
	w.Add(TokensWithWords("spaced repetition, spaced practice"))
	if got := w.Of(stem("spaced")); got != "spaced" {
		t.Fatalf("Of = %q", got)
	}
}

func TestLanguageAndFrenchStemming(t *testing.T) {
	fr := "Les étudiants apprennent mieux quand ils révisent leurs notes et les cartes."
	if Language(fr) != "fr" || Language("The students learn better when they review their notes.") != "en" {
		t.Fatal("language detection")
	}
	got := tokens(fr)
	if !slices.Contains(got, "étudi") || !slices.Contains(got, "cart") {
		t.Fatalf("French tokens = %v", got)
	}
}

func TestWordNetSynonyms(t *testing.T) {
	// From Open English WordNet: each word's most common sense.
	if !Synonyms(stem("remark"), stem("note")) {
		t.Error("remark and note should be synonyms")
	}
	if Synonyms(stem("note"), stem("bill")) {
		t.Error("a rare sense linked note to bill")
	}
}

// Transcripts and news come in more than one language (EP-123): Spanish is
// recognised and stemmed too, and English stays English.
func TestSpanish(t *testing.T) {
	es := "Las bases de datos distribuidas replican los datos entre varios servidores para que el sistema siga funcionando cuando uno falla."
	if Language(es) != "es" {
		t.Fatalf("Language = %q", Language(es))
	}
	stems, _ := TokensIn("bases distribuidas replicando", "es")
	if strings.Join(stems, " ") != "bas distribu replic" {
		t.Fatalf("stems = %v", stems)
	}
	if Language("The solo developer shipped a new era of tools for the Han dynasty museum.") != "en" {
		t.Fatal("English read as Spanish")
	}
}
