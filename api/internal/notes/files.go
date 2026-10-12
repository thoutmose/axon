package notes

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/images"
)

// Files that notes embed (EP-51): pictures and PDFs, stored by vault path so
// the vault sync writes them next to the notes and Obsidian shows them too.
// A picture added in the editor is re-encoded (no EXIF, no location) and
// scaled down; a file pushed by the vault sync is kept byte for byte, since
// it is already the person's own file and must round-trip unchanged.

// Limits for one file and for everything one person stores.
const (
	maxFile        = 10 << 20
	maxFilesUser   = 500 << 20
	maxPictureSide = 2400
)

// UploadFolder is where pictures added in the editor go.
const UploadFolder = "Attachments"

var fileTypes = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".svg": "image/svg+xml", ".pdf": "application/pdf",
}

// Errors callers translate to HTTP responses.
var (
	ErrBadFilePath = errors.New("bad file path")
	ErrBadFile     = errors.New("the file is not what its name says")
	ErrQuota       = errors.New("file storage is full")
)

// File is a stored file's metadata.
type File struct {
	Path        string    `json:"path"`
	ContentType string    `json:"content_type"`
	Size        int       `json:"size"`
	Hash        string    `json:"hash"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CleanFilePath validates a vault-relative path to an embeddable file.
func CleanFilePath(p string) (string, string, bool) {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	clean := path.Clean(p)
	ct, known := fileTypes[strings.ToLower(path.Ext(clean))]
	ok := known && clean == p && !strings.HasPrefix(clean, "/") && !strings.HasPrefix(clean, ".") &&
		!strings.Contains(clean, "/.") && len(clean) <= 500 && strings.TrimSuffix(path.Base(clean), path.Ext(clean)) != ""
	return clean, ct, ok
}

// sniff reports whether data really is a file of type ct.
func sniff(data []byte, ct string) bool {
	switch ct {
	case "image/png":
		return bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n"))
	case "image/jpeg":
		return bytes.HasPrefix(data, []byte{0xff, 0xd8, 0xff})
	case "image/gif":
		return bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a"))
	case "image/webp":
		return len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	case "application/pdf":
		return bytes.HasPrefix(data, []byte("%PDF-"))
	case "image/svg+xml":
		dec := xml.NewDecoder(bytes.NewReader(data))
		for {
			tok, err := dec.Token()
			if err != nil {
				return false
			}
			if el, ok := tok.(xml.StartElement); ok {
				return el.Name.Local == "svg"
			}
		}
	}
	return false
}

// PutFile stores data at p, replacing what is there.
func PutFile(ctx context.Context, tx pgx.Tx, userID uuid.UUID, p string, data []byte) (File, error) {
	p, ct, ok := CleanFilePath(p)
	if !ok {
		return File{}, ErrBadFilePath
	}
	if len(data) > maxFile {
		return File{}, ErrTooLarge
	}
	if !sniff(data, ct) {
		return File{}, ErrBadFile
	}
	var used int64
	if err := tx.QueryRow(ctx, `SELECT coalesce(sum(length(data)), 0) FROM note_files WHERE user_id = $1 AND path <> $2`, userID, p).Scan(&used); err != nil {
		return File{}, err
	}
	if used+int64(len(data)) > maxFilesUser {
		return File{}, ErrQuota
	}
	var f File
	err := tx.QueryRow(ctx, `INSERT INTO note_files (user_id, path, content_type, data, hash) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, path) DO UPDATE SET content_type = $3, data = $4, hash = $5, updated_at = clock_timestamp()
		RETURNING path, content_type, length(data), hash, updated_at`, userID, p, ct, data, Hash(string(data))).
		Scan(&f.Path, &f.ContentType, &f.Size, &f.Hash, &f.UpdatedAt)
	if err != nil {
		return File{}, err
	}
	_, err = tx.Exec(ctx, `DELETE FROM note_file_tombstones WHERE user_id = $1 AND path = $2`, userID, p)
	return f, err
}

func getFile(ctx context.Context, q db.Querier, userID uuid.UUID, p string) (File, error) {
	var f File
	err := q.QueryRow(ctx, `SELECT path, content_type, length(data), hash, updated_at FROM note_files WHERE user_id = $1 AND path = $2`, userID, p).
		Scan(&f.Path, &f.ContentType, &f.Size, &f.Hash, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, errNoFile
	}
	return f, err
}

var errNoFile = errors.New("no such file")

func deleteFile(ctx context.Context, tx pgx.Tx, userID uuid.UUID, p string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM note_files WHERE user_id = $1 AND path = $2`, userID, p); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO note_file_tombstones (user_id, path) VALUES ($1, $2)
		ON CONFLICT (user_id, path) DO UPDATE SET deleted_at = now()`, userID, p)
	return err
}

func (h *Handlers) fileFail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNoFile):
		httpx.Problem(w, http.StatusNotFound, "No such file.")
	case errors.Is(err, ErrBadFilePath):
		httpx.Problem(w, http.StatusUnprocessableEntity, "File paths look like Attachments/photo.png: relative, ending in .png, .jpg, .gif, .webp, .svg, or .pdf, with no '..' or hidden folders.")
	case errors.Is(err, ErrBadFile), errors.Is(err, images.ErrNotImage):
		httpx.Problem(w, http.StatusUnprocessableEntity, "The file is not a picture or PDF of the kind its name says.")
	case errors.Is(err, ErrTooLarge):
		httpx.Problem(w, http.StatusRequestEntityTooLarge, "Files are limited to 10 MB.")
	case errors.Is(err, ErrQuota):
		httpx.Problem(w, http.StatusInsufficientStorage, "Your notes' files already use 500 MB. Delete some before adding more.")
	default:
		h.fail(w, err)
	}
}

func readBody(r *http.Request, limit int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r.Body, int64(limit)+1))
	if err != nil {
		return nil, ErrBadFile
	}
	if len(data) > limit {
		return nil, ErrTooLarge
	}
	return data, nil
}

// files lists every file, and the paths deleted in the last 30 days, for the
// editor and the vault sync.
func (h *Handlers) files(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rows, err := h.db.Query(r.Context(), `SELECT path, content_type, length(data), hash, updated_at FROM note_files WHERE user_id = $1 ORDER BY path`, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[File])
	if err != nil {
		h.fail(w, err)
		return
	}
	rows, err = h.db.Query(r.Context(), `SELECT path, deleted_at FROM note_file_tombstones WHERE user_id = $1 AND deleted_at > now() - interval '30 days' ORDER BY deleted_at`, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	stones, err := pgx.CollectRows(rows, pgx.RowToStructByPos[tombstone])
	if err != nil {
		h.fail(w, err)
		return
	}
	if items == nil {
		items = []File{}
	}
	if stones == nil {
		stones = []tombstone{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "tombstones": stones})
}

// upload stores a file added in the editor under Attachments/, with a name
// made unique by its content. PNG and JPEG pictures are re-encoded, so a
// phone photo up to 20 MB fits once scaled.
func (h *Handlers) upload(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	data, err := readBody(r, 2*maxFile)
	if err != nil {
		h.fileFail(w, err)
		return
	}
	name := path.Base(strings.ReplaceAll(r.URL.Query().Get("name"), "\\", "/"))
	ext := strings.ToLower(path.Ext(name))
	ct, ok := fileTypes[ext]
	if !ok || ct != strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]) {
		httpx.Problem(w, http.StatusUnsupportedMediaType, "Add a PNG, JPEG, GIF, WebP, SVG, or PDF file, with a name ending in its type.")
		return
	}
	switch ct {
	case "image/png":
		data, err = images.Fit(data, maxPictureSide)
	case "image/jpeg":
		data, err = images.FitJPEG(data, maxPictureSide)
		ext = ".jpg"
	}
	if err != nil {
		h.fileFail(w, err)
		return
	}
	stem := Slug(strings.TrimSuffix(name, path.Ext(name)))
	if stem == "note" {
		stem = "picture"
	}
	if r := []rune(stem); len(r) > 60 {
		stem = strings.Trim(string(r[:60]), "-")
	}
	p := fmt.Sprintf("%s/%s-%s%s", UploadFolder, stem, Hash(string(data))[:8], ext)
	var f File
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		f, err = PutFile(r.Context(), tx, who.UserID, p, data)
		return err
	})
	if err != nil {
		h.fileFail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, f)
}

// file serves one file. Nothing in it can run: the policy sandboxes an SVG
// opened on its own, and PDFs download instead of opening in this origin.
func (h *Handlers) file(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	p, _, ok := CleanFilePath(r.URL.Query().Get("path"))
	if !ok {
		h.fileFail(w, ErrBadFilePath)
		return
	}
	var ct, hash string
	var data []byte
	err := h.db.QueryRow(r.Context(), `SELECT content_type, hash, data FROM note_files WHERE user_id = $1 AND path = $2`, who.UserID, p).Scan(&ct, &hash, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		err = errNoFile
	}
	if err != nil {
		h.fileFail(w, err)
		return
	}
	etag := `"` + hash + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox")
	if ct == "application/pdf" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(p)))
	}
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", ct)
	_, _ = w.Write(data)
}

