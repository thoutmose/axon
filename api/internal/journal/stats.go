package journal

import (
	"math"
	"math/rand/v2"
	"slices"
	"sort"
)

// Statistics for "What helps you" (EP-92), small and checked against
// reference implementations: rank correlation, a permutation test, and the
// Benjamini–Hochberg correction for testing many pairs at once.

// ranks gives each value its rank from 1, ties sharing their average rank.
func ranks(v []float64) []float64 {
	idx := make([]int, len(v))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return v[idx[a]] < v[idx[b]] })
	out := make([]float64, len(v))
	for i := 0; i < len(idx); {
		j := i
		for j+1 < len(idx) && v[idx[j+1]] == v[idx[i]] {
			j++
		}
		avg := float64(i+j)/2 + 1
		for k := i; k <= j; k++ {
			out[idx[k]] = avg
		}
		i = j + 1
	}
	return out
}

// pearson is the correlation of two series; NaN when one does not vary.
func pearson(x, y []float64) float64 {
	n := float64(len(x))
	var mx, my float64
	for i := range x {
		mx += x[i]
		my += y[i]
	}
	mx, my = mx/n, my/n
	var sxy, sxx, syy float64
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 || syy == 0 {
		return math.NaN()
	}
	return sxy / math.Sqrt(sxx*syy)
}

// spearman is the rank correlation, robust to outliers and to scales
// like 1 to 5 that are ordered but not evenly spaced.
func spearman(x, y []float64) float64 { return pearson(ranks(x), ranks(y)) }

// permutationP is the two-sided p-value of x and y's rank correlation:
// how often shuffling y gives one at least as strong. Seeded, so the same
// data always gives the same answer.
func permutationP(x, y []float64, rounds int, seed uint64) float64 {
	rx, ry := ranks(x), ranks(y)
	observed := math.Abs(pearson(rx, ry))
	if math.IsNaN(observed) {
		return 1
	}
	r := rand.New(rand.NewPCG(seed, 0x5eed)) //nolint:gosec // seeded so the same data gives the same answer, not security
	shuffled := slices.Clone(ry)
	hits := 0
	for range rounds {
		r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if math.Abs(pearson(rx, shuffled)) >= observed-1e-12 {
			hits++
		}
	}
	return float64(hits+1) / float64(rounds+1)
}

// benjaminiHochberg turns p-values into q-values: the share of false
// findings expected among those kept at that level.
func benjaminiHochberg(p []float64) []float64 {
	m := len(p)
	idx := make([]int, m)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return p[idx[a]] < p[idx[b]] })
	q := make([]float64, m)
	next := 1.0
	for rank := m; rank >= 1; rank-- {
		i := idx[rank-1]
		v := math.Min(next, p[i]*float64(m)/float64(rank))
		q[i], next = v, v
	}
	return q
}
