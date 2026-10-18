package imports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/budget"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/media"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Result says what an import did, or with a dry run, would do.
type Result struct {
	Source    string   `json:"source"`
	DryRun    bool     `json:"dry_run"`
	Found     int      `json:"found"`     // items read from the file
	Created   int      `json:"created"`   // added to Axon
	Skipped   int      `json:"skipped"`   // already in Axon
	Conflicts int      `json:"conflicts"` // a note at the same path with other content, left alone
	Problems  []string `json:"problems"`
	Sample    []string `json:"sample"` // the first few titles
}

// maxFile bounds one upload; a Notion export with pictures can be large.
const maxFile = 50 << 20

// Handlers serves imports.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the import endpoint.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoint; the caller wraps it in auth.Require and must
// not cap bodies below maxFile.
func (h *Handlers) Routes(r chi.Router) {
	r.Post("/v1/import/{source}", h.importFile)
}

var sources = map[string]bool{"todoist": true, "ticktick": true, "goodreads": true, "letterboxd": true, "firefly": true, "ofx": true, "notion": true}

func (h *Handlers) importFile(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	source := chi.URLParam(r, "source")
	if !sources[source] {
		httpx.Problem(w, http.StatusNotFound, "Axon imports from todoist, ticktick, goodreads, letterboxd, firefly, ofx, and notion.")
		return
	}
	// A 50 MB export over a slow link outlasts the server's 15-second limits.
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Now().Add(10 * time.Minute))
	_ = rc.SetWriteDeadline(time.Now().Add(10 * time.Minute))
	data, err := io.ReadAll(io.LimitReader(r.Body, maxFile+1))
	if err != nil || len(data) > maxFile {
		httpx.Problem(w, http.StatusRequestEntityTooLarge, "Imports are limited to 50 MB.")
		return
	}
	var account *uuid.UUID
	if v := r.URL.Query().Get("account_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.Problem(w, http.StatusUnprocessableEntity, "No such account.")
			return
		}
		account = &id
	}
	res := Result{Source: source, DryRun: r.URL.Query().Get("dry_run") == "true", Problems: []string{}, Sample: []string{}}
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return h.run(r.Context(), tx, who.UserID, source, data, account, &res)
	})
	var ef ErrFormat
	var bv budget.ValidationError
	switch {
	case errors.As(err, &ef):
		httpx.Problem(w, http.StatusUnprocessableEntity, ef.Msg)
	case errors.As(err, &bv):
		httpx.Problem(w, http.StatusUnprocessableEntity, bv.Msg)
	case err != nil:
		h.log.Error("import failed", "source", source, "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	default:
		httpx.JSON(w, http.StatusOK, res)
	}
}

// What happened to one item.
type outcome int

const (
	skipped outcome = iota
	created
	conflict
)

// each runs one item in a savepoint, so a refused item does not undo the
// others; a problem is reported by its title.
func each(ctx context.Context, tx pgx.Tx, res *Result, title string, f func(pgx.Tx) (outcome, error)) error {
	if len(res.Sample) < 8 {
		res.Sample = append(res.Sample, title)
	}
	if res.DryRun {
		return nil
	}
	sp, err := tx.Begin(ctx)
	if err != nil {
		return err
	}
	got, err := f(sp)
	if err != nil {
		_ = sp.Rollback(ctx)
		var tv tasks.ValidationError
		var mv media.ValidationError
		if errors.As(err, &tv) || errors.As(err, &mv) || errors.Is(err, notes.ErrBadPath) || errors.Is(err, notes.ErrTooLarge) ||
			errors.Is(err, notes.ErrBadFilePath) || errors.Is(err, notes.ErrBadFile) || errors.Is(err, notes.ErrQuota) {
			if len(res.Problems) < 50 {
				res.Problems = append(res.Problems, fmt.Sprintf("%s: %s", title, err.Error()))
			}
			return nil
		}
		return err
	}
	switch got {
	case created:
		res.Created++
	case conflict:
		res.Conflicts++
	default:
		res.Skipped++
	}
	return sp.Commit(ctx)
}

