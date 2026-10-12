package knowledge

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

// synthetic returns n notes of about 60 distinct terms each, drawn from a
// 5,000-word vocabulary with a few shared themes, as unit TF-IDF vectors.
func synthetic(n int) []*analysed {
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // deterministic test data
	docs := make([]*analysed, n)
	for i := range docs {
		v := Vector{}
		theme := r.IntN(40) * 100
		for range 60 {
			t := r.IntN(5000)
			if r.IntN(3) == 0 {
				t = theme + r.IntN(100)
			}
			v[fmt.Sprint("t", t)] += 1
		}
		norm := 0.0
		for _, x := range v {
			norm += x * x
		}
		for t := range v {
			v[t] /= math.Sqrt(norm)
		}
		docs[i] = &analysed{doc: &doc{vec: v}}
	}
	return docs
}

func BenchmarkTopics(b *testing.B) {
	for _, n := range []int{500, 1500, 3000} {
		docs := synthetic(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for b.Loop() {
				topics(docs)
			}
		})
	}
}

// similarPairs finds the same neighbours as comparing every pair, with the
// same scores, each list in ascending order.
func TestSimilarPairsMatchesEveryPair(t *testing.T) {
	docs := synthetic(300)
	got := similarPairs(docs, topicSimilarity)
	edges := 0
	for i := range docs {
		var want []neighbour
		for j := range docs {
			if j != i {
				if s := Cosine(docs[i].vec, docs[j].vec); s >= topicSimilarity {
					want = append(want, neighbour{j, s})
				}
			}
		}
		if len(got[i]) != len(want) {
			t.Fatalf("note %d: %d neighbours, want %d", i, len(got[i]), len(want))
		}
		for k := range want {
			if got[i][k].j != want[k].j || math.Abs(got[i][k].w-want[k].w) > 1e-12 {
				t.Fatalf("note %d, neighbour %d = %+v, want %+v", i, k, got[i][k], want[k])
			}
		}
		edges += len(want)
	}
	if edges == 0 {
		t.Fatal("the synthetic notes should have similar pairs")
	}
}
