package notes

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Request size limits: one note, and a whole-vault import.
const (
	maxNoteBody   = 2 << 20
	maxImportBody = 50 << 20
	importTime    = 10 * time.Minute // reading and importing one archive
)

// Handlers serves notes, import, and sync.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the note endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require and must
// not cap bodies below maxImportBody.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/notes", h.list)
	r.Get("/v1/notes/tags", h.tags)
	r.Post("/v1/notes", h.create)
	r.Get("/v1/notes/{id}", h.get)
	r.Put("/v1/notes/{id}", h.update)
	r.Delete("/v1/notes/{id}", h.delete)
	r.Get("/v1/notes/{id}/mentions", h.mentions)
	r.Post("/v1/notes/import", h.importZip)
	r.Get("/v1/notes/sync", h.changes)
	r.Get("/v1/notes/sync/version", h.syncVersion)
	r.Post("/v1/notes/sync", h.push)
	r.Post("/v1/notes/sync/delete", h.pushDelete)
	r.Get("/v1/notes/files", h.files)
	r.Post("/v1/notes/files", h.upload)
	r.Get("/v1/notes/file", h.file)
	r.Put("/v1/notes/file", h.putFile)
	r.Delete("/v1/notes/file", h.removeFile)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such note.")
	case errors.Is(err, ErrBadPath):
		httpx.Problem(w, http.StatusUnprocessableEntity, "Paths look like Zettel/idea.md: relative, ending in .md, with no '..' or hidden folders.")
	case errors.Is(err, ErrTooLarge):
		httpx.Problem(w, http.StatusRequestEntityTooLarge, "Notes are limited to 1,000,000 characters.")
	case errors.Is(err, ErrPathTaken):
		httpx.Problem(w, http.StatusConflict, "Another note already uses that path.")
	case errors.Is(err, ErrStale):
		httpx.Problem(w, http.StatusPreconditionFailed, "This note changed since you loaded it. Reload, merge your edit, and save again.")
	default:
		h.log.Error("notes request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	q := r.URL.Query()
	limit, ok := httpx.Limit(r)
	if !ok {
		httpx.Problem(w, http.StatusBadRequest, "?limit= must be between 1 and 100.")
		return
	}
	var para, after, tag, exact *string
	if v := q.Get("para"); v != "" {
		para = &v
	}
	if v := q.Get("tag"); v != "" {
		v = strings.ToLower(v)
		tag = &v
	}
	if v := q.Get("path"); v != "" {
		exact = &v
	}
	var kind *string
	if v := q.Get("kind"); v != "" {
		kind = &v
	}
	var folder *string
	if v := q.Get("folder"); v != "" {
		f := escapeLike(strings.TrimSuffix(v, "/")) + "/%"
		folder = &f
	}
	if v := q.Get("after"); v != "" {
		after = &v // the last path of the previous page
	}
	// ?mirrored=false leaves out the notes the vault mirror keeps for items (EP-48).
	hideMirrored := q.Get("mirrored") == "false"
	rows, err := h.db.Query(r.Context(), `SELECT `+columns+` FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND ($2::text IS NULL OR n.para = $2) AND ($3::text IS NULL OR n.path > $3)
		AND ($5::text IS NULL OR n.tags @> ARRAY[$5::text]) AND ($6::text IS NULL OR n.path = $6)
		AND ($7::text IS NULL OR n.path LIKE $7)
		AND (NOT $8 OR NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = n.id))
		AND ($9::text IS NULL OR n.kind = $9)
		ORDER BY n.path LIMIT $4`, who.UserID, para, after, limit, tag, exact, folder, hideMirrored, kind)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Note, error) { return scan(row, false) })
	if err != nil {
		h.fail(w, err)
		return
	}
	out := httpx.List[Note]{Items: items}
	if out.Items == nil {
		out.Items = []Note{}
	}
	if len(items) == limit {
		out.NextCursor = &items[len(items)-1].Path
	}
	httpx.JSON(w, http.StatusOK, out)
}

type noteBody struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body noteBody
	if err := httpx.DecodeN(r, &body, maxNoteBody); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Path == "" { // new notes land in the inbox, named after their title
		stem := time.Now().UTC().Format("200601021504") + "-note"
		if title := Parse("untitled.md", body.Content).Title; title != "untitled" {
			stem = Slug(title)
		}
		body.Path = "0-Inbox/" + stem + ".md"
	}
	var n Note
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		n, err = Create(r.Context(), tx, who.UserID, graph.UserActor, body.Path, body.Content)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(n.UpdatedAt))
	httpx.JSON(w, http.StatusCreated, n)
}

