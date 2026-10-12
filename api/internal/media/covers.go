package media

import (
	"context"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/text/unicode/norm"
)

// coversPerRun is how many items one "Find missing covers" looks up.
const coversPerRun = 20

// FindCovers looks up the person's items that have no cover, oldest first,
// and gives each the cover of the same work when the catalogue has one.
// Books use Open Library; films and series only when TMDB is set up. allow
// spends one lookup from the person's quota; when it is spent, the rest
// wait for the next run. Lookups are cached, so a run again is cheap.
func (c *Catalogs) FindCovers(ctx context.Context, userID uuid.UUID, allow func() bool) (checked, found int, err error) {
	kinds := []string{"book"}
	if c.TMDBEnabled() {
		kinds = append(kinds, "film", "series")
	}
	rows, err := c.db.Query(ctx, `SELECT `+columns+from+`
		WHERE e.user_id = $1 AND m.cover_id IS NULL AND m.kind = ANY($2) ORDER BY e.created_at, m.id LIMIT $3`, userID, kinds, coversPerRun)
	if err != nil {
		return 0, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) { return scan(r) })
	if err != nil {
		return 0, 0, err
	}
	for _, it := range items {
		if !allow() {
			break
		}
		query := it.Title
		if it.Kind == "book" && it.Creators != "" {
			query += " " + it.Creators
		}
		cands, err := c.Search(ctx, it.Kind, query)
		if err != nil {
			return checked, found, err
		}
		checked++
		cand := bestMatch(it, cands)
		if cand == nil {
			continue
		}
		cover := c.Cover(ctx, *cand)
		if cover == nil {
			continue
		}
		if _, err := c.db.Exec(ctx, `UPDATE media_items SET cover_id = $2, updated_at = now() WHERE id = $1 AND cover_id IS NULL`, it.ID, cover); err != nil {
			return checked, found, err
		}
		found++
	}
	return checked, found, nil
}

// bestMatch is the first candidate that is clearly the same work as it,
// with a cover: the same title, and the same author or year whenever both
// sides know one. A wrong cover is worse than none, so anything less is no
// match.
func bestMatch(it Item, cands []Candidate) *Candidate {
	title := normal(it.Title)
	mine := names(it.Creators)
	for i, c := range cands {
		switch {
		case !c.HasCover || c.CoverKey == "" || normal(c.Title) != title:
		case len(mine) > 0 && c.Creators != "" && !overlap(mine, names(c.Creators)):
		case it.Year != nil && c.Year != nil && *it.Year != *c.Year:
		default:
			return &cands[i]
		}
	}
	return nil
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
