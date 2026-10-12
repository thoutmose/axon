package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Trash (EP-66). Before an entity is deleted, every row the deletion
// removes is saved: the entity, its domain row, the rows that cascade from
// it (a habit's check-ins, a card's reviews, a goal's milestones and their
// own entities), and its links. Putting it back inserts the same rows. Rows
// that only derive from others (note embeddings) are left to be rebuilt.

// Kept is how long the trash keeps a deletion.
const Kept = 30 * 24 * time.Hour

// derived rows are rebuilt, not restored.
var derived = map[string]bool{"note_embeddings": true}

type fk struct{ child, col string }

var (
	schemaOnce sync.Once
	schemaErr  error
	cascades   map[string][]fk     // parent table → children that cascade from it by "id"
	entityRow  map[string]bool     // tables whose id is an entity
	columns    map[string][]string // writable columns (no generated ones)
)

func loadSchema(ctx context.Context, q db.Querier) error {
	schemaOnce.Do(func() {
		cascades, entityRow, columns = map[string][]fk{}, map[string]bool{}, map[string][]string{}
		rows, err := q.Query(ctx, `SELECT c.conrelid::regclass::text, c.confrelid::regclass::text, a.attname, c.confdeltype
			FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
			JOIN pg_attribute pa ON pa.attrelid = c.confrelid AND pa.attnum = c.confkey[1]
			WHERE c.contype = 'f' AND array_length(c.conkey, 1) = 1 AND pa.attname = 'id'`)
		if err != nil {
			schemaErr = err
			return
		}
		for rows.Next() {
			var child, parent, col, del string
			if schemaErr = rows.Scan(&child, &parent, &col, &del); schemaErr != nil {
				rows.Close()
				return
			}
			if del != "c" || child == "links" || child == "events" {
				continue
			}
			if parent == "entities" && col == "id" {
				entityRow[child] = true
				continue
			}
			cascades[parent] = append(cascades[parent], fk{child, col})
		}
		if schemaErr = rows.Err(); schemaErr != nil {
			return
		}
		cols, err := q.Query(ctx, `SELECT c.relname, a.attname FROM pg_attribute a JOIN pg_class c ON c.oid = a.attrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = current_schema() AND c.relkind = 'r' AND a.attnum > 0 AND NOT a.attisdropped AND a.attgenerated = ''
			ORDER BY c.relname, a.attnum`)
		if err != nil {
			schemaErr = err
			return
		}
		for cols.Next() {
			var t, c string
			if schemaErr = cols.Scan(&t, &c); schemaErr != nil {
				cols.Close()
				return
			}
			columns[t] = append(columns[t], c)
		}
		schemaErr = cols.Err()
	})
	return schemaErr
}

type saved struct {
	Table string          `json:"table"`
	Row   json.RawMessage `json:"row"`
}

func rowsOf(ctx context.Context, q db.Querier, table, col string, ids []uuid.UUID) ([]json.RawMessage, []uuid.UUID, error) {
	t := pgx.Identifier{table}.Sanitize()
	c := pgx.Identifier{col}.Sanitize()
	rows, err := q.Query(ctx, fmt.Sprintf(`SELECT row_to_json(x), %s FROM %s x WHERE x.%s = ANY($1)`,
		map[bool]string{true: "x.id", false: "NULL::uuid"}[hasID(table)], t, c), ids)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []json.RawMessage
	var keys []uuid.UUID
	for rows.Next() {
		var r json.RawMessage
		var id *uuid.UUID
		if err := rows.Scan(&r, &id); err != nil {
			return nil, nil, err
		}
		out = append(out, r)
		if id != nil {
			keys = append(keys, *id)
		}
	}
	return out, keys, rows.Err()
}

func hasID(table string) bool {
	for _, c := range columns[table] {
		if c == "id" {
			return true
		}
	}
	return false
}