func noteID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such note.")
	}
	return id, err == nil
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := noteID(w, r)
	if !ok {
		return
	}
	n, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(n.UpdatedAt))
	httpx.JSON(w, http.StatusOK, n)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := noteID(w, r)
	if !ok {
		return
	}
	version, ok := httpx.IfMatch(w, r)
	if !ok {
		return
	}
	var body noteBody
	if err := httpx.DecodeN(r, &body, maxNoteBody); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var n Note
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		n, err = Update(r.Context(), tx, who.UserID, id, graph.UserActor, version, body.Path, body.Content)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(n.UpdatedAt))
	httpx.JSON(w, http.StatusOK, n)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := noteID(w, r)
	if !ok {
		return
	}
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return Delete(r.Context(), tx, who.UserID, id, graph.UserActor)
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Import ---

// ImportResult reports what a ZIP import did. Existing notes with different
// content are never overwritten: they are listed as conflicts.
type ImportResult struct {
	Created   []string `json:"created"`
	Unchanged []string `json:"unchanged"`
	Conflicts []string `json:"conflicts"`
	Skipped   []string `json:"skipped"`
}

func (h *Handlers) importZip(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	// A vault of thousands of notes, sent over a slow link, outlasts the
	// server's 15-second limits: this route gets its own.
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Now().Add(importTime))
	_ = rc.SetWriteDeadline(time.Now().Add(importTime))
	data, err := io.ReadAll(io.LimitReader(r.Body, maxImportBody+1))
	if err != nil || len(data) > maxImportBody {
		httpx.Problem(w, http.StatusRequestEntityTooLarge, "Imports are limited to 50 MB.")
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		httpx.Problem(w, http.StatusBadRequest, "Send a ZIP archive of Markdown files.")
		return
	}
	res := ImportResult{Created: []string{}, Unchanged: []string{}, Conflicts: []string{}, Skipped: []string{}}
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var batch Batch
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			name := strings.TrimPrefix(f.Name, "notes/")
			if fp, _, ok := CleanFilePath(name); ok && f.UncompressedSize64 <= maxFile {
				if err := importFile(r.Context(), tx, who.UserID, f, fp, &res); err != nil {
					return err
				}
				continue
			}
			p, ok := CleanPath(name)
			if !ok || f.UncompressedSize64 > 1_000_000 {
				res.Skipped = append(res.Skipped, f.Name)
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			content, err := io.ReadAll(io.LimitReader(rc, 1_000_001))
			_ = rc.Close()
			if err != nil {
				return err
			}
			cur, err := GetByPath(r.Context(), tx, who.UserID, p)
			switch {
			case errors.Is(err, graph.ErrNotFound):
				if _, err := batch.Add(r.Context(), tx, who.UserID, graph.UserActor, p, string(content)); err != nil {
					return err
				}
				res.Created = append(res.Created, p)
			case err != nil:
				return err
			case *cur.Content == string(content):
				res.Unchanged = append(res.Unchanged, p)
			default:
				res.Conflicts = append(res.Conflicts, p)
			}
		}
		return batch.Finish(r.Context(), tx, who.UserID, graph.UserActor)
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// importFile adds a picture or PDF from the archive; one that differs from
// the stored file is a conflict, and one that is not what its name says is
// skipped.
func importFile(ctx context.Context, tx pgx.Tx, userID uuid.UUID, f *zip.File, p string, res *ImportResult) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(rc, maxFile+1))
	_ = rc.Close()
	if err != nil {
		return err
	}
	cur, err := getFile(ctx, tx, userID, p)
	switch {
	case errors.Is(err, errNoFile):
		_, err = PutFile(ctx, tx, userID, p, data)
		switch {
		case errors.Is(err, ErrBadFile), errors.Is(err, ErrTooLarge):
			res.Skipped = append(res.Skipped, f.Name)
			return nil
		case err != nil:
			return err
		}
		res.Created = append(res.Created, p)
	case err != nil:
		return err
	case cur.Hash == Hash(string(data)):
		res.Unchanged = append(res.Unchanged, p)
	default:
		res.Conflicts = append(res.Conflicts, p)
	}
	return nil
}

