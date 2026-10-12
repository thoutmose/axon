// Package learning is the learning and memory engine around flashcards and
// notes (EP-52): memory palaces, free recall of whole notes, cards drafted
// from a note's own markup, practice tests, and the day's plan.
package learning

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// A memory palace (the method of loci, Moonwalking with Einstein) is a
// place the person knows well, walked in a fixed order. Each card placed at
// a locus is pictured there as a vivid image; walking the palace in the
// mind brings the cards back in order.

// PalaceType is the graph entity type of a palace.
const PalaceType = "palace"

func init() {
	graph.RegisterType(graph.Type{Name: PalaceType, Table: "memory_palaces", Export: exportPalaces})
}

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Placed is a card at a locus.
type Placed struct {
	ID       uuid.UUID `json:"id"`
	Locus    int       `json:"locus"`
	Deck     string    `json:"deck"`
	Front    string    `json:"front"`
	Back     string    `json:"back"`
	Mnemonic string    `json:"mnemonic"`
	HasImage bool      `json:"has_image"`
}

// Palace is a memory palace and, when read alone, the cards placed in it.
type Palace struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Loci        []string  `json:"loci"`
	Cards       []Placed  `json:"cards,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PalaceFields are the editable parts of a palace.
type PalaceFields struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Loci        *[]string `json:"loci"`
}

func (f *PalaceFields) validate() error {
	if f.Title != nil {
		*f.Title = strings.TrimSpace(*f.Title)
		if *f.Title == "" || len([]rune(*f.Title)) > 200 {
			return ValidationError{"A palace's name is 1 to 200 characters."}
		}
	}
	if f.Description != nil && len(*f.Description) > 5000 {
		return ValidationError{"The description is up to 5,000 characters."}
	}
	if f.Loci != nil {
		if len(*f.Loci) > 100 {
			return ValidationError{"A palace has up to 100 places."}
		}
		for i, l := range *f.Loci {
			l = strings.TrimSpace(l)
			if l == "" || len([]rune(l)) > 200 {
				return ValidationError{"Each place is 1 to 200 characters, like \"front door\" or \"the sofa\"."}
			}
			(*f.Loci)[i] = l
		}
	}
	return nil
}

const palaceColumns = `p.id, e.title, coalesce(e.search_text, ''), p.loci, e.created_at, p.updated_at`

func scanPalace(row pgx.Row) (Palace, error) {
	var p Palace
	err := row.Scan(&p.ID, &p.Title, &p.Description, &p.Loci, &p.CreatedAt, &p.UpdatedAt)
	if p.Loci == nil {
		p.Loci = []string{}
	}
	return p, err
}

// Palaces lists the person's palaces by name.
func Palaces(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Palace, error) {
	rows, err := q.Query(ctx, `SELECT `+palaceColumns+` FROM memory_palaces p JOIN entities e ON e.id = p.id
		WHERE p.user_id = $1 ORDER BY lower(e.title), p.id LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Palace, error) { return scanPalace(r) })
	if out == nil {
		out = []Palace{}
	}
	return out, err
}

// GetPalace returns a palace with its cards in walking order.
func GetPalace(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Palace, error) {
	p, err := scanPalace(q.QueryRow(ctx, `SELECT `+palaceColumns+` FROM memory_palaces p JOIN entities e ON e.id = p.id
		WHERE p.id = $1 AND p.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, graph.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	rows, err := q.Query(ctx, `SELECT c.id, c.locus, c.deck, c.front, c.back, c.mnemonic, c.image IS NOT NULL FROM cards c
		JOIN entities e ON e.id = c.id WHERE e.user_id = $1 AND c.palace_id = $2 ORDER BY c.locus, e.created_at`, userID, id)
	if err != nil {
		return p, err
	}
	p.Cards, err = pgx.CollectRows(rows, pgx.RowToStructByPos[Placed])
	if p.Cards == nil {
		p.Cards = []Placed{}
	}
	return p, err
}

// CreatePalace adds a palace.
func CreatePalace(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f PalaceFields) (Palace, error) {
	if f.Title == nil {
		return Palace{}, ValidationError{"A palace needs a name, like \"My flat\"."}
	}
	if err := f.validate(); err != nil {
		return Palace{}, err
	}
	desc, loci := "", []string{}
	if f.Description != nil {
		desc = *f.Description
	}
	if f.Loci != nil {
		loci = *f.Loci
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, PalaceType, *f.Title, desc)
	if err != nil {
		return Palace{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO memory_palaces (id, user_id, loci) VALUES ($1, $2, $3)`, id, userID, loci); err != nil {
		return Palace{}, err
	}
	return GetPalace(ctx, tx, userID, id)
}

// UpdatePalace renames a palace or changes its places. Cards at places that
// no longer exist leave the palace.
func UpdatePalace(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, f PalaceFields) (Palace, error) {
	if err := f.validate(); err != nil {
		return Palace{}, err
	}
	p, err := GetPalace(ctx, tx, userID, id)
	if err != nil {
		return p, err
	}
	var changed []string
	if f.Title != nil && *f.Title != p.Title {
		p.Title, changed = *f.Title, append(changed, "title")
	}
	if f.Description != nil && *f.Description != p.Description {
		p.Description, changed = *f.Description, append(changed, "description")
	}
	if len(changed) > 0 {
		if err := graph.UpdateEntity(ctx, tx, userID, id, actor, p.Title, p.Description, changed); err != nil {
			return p, err
		}
	}
	if f.Loci != nil {
		if _, err := tx.Exec(ctx, `UPDATE memory_palaces SET loci = $2, updated_at = clock_timestamp() WHERE id = $1`, id, *f.Loci); err != nil {
			return p, err
		}
		if _, err := tx.Exec(ctx, `UPDATE cards SET palace_id = NULL, locus = NULL WHERE palace_id = $1 AND locus >= $2`, id, len(*f.Loci)); err != nil {
			return p, err
		}
	}
	return GetPalace(ctx, tx, userID, id)
}

func exportPalaces(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return Palaces(ctx, q, userID)
}
