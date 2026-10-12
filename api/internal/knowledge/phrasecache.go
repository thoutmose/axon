package knowledge

import (
	"crypto/sha256"
	"slices"
	"strconv"
	"sync"
)

// phraseCache keeps each text's key phrases. TextRank was a third of the
// time of the Notes page's graph and analysis, which run it on every note at
// every visit, while notes rarely change in between (EP-79). Keyed by a hash
// of k and the text, so an edited note is simply a new entry; cleared when
// full, so it stays a few megabytes.
var phraseCache = struct {
	sync.Mutex
	m map[[32]byte][]Phrase
}{m: map[[32]byte][]Phrase{}}

const phraseCacheSize = 20000

// cachedKeyPhrases is KeyPhrases, from the cache when it can. Callers get
// their own copy: they reweigh it.
func cachedKeyPhrases(text string, k int) []Phrase {
	key := sha256.Sum256([]byte(strconv.Itoa(k) + "\x00" + text))
	phraseCache.Lock()
	p, ok := phraseCache.m[key]
	phraseCache.Unlock()
	if !ok {
		p = KeyPhrases(text, k)
		phraseCache.Lock()
		if len(phraseCache.m) >= phraseCacheSize {
			clear(phraseCache.m)
		}
		phraseCache.m[key] = p
		phraseCache.Unlock()
	}
	return slices.Clone(p)
}
