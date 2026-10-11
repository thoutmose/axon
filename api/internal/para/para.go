// Package para implements the Areas and Resources of PARA (Projects, Areas,
// Resources, Archives) as entities, so anything can be filed under them with
// a link. Projects already exist; archives are the archived items of every
// kind.
package para

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// Kind is areas or resources: same shape, different table and wording.
type Kind struct {
	Type  string // entity type
	Table string // fixed table name, never user input
	Name  string // for messages
}

// The two kinds.
var (
	Areas     = Kind{Type: "area", Table: "areas", Name: "area"}
	Resources = Kind{Type: "resource", Table: "resources", Name: "resource"}
)

// InArea files anything under an area.
const InArea = "in_area"

func init() {
	for _, k := range []Kind{Areas, Resources} {
		graph.RegisterType(graph.Type{Name: k.Type, Table: k.Table, Export: k.export})
	}
	graph.RegisterRelation(graph.Relation{Name: InArea, Targets: []string{Areas.Type}})
}

var statuses = []string{"active", "archived"}

// Errors callers translate to HTTP responses.
var ErrStale = errors.New("stale version")

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Item is an area or a resource.
type Item struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         *string   `json:"url"`
	Status      string    `json:"status"`
	Linked      int       `json:"linked"` // items connected to it
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (k Kind) columns() string {
	return `x.id, e.title, x.description, x.url, x.status,
		(SELECT count(*) FROM links l WHERE l.source_id = x.id OR l.target_id = x.id),
		e.created_at, x.updated_at FROM ` + k.Table + ` x JOIN entities e ON e.id = x.id`
}

func scan(row pgx.Row) (Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.Title, &it.Description, &it.URL, &it.Status, &it.Linked, &it.CreatedAt, &it.UpdatedAt)
	return it, err
}

// Get returns one of the user's items.
func (k Kind) Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Item, error) {
	it, err := scan(q.QueryRow(ctx, `SELECT `+k.columns()+` WHERE x.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return it, graph.ErrNotFound
	}
	return it, err
}

// List returns the user's items with a status, by title.
func (k Kind) List(ctx context.Context, q db.Querier, userID uuid.UUID, status string) ([]Item, error) {
	rows, err := q.Query(ctx, `SELECT `+k.columns()+` WHERE e.user_id = $1 AND x.status = $2 ORDER BY lower(e.title), x.id LIMIT 500`, userID, status)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) { return scan(r) })
	if out == nil {
		out = []Item{}
	}
	return out, err
}

// Fields are the editable parts; nil means "not provided".
type Fields struct {
	Title, Description, URL, Status *string
}

func (k Kind) validate(f *Fields) error {
	if f.Title != nil {
		*f.Title = strings.TrimSpace(*f.Title)
		if *f.Title == "" || len([]rune(*f.Title)) > 500 {
			return ValidationError{fmt.Sprintf("An %s needs a title of 1 to 500 characters.", k.Name)}
		}
	}
	if f.Description != nil && len(*f.Description) > 20000 {
		return ValidationError{"The description is limited to 20,000 characters."}
	}
	if f.URL != nil && *f.URL != "" {
		u, err := url.Parse(*f.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(*f.URL) > 1000 {
			return ValidationError{"A link starts with http:// or https://."}
		}
	}
	if f.Status != nil && *f.Status != "active" && *f.Status != "archived" {
		return ValidationError{"Status is active or archived."}
	}
	return nil
}

func nullable(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// Create adds an item.
func (k Kind) Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f Fields) (Item, error) {
	if f.Title == nil {
		return Item{}, ValidationError{fmt.Sprintf("An %s needs a title.", k.Name)}
	}
	if err := k.validate(&f); err != nil {
		return Item{}, err
	}
	desc, status := "", "active"
	if f.Description != nil {
		desc = *f.Description
	}
	if f.Status != nil {
		status = *f.Status
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, k.Type, *f.Title, desc)
	if err != nil {
		return Item{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO `+k.Table+` (id, description, url, status) VALUES ($1, $2, $3, $4)`,
		id, desc, nullable(f.URL), status); err != nil {
		return Item{}, err
	}
	return k.Get(ctx, tx, userID, id)
}

// Update applies f if the item's version still equals version.
func (k Kind) Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, version time.Time, f Fields) (Item, error) {
	if err := k.validate(&f); err != nil {
		return Item{}, err
	}
	cur, err := k.Get(ctx, tx, userID, id)
	if err != nil {
		return Item{}, err
	}
	if !cur.UpdatedAt.Equal(version) {
		return Item{}, ErrStale
	}
	next := cur
	var changed []string
	if f.Title != nil && *f.Title != cur.Title {
		next.Title, changed = *f.Title, append(changed, "title")
	}
	if f.Description != nil && *f.Description != cur.Description {
		next.Description, changed = *f.Description, append(changed, "description")
	}
	if f.URL != nil {
		next.URL, changed = nullable(f.URL), append(changed, "url")
	}
	if f.Status != nil && *f.Status != cur.Status {
		next.Status, changed = *f.Status, append(changed, "status")
	}
	if len(changed) == 0 {
		return cur, nil
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, next.Title, next.Description, changed); err != nil {
		return Item{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE `+k.Table+` SET description = $2, url = $3, status = $4, updated_at = clock_timestamp() WHERE id = $1`,
		id, next.Description, next.URL, next.Status); err != nil {
		return Item{}, err
	}
	return k.Get(ctx, tx, userID, id)
}

// Delete removes an item and its links; what was filed under it stays.
func (k Kind) Delete(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	if _, err := k.Get(ctx, tx, userID, id); err != nil {
		return err
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

func (k Kind) export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+k.columns()+` WHERE e.user_id = $1 ORDER BY x.id`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) { return scan(r) })
}
