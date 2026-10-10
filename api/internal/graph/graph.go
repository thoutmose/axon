// Package graph implements the entity registry, universal links, the event
// log, and search (ADR 0006). Domain packages register their entity type and
// create rows through CreateEntity so links, search, and export work for them.
package graph

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/thoutmose/axon/api/internal/db"
)

// Errors callers translate to HTTP responses.
var (
	ErrNotFound         = errors.New("not found")
	ErrRelationRejected = errors.New("relation not allowed between these types")
	ErrDuplicate        = errors.New("already exists")
)

// Type describes one kind of entity. Table is the domain table that uses
// entities.id as its primary key ("" for types with no extra columns).
type Type struct {
	Name  string
	Table string
	// Export returns the domain rows for the user's export. Required.
	Export func(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error)
}

// Relation is a typed edge. Empty Sources/Targets allow any type.
type Relation struct {
	Name    string
	Sources []string
	Targets []string
}

// RelatesTo is always available between two distinct entities.
const RelatesTo = "relates_to"

var (
	mu        sync.RWMutex
	types     = map[string]Type{}
	relations = map[string]Relation{RelatesTo: {Name: RelatesTo}}
)

// RegisterType adds an entity type. Call it from an init function.
func RegisterType(t Type) {
	mu.Lock()
	defer mu.Unlock()
	if t.Export == nil {
		panic(fmt.Sprintf("graph: type %q has no Export", t.Name))
	}
	types[t.Name] = t
}

// RegisterRelation adds a typed relation.
func RegisterRelation(r Relation) {
	mu.Lock()
	defer mu.Unlock()
	relations[r.Name] = r
}

// Types returns the registered types, sorted by name.
func Types() []Type {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Type, 0, len(types))
	for _, t := range types {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b Type) int { return compare(a.Name, b.Name) })
	return out
}

func compare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func allowed(relation, source, target string) bool {
	mu.RLock()
	defer mu.RUnlock()
	r, ok := relations[relation]
	if !ok {
		return false
	}
	return (len(r.Sources) == 0 || slices.Contains(r.Sources, source)) &&
		(len(r.Targets) == 0 || slices.Contains(r.Targets, target))
}

// Actor names who made a change: "user", "agent:<name>", or "system".
type Actor string

// UserActor is the account owner acting directly.
const UserActor Actor = "user"

// Entity is the shared part of every linkable row.
type Entity struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateEntity inserts the entities row and a "created" event. The caller
// inserts its domain row with the returned ID in the same transaction.
func CreateEntity(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor Actor, typ, title, searchText string) (uuid.UUID, error) {
	mu.RLock()
	_, known := types[typ]
	mu.RUnlock()
	if !known {
		return uuid.Nil, fmt.Errorf("graph: unregistered entity type %q", typ)
	}
	id := uuid.Must(uuid.NewV7())
	_, err := tx.Exec(ctx,
		`INSERT INTO entities (id, user_id, type, title, search_text) VALUES ($1, $2, $3, $4, $5)`,
		id, userID, typ, title, searchText)
	if err != nil {
		return uuid.Nil, err
	}
	return id, recordEvent(ctx, tx, userID, actor, typ, id, "created", nil)
}

// UpdateEntity refreshes title and search text and records which fields changed.
func UpdateEntity(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor Actor, title, searchText string, fields []string) error {
	var typ string
	err := tx.QueryRow(ctx,
		`UPDATE entities SET title = $3, search_text = $4, updated_at = now()
		 WHERE id = $1 AND user_id = $2 RETURNING type`, id, userID, title, searchText).Scan(&typ)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return recordEvent(ctx, tx, userID, actor, typ, id, "updated", fields)
}

// DeleteEntity removes the entity; its domain row and links go with it.
func DeleteEntity(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor Actor) error {
	var typ string
	err := tx.QueryRow(ctx, `DELETE FROM entities WHERE id = $1 AND user_id = $2 RETURNING type`, id, userID).Scan(&typ)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return recordEvent(ctx, tx, userID, actor, typ, id, "deleted", nil)
}

