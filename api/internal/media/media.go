// Package media tracks films, series, and books (EP-17): what a person
// wants, is on, has finished, or dropped, with ratings, progress, and
// reviews. Catalogue data from TMDB and Open Library fills items in but is
// never required.
package media

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// Type is the entity type.
const Type = "media"

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "media_items", Export: export})
}

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Kinds and statuses.
var (
	Kinds    = []string{"film", "series", "book", "game"}
	Statuses = []string{"want", "doing", "done", "dropped"}
)

// Item is one film, series, book, or video game (EP-149).
type Item struct {
	ID         uuid.UUID  `json:"id"`
	Title      string     `json:"title"`
	Kind       string     `json:"kind"`
	Status     string     `json:"status"`
	Year       *int       `json:"year"`
	Creators   string     `json:"creators"`
	Genres     []string   `json:"genres"`
	Rating     *int       `json:"rating"`
	Progress   int        `json:"progress"`
	Total      *int       `json:"total"`
	StartedOn  *string    `json:"started_on"`
	FinishedOn *string    `json:"finished_on"`
	Review     string     `json:"review"`
	Source     string     `json:"source"`
	SourceID   string     `json:"source_id"`
	Overview   string     `json:"overview"`
	CoverID    *uuid.UUID `json:"cover_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Fields are the changeable fields; nil leaves a field as it is.
type Fields struct {
	Title      *string   `json:"title"`
	Kind       *string   `json:"kind"`
	Status     *string   `json:"status"`
	Year       *int      `json:"year"`
	Creators   *string   `json:"creators"`
	Genres     *[]string `json:"genres"`
	Rating     *int      `json:"rating"` // 0 clears it
	Progress   *int      `json:"progress"`
	Total      *int      `json:"total"` // 0 clears it
	StartedOn  *string   `json:"started_on"`
	FinishedOn *string   `json:"finished_on"`
	Review     *string   `json:"review"`
}

const columns = `m.id, e.title, m.kind, m.status, m.year, m.creators, m.genres, m.rating, m.progress, m.total,
	to_char(m.started_on, 'YYYY-MM-DD'), to_char(m.finished_on, 'YYYY-MM-DD'), m.review, m.source, m.source_id,
	m.overview, m.cover_id, e.created_at, m.updated_at`

const from = ` FROM media_items m JOIN entities e ON e.id = m.id`

func scan(row pgx.Row) (Item, error) {
	var it Item
	var rating *int16
	err := row.Scan(&it.ID, &it.Title, &it.Kind, &it.Status, &it.Year, &it.Creators, &it.Genres, &rating, &it.Progress, &it.Total,
		&it.StartedOn, &it.FinishedOn, &it.Review, &it.Source, &it.SourceID, &it.Overview, &it.CoverID, &it.CreatedAt, &it.UpdatedAt)
	if rating != nil {
		r := int(*rating)
		it.Rating = &r
	}
	if it.Genres == nil {
		it.Genres = []string{}
	}
	return it, err
}

// Get returns one of the person's items.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Item, error) {
	it, err := scan(q.QueryRow(ctx, `SELECT `+columns+from+` WHERE m.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return it, graph.ErrNotFound
	}
	return it, err
}

// Filter narrows List; empty fields match everything.
type Filter struct {
	Kind   string
	Status string
}

// List returns the person's items, most recently changed first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, f Filter) ([]Item, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+from+` WHERE e.user_id = $1 AND ($2 = '' OR m.kind = $2) AND ($3 = '' OR m.status = $3)
		ORDER BY m.updated_at DESC, m.id LIMIT 2000`, userID, f.Kind, f.Status)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) { return scan(r) })
	if out == nil {
		out = []Item{}
	}
	return out, err
}

// Catalog is what a catalogue entry adds to a new item.
type Catalog struct {
	Source   string
	SourceID string
	Overview string
	CoverID  *uuid.UUID
}

// Create adds an item. Title and kind are required; status defaults to want.
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f Fields, c *Catalog, now time.Time) (Item, error) {
	if f.Title == nil || strings.TrimSpace(*f.Title) == "" {
		return Item{}, ValidationError{"Give it a title."}
	}
	if f.Kind == nil {
		return Item{}, ValidationError{"Say whether it is a film, a series, a book, or a game."}
	}
	it := Item{Status: "want", Genres: []string{}, Source: "manual"}
	if err := apply(&it, f, now); err != nil {
		return Item{}, err
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, it.Title, searchText(it))
	if err != nil {
		return Item{}, err
	}
	var fetched *time.Time
	if c != nil {
		it.Source, it.SourceID, it.Overview, it.CoverID = c.Source, c.SourceID, truncate(c.Overview, 4000), c.CoverID
		fetched = &now
	}
	if _, err := tx.Exec(ctx, `INSERT INTO media_items (id, kind, status, year, creators, genres, rating, progress, total,
		started_on, finished_on, review, source, source_id, overview, cover_id, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		id, it.Kind, it.Status, it.Year, it.Creators, it.Genres, it.Rating, it.Progress, it.Total,
		it.StartedOn, it.FinishedOn, it.Review, it.Source, it.SourceID, it.Overview, it.CoverID, fetched); err != nil {
		return Item{}, err
	}
	return Get(ctx, tx, userID, id)
}