func (h *Handlers) run(ctx context.Context, tx pgx.Tx, userID uuid.UUID, source string, data []byte, account *uuid.UUID, res *Result) error {
	switch source {
	case "todoist", "ticktick":
		parse := Todoist
		if source == "ticktick" {
			parse = TickTick
		}
		items, err := parse(data)
		if err != nil {
			return err
		}
		res.Found = len(items)
		for _, t := range items {
			if err := each(ctx, tx, res, t.Title, func(sp pgx.Tx) (outcome, error) { return h.task(ctx, sp, userID, t) }); err != nil {
				return err
			}
		}
	case "goodreads", "letterboxd":
		parse := Goodreads
		if source == "letterboxd" {
			parse = Letterboxd
		}
		items, err := parse(data)
		if err != nil {
			return err
		}
		res.Found = len(items)
		for _, m := range items {
			if err := each(ctx, tx, res, m.Title, func(sp pgx.Tx) (outcome, error) { return h.media(ctx, sp, userID, m) }); err != nil {
				return err
			}
		}
	case "firefly", "ofx":
		parse := Firefly
		if source == "ofx" {
			parse = OFX
		}
		rows, err := parse(data)
		if err != nil {
			return err
		}
		res.Found = len(rows)
		for i, row := range rows {
			if i < 8 {
				res.Sample = append(res.Sample, fmt.Sprintf("%s %s %.2f", row.On, row.Payee, float64(row.Cents)/100))
			}
		}
		if res.DryRun {
			return nil
		}
		got, err := budget.ImportRows(ctx, tx, userID, account, rows)
		res.Created, res.Skipped, res.Problems = got.Created, got.Skipped, got.Problems
		return err
	case "notion":
		pages, files, err := Notion(data)
		if err != nil {
			return err
		}
		res.Found = len(pages) + len(files)
		for _, f := range files {
			if err := each(ctx, tx, res, f.Path, func(sp pgx.Tx) (outcome, error) { return h.file(ctx, sp, userID, f) }); err != nil {
				return err
			}
		}
		var batch notes.Batch // links resolve once, after every page
		for _, n := range pages {
			if err := each(ctx, tx, res, n.Path, func(sp pgx.Tx) (outcome, error) { return h.note(ctx, sp, userID, &batch, n) }); err != nil {
				return err
			}
		}
		return batch.Finish(ctx, tx, userID, graph.UserActor)
	}
	return nil
}

func (h *Handlers) task(ctx context.Context, tx pgx.Tx, userID uuid.UUID, t Task) (outcome, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tasks k JOIN entities e ON e.id = k.id WHERE e.user_id = $1 AND e.title = $2
		AND k.due_on IS NOT DISTINCT FROM $3::date)`, userID, t.Title, t.Due).Scan(&exists); err != nil || exists {
		return skipped, err
	}
	status := "next"
	if t.Done {
		status = "done"
	}
	f := tasks.Fields{Title: &t.Title, Notes: &t.Notes, Status: &status, DueOn: t.Due}
	if t.Context != "" {
		f.Context = &t.Context
	}
	if _, err := tasks.Create(ctx, tx, userID, graph.UserActor, f); err != nil {
		return skipped, err
	}
	return created, nil
}

func (h *Handlers) media(ctx context.Context, tx pgx.Tx, userID uuid.UUID, m Media) (outcome, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM media_items i JOIN entities e ON e.id = i.id WHERE e.user_id = $1
		AND i.kind = $2 AND lower(e.title) = lower($3) AND coalesce(i.year, 0) = $4)`, userID, m.Kind, m.Title, m.Year).Scan(&exists); err != nil || exists {
		return skipped, err
	}
	f := media.Fields{Title: &m.Title, Kind: &m.Kind, Status: &m.Status}
	if m.Year > 0 {
		f.Year = &m.Year
	}
	if m.Creators != "" {
		f.Creators = &m.Creators
	}
	if m.Rating > 0 {
		f.Rating = &m.Rating
	}
	if m.Finished != "" {
		// The start is not in these exports; without one, Axon would take today.
		f.StartedOn, f.FinishedOn = &m.Finished, &m.Finished
	}
	if _, err := media.Create(ctx, tx, userID, graph.UserActor, f, nil, h.now()); err != nil {
		return skipped, err
	}
	return created, nil
}

func (h *Handlers) note(ctx context.Context, tx pgx.Tx, userID uuid.UUID, batch *notes.Batch, n Note) (outcome, error) {
	p, ok := notes.CleanPath(n.Path)
	if !ok {
		return skipped, notes.ErrBadPath
	}
	cur, err := notes.GetByPath(ctx, tx, userID, p)
	switch {
	case errors.Is(err, graph.ErrNotFound):
		if _, err := batch.Add(ctx, tx, userID, graph.UserActor, p, n.Content); err != nil {
			return skipped, err
		}
		return created, nil
	case err != nil:
		return skipped, err
	case cur.Content != nil && *cur.Content != n.Content:
		return conflict, nil
	}
	return skipped, nil
}

func (h *Handlers) file(ctx context.Context, tx pgx.Tx, userID uuid.UUID, f File) (outcome, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM note_files WHERE user_id = $1 AND path = $2)`, userID, f.Path).Scan(&exists); err != nil || exists {
		return skipped, err
	}
	if _, err := notes.PutFile(ctx, tx, userID, f.Path, f.Data); err != nil {
		return skipped, err
	}
	return created, nil
}