// GetEntity returns the shared fields of one of the user's entities.
func GetEntity(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Entity, error) {
	var e Entity
	err := q.QueryRow(ctx,
		`SELECT id, type, title, created_at, updated_at FROM entities WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&e.ID, &e.Type, &e.Title, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

// --- Links ---

// Link is a typed edge as seen from one of its ends.
type Link struct {
	ID        uuid.UUID `json:"id"`
	Relation  string    `json:"relation"`
	Other     Entity    `json:"entity"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateLink connects two of the user's entities if the relation allows their types.
func CreateLink(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor Actor, sourceID uuid.UUID, relation string, targetID uuid.UUID) (uuid.UUID, error) {
	if sourceID == targetID {
		return uuid.Nil, ErrRelationRejected
	}
	src, err := GetEntity(ctx, tx, userID, sourceID)
	if err != nil {
		return uuid.Nil, err
	}
	dst, err := GetEntity(ctx, tx, userID, targetID)
	if err != nil {
		return uuid.Nil, err
	}
	if !allowed(relation, src.Type, dst.Type) {
		return uuid.Nil, ErrRelationRejected
	}
	id := uuid.Must(uuid.NewV7())
	_, err = tx.Exec(ctx,
		`INSERT INTO links (id, user_id, source_id, relation, target_id) VALUES ($1, $2, $3, $4, $5)`,
		id, userID, sourceID, relation, targetID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return uuid.Nil, ErrDuplicate
	}
	if err != nil {
		return uuid.Nil, err
	}
	if err := recordEvent(ctx, tx, userID, actor, src.Type, sourceID, "linked", []string{relation}); err != nil {
		return uuid.Nil, err
	}
	return id, recordEvent(ctx, tx, userID, actor, dst.Type, targetID, "linked", []string{relation})
}

// DeleteLink removes one of the user's links.
func DeleteLink(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor Actor, linkID uuid.UUID) error {
	var src, dst uuid.UUID
	var relation string
	err := tx.QueryRow(ctx,
		`DELETE FROM links WHERE id = $1 AND user_id = $2 RETURNING source_id, relation, target_id`, linkID, userID).
		Scan(&src, &relation, &dst)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	for _, id := range []uuid.UUID{src, dst} {
		e, err := GetEntity(ctx, tx, userID, id)
		if err != nil {
			return err
		}
		if err := recordEvent(ctx, tx, userID, actor, e.Type, id, "unlinked", []string{relation}); err != nil {
			return err
		}
	}
	return nil
}

// Page bounds list queries. After is the last ID of the previous page (IDs are
// UUIDv7, so ID order is creation order).
type Page struct {
	After *uuid.UUID
	Limit int
}

// MaxPage caps every list.
const MaxPage = 100

func (p Page) limit() int {
	if p.Limit <= 0 || p.Limit > MaxPage {
		return MaxPage
	}
	return p.Limit
}

// Outgoing lists links from the entity; Incoming lists backlinks to it.
func Outgoing(ctx context.Context, q db.Querier, userID, id uuid.UUID, p Page) ([]Link, error) {
	return listLinks(ctx, q, userID, id, p, "source_id", "target_id")
}

// Incoming lists links that point at the entity (backlinks).
func Incoming(ctx context.Context, q db.Querier, userID, id uuid.UUID, p Page) ([]Link, error) {
	return listLinks(ctx, q, userID, id, p, "target_id", "source_id")
}

func listLinks(ctx context.Context, q db.Querier, userID, id uuid.UUID, p Page, self, other string) ([]Link, error) {
	// self and other are fixed column names chosen above, never user input.
	rows, err := q.Query(ctx, `
		SELECT l.id, l.relation, l.created_at, e.id, e.type, e.title, e.created_at, e.updated_at
		FROM links l JOIN entities e ON e.id = l.`+other+`
		WHERE l.user_id = $1 AND l.`+self+` = $2 AND ($3::uuid IS NULL OR l.id > $3)
		ORDER BY l.id LIMIT $4`, userID, id, p.After, p.limit())
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Link, error) {
		var l Link
		err := row.Scan(&l.ID, &l.Relation, &l.CreatedAt, &l.Other.ID, &l.Other.Type, &l.Other.Title, &l.Other.CreatedAt, &l.Other.UpdatedAt)
		return l, err
	})
}

// --- Events ---

// Event is one entry in an entity's timeline.
type Event struct {
	ID         uuid.UUID `json:"id"`
	Actor      string    `json:"actor"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Kind       string    `json:"kind"`
	Fields     []string  `json:"fields"`
	CreatedAt  time.Time `json:"created_at"`
}

func recordEvent(ctx context.Context, q db.Querier, userID uuid.UUID, actor Actor, typ string, entityID uuid.UUID, kind string, fields []string) error {
	if fields == nil {
		fields = []string{}
	}
	_, err := q.Exec(ctx,
		`INSERT INTO events (id, user_id, actor, entity_type, entity_id, kind, fields) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.Must(uuid.NewV7()), userID, string(actor), typ, entityID, kind, fields)
	return err
}