// Update changes the given fields.
func Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, f Fields, now time.Time) (Item, error) {
	it, err := Get(ctx, tx, userID, id)
	if err != nil {
		return it, err
	}
	if err := apply(&it, f, now); err != nil {
		return it, err
	}
	if _, err := tx.Exec(ctx, `UPDATE media_items SET kind = $2, status = $3, year = $4, creators = $5, genres = $6, rating = $7,
		progress = $8, total = $9, started_on = $10, finished_on = $11, review = $12, updated_at = clock_timestamp() WHERE id = $1`,
		id, it.Kind, it.Status, it.Year, it.Creators, it.Genres, it.Rating, it.Progress, it.Total, it.StartedOn, it.FinishedOn, it.Review); err != nil {
		return it, err
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, it.Title, searchText(it), changed(f)); err != nil {
		return it, err
	}
	return Get(ctx, tx, userID, id)
}

// apply validates and sets fields. Moving to doing or done fills in the
// start and finish dates when they are empty.
func apply(it *Item, f Fields, now time.Time) error {
	if f.Title != nil {
		t := strings.TrimSpace(*f.Title)
		if t == "" || len([]rune(t)) > 500 {
			return ValidationError{"The title is 1 to 500 characters."}
		}
		it.Title = t
	}
	if f.Kind != nil {
		if !slices.Contains(Kinds, *f.Kind) {
			return ValidationError{"Kind is film, series, book, or game."}
		}
		it.Kind = *f.Kind
	}
	if f.Year != nil {
		if *f.Year == 0 {
			it.Year = nil
		} else if *f.Year < 1000 || *f.Year > 3000 {
			return ValidationError{"The year looks wrong."}
		} else {
			it.Year = f.Year
		}
	}
	if f.Creators != nil {
		if len([]rune(*f.Creators)) > 500 {
			return ValidationError{"Creators are up to 500 characters."}
		}
		it.Creators = strings.TrimSpace(*f.Creators)
	}
	if f.Genres != nil {
		g := cleanGenres(*f.Genres)
		if len(g) > 12 {
			return ValidationError{"Up to 12 genres."}
		}
		it.Genres = g
	}
	if f.Rating != nil {
		switch {
		case *f.Rating == 0:
			it.Rating = nil
		case *f.Rating < 1 || *f.Rating > 10:
			return ValidationError{"Rate from 1 to 10."}
		default:
			it.Rating = f.Rating
		}
	}
	if f.Total != nil {
		switch {
		case *f.Total == 0:
			it.Total = nil
		case *f.Total < 1 || *f.Total > 100000:
			return ValidationError{"The total is 1 to 100,000."}
		default:
			it.Total = f.Total
		}
	}
	if f.Progress != nil {
		if *f.Progress < 0 || *f.Progress > 100000 {
			return ValidationError{"Progress is 0 to 100,000."}
		}
		it.Progress = *f.Progress
	}
	if it.Total != nil && it.Progress > *it.Total {
		return ValidationError{"Progress cannot be more than the total."}
	}
	for _, d := range []struct {
		in  *string
		out **string
	}{{f.StartedOn, &it.StartedOn}, {f.FinishedOn, &it.FinishedOn}} {
		if d.in == nil {
			continue
		}
		if *d.in == "" {
			*d.out = nil
			continue
		}
		if _, err := time.Parse(time.DateOnly, *d.in); err != nil {
			return ValidationError{"Dates are YYYY-MM-DD."}
		}
		v := *d.in
		*d.out = &v
	}
	if f.Review != nil {
		if len([]rune(*f.Review)) > 10000 {
			return ValidationError{"The review is up to 10,000 characters."}
		}
		it.Review = *f.Review
	}
	if f.Status != nil {
		if !slices.Contains(Statuses, *f.Status) {
			return ValidationError{"Status is want, doing, done, or dropped."}
		}
		today := now.Format(time.DateOnly)
		if (*f.Status == "doing" || *f.Status == "done") && it.StartedOn == nil {
			it.StartedOn = &today
		}
		if *f.Status == "done" && it.FinishedOn == nil {
			it.FinishedOn = &today
		}
		if *f.Status == "done" && it.Total != nil {
			it.Progress = *it.Total
		}
		it.Status = *f.Status
	}
	if it.StartedOn != nil && it.FinishedOn != nil && *it.FinishedOn < *it.StartedOn {
		return ValidationError{"It cannot finish before it started."}
	}
	return nil
}

func cleanGenres(in []string) []string {
	out := []string{}
	for _, g := range in {
		g = strings.TrimSpace(g)
		if g != "" && len([]rune(g)) <= 60 && !slices.ContainsFunc(out, func(o string) bool { return strings.EqualFold(o, g) }) {
			out = append(out, g)
		}
	}
	return out
}

func searchText(it Item) string {
	return strings.Join(append([]string{it.Creators}, it.Genres...), " ")
}

func changed(f Fields) []string {
	var out []string
	for name, set := range map[string]bool{"title": f.Title != nil, "kind": f.Kind != nil, "status": f.Status != nil, "year": f.Year != nil,
		"creators": f.Creators != nil, "genres": f.Genres != nil, "rating": f.Rating != nil, "progress": f.Progress != nil,
		"total": f.Total != nil, "started_on": f.StartedOn != nil, "finished_on": f.FinishedOn != nil, "review": f.Review != nil} {
		if set {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return List(ctx, q, userID, Filter{})
}
