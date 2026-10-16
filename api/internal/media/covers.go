package media

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/text/unicode/norm"
)

// coversPerRun is how many items one "Find missing covers" looks up.
const coversPerRun = 20

// CoverBudget is how long one run may take: covers can be slow (Open
// Library's redirect to archive.org), and the answer must come before the
// server's 15-second write timeout. Tests shorten it.
var CoverBudget = 12 * time.Second

// FindCovers looks up the person's items that have no cover, oldest first,
// and gives each the cover of the same work when the catalogue has one.
// Books use Open Library; films and series only when TMDB is set up. allow
// spends one lookup from the person's quota. When the quota or the time
// budget runs out, more is true and the rest wait for the next run; lookups
// are cached, so a run again is cheap.
func (c *Catalogs) FindCovers(ctx context.Context, userID uuid.UUID, allow func() bool) (checked, found int, more bool, err error) {
	kinds := []string{"book"}
	if c.TMDBEnabled() {
		kinds = append(kinds, "film", "series")
	}
	rows, err := c.db.Query(ctx, `SELECT `+columns+from+`
		WHERE e.user_id = $1 AND m.cover_id IS NULL AND m.kind = ANY($2) ORDER BY e.created_at, m.id LIMIT $3`, userID, kinds, coversPerRun)
	if err != nil {
		return 0, 0, false, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) { return scan(r) })
	if err != nil {
		return 0, 0, false, err
	}
	save := context.WithoutCancel(ctx) // a cover that arrived is kept, even at the deadline
	ctx, cancel := context.WithTimeout(ctx, CoverBudget)
	defer cancel()
	for _, it := range items {
		if ctx.Err() != nil || !allow() {
			return checked, found, true, nil
		}
		// The title alone: with its author, Open Library often finds nothing;
		// bestMatch checks the author.
		cands, err := c.Search(ctx, it.Kind, it.Title)
		if err != nil {
			if ctx.Err() != nil {
				return checked, found, true, nil
			}
			return checked, found, false, err
		}
		checked++
		cand := bestMatch(it, cands)
		if cand == nil {
			continue
		}
		// Cached search results leave the cover's key out; the work's details
		// have it (as when adding from a search).
		full, err := c.Details(ctx, cand.Source, it.Kind, cand.SourceID)
		if err != nil {
			if ctx.Err() != nil {
				return checked, found, true, nil
			}
			return checked, found, false, err
		}
		cover := c.Cover(ctx, full)
		if cover == nil {
			if ctx.Err() != nil {
				return checked, found, true, nil
			}
			continue
		}
		if _, err := c.db.Exec(save, `UPDATE media_items SET cover_id = $2, updated_at = now() WHERE id = $1 AND cover_id IS NULL`, it.ID, cover); err != nil {
			return checked, found, false, err
		}
		found++
	}
	return checked, found, false, nil
}

// bestMatch is the first candidate that is clearly the same work as it,
// with a cover: the same title, and the same author or year whenever both
// sides know one. A wrong cover is worse than none, so anything less is no
// match.
//
// When no title is the same, a catalogue that names the work a little
// differently (TMDB's "Dune" for "Dune: Part One") still matches if one title
// starts the other at a word and both sides know the same year (EP-134).
func bestMatch(it Item, cands []Candidate) *Candidate {
	title := normal(it.Title)
	mine := names(it.Creators)
	fits := func(c Candidate) bool {
		switch {
		case !c.HasCover:
		case len(mine) > 0 && c.Creators != "" && !overlap(mine, names(c.Creators)):
		case it.Year != nil && c.Year != nil && *it.Year != *c.Year:
		default:
			return true
		}
		return false
	}
	for i, c := range cands {
		if normal(c.Title) == title && fits(c) {
			return &cands[i]
		}
	}
	if it.Year == nil {
		return nil
	}
	for i, c := range cands {
		if t := normal(c.Title); c.Year != nil && startsAtWord(title, t) && fits(c) {
			return &cands[i]
		}
	}
	return nil
}

// startsAtWord reports whether one of two normal titles begins the other,
// ending at a word: "dune" and "dune part one", not "dun" and "dune".
func startsAtWord(a, b string) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	return a != "" && strings.HasPrefix(b, a) && (len(b) == len(a) || b[len(a)] == ' ')
}

// normal lower-cases s and keeps its letters and digits, without accents,
// one space apart: "Dune!" and "dune" are the same title.
func normal(s string) string {
	var b strings.Builder
	space := false
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			space = false
		default:
			space = true
		}
	}
	return b.String()
}

// names splits a list of creators ("A, B") into normal names.
func names(s string) []string {
	var out []string
	for _, n := range strings.Split(s, ",") {
		if n = normal(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func overlap(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}
