package knowledge

import (
	"slices"
	"testing"
)

func TestKeyPhrasesAreCachedByText(t *testing.T) {
	clear(phraseCache.m)
	text := "Spaced repetition beats massed practice. Spaced repetition needs retrieval practice, and retrieval practice needs effort."
	first := cachedKeyPhrases(text, 6)
	if len(first) == 0 || !slices.Equal(first, KeyPhrases(text, 6)) {
		t.Fatalf("cached = %+v, want the same as KeyPhrases", first)
	}
	if len(phraseCache.m) != 1 {
		t.Fatalf("cache holds %d entries, want 1", len(phraseCache.m))
	}
	// Callers reweigh what they get: that must not change the cache.
	first[0].Score = -1
	if again := cachedKeyPhrases(text, 6); again[0].Score == -1 || len(phraseCache.m) != 1 {
		t.Fatal("a caller's change reached the cache, or the second call missed it")
	}
	// Another k is another entry.
	cachedKeyPhrases(text, 3)
	if len(phraseCache.m) != 2 {
		t.Fatalf("cache holds %d entries, want 2", len(phraseCache.m))
	}
}
