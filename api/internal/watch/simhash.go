package watch

import (
	"hash/fnv"
	"math/bits"
)

// SimHash fingerprints a text from its word shingles (Charikar 2002): two
// texts that say nearly the same thing differ in few of the 64 bits, so a
// story reposted with a new headline or a tracking footer is still found.
func SimHash(tokens []string) uint64 {
	var v [64]int
	add := func(s string, w int) {
		h := fnv.New64a()
		_, _ = h.Write([]byte(s))
		x := h.Sum64()
		for i := range 64 {
			if x&(1<<uint(i)) != 0 {
				v[i] += w
			} else {
				v[i] -= w
			}
		}
	}
	// Single words and pairs of words: pairs keep some word order.
	for i, t := range tokens {
		add(t, 1)
		if i+1 < len(tokens) {
			add(t+" "+tokens[i+1], 2)
		}
	}
	var out uint64
	for i := range 64 {
		if v[i] > 0 {
			out |= 1 << uint(i)
		}
	}
	return out
}

// Distance is the number of bits two fingerprints differ in.
func Distance(a, b uint64) int { return bits.OnesCount64(a ^ b) }

// NearDuplicate is the largest distance at which two texts are the same story.
const NearDuplicate = 3

// Jaccard is the overlap of two sets of words, 0 to 1.
func Jaccard(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	sa := map[string]bool{}
	for _, x := range a {
		sa[x] = true
	}
	sb := map[string]bool{}
	inter := 0
	for _, x := range b {
		if !sb[x] {
			sb[x] = true
			if sa[x] {
				inter++
			}
		}
	}
	return float64(inter) / float64(len(sa)+len(sb)-inter)
}
