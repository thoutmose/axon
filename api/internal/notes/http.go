package notes

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
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
	r.Post("/v1/notes", h.create)
	r.Get("/v1/notes/{id}", h.get)
	r.Put("/v1/notes/{id}", h.update)
	r.Delete("/v1/notes/{id}", h.delete)
	r.Post("/v1/notes/import", h.importZip)
	r.Get("/v1/notes/sync", h.changes)
	r.Post("/v1/notes/sync", h.push)
	r.Post("/v1/notes/sync/delete", h.pushDelete)
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
	var para, after *string
	if v := q.Get("para"); v != "" {
		para = &v
	}
	if v := q.Get("after"); v != "" {
		after = &v // the last path of the previous page
	}
	rows, err := h.db.Query(r.Context(), `SELECT `+columns+` FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND ($2::text IS NULL OR n.para = $2) AND ($3::text IS NULL OR n.path > $3)
		ORDER BY n.path LIMIT $4`, who.UserID, para, after, limit)
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
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			p, ok := CleanPath(strings.TrimPrefix(f.Name, "notes/"))
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
				if _, err := Create(r.Context(), tx, who.UserID, graph.UserActor, p, string(content)); err != nil {
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
		return nil
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
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
