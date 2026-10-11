// Package notes stores Markdown notes verbatim, links them through
// [[wikilinks]], and serves the API that keeps a Git vault in sync (EP-11).
package notes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// Type is the graph entity type of a note.
const Type = "note"

// References is the relation a [[wikilink]] creates: from a note to any entity.
const References = "references"

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "notes", Export: export})
	graph.RegisterRelation(graph.Relation{Name: References, Sources: []string{Type}})
}

// Errors callers translate to HTTP responses.
var (
	ErrPathTaken = errors.New("path already used")
	ErrBadPath   = errors.New(`paths look like "Zettel/idea.md": a relative path ending in .md, no "..", no hidden folders`)
	ErrStale     = errors.New("stale version")
	ErrTooLarge  = errors.New("notes are limited to 1,000,000 characters")
)

// Note is the API representation. Content is omitted from lists.
type Note struct {
	ID        uuid.UUID `json:"id"`
	Path      string    `json:"path"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"`
	Para      string    `json:"para"`
	Tags      []string  `json:"tags"`
	Hash      string    `json:"hash"` // SHA-256 of content, for sync
	Content   *string   `json:"content,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Hash returns the hex SHA-256 of content.
func Hash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

const columns = `n.id, n.path, e.title, n.kind, n.para, coalesce(n.tags, '{}'), n.content, e.created_at, n.updated_at`

func scan(row pgx.Row, withContent bool) (Note, error) {
	var n Note
	var content string
	err := row.Scan(&n.ID, &n.Path, &n.Title, &n.Kind, &n.Para, &n.Tags, &content, &n.CreatedAt, &n.UpdatedAt)
	n.Hash = Hash(content)
	if withContent {
		n.Content = &content
	}
	return n, err
}

// Get returns one of the user's notes with its content.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Note, error) {
	n, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.id = $1 AND n.user_id = $2`, id, userID), true)
	if errors.Is(err, pgx.ErrNoRows) {
		return n, graph.ErrNotFound
	}
	return n, err
}

// GetByPath returns the user's note at path.
func GetByPath(ctx context.Context, q db.Querier, userID uuid.UUID, p string) (Note, error) {
	n, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND n.path = $2`, userID, p), true)
	if errors.Is(err, pgx.ErrNoRows) {
		return n, graph.ErrNotFound
	}
	return n, err
}

// Create stores a new note.
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, p, content string) (Note, error) {
	p, ok := CleanPath(p)
	if !ok {
		return Note{}, ErrBadPath
	}
	if len(content) > 1_000_000 {
		return Note{}, ErrTooLarge
	}
	m := Parse(p, content)
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, m.Title, content)
	if err != nil {
		return Note{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notes (id, user_id, path, content, kind, para, tags) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, userID, p, content, m.Kind, m.Para, m.Tags)
	if isUnique(err) {
		return Note{}, ErrPathTaken
	}
	if err != nil {
		return Note{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM note_tombstones WHERE user_id = $1 AND path = $2`, userID, p); err != nil {
		return Note{}, err
	}
	if err := relink(ctx, tx, userID, actor, id, content); err != nil {
		return Note{}, err
	}
	// Notes that already linked to this name, before it existed, now resolve.
	if err := relinkReferrers(ctx, tx, userID, actor, []string{Stem(p), m.Title}); err != nil {
		return Note{}, err
	}
	return Get(ctx, tx, userID, id)
}