// --- Sync ---
//
// A sync client (axon notes-sync) keeps a folder of Markdown files and Axon
// identical. It tracks, per path, the hash both sides last agreed on, and
// sends it back as base_hash: Axon only accepts a write whose base matches its
// current content, so a change made on both sides becomes a conflict for the
// client to keep side by side, never a silent overwrite.

type tombstone struct {
	Path      string    `json:"path"`
	DeletedAt time.Time `json:"deleted_at"`
}

// changes returns notes changed and deleted since a cursor (oldest first).
// syncVersion is a cheap fingerprint of the person's notes: it changes
// whenever a note is created, edited, renamed, or deleted, so a watching
// sync knows when to run (EP-36).
func (h *Handlers) syncVersion(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var count int
	var lastEdit, lastDelete *time.Time
	if err := h.db.QueryRow(r.Context(), `SELECT
		(SELECT count(*) FROM notes WHERE user_id = $1),
		(SELECT max(updated_at) FROM notes WHERE user_id = $1),
		(SELECT max(deleted_at) FROM note_tombstones WHERE user_id = $1)`, who.UserID).Scan(&count, &lastEdit, &lastDelete); err != nil {
		h.fail(w, err)
		return
	}
	// Attachments count too, so a picture added in Axon wakes the sync.
	var files int
	var lastFile, lastFileDelete *time.Time
	if err := h.db.QueryRow(r.Context(), `SELECT
		(SELECT count(*) FROM note_files WHERE user_id = $1),
		(SELECT max(updated_at) FROM note_files WHERE user_id = $1),
		(SELECT max(deleted_at) FROM note_file_tombstones WHERE user_id = $1)`, who.UserID).Scan(&files, &lastFile, &lastFileDelete); err != nil {
		h.fail(w, err)
		return
	}
	stamp := func(t *time.Time) string {
		if t == nil {
			return "-"
		}
		return t.UTC().Format(time.RFC3339Nano)
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"version": fmt.Sprintf("%d|%s|%s|%d|%s|%s", count, stamp(lastEdit), stamp(lastDelete), files, stamp(lastFile), stamp(lastFileDelete))})
}

func (h *Handlers) changes(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	since := time.Time{}
	var afterID *uuid.UUID
	if c := r.URL.Query().Get("cursor"); c != "" {
		raw, _ := base64.RawURLEncoding.DecodeString(c)
		ts, id, ok := strings.Cut(string(raw), "|")
		t, err1 := time.Parse(time.RFC3339Nano, ts)
		uid, err2 := uuid.Parse(id)
		if !ok || err1 != nil || err2 != nil {
			httpx.Problem(w, http.StatusBadRequest, "The ?cursor= value is not valid.")
			return
		}
		since, afterID = t, &uid
	}
	const page = 100
	rows, err := h.db.Query(r.Context(), `SELECT `+columns+` FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND (n.updated_at, n.id) > ($2, coalesce($3, '00000000-0000-0000-0000-000000000000'::uuid))
		ORDER BY n.updated_at, n.id LIMIT $4`, who.UserID, since, afterID, page)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Note, error) { return scan(row, true) })
	if err != nil {
		h.fail(w, err)
		return
	}
	rows, err = h.db.Query(r.Context(), `SELECT path, deleted_at FROM note_tombstones WHERE user_id = $1 AND deleted_at > $2 ORDER BY deleted_at`, who.UserID, since)
	if err != nil {
		h.fail(w, err)
		return
	}
	stones, err := pgx.CollectRows(rows, pgx.RowToStructByPos[tombstone])
	if err != nil {
		h.fail(w, err)
		return
	}
	_, _ = h.db.Exec(r.Context(), `DELETE FROM note_tombstones WHERE deleted_at < now() - interval '30 days'`)

	next := since
	nextID := uuid.Nil
	if afterID != nil {
		nextID = *afterID
	}
	if len(items) > 0 {
		last := items[len(items)-1]
		next, nextID = last.UpdatedAt, last.ID
	}
	if items == nil {
		items = []Note{}
	}
	if stones == nil {
		stones = []tombstone{}
	}
	cursor := base64.RawURLEncoding.EncodeToString([]byte(next.UTC().Format(time.RFC3339Nano) + "|" + nextID.String()))
	httpx.JSON(w, http.StatusOK, map[string]any{"notes": items, "tombstones": stones, "cursor": cursor, "more": len(items) == page})
}

