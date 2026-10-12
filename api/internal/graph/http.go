package graph

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves entities, links, timelines, and search.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the graph endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/relations", h.relations)
	r.Get("/v1/entities/{id}", h.entity)
	r.Get("/v1/entities/{id}/links", h.links)
	r.Get("/v1/entities/{id}/timeline", h.timeline)
	r.Post("/v1/links", h.createLink)
	r.Delete("/v1/links/{id}", h.deleteLink)
	r.Get("/v1/search", h.search)
	r.Get("/v1/trash", h.trash)
	r.Delete("/v1/trash", h.empty)
	r.Post("/v1/trash/undo", h.undo)
	r.Post("/v1/trash/{id}/restore", h.restoreOne)
	r.Delete("/v1/trash/{id}", h.forget)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	h.log.Error("graph request failed", "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "Not found.")
	}
	return id, err == nil
}

func cursor(w http.ResponseWriter, r *http.Request, name string) (*uuid.UUID, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil, true
	}
	id, err := uuid.Parse(v)
	if err != nil {
		httpx.Problem(w, http.StatusBadRequest, "The ?"+name+"= cursor is not valid. Use next_cursor from the previous page.")
		return nil, false
	}
	return &id, true
}

func limit(w http.ResponseWriter, r *http.Request) (int, bool) {
	n, ok := httpx.Limit(r)
	if !ok {
		httpx.Problem(w, http.StatusBadRequest, "?limit= must be between 1 and 100.")
	}
	return n, ok
}

func page[T any](items []T, limit int, id func(T) uuid.UUID) httpx.List[T] {
	out := httpx.List[T]{Items: items}
	if out.Items == nil {
		out.Items = []T{}
	}
	if len(items) == limit {
		next := id(items[len(items)-1]).String()
		out.NextCursor = &next
	}
	return out
}

func (h *Handlers) relations(w http.ResponseWriter, _ *http.Request) {
	mu.RLock()
	list := make([]Relation, 0, len(relations))
	for _, r := range relations {
		r.Sources, r.Targets = orEmpty(r.Sources), orEmpty(r.Targets)
		list = append(list, r)
	}
	mu.RUnlock()
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (h *Handlers) entity(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	e, err := GetEntity(r.Context(), h.db, who.UserID, id)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "Not found.")
	case err != nil:
		h.fail(w, err)
	default:
		httpx.JSON(w, http.StatusOK, e)
	}
}

func (h *Handlers) links(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	after, ok := cursor(w, r, "after")
	if !ok {
		return
	}
	n, ok := limit(w, r)
	if !ok {
		return
	}
	if _, err := GetEntity(r.Context(), h.db, who.UserID, id); errors.Is(err, ErrNotFound) {
		httpx.Problem(w, http.StatusNotFound, "Not found.")
		return
	}
	list := Outgoing
	switch r.URL.Query().Get("direction") {
	case "", "outgoing":
	case "incoming":
		list = Incoming
	default:
		httpx.Problem(w, http.StatusBadRequest, `?direction= must be "outgoing" or "incoming".`)
		return
	}
	items, err := list(r.Context(), h.db, who.UserID, id, Page{After: after, Limit: n})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page(items, n, func(l Link) uuid.UUID { return l.ID }))
}

func (h *Handlers) timeline(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	before, ok := cursor(w, r, "before")
	if !ok {
		return
	}
	n, ok := limit(w, r)
	if !ok {
		return
	}
	items, err := Timeline(r.Context(), h.db, who.UserID, id, before, n)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page(items, n, func(e Event) uuid.UUID { return e.ID }))
}