// Update replaces a note's content and/or path if version is still current.
// Moving a note rewrites [[old-name]] links in other notes, so none break.
func Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, version time.Time, p, content string) (Note, error) {
	p, ok := CleanPath(p)
	if !ok {
		return Note{}, ErrBadPath
	}
	if len(content) > 1_000_000 {
		return Note{}, ErrTooLarge
	}
	cur, err := Get(ctx, tx, userID, id)
	if err != nil {
		return Note{}, err
	}
	if !cur.UpdatedAt.Equal(version) {
		return Note{}, ErrStale
	}
	if p == cur.Path && content == *cur.Content {
		return cur, nil
	}
	var fields []string
	if content != *cur.Content {
		fields = append(fields, "content")
	}
	m := Parse(p, content)
	if p != cur.Path {
		fields = append(fields, "path")
		if _, err := tx.Exec(ctx, `INSERT INTO note_tombstones (user_id, path) VALUES ($1, $2)
			ON CONFLICT (user_id, path) DO UPDATE SET deleted_at = now()`, userID, cur.Path); err != nil {
			return Note{}, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM note_tombstones WHERE user_id = $1 AND path = $2`, userID, p); err != nil {
			return Note{}, err
		}
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, m.Title, content, fields); err != nil {
		return Note{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE notes SET path = $2, content = $3, kind = $4, para = $5, tags = $6, updated_at = clock_timestamp() WHERE id = $1`,
		id, p, content, m.Kind, m.Para, m.Tags)
	if isUnique(err) {
		return Note{}, ErrPathTaken
	}
	if err != nil {
		return Note{}, err
	}
	if oldStem, newStem := Stem(cur.Path), Stem(p); oldStem != newStem {
		if err := renameReferences(ctx, tx, userID, actor, id, oldStem, newStem); err != nil {
			return Note{}, err
		}
	}
	if err := relink(ctx, tx, userID, actor, id, content); err != nil {
		return Note{}, err
	}
	if err := relinkReferrers(ctx, tx, userID, actor, []string{Stem(p), m.Title}); err != nil {
		return Note{}, err
	}
	return Get(ctx, tx, userID, id)
}

// Delete removes a note and leaves a tombstone so sync deletes the file too.
func Delete(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	cur, err := Get(ctx, tx, userID, id)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO note_tombstones (user_id, path) VALUES ($1, $2)
		ON CONFLICT (user_id, path) DO UPDATE SET deleted_at = now()`, userID, cur.Path); err != nil {
		return err
	}
	if err := graph.DeleteEntity(ctx, tx, userID, id, actor); err != nil {
		return err
	}
	// Links to it are gone; notes that named it keep the text and will
	// resolve again if a note with that name comes back.
	return nil
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// resolve maps link targets to the user's notes: by file stem first (as
// Obsidian does), then by title, case-insensitively. Ambiguous names pick the
// shortest path.
func resolve(ctx context.Context, q db.Querier, userID uuid.UUID, targets []string) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	if len(targets) == 0 {
		return out, nil
	}
	lower := make([]string, len(targets))
	for i, t := range targets {
		lower[i] = strings.ToLower(t)
	}
	rows, err := q.Query(ctx, `
		SELECT DISTINCT ON (key) key, id FROM (
			SELECT lower(regexp_replace(n.path, '^.*/|\.md$', '', 'g')) AS key, n.id, 0 AS rank, length(n.path) AS len
			FROM notes n WHERE n.user_id = $1
			UNION ALL
			SELECT lower(e.title), n.id, 1, length(n.path)
			FROM notes n JOIN entities e ON e.id = n.id WHERE n.user_id = $1
		) candidates WHERE key = ANY($2)
		ORDER BY key, rank, len, id`, userID, lower)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var id uuid.UUID
		if err := rows.Scan(&key, &id); err != nil {
			return nil, err
		}
		out[key] = id
	}
	return out, rows.Err()
}

// relink makes the note's outgoing "references" links match its wikilinks.
func relink(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, id uuid.UUID, content string) error {
	targets := Wikilinks(content)
	ids, err := resolve(ctx, tx, userID, targets)
	if err != nil {
		return err
	}
	want := map[uuid.UUID]bool{}
	for _, t := range targets {
		if target, ok := ids[strings.ToLower(t)]; ok && target != id {
			want[target] = true
		}
	}
	rows, err := tx.Query(ctx, `SELECT id, target_id FROM links WHERE user_id = $1 AND source_id = $2 AND relation = $3`, userID, id, References)
	if err != nil {
		return err
	}
	type link struct{ ID, Target uuid.UUID }
	existing, err := pgx.CollectRows(rows, pgx.RowToStructByPos[link])
	if err != nil {
		return err
	}
	for _, l := range existing {
		if want[l.Target] {
			delete(want, l.Target)
			continue
		}
		if err := graph.DeleteLink(ctx, tx, userID, actor, l.ID); err != nil {
			return err
		}
	}
	for target := range want {
		if _, err := graph.CreateLink(ctx, tx, userID, actor, id, References, target); err != nil && !errors.Is(err, graph.ErrDuplicate) {
			return err
		}
	}
	return nil
}

// relinkReferrers re-resolves notes whose text mentions any of the names.
func relinkReferrers(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, names []string) error {
	patterns := make([]string, 0, len(names))
	for _, n := range names {
		if n != "" {
			patterns = append(patterns, "%[["+escapeLike(n)+"%")
		}
	}
	rows, err := tx.Query(ctx, `SELECT id, content FROM notes WHERE user_id = $1 AND content ILIKE ANY($2)`, userID, patterns)
	if err != nil {
		return err
	}
	type ref struct {
		ID      uuid.UUID
		Content string
	}
	refs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[ref])
	if err != nil {
		return err
	}
	for _, r := range refs {
		if err := relink(ctx, tx, userID, actor, r.ID, r.Content); err != nil {
			return err
		}
	}
	return nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// renameReferences rewrites [[old]] in every note that links to the moved note.
func renameReferences(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, target uuid.UUID, oldStem, newStem string) error {
	rows, err := tx.Query(ctx, `SELECT n.id, n.content, e.title FROM links l JOIN notes n ON n.id = l.source_id JOIN entities e ON e.id = n.id
		WHERE l.user_id = $1 AND l.target_id = $2 AND l.relation = $3`, userID, target, References)
	if err != nil {
		return err
	}
	type ref struct {
		ID             uuid.UUID
		Content, Title string
	}
	refs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[ref])
	if err != nil {
		return err
	}
	for _, r := range refs {
		updated := RenameLinks(r.Content, oldStem, newStem)
		if updated == r.Content {
			continue
		}
		if err := graph.UpdateEntity(ctx, tx, userID, r.ID, actor, r.Title, updated, []string{"content"}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE notes SET content = $2, updated_at = clock_timestamp() WHERE id = $1`, r.ID, updated); err != nil {
			return err
		}
	}
	return nil
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM notes n JOIN entities e ON e.id = n.id WHERE n.user_id = $1 ORDER BY n.path`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Note, error) { return scan(row, true) })
}

// Files returns every note as path → Markdown, for the export archive.
func Files(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string][]byte, error) {
	rows, err := q.Query(ctx, `SELECT path, content FROM notes WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]byte{}
	for rows.Next() {
		var p, c string
		if err := rows.Scan(&p, &c); err != nil {
			return nil, err
		}
		out[p] = []byte(c)
	}
	return out, rows.Err()
}

// BackfillTags parses tags for notes stored before tags existed. It runs
// after migrations ("axon migrate up") and is safe to repeat.
func BackfillTags(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	total := 0
	for {
		rows, err := pool.Query(ctx, `SELECT id, path, content FROM notes WHERE tags IS NULL LIMIT 200`)
		if err != nil {
			return total, err
		}
		type row struct {
			ID            uuid.UUID
			Path, Content string
		}
		batch, err := pgx.CollectRows(rows, pgx.RowToStructByPos[row])
		if err != nil || len(batch) == 0 {
			return total, err
		}
		for _, r := range batch {
			if _, err := pool.Exec(ctx, `UPDATE notes SET tags = $2 WHERE id = $1`, r.ID, Parse(r.Path, r.Content).Tags); err != nil {
				return total, err
			}
		}
		total += len(batch)
	}
}
