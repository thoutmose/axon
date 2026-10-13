// Package mirror keeps Axon's items and Markdown notes in step (EP-48).
// Every task, goal, milestone, project, area, resource, habit, and film,
// series, or book has a note under the person's vault folder ("Axon" by
// default); vault sync carries those notes to their Markdown vault.
//
// Both ways: changing an item rewrites its note; editing a note's title,
// front matter, or editable sections updates the item; a new note with a
// known type under the folder creates an item; deleting the note deletes
// the item. Moving a note into or out of Archive/ archives or restores it.
// Health logs and journal days are written one way and only when the
// person turned them on.
//
// A single background worker reads the event log (and a small queue for
// habit check-ins and health logs) every few seconds, so no request ever
// waits for the vault.
package mirror

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/health"
	"github.com/thoutmose/axon/api/internal/journal"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/privacy"
	"github.com/thoutmose/axon/api/internal/settings"
)

func enqueue(ctx context.Context, q db.Querier, user uuid.UUID, key string) error {
	_, err := q.Exec(ctx, `INSERT INTO vault_mirror_queue (user_id, key) VALUES ($1, $2)
		ON CONFLICT (user_id, key) DO UPDATE SET queued_at = now()`, user, key)
	return err
}

func init() {
	habits.OnCheckin = func(ctx context.Context, q db.Querier, user, habit uuid.UUID) {
		_ = enqueue(ctx, q, user, habit.String())
	}
	health.OnChange = enqueue
	journal.OnChange = enqueue
}

// Worker runs the mirror.
type Worker struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// New returns a worker; call Run in a goroutine, or Step in tests.
func New(pool *pgxpool.Pool, log *slog.Logger) *Worker {
	return &Worker{db: pool, log: log, now: time.Now}
}