type pushBody struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	BaseHash string `json:"base_hash"` // "" when the client has never seen this path in Axon
}

// conflict answers 409 with Axon's current version so the client can keep both.
func conflict(w http.ResponseWriter, current *Note) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusConflict)
	httpx.WriteJSON(w, map[string]any{
		"type": "about:blank", "title": "Conflict", "status": http.StatusConflict,
		"detail":  "The note changed in Axon since your last sync. Keep both versions and merge them.",
		"current": current,
	})
}

func (h *Handlers) push(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body pushBody
	if err := httpx.DecodeN(r, &body, maxNoteBody); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	p, ok := CleanPath(body.Path)
	if !ok {
		h.fail(w, ErrBadPath)
		return
	}
	var out Note
	var clash *Note
	clashed := errors.New("conflict")
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		cur, err := GetByPath(r.Context(), tx, who.UserID, p)
		switch {
		case errors.Is(err, graph.ErrNotFound):
			if body.BaseHash != "" { // deleted in Axon after the client last synced
				return clashed
			}
			out, err = Create(r.Context(), tx, who.UserID, graph.UserActor, p, body.Content)
			return err
		case err != nil:
			return err
		case cur.Hash == Hash(body.Content):
			out = cur // already identical
			return nil
		case cur.Hash != body.BaseHash:
			clash = &cur
			return clashed
		}
		out, err = Update(r.Context(), tx, who.UserID, cur.ID, graph.UserActor, cur.UpdatedAt, p, body.Content)
		return err
	})
	switch {
	case errors.Is(err, clashed):
		conflict(w, clash)
	case err != nil:
		h.fail(w, err)
	default:
		out.Content = nil
		httpx.JSON(w, http.StatusOK, out)
	}
}

func (h *Handlers) pushDelete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Path     string `json:"path"`
		BaseHash string `json:"base_hash"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var clash *Note
	clashed := errors.New("conflict")
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		cur, err := GetByPath(r.Context(), tx, who.UserID, body.Path)
		if errors.Is(err, graph.ErrNotFound) {
			return nil // already gone
		}
		if err != nil {
			return err
		}
		if cur.Hash != body.BaseHash {
			clash = &cur // edited in Axon: keep it
			return clashed
		}
		return Delete(r.Context(), tx, who.UserID, cur.ID, graph.UserActor)
	})
	switch {
	case errors.Is(err, clashed):
		conflict(w, clash)
	case err != nil:
		h.fail(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// tags lists the tags in use with how many notes carry each.
func (h *Handlers) tags(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rows, err := h.db.Query(r.Context(), `SELECT t, count(*) FROM notes, unnest(coalesce(tags, '{}')) AS t
		WHERE user_id = $1 GROUP BY t ORDER BY count(*) DESC, t LIMIT 200`, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	type tagCount struct {
		Tag   string `json:"tag"`
		Count int    `json:"count"`
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[tagCount])
	if err != nil {
		h.fail(w, err)
		return
	}
	if items == nil {
		items = []tagCount{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

// mentions finds notes that name this note in plain text but do not link
// to it yet: candidates for a [[wikilink]].
func (h *Handlers) mentions(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := noteID(w, r)
	if !ok {
		return
	}
	n, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	type mention struct {
		ID    uuid.UUID `json:"id"`
		Title string    `json:"title"`
		Path  string    `json:"path"`
	}
	items := []mention{}
	if len([]rune(n.Title)) >= 3 {
		like := "%" + escapeLike(n.Title) + "%"
		rows, err := h.db.Query(r.Context(), `SELECT n.id, e.title, n.path FROM notes n JOIN entities e ON e.id = n.id
			WHERE n.user_id = $1 AND n.id <> $2 AND n.content ILIKE $3
			AND NOT EXISTS (SELECT 1 FROM links l WHERE l.source_id = n.id AND l.target_id = $2)
			ORDER BY n.updated_at DESC LIMIT 20`, who.UserID, id, like)
		if err != nil {
			h.fail(w, err)
			return
		}
		if items, err = pgx.CollectRows(rows, pgx.RowToStructByPos[mention]); err != nil {
			h.fail(w, err)
			return
		}
		if items == nil {
			items = []mention{}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}