func (h *Handlers) createLink(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		SourceID uuid.UUID `json:"source_id"`
		Relation string    `json:"relation"`
		TargetID uuid.UUID `json:"target_id"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var id uuid.UUID
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		id, err = CreateLink(r.Context(), tx, who.UserID, UserActor, body.SourceID, body.Relation, body.TargetID)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "One of the linked items does not exist.")
	case errors.Is(err, ErrRelationRejected):
		httpx.Problem(w, http.StatusUnprocessableEntity, "That relation is not allowed between these kinds of items. See GET /v1/relations.")
	case errors.Is(err, ErrDuplicate):
		httpx.Problem(w, http.StatusConflict, "These items are already linked that way.")
	case err != nil:
		h.fail(w, err)
	default:
		httpx.JSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
	}
}

func (h *Handlers) deleteLink(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return DeleteLink(r.Context(), tx, who.UserID, UserActor, id)
	})
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "Not found.")
	case err != nil:
		h.fail(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	q := r.URL.Query().Get("q")
	if q == "" || len(q) > 500 {
		httpx.Problem(w, http.StatusBadRequest, "?q= must be 1 to 500 characters.")
		return
	}
	n, ok := limit(w, r)
	if !ok {
		return
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		var err error
		if offset, err = strconv.Atoi(v); err != nil || offset < 0 || offset > 1000 {
			httpx.Problem(w, http.StatusBadRequest, "?offset= must be between 0 and 1000.")
			return
		}
	}
	typ := r.URL.Query().Get("type")
	if len(typ) > 40 {
		httpx.Problem(w, http.StatusBadRequest, "?type= is an entity type such as note or task.")
		return
	}
	hits, err := Search(r.Context(), h.db, who.UserID, q, offset, n, SearchOptions{Type: typ})
	if err != nil {
		h.fail(w, err)
		return
	}
	if hits == nil {
		hits = []Hit{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": hits})
}

// --- Trash (EP-66) ---

func (h *Handlers) trash(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if err := PurgeTrash(r.Context(), h.db, time.Now()); err != nil {
		h.fail(w, err)
		return
	}
	items, err := TrashList(r.Context(), h.db, who.UserID, time.Now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) restore(w http.ResponseWriter, r *http.Request, find func(pgx.Tx, uuid.UUID) (uuid.UUID, error)) {
	who, _ := auth.FromContext(r.Context())
	var id uuid.UUID
	var typ string
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		trashID, err := find(tx, who.UserID)
		if err != nil {
			return err
		}
		id, typ, err = Restore(r.Context(), tx, who.UserID, trashID)
		return err
	})
	var er ErrRestore
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "Not in the trash (it keeps deletions for 30 days).")
	case errors.As(err, &er):
		httpx.Problem(w, http.StatusConflict, er.Msg)
	case err != nil:
		h.fail(w, err)
	default:
		httpx.JSON(w, http.StatusOK, map[string]any{"entity_id": id, "type": typ})
	}
}

func (h *Handlers) restoreOne(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	h.restore(w, r, func(_ pgx.Tx, _ uuid.UUID) (uuid.UUID, error) {
		if err != nil {
			return uuid.Nil, ErrNotFound
		}
		return id, nil
	})
}

// undo restores the latest deletion of an entity: what an Undo button needs.
func (h *Handlers) undo(w http.ResponseWriter, r *http.Request) {
	var b struct {
		EntityID uuid.UUID `json:"entity_id"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	h.restore(w, r, func(tx pgx.Tx, userID uuid.UUID) (uuid.UUID, error) {
		var id uuid.UUID
		err := tx.QueryRow(r.Context(), `SELECT id FROM trash WHERE user_id = $1 AND entity_id = $2 ORDER BY seq DESC LIMIT 1`, userID, b.EntityID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return id, ErrNotFound
		}
		return id, err
	})
}

func (h *Handlers) forget(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "Not in the trash.")
		return
	}
	tag, err := h.db.Exec(r.Context(), `DELETE FROM trash WHERE user_id = $1 AND batch = (SELECT batch FROM trash WHERE id = $2 AND user_id = $1)`, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Problem(w, http.StatusNotFound, "Not in the trash.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) empty(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if _, err := h.db.Exec(r.Context(), `DELETE FROM trash WHERE user_id = $1`, who.UserID); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