// snapshot saves what deleting id will remove, before it is removed.
func snapshot(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, typ, title string) error {
	if err := loadSchema(ctx, tx); err != nil {
		return err
	}
	mu.RLock()
	t, known := types[typ]
	mu.RUnlock()
	if !known {
		return nil
	}
	var out []saved
	var entities []uuid.UUID
	add := func(table string, rows []json.RawMessage) {
		for _, r := range rows {
			out = append(out, saved{table, r})
		}
	}
	// The entity itself, then its domain row.
	er, _, err := rowsOf(ctx, tx, "entities", "id", []uuid.UUID{id})
	if err != nil {
		return err
	}
	add("entities", er)
	entities = append(entities, id)
	dr, _, err := rowsOf(ctx, tx, t.Table, "id", []uuid.UUID{id})
	if err != nil {
		return err
	}
	add(t.Table, dr)
	// What cascades from it, two levels down.
	level := []struct {
		table string
		ids   []uuid.UUID
	}{{t.Table, []uuid.UUID{id}}}
	for depth := 0; depth < 2 && len(level) > 0; depth++ {
		var next []struct {
			table string
			ids   []uuid.UUID
		}
		for _, p := range level {
			for _, f := range cascades[p.table] {
				if derived[f.child] {
					continue
				}
				rows, keys, err := rowsOf(ctx, tx, f.child, f.col, p.ids)
				if err != nil {
					return err
				}
				if len(rows) == 0 {
					continue
				}
				if entityRow[f.child] && len(keys) > 0 {
					// Children that are entities themselves: their entity rows first.
					cr, _, err := rowsOf(ctx, tx, "entities", "id", keys)
					if err != nil {
						return err
					}
					add("entities", cr)
					entities = append(entities, keys...)
				}
				add(f.child, rows)
				if len(keys) > 0 {
					next = append(next, struct {
						table string
						ids   []uuid.UUID
					}{f.child, keys})
				}
			}
		}
		level = next
	}
	// Links last, from and to every entity saved.
	lr, err := tx.Query(ctx, `SELECT row_to_json(l) FROM links l WHERE l.user_id = $1 AND (l.source_id = ANY($2) OR l.target_id = ANY($2))`, userID, entities)
	if err != nil {
		return err
	}
	links, err := pgx.CollectRows(lr, pgx.RowTo[json.RawMessage])
	if err != nil {
		return err
	}
	add("links", links)
	body, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO trash (id, user_id, batch, entity_id, entity_type, title, rows) VALUES ($1, $2, txid_current(), $3, $4, $5, $6)`,
		uuid.Must(uuid.NewV7()), userID, id, typ, title, body)
	return err
}

// ErrRestore says why something could not be put back, in words a person reads.
type ErrRestore struct{ Msg string }

func (e ErrRestore) Error() string { return e.Msg }

// OnRestore lets a domain package tidy up after its rows come back (a note
// leaves the sync's tombstones, for instance).
var restored = map[string]func(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID) error{}

// OnRestore registers fn for entities of typ.
func OnRestore(typ string, fn func(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID) error) {
	mu.Lock()
	defer mu.Unlock()
	restored[typ] = fn
}

// Restore puts back a deletion and everything deleted with it, newest last
// deleted first, so parents return before their children.
func Restore(ctx context.Context, tx pgx.Tx, userID, trashID uuid.UUID) (uuid.UUID, string, error) {
	if err := loadSchema(ctx, tx); err != nil {
		return uuid.Nil, "", err
	}
	var batch int64
	if err := tx.QueryRow(ctx, `SELECT batch FROM trash WHERE id = $1 AND user_id = $2`, trashID, userID).Scan(&batch); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, "", ErrNotFound
		}
		return uuid.Nil, "", err
	}
	rows, err := tx.Query(ctx, `SELECT entity_id, entity_type, rows FROM trash WHERE user_id = $1 AND batch = $2 ORDER BY seq DESC`, userID, batch)
	if err != nil {
		return uuid.Nil, "", err
	}
	type entry struct {
		id   uuid.UUID
		typ  string
		rows []saved
	}
	var entries []entry
	for rows.Next() {
		var e entry
		var raw []byte
		if err := rows.Scan(&e.id, &e.typ, &raw); err != nil {
			rows.Close()
			return uuid.Nil, "", err
		}
		if err := json.Unmarshal(raw, &e.rows); err != nil {
			rows.Close()
			return uuid.Nil, "", err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return uuid.Nil, "", err
	}
	for _, e := range entries {
		for _, r := range e.rows {
			cols := columns[r.Table]
			if len(cols) == 0 {
				continue
			}
			list := make([]string, len(cols))
			for i, c := range cols {
				list[i] = pgx.Identifier{c}.Sanitize()
			}
			t := pgx.Identifier{r.Table}.Sanitize()
			sql := fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM json_populate_record(NULL::%s, $1)`, t, strings.Join(list, ", "), strings.Join(list, ", "), t)
			sp, err := tx.Begin(ctx)
			if err != nil {
				return uuid.Nil, "", err
			}
			if _, err := sp.Exec(ctx, sql, string(r.Row)); err != nil {
				_ = sp.Rollback(ctx)
				if r.Table == "links" {
					continue // the other end is gone: the link cannot come back
				}
				if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
					return uuid.Nil, "", ErrRestore{"Something with the same name or path exists now; rename or delete it, then restore."}
				}
				return uuid.Nil, "", err
			}
			if err := sp.Commit(ctx); err != nil {
				return uuid.Nil, "", err
			}
		}
		// It changed now, for whoever syncs or mirrors it.
		if _, err := tx.Exec(ctx, `UPDATE entities SET updated_at = now() WHERE id = $1`, e.id); err != nil {
			return uuid.Nil, "", err
		}
		mu.RLock()
		fn := restored[e.typ]
		mu.RUnlock()
		if fn != nil {
			if err := fn(ctx, tx, userID, e.id); err != nil {
				return uuid.Nil, "", err
			}
		}
		if err := recordEvent(ctx, tx, userID, UserActor, e.typ, e.id, "created", []string{"restored"}); err != nil {
			return uuid.Nil, "", err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM trash WHERE user_id = $1 AND batch = $2`, userID, batch); err != nil {
		return uuid.Nil, "", err
	}
	last := entries[len(entries)-1]
	return last.id, last.typ, nil
}

// Trashed is one deletion, as the trash lists it.
type Trashed struct {
	ID        uuid.UUID `json:"id"`
	EntityID  uuid.UUID `json:"entity_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	With      int       `json:"with"` // other items deleted with it
	DeletedAt time.Time `json:"deleted_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TrashList lists deletions, newest first, one per batch: the item deleted
// last in it (the one the person chose), with how many came along.
// Deletions older than 30 days are left out (and purged by PurgeTrash).
func TrashList(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) ([]Trashed, error) {
	rows, err := q.Query(ctx, `SELECT DISTINCT ON (batch) id, entity_id, entity_type, title,
		(SELECT count(*) - 1 FROM trash t2 WHERE t2.user_id = t.user_id AND t2.batch = t.batch), deleted_at
		FROM trash t WHERE user_id = $1 AND deleted_at >= $2 ORDER BY batch, seq DESC`, userID, now.Add(-Kept))
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Trashed, error) {
		var t Trashed
		err := r.Scan(&t.ID, &t.EntityID, &t.Type, &t.Title, &t.With, &t.DeletedAt)
		t.ExpiresAt = t.DeletedAt.Add(Kept)
		return t, err
	})
	if err != nil {
		return nil, err
	}
	// Newest first.
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	if items == nil {
		items = []Trashed{}
	}
	return items, nil
}

// PurgeTrash forgets deletions older than 30 days.
func PurgeTrash(ctx context.Context, q db.Querier, now time.Time) error {
	_, err := q.Exec(ctx, `DELETE FROM trash WHERE deleted_at < $1`, now.Add(-Kept))
	return err
}

// TrashExport lists what the trash holds, for a personal data export.
func TrashExport(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return TrashList(ctx, q, userID, time.Now())
}
