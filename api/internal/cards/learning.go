package cards

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/fsrs"
)

// How reviews are planned (EP-52), after Understanding How We Learn:
//
//   - Spaced practice: FSRS schedules each card just before it would be
//     forgotten, at the retention the person chose.
//   - New cards arrive a few a day, so the daily load stays steady instead
//     of piling up after a big import.
//   - Interleaving: a review of every deck mixes subjects, so consecutive
//     cards come from different decks and the person must first work out
//     which idea applies.
//   - Deliberate practice: leeches and the cards closest to being forgotten
//     can be practised on their own, without changing their schedule.

// Prefs are the person's learning settings ("learning" in their settings).
// They are read here rather than through package settings, which depends on
// the agents that depend on this package.
type Prefs struct {
	NewPerDay  int     `json:"new_per_day"`
	Retention  float64 `json:"retention"`
	Interleave bool    `json:"interleave"`
	TypeAnswer bool    `json:"type_answer"`
}

// DefaultPrefs: 20 new cards a day, 90% retention, decks mixed.
func DefaultPrefs() Prefs {
	return Prefs{NewPerDay: 20, Retention: 0.9, Interleave: true}
}

// LoadPrefs returns the person's learning settings, filled with defaults.
func LoadPrefs(ctx context.Context, q db.Querier, userID uuid.UUID) (Prefs, error) {
	p := DefaultPrefs()
	var raw []byte
	err := q.QueryRow(ctx, `SELECT doc->'learning' FROM user_settings WHERE user_id = $1`, userID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && raw == nil) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return DefaultPrefs(), nil //nolint:nilerr // a malformed stored value falls back to the defaults
	}
	return p, nil
}

func (p Prefs) scheduler() fsrs.Scheduler {
	s := fsrs.Default()
	if p.Retention >= 0.7 && p.Retention <= 0.97 {
		s.DesiredRetention = p.Retention
	}
	return s
}

// newIntroduced counts cards first reviewed in the last 24 hours.
func newIntroduced(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM (SELECT r.card_id FROM card_reviews r JOIN entities e ON e.id = r.card_id
		WHERE e.user_id = $1 GROUP BY r.card_id HAVING min(r.reviewed_at) > $2) x`, userID, now.Add(-24*time.Hour)).Scan(&n)
	return n, err
}

// Due returns what to review now: cards whose time has come, oldest due
// first, then new cards up to today's allowance. Across every deck, with
// interleaving on, decks take turns.
func Due(ctx context.Context, q db.Querier, userID uuid.UUID, deck string, now time.Time, limit int) ([]Card, error) {
	p, err := LoadPrefs(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM cards c JOIN entities e ON e.id = c.id
		WHERE e.user_id = $1 AND NOT c.suspended AND c.reps > 0 AND c.due <= $2 AND ($3 = '' OR c.deck = $3)
		ORDER BY c.due, c.id LIMIT $4`, userID, now, deck, limit)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Card, error) { return scan(r) })
	if err != nil {
		return nil, err
	}
	introduced, err := newIntroduced(ctx, q, userID, now)
	if err != nil {
		return nil, err
	}
	if allowance := min(p.NewPerDay-introduced, limit-len(out)); allowance > 0 {
		rows, err := q.Query(ctx, `SELECT `+columns+` FROM cards c JOIN entities e ON e.id = c.id
			WHERE e.user_id = $1 AND NOT c.suspended AND c.reps = 0 AND ($2 = '' OR c.deck = $2)
			ORDER BY e.created_at, c.id LIMIT $3`, userID, deck, allowance)
		if err != nil {
			return nil, err
		}
		fresh, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Card, error) { return scan(r) })
		if err != nil {
			return nil, err
		}
		out = append(out, fresh...)
	}
	if deck == "" && p.Interleave {
		out = interleave(out)
	}
	if out == nil {
		out = []Card{}
	}
	return out, nil
}

// interleave lets decks take turns, keeping each deck's own order.
func interleave(list []Card) []Card {
	var order []string
	byDeck := map[string][]Card{}
	for _, c := range list {
		if _, seen := byDeck[c.Deck]; !seen {
			order = append(order, c.Deck)
		}
		byDeck[c.Deck] = append(byDeck[c.Deck], c)
	}
	out := make([]Card, 0, len(list))
	for len(out) < len(list) {
		for _, d := range order {
			if q := byDeck[d]; len(q) > 0 {
				out = append(out, q[0])
				byDeck[d] = q[1:]
			}
		}
	}
	return out
}

// DueCount is how many cards Due would offer now, for the dashboard.
func DueCount(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (int, error) {
	p, err := LoadPrefs(ctx, q, userID)
	if err != nil {
		return 0, err
	}
	var reviews, fresh int
	if err := q.QueryRow(ctx, `SELECT count(*) FILTER (WHERE c.reps > 0 AND c.due <= $2), count(*) FILTER (WHERE c.reps = 0)
		FROM cards c JOIN entities e ON e.id = c.id WHERE e.user_id = $1 AND NOT c.suspended`, userID, now).Scan(&reviews, &fresh); err != nil {
		return 0, err
	}
	introduced, err := newIntroduced(ctx, q, userID, now)
	if err != nil {
		return 0, err
	}
	return reviews + max(0, min(fresh, p.NewPerDay-introduced)), nil
}

// Practice returns the cards most worth deliberate practice: leeches first,
// then the reviewed cards closest to being forgotten. Practising them does
// not change their schedule.
func Practice(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time, limit int) ([]Card, error) {
	p, err := LoadPrefs(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM cards c JOIN entities e ON e.id = c.id
		WHERE e.user_id = $1 AND NOT c.suspended AND c.reps > 0 ORDER BY c.lapses DESC, c.id LIMIT 2000`, userID)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Card, error) { return scan(r) })
	if err != nil {
		return nil, err
	}
	s := p.scheduler()
	recall := make(map[uuid.UUID]float64, len(list))
	for _, c := range list {
		recall[c.ID] = s.Retrievability(c.schedule(), now)
	}
	slices.SortStableFunc(list, func(a, b Card) int {
		switch {
		case a.Leech != b.Leech:
			if a.Leech {
				return -1
			}
			return 1
		case recall[a.ID] < recall[b.ID]:
			return -1
		case recall[a.ID] > recall[b.ID]:
			return 1
		}
		return 0
	})
	if len(list) > limit {
		list = list[:limit]
	}
	if list == nil {
		list = []Card{}
	}
	return list, nil
}

// Retention is the share of reviews of known cards (state review) answered
// hard, good, or easy over the last 30 days, and how many there were.
func Retention(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (float64, int, error) {
	var total, kept int
	err := q.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE r.rating > 1) FROM card_reviews r JOIN entities e ON e.id = r.card_id
		WHERE e.user_id = $1 AND r.state = 2 AND r.reviewed_at > $2`, userID, now.AddDate(0, 0, -30)).Scan(&total, &kept)
	if err != nil || total == 0 {
		return 0, total, err
	}
	return float64(kept) / float64(total), total, nil
}