// Run steps every few seconds until ctx ends.
func (w *Worker) Run(ctx context.Context) {
	for {
		if err := w.Step(ctx); err != nil && ctx.Err() == nil {
			w.log.Error("vault mirror step failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

type user struct {
	folder string
	health bool
}

// Step brings everything in step once: new or changed settings, the event
// log since the last step, and the queue.
func (w *Worker) Step(ctx context.Context) error {
	users, err := w.syncUsers(ctx)
	if err != nil {
		return err
	}
	if err := w.events(ctx, users); err != nil {
		return err
	}
	return w.queue(ctx, users)
}

func (w *Worker) syncUsers(ctx context.Context) (map[uuid.UUID]user, error) {
	rows, err := w.db.Query(ctx, `SELECT u.id, m.folder, m.health, m.user_id IS NOT NULL FROM users u LEFT JOIN vault_mirror_users m ON m.user_id = u.id`)
	if err != nil {
		return nil, err
	}
	type row struct {
		id     uuid.UUID
		folder *string
		health *bool
		has    bool
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.folder, &r.health, &r.has); err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]user{}
	for _, r := range all {
		doc, err := settings.Get(ctx, w.db, r.id)
		if err != nil {
			return nil, err
		}
		if !doc.Vault.Mirror {
			if r.has {
				if _, err := w.db.Exec(ctx, `DELETE FROM vault_mirror_users WHERE user_id = $1`, r.id); err != nil {
					return nil, err
				}
			}
			continue
		}
		consent, err := privacy.Granted(ctx, w.db, r.id, health.Consent)
		if err != nil {
			return nil, err
		}
		u := user{folder: doc.Vault.Folder, health: doc.Vault.Health && consent}
		moved := !r.has || *r.folder != u.folder
		if moved {
			if err := w.syncAll(ctx, r.id, u); err != nil {
				return nil, err
			}
		}
		switch {
		case u.health && (moved || !*r.health):
			if err := w.syncHealth(ctx, r.id, u); err != nil {
				return nil, err
			}
		case !u.health && r.has && *r.health:
			if err := w.dropHealth(ctx, r.id); err != nil {
				return nil, err
			}
		}
		if moved || *r.health != u.health {
			if _, err := w.db.Exec(ctx, `INSERT INTO vault_mirror_users (user_id, folder, health) VALUES ($1, $2, $3)
				ON CONFLICT (user_id) DO UPDATE SET folder = $2, health = $3, synced_at = now()`, r.id, u.folder, u.health); err != nil {
				return nil, err
			}
		}
		out[r.id] = u
	}
	return out, nil
}

// syncAll writes every item's note (the first time, or after the folder changed).
func (w *Worker) syncAll(ctx context.Context, id uuid.UUID, u user) error {
	types := make([]string, 0, len(kinds))
	for t := range kinds {
		types = append(types, t)
	}
	rows, err := w.db.Query(ctx, `SELECT id, type FROM entities WHERE user_id = $1 AND type = ANY($2) ORDER BY created_at, id`, id, types)
	if err != nil {
		return err
	}
	type item struct {
		id  uuid.UUID
		typ string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.typ); err != nil {
			rows.Close()
			return err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, it := range items {
		if err := w.refresh(ctx, id, u, it.typ, it.id); err != nil {
			w.log.Warn("vault mirror: cannot write an item's note", "type", it.typ, "err", err)
		}
	}
	return nil
}

func (w *Worker) syncHealth(ctx context.Context, id uuid.UUID, u user) error {
	rows, err := w.db.Query(ctx, `SELECT 'health:food:' || to_char(eaten_on, 'YYYY-MM-DD') FROM food_entries WHERE user_id = $1
		UNION SELECT 'health:workout:' || id FROM workouts WHERE user_id = $1
		UNION SELECT 'health:journal:' || to_char(day, 'YYYY-MM-DD') FROM journal_entries WHERE user_id = $1`, id)
	if err != nil {
		return err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := w.refreshHealth(ctx, id, u, k); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) dropHealth(ctx context.Context, id uuid.UUID) error {
	rows, err := w.db.Query(ctx, `SELECT key FROM vault_mirror WHERE user_id = $1 AND key LIKE 'health:%'`, id)
	if err != nil {
		return err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := w.drop(ctx, id, k); err != nil {
			return err
		}
	}
	return nil
}

type mapping struct {
	noteID uuid.UUID
	kind   string
	hash   string
	path   string
	extra  string
}

func lookup(ctx context.Context, q db.Querier, where string, args ...any) (mapping, bool, error) {
	var m mapping
	err := q.QueryRow(ctx, `SELECT note_id, kind, hash, path, extra FROM vault_mirror WHERE `+where, args...).Scan(&m.noteID, &m.kind, &m.hash, &m.path, &m.extra)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, false, nil
	}
	return m, err == nil, err
}

// refresh renders an item and writes its note, or removes the note when
// the item is gone.
func (w *Worker) refresh(ctx context.Context, id uuid.UUID, u user, entity string, item uuid.UUID) error {
	k, ok := kinds[entity]
	if !ok {
		return nil
	}
	r, err := k.render(ctx, w.db, id, item, w.now())
	if errors.Is(err, graph.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
		return w.drop(ctx, id, item.String())
	}
	if err != nil {
		return err
	}
	m, _, err := lookup(ctx, w.db, `user_id = $1 AND key = $2`, id, item.String())
	if err != nil {
		return err
	}
	if !k.ownNotes {
		r.page.sections = append(r.page.sections, section{heading: "Notes", body: m.extra})
	}
	return w.write(ctx, id, u, item.String(), entity, r)
}

func (w *Worker) refreshHealth(ctx context.Context, id uuid.UUID, u user, key string) error {
	var r rendered
	var ok bool
	var err error
	switch {
	case strings.HasPrefix(key, "health:food:"):
		r, ok, err = renderFoodDay(ctx, w.db, id, strings.TrimPrefix(key, "health:food:"))
	case strings.HasPrefix(key, "health:workout:"):
		wid, perr := uuid.Parse(strings.TrimPrefix(key, "health:workout:"))
		if perr != nil {
			return nil
		}
		r, ok, err = renderWorkout(ctx, w.db, id, wid)
	case strings.HasPrefix(key, "health:journal:"):
		r, ok, err = renderJournalDay(ctx, w.db, id, strings.TrimPrefix(key, "health:journal:"))
	default:
		return nil
	}
	if err != nil {
		return err
	}
	if !ok {
		return w.drop(ctx, id, key)
	}
	return w.write(ctx, id, u, key, "health", r)
}

// write stores the note for key: created the first time, then updated (and
// moved when its title or folder changed) only when it differs.
func (w *Worker) write(ctx context.Context, id uuid.UUID, u user, key, kindName string, r rendered) error {
	folder := r.folder
	if r.archived {
		folder = "Archive/" + folder
	}
	dir := u.folder + "/" + folder
	content := r.page.markdown()
	return pgx.BeginFunc(ctx, w.db, func(tx pgx.Tx) error {
		m, found, err := lookup(ctx, tx, `user_id = $1 AND key = $2`, id, key)
		if err != nil {
			return err
		}
		var cur *notes.Note
		if found {
			n, err := notes.Get(ctx, tx, id, m.noteID)
			switch {
			case err == nil:
				cur = &n
			case !errors.Is(err, graph.ErrNotFound):
				return err
			}
		}
		path, err := pathFor(ctx, tx, id, dir, fileName(r.page.title), cur, key)
		if err != nil {
			return err
		}
		var n notes.Note
		switch {
		case cur != nil && cur.Path == path && *cur.Content == content:
			n = *cur
		case cur != nil:
			if n, err = notes.Update(ctx, tx, id, cur.ID, Actor, cur.UpdatedAt, path, content); err != nil {
				return err
			}
		default:
			if n, err = notes.Create(ctx, tx, id, Actor, path, content); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO vault_mirror (user_id, key, kind, note_id, hash, path) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (user_id, key) DO UPDATE SET note_id = $4, hash = $5, kind = $3, path = $6`, id, key, kindName, n.ID, notes.Hash(content), n.Path)
		return err
	})
}

// pathFor keeps a note's file name stable and unique: the title as the
// file name, with a short id when another note already has that name.
func pathFor(ctx context.Context, q db.Querier, id uuid.UUID, dir, name string, cur *notes.Note, key string) (string, error) {
	desired := dir + "/" + name + ".md"
	other, err := notes.GetByPath(ctx, q, id, desired)
	switch {
	case errors.Is(err, graph.ErrNotFound):
		return desired, nil
	case err != nil:
		return "", err
	case cur != nil && other.ID == cur.ID:
		return desired, nil
	}
	short := key
	if len(short) > 6 {
		short = short[len(short)-6:]
	}
	return dir + "/" + name + " (" + short + ").md", nil
}

// drop removes the note for key and forgets it.
func (w *Worker) drop(ctx context.Context, id uuid.UUID, key string) error {
	return pgx.BeginFunc(ctx, w.db, func(tx pgx.Tx) error {
		m, found, err := lookup(ctx, tx, `user_id = $1 AND key = $2`, id, key)
		if err != nil || !found {
			return err
		}
		if err := notes.Delete(ctx, tx, id, m.noteID, Actor); err != nil && !errors.Is(err, graph.ErrNotFound) {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM vault_mirror WHERE user_id = $1 AND key = $2`, id, key)
		return err
	})
}

type event struct {
	id, user, entity uuid.UUID
	actor, typ, kind string
}

func (w *Worker) events(ctx context.Context, users map[uuid.UUID]user) error {
	var cursor uuid.UUID
	err := w.db.QueryRow(ctx, `SELECT event_id FROM vault_mirror_cursor WHERE id = 1`).Scan(&cursor)
	if errors.Is(err, pgx.ErrNoRows) {
		// First run: everything already written by syncUsers; start from now.
		_, err = w.db.Exec(ctx, `INSERT INTO vault_mirror_cursor (id, event_id)
			SELECT 1, coalesce((SELECT id FROM events ORDER BY id DESC LIMIT 1), '00000000-0000-0000-0000-000000000000') ON CONFLICT DO NOTHING`)
		return err
	}
	if err != nil {
		return err
	}
	rows, err := w.db.Query(ctx, `SELECT id, user_id, actor, entity_type, entity_id, kind FROM events WHERE id > $1 ORDER BY id LIMIT 1000`, cursor)
	if err != nil {
		return err
	}
	var evs []event
	for rows.Next() {
		var e event
		if err := rows.Scan(&e.id, &e.user, &e.actor, &e.typ, &e.entity, &e.kind); err != nil {
			rows.Close()
			return err
		}
		evs = append(evs, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, e := range evs {
		if u, ok := users[e.user]; ok {
			if err := w.event(ctx, e, u); err != nil {
				w.log.Warn("vault mirror: event not applied", "type", e.typ, "kind", e.kind, "err", err)
			}
		}
		cursor = e.id
	}
	if len(evs) > 0 {
		_, err = w.db.Exec(ctx, `UPDATE vault_mirror_cursor SET event_id = $1 WHERE id = 1`, cursor)
	}
	return err
}

func (w *Worker) event(ctx context.Context, e event, u user) error {
	if e.typ == notes.Type {
		if e.actor == string(Actor) {
			return nil
		}
		return w.noteChanged(ctx, e, u)
	}
	if _, ok := kinds[e.typ]; !ok {
		return nil
	}
	if e.kind == "deleted" {
		return w.drop(ctx, e.user, e.entity.String())
	}
	if err := w.refresh(ctx, e.user, u, e.typ, e.entity); err != nil {
		return err
	}
	// Notes that show this item elsewhere: a task's projects and goals, a milestone's goal.
	var deps []struct {
		id  uuid.UUID
		typ string
	}
	var rows pgx.Rows
	var err error
	switch e.typ {
	case "task":
		rows, err = w.db.Query(ctx, `SELECT project_id, 'project' FROM project_cards WHERE task_id = $1
			UNION SELECT l.target_id, t.type FROM links l JOIN entities t ON t.id = l.target_id WHERE l.source_id = $1 AND t.type = 'goal'`, e.entity)
	case "milestone":
		rows, err = w.db.Query(ctx, `SELECT goal_id, 'goal' FROM milestones WHERE id = $1`, e.entity)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	for rows.Next() {
		var d struct {
			id  uuid.UUID
			typ string
		}
		if err := rows.Scan(&d.id, &d.typ); err != nil {
			rows.Close()
			return err
		}
		deps = append(deps, d)
	}
	rows.Close()
	for _, d := range deps {
		if err := w.refresh(ctx, e.user, u, d.typ, d.id); err != nil {
			return err
		}
	}
	return rows.Err()
}

// noteChanged reads back a note the person edited, created, or deleted.
func (w *Worker) noteChanged(ctx context.Context, e event, u user) error {
	if e.kind == "deleted" {
		m, found, err := lookup(ctx, w.db, `note_id = $1`, e.entity)
		if err != nil || !found {
			return err
		}
		return pgx.BeginFunc(ctx, w.db, func(tx pgx.Tx) error {
			if k, ok := kinds[m.kind]; ok {
				item, err := uuid.Parse(mKey(ctx, tx, e.entity))
				if err == nil {
					if err := k.remove(ctx, tx, e.user, item, w.now()); err != nil && !errors.Is(err, graph.ErrNotFound) {
						return err
					}
				}
			}
			_, err := tx.Exec(ctx, `DELETE FROM vault_mirror WHERE note_id = $1`, e.entity)
			return err
		})
	}
	n, err := notes.Get(ctx, w.db, e.user, e.entity)
	if errors.Is(err, graph.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	rel, ok := strings.CutPrefix(n.Path, u.folder+"/")
	if !ok || strings.HasPrefix(rel, "Health/") || strings.HasPrefix(rel, "Archive/Health/") {
		return nil
	}
	inArchive := strings.HasPrefix(rel, "Archive/")
	content := *n.Content
	p := parse(content)
	m, found, err := lookup(ctx, w.db, `note_id = $1`, n.ID)
	if err != nil {
		return err
	}
	var item uuid.UUID
	var k kind
	if found {
		if notes.Hash(content) == m.hash && n.Path == m.path {
			return nil // what Axon wrote
		}
		if k, ok = kinds[m.kind]; !ok {
			return nil // health: one way
		}
		if item, err = uuid.Parse(mKey(ctx, w.db, n.ID)); err != nil {
			return nil
		}
		err = pgx.BeginFunc(ctx, w.db, func(tx pgx.Tx) error {
			if err := k.apply(ctx, tx, e.user, item, p, inArchive, w.now()); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `UPDATE vault_mirror SET extra = $2 WHERE note_id = $1`, n.ID, freeNotes(k, p))
			return err
		})
	} else {
		if k, ok = kindForType(p.fm["type"]); !ok {
			return nil
		}
		err = pgx.BeginFunc(ctx, w.db, func(tx pgx.Tx) error {
			var err error
			if item, err = k.create(ctx, tx, e.user, p.fm["type"], p, inArchive, w.now()); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `INSERT INTO vault_mirror (user_id, key, kind, note_id, hash, extra) VALUES ($1, $2, $3, $4, $5, $6)`,
				e.user, item.String(), k.entity, n.ID, notes.Hash(content), freeNotes(k, p))
			return err
		})
	}
	if err != nil {
		w.log.Info("vault mirror: a note could not update its item", "path", n.Path, "err", err)
		return nil
	}
	return w.refresh(ctx, e.user, u, k.entity, item)
}

func freeNotes(k kind, p parsed) string {
	if k.ownNotes {
		return ""
	}
	return p.sections["Notes"]
}

func mKey(ctx context.Context, q db.Querier, noteID uuid.UUID) string {
	var key string
	_ = q.QueryRow(ctx, `SELECT key FROM vault_mirror WHERE note_id = $1`, noteID).Scan(&key)
	return key
}

func (w *Worker) queue(ctx context.Context, users map[uuid.UUID]user) error {
	rows, err := w.db.Query(ctx, `DELETE FROM vault_mirror_queue WHERE (user_id, key) IN
		(SELECT user_id, key FROM vault_mirror_queue ORDER BY queued_at LIMIT 500) RETURNING user_id, key`)
	if err != nil {
		return err
	}
	type item struct {
		user uuid.UUID
		key  string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.user, &it.key); err != nil {
			rows.Close()
			return err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, it := range items {
		u, mirrored := users[it.user]
		var err error
		switch {
		case it.key == "health:*":
			err = w.dropHealth(ctx, it.user) // always: the health logs are gone
		case !mirrored:
		case strings.HasPrefix(it.key, "health:"):
			if u.health {
				err = w.refreshHealth(ctx, it.user, u, it.key)
			}
		default:
			if id, perr := uuid.Parse(it.key); perr == nil {
				err = w.refresh(ctx, it.user, u, "habit", id)
			}
		}
		if err != nil {
			w.log.Warn("vault mirror: queued change not applied", "err", err)
		}
	}
	return nil
}
