package digest

import (
	"fmt"
	"math/bits"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The news digest (EP-155): the last day's items from the person's general
// news sources (a FreshRSS server's feeds, for one), told as stories. The
// tech watch already stores a story carried by several feeds once, naming
// the other outlets; beyond that, items
// that say the same thing (their SimHash fingerprints close, or their terms
// largely shared) are one story; a story counts for more the more distinct
// outlets carry it, and the newer it is. The ten that count most are told
// with their outlets, the headline of the outlet that covered it first, and
// a line of what it says.

const (
	maxHeadlines = 10
	newsWindow   = 36 * time.Hour
)

type newsItem struct {
	id        uuid.UUID
	title     string
	source    string
	also      []string // outlets whose copy of it was stored once, with this one
	summary   string
	simhash   uint64
	terms     []string
	published time.Time
}

// story is a cluster of items that say the same thing.
type story struct {
	items   []newsItem
	outlets []string
	score   float64
}

func overlap(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := make(map[string]bool, len(a))
	for _, t := range a {
		set[t] = true
	}
	shared := 0
	for _, t := range b {
		if set[t] {
			shared++
		}
	}
	return float64(shared) / float64(min(len(a), len(b)))
}

// same tells two items about one story apart from two stories.
func same(a, b newsItem) bool {
	return bits.OnesCount64(a.simhash^b.simhash) <= 6 || overlap(a.terms, b.terms) >= 0.45
}

// stories groups the items, most important first.
func stories(items []newsItem, now time.Time) []story {
	parent := make([]int, len(items))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	for i := range items {
		for j := i + 1; j < len(items); j++ {
			if same(items[i], items[j]) {
				parent[find(i)] = find(j)
			}
		}
	}
	groups := map[int]*story{}
	for i, it := range items {
		g := groups[find(i)]
		if g == nil {
			g = &story{}
			groups[find(i)] = g
		}
		g.items = append(g.items, it)
		for _, o := range append([]string{it.source}, it.also...) {
			if !slices.Contains(g.outlets, o) {
				g.outlets = append(g.outlets, o)
			}
		}
	}
	out := make([]story, 0, len(groups))
	for _, g := range groups {
		slices.SortFunc(g.items, func(a, b newsItem) int { return a.published.Compare(b.published) })
		age := now.Sub(g.items[len(g.items)-1].published).Hours()
		// Outlets count most: a story many newsrooms chose is the news.
		g.score = 3*float64(len(g.outlets)) + float64(len(g.items)) + max(0, 2-age/12)
		out = append(out, *g)
	}
	slices.SortFunc(out, func(a, b story) int {
		switch {
		case a.score > b.score:
			return -1
		case a.score < b.score:
			return 1
		}
		return b.items[0].published.Compare(a.items[0].published)
	})
	return out
}

// headlines builds the news digest.
func (b *builder) headlines(d *Digest) error {
	d.Title = b.tr["The news"]
	d.Label = b.longDate(b.today)
	d.URL = "/watch?kind=news"
	// Outliers stay: a short excerpt is common in news feeds.
	rows, err := b.q.Query(b.ctx, `SELECT i.id, i.title, i.source_title, i.also_in, i.summary, i.simhash, i.terms, i.published_at
		FROM watch_items i JOIN watch_sources s ON s.id = i.source_id
		WHERE i.user_id = $1 AND s.news AND i.published_at >= $2 AND i.status <> 'archived'
		ORDER BY i.published_at DESC LIMIT 400`, b.user, b.now.Add(-newsWindow))
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (newsItem, error) {
		var it newsItem
		var h int64
		err := r.Scan(&it.id, &it.title, &it.source, &it.also, &it.summary, &h, &it.terms, &it.published)
		it.simhash = uint64(h) //nolint:gosec // the same 64 bits
		if len(it.terms) > 60 {
			it.terms = it.terms[:60]
		}
		return it, err
	})
	if err != nil {
		return err
	}
	var list []Item
	for _, s := range stories(items, b.now) {
		if len(list) == maxHeadlines {
			break
		}
		first := s.items[0]
		detail := strings.Join(s.outlets[:min(3, len(s.outlets))], ", ")
		if more := len(s.outlets) - 3; more > 0 {
			detail += fmt.Sprintf(b.tr[" and %d more"], more)
		}
		if line := strings.TrimSpace(first.summary); line != "" {
			if r := []rune(line); len(r) > 220 {
				line = string(r[:220]) + "…"
			}
			detail += " — " + line
		}
		list = append(list, Item{ID: first.id, Type: "news", Text: first.title, Detail: detail, URL: "/watch?item=" + first.id.String()})
	}
	if len(list) > 0 {
		d.Sections = append(d.Sections, b.section("Top stories", "📰", list, 0))
	}
	return nil
}
