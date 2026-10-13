package media

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/l10n"
)

// Recommendation is an item from the person's own list, with why it is
// suggested now. No AI and no catalogue are involved: the reasons come from
// what they rated highly and what they left unfinished.
type Recommendation struct {
	Item    Item     `json:"item"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons"`
}

// lovedFrom is the rating from which a finished item counts as loved.
const lovedFrom = 8

// Recommend ranks the person's want list and paused items, best first.
func Recommend(ctx context.Context, q db.Querier, userID uuid.UUID, kind string, now time.Time, limit int) ([]Recommendation, error) {
	items, err := List(ctx, q, userID, Filter{Kind: kind})
	if err != nil {
		return nil, err
	}
	return rank(items, now, limit, l10n.Lang(ctx)), nil
}

func rank(items []Item, now time.Time, limit int, lang string) []Recommendation {
	var loved []Item
	for _, it := range items {
		if it.Status == "done" && it.Rating != nil && *it.Rating >= lovedFrom {
			loved = append(loved, it)
		}
	}
	out := []Recommendation{}
	for _, it := range items {
		if it.Status != "want" && it.Status != "doing" {
			continue
		}
		r := Recommendation{Item: it}
		if it.Status == "doing" {
			r.Score += 2
			at := it.Total != nil && it.Progress > 0
			days := int(now.Sub(it.UpdatedAt).Hours() / 24)
			var msg string
			switch {
			case at && days >= 14:
				msg = l10n.Tr(lang, "You started it and are at %d of %d; untouched for %d days.", it.Progress, *it.Total, days)
			case at:
				msg = l10n.Tr(lang, "You started it and are at %d of %d.", it.Progress, *it.Total)
			case days >= 14:
				msg = l10n.Tr(lang, "You started it; untouched for %d days.", days)
			default:
				msg = l10n.Tr(lang, "You started it.")
			}
			r.Reasons = append(r.Reasons, msg)
		}
		for _, name := range people(it.Creators) {
			if l := find(loved, func(o Item) bool { return o.ID != it.ID && slices.Contains(people(o.Creators), name) }); l != nil {
				r.Score += 3
				r.Reasons = append(r.Reasons, l10n.Tr(lang, "By %s, like %s, which you rated %d/10.", name, l.Title, *l.Rating))
			}
		}
		for _, g := range it.Genres {
			var like []string
			for _, o := range loved {
				if slices.ContainsFunc(o.Genres, func(x string) bool { return strings.EqualFold(x, g) }) {
					like = append(like, o.Title)
				}
			}
			if len(like) > 0 {
				r.Score++
				r.Reasons = append(r.Reasons, l10n.Tr(lang, "%s, like %s, which you loved.", g, list(like, lang)))
			}
		}
		if len(r.Reasons) == 0 {
			r.Reasons = append(r.Reasons, l10n.Tr(lang, "On your list since %s.", l10n.Date(lang, it.CreatedAt)))
		}
		out = append(out, r)
	}
	slices.SortStableFunc(out, func(a, b Recommendation) int {
		if a.Score != b.Score {
			return b.Score - a.Score
		}
		return a.Item.CreatedAt.Compare(b.Item.CreatedAt) // the longest-waiting first
	})
	return first(out, limit)
}

func people(creators string) []string {
	var out []string
	for _, p := range strings.Split(creators, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func find(items []Item, ok func(Item) bool) *Item {
	for i := range items {
		if ok(items[i]) {
			return &items[i]
		}
	}
	return nil
}

func list(titles []string, lang string) string {
	if len(titles) > 2 {
		return l10n.Tr(lang, "%s, %s, and others", titles[0], titles[1])
	}
	return strings.Join(titles, l10n.Tr(lang, " and "))
}