// Timeline lists an entity's events, newest first. Before is the oldest ID already seen.
func Timeline(ctx context.Context, q db.Querier, userID, entityID uuid.UUID, before *uuid.UUID, limit int) ([]Event, error) {
	rows, err := q.Query(ctx, `
		SELECT id, actor, entity_type, entity_id, kind, fields, created_at FROM events
		WHERE user_id = $1 AND entity_id = $2 AND ($3::uuid IS NULL OR id < $3)
		ORDER BY id DESC LIMIT $4`, userID, entityID, before, Page{Limit: limit}.limit())
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Event])
}

// --- Search ---

// Hit is one search result.
type Hit struct {
	Entity
	Rank float32 `json:"rank"`
}

// Search finds the user's entities matching a web-style query ("quoted
// phrases", -exclusions, or). Offset pagination is capped: search is for
// finding, not for paging through everything.
func Search(ctx context.Context, q db.Querier, userID uuid.UUID, query string, offset, limit int) ([]Hit, error) {
	if offset < 0 || offset > 1000 {
		offset = 0
	}
	rows, err := q.Query(ctx, `
		SELECT id, type, title, created_at, updated_at, ts_rank(search, query) AS rank
		FROM entities, websearch_to_tsquery('simple', $2) AS query
		WHERE user_id = $1 AND search @@ query
		ORDER BY rank DESC, id DESC OFFSET $3 LIMIT $4`, userID, query, offset, Page{Limit: limit}.limit())
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Hit, error) {
		var h Hit
		err := row.Scan(&h.ID, &h.Type, &h.Title, &h.CreatedAt, &h.UpdatedAt, &h.Rank)
		return h, err
	})
}

// --- Export and integrity ---

// Export returns the shared graph data plus every registered type's rows.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	out := map[string]any{}
	rows, err := q.Query(ctx, `SELECT id, type, title, created_at, updated_at FROM entities WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	if out["entities"], err = pgx.CollectRows(rows, pgx.RowToStructByPos[Entity]); err != nil {
		return nil, err
	}
	type link struct {
		ID        uuid.UUID `json:"id"`
		SourceID  uuid.UUID `json:"source_id"`
		Relation  string    `json:"relation"`
		TargetID  uuid.UUID `json:"target_id"`
		CreatedAt time.Time `json:"created_at"`
	}
	rows, err = q.Query(ctx, `SELECT id, source_id, relation, target_id, created_at FROM links WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	if out["links"], err = pgx.CollectRows(rows, pgx.RowToStructByPos[link]); err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT id, actor, entity_type, entity_id, kind, fields, created_at FROM events WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	if out["events"], err = pgx.CollectRows(rows, pgx.RowToStructByPos[Event]); err != nil {
		return nil, err
	}
	for _, t := range Types() {
		section, err := t.Export(ctx, q, userID)
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", t.Name, err)
		}
		out[t.Name] = section
	}
	return out, nil
}

// Problem is one integrity violation.
type Problem struct {
	Check string `json:"check"`
	Count int    `json:"count"`
}

// CheckIntegrity finds rows the schema cannot rule out on its own: entities
// with no domain row, domain rows typed differently from their entity, and
// entity types nobody registered. Links cannot dangle (foreign keys).
func CheckIntegrity(ctx context.Context, q db.Querier) ([]Problem, error) {
	var problems []Problem
	add := func(check, sql string, args ...any) error {
		var n int
		if err := q.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			return fmt.Errorf("%s: %w", check, err)
		}
		if n > 0 {
			problems = append(problems, Problem{check, n})
		}
		return nil
	}
	names := []string{}
	for _, t := range Types() {
		names = append(names, t.Name)
		if t.Table == "" {
			continue
		}
		// Table names come from code registrations, never from users.
		ident := pgx.Identifier{t.Table}.Sanitize()
		if err := add("entity without "+t.Table+" row",
			`SELECT count(*) FROM entities e WHERE e.type = $1 AND NOT EXISTS (SELECT 1 FROM `+ident+` d WHERE d.id = e.id)`, t.Name); err != nil {
			return nil, err
		}
		if err := add(t.Table+" row typed differently",
			`SELECT count(*) FROM `+ident+` d JOIN entities e ON e.id = d.id WHERE e.type <> $1`, t.Name); err != nil {
			return nil, err
		}
	}
	if err := add("entity of unregistered type", `SELECT count(*) FROM entities WHERE NOT (type = ANY($1))`, names); err != nil {
		return nil, err
	}
	return problems, nil
}