// fileConflict answers 409 with the file Axon holds now (or none).
func fileConflict(w http.ResponseWriter, current *File) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusConflict)
	httpx.WriteJSON(w, map[string]any{
		"type": "about:blank", "title": "Conflict", "status": http.StatusConflict,
		"detail":  "The file changed in Axon since your last sync. Keep both versions.",
		"current": current,
	})
}

// putFile is the vault sync's write: the bytes are kept as they are, and
// ?base_hash= must match what Axon holds ("" for a new file), as for notes.
func (h *Handlers) putFile(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	p, _, ok := CleanFilePath(r.URL.Query().Get("path"))
	if !ok {
		h.fileFail(w, ErrBadFilePath)
		return
	}
	base := r.URL.Query().Get("base_hash")
	data, err := readBody(r, maxFile)
	if err != nil {
		h.fileFail(w, err)
		return
	}
	var out File
	var clash *File
	clashed := errors.New("conflict")
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		cur, err := getFile(r.Context(), tx, who.UserID, p)
		switch {
		case errors.Is(err, errNoFile):
			if base != "" { // deleted in Axon after the client last synced
				return clashed
			}
		case err != nil:
			return err
		case cur.Hash == Hash(string(data)):
			out = cur
			return nil
		case cur.Hash != base:
			clash = &cur
			return clashed
		}
		out, err = PutFile(r.Context(), tx, who.UserID, p, data)
		return err
	})
	switch {
	case errors.Is(err, clashed):
		fileConflict(w, clash)
	case err != nil:
		h.fileFail(w, err)
	default:
		httpx.JSON(w, http.StatusOK, out)
	}
}

// removeFile deletes a file; with ?base_hash= (the vault sync) only if Axon
// still holds that version.
func (h *Handlers) removeFile(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	p, _, ok := CleanFilePath(r.URL.Query().Get("path"))
	if !ok {
		h.fileFail(w, ErrBadFilePath)
		return
	}
	base, checked := r.URL.Query()["base_hash"]
	var clash *File
	clashed := errors.New("conflict")
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		cur, err := getFile(r.Context(), tx, who.UserID, p)
		if errors.Is(err, errNoFile) {
			return nil // already gone
		}
		if err != nil {
			return err
		}
		if checked && cur.Hash != base[0] {
			clash = &cur
			return clashed
		}
		return deleteFile(r.Context(), tx, who.UserID, p)
	})
	switch {
	case errors.Is(err, clashed):
		fileConflict(w, clash)
	case err != nil:
		h.fileFail(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// fileData returns every file as path → bytes, for the export archive.
func fileData(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string][]byte, error) {
	rows, err := q.Query(ctx, `SELECT path, data FROM note_files WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]byte{}
	for rows.Next() {
		var p string
		var d []byte
		if err := rows.Scan(&p, &d); err != nil {
			return nil, err
		}
		out[p] = d
	}
	return out, rows.Err()
}
