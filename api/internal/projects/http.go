package projects

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves projects and boards.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the project endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/projects", h.list)
	r.Post("/v1/projects", h.create)
	r.Get("/v1/projects/{id}", h.get)
	r.Patch("/v1/projects/{id}", h.update)
	r.Delete("/v1/projects/{id}", h.delete)
	r.Get("/v1/projects/{id}/board", h.board)
	r.Get("/v1/projects/{id}/docs", h.docs)
	r.Post("/v1/projects/{id}/cards", h.addCard)
	r.Put("/v1/projects/{id}/cards/{task}", h.placeCard)
	r.Delete("/v1/cards/{task}", h.removeCard)
	r.Get("/v1/tasks/{id}/card", h.taskCard)
}

// TaskCard is where a task sits on a project's board (owner request: a task
// opened from the calendar shows its project, its parent, and its subtasks).
type TaskCard struct {
	Project  CardRef   `json:"project"`
	Column   string    `json:"column"`
	Kind     string    `json:"kind"`
	Parent   *CardRef  `json:"parent"`
	Children []CardRef `json:"children"`
}

// CardRef names a project, or a task with its status.
type CardRef struct {
	ID     uuid.UUID `json:"id"`
	Title  string    `json:"title"`
	Status string    `json:"status,omitempty"`
}

// taskCard answers {"card": null} for a task on no board.
func (h *Handlers) taskCard(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuidParam(r, "id")
	if err != nil {
		h.fail(w, err)
		return
	}
	var c TaskCard
	var parent *uuid.UUID
	err = h.db.QueryRow(r.Context(), `SELECT c.project_id, p.title, c.col, c.kind, c.parent_id
		FROM project_cards c JOIN entities p ON p.id = c.project_id JOIN entities t ON t.id = c.task_id
		WHERE c.task_id = $1 AND t.user_id = $2`, id, who.UserID).Scan(&c.Project.ID, &c.Project.Title, &c.Column, &c.Kind, &parent)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, map[string]any{"card": nil})
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if parent != nil {
		c.Parent = &CardRef{ID: *parent}
		if err := h.db.QueryRow(r.Context(), `SELECT e.title, t.status FROM entities e JOIN tasks t ON t.id = e.id WHERE e.id = $1`, *parent).
			Scan(&c.Parent.Title, &c.Parent.Status); err != nil {
			h.fail(w, err)
			return
		}
	}
	rows, err := h.db.Query(r.Context(), `SELECT e.id, e.title, t.status FROM project_cards c JOIN entities e ON e.id = c.task_id
		JOIN tasks t ON t.id = c.task_id WHERE c.parent_id = $1 ORDER BY c.position`, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if c.Children, err = pgx.CollectRows(rows, pgx.RowToStructByPos[CardRef]); err != nil {
		h.fail(w, err)
		return
	}
	if c.Children == nil {
		c.Children = []CardRef{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"card": c})
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such project, task, or card.")
	case errors.Is(err, ErrStale):
		httpx.Problem(w, http.StatusPreconditionFailed, "This project changed since you loaded it. Reload and try again.")
	case errors.Is(err, ErrOverLimit):
		count := strings.TrimPrefix(err.Error(), ErrOverLimit.Error()+": ")
		httpx.Problem(w, http.StatusConflict, fmt.Sprintf("Doing already holds %s cards, its WIP limit. Send over_limit to move it anyway.", count))
	default:
		h.log.Error("projects request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func uuidParam(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		return id, graph.ErrNotFound
	}
	return id, nil
}

func decodeFields(r *http.Request) (Fields, error) {
	var raw map[string]json.RawMessage
	if err := httpx.Decode(r, &raw); err != nil {
		return Fields{}, ValidationError{err.Error()}
	}
	var f Fields
	for k, v := range raw {
		var err error
		switch k {
		case "title":
			err = json.Unmarshal(v, &f.Title)
		case "description":
			err = json.Unmarshal(v, &f.Description)
		case "status":
			err = json.Unmarshal(v, &f.Status)
		case "docs_folder":
			err = json.Unmarshal(v, &f.DocsFolder)
		case "wip_limit":
			if string(v) == "null" {
				f.ClearWIPLimit = true
			} else {
				err = json.Unmarshal(v, &f.WIPLimit)
			}
		default:
			return Fields{}, ValidationError{fmt.Sprintf("Unknown field %q.", k)}
		}
		if err != nil {
			return Fields{}, ValidationError{fmt.Sprintf("%s has the wrong type.", k)}
		}
	}
	return f, nil
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}
	if !slices.Contains(statuses, status) {
		httpx.Problem(w, http.StatusBadRequest, "?status= must be active, on_hold, done, or archived.")
		return
	}
	items, err := List(r.Context(), h.db, who.UserID, status)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := decodeFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var p Project
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		p, err = Create(r.Context(), tx, who.UserID, graph.UserActor, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(p.UpdatedAt))
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuidParam(r, "id")
	if err != nil {
		h.fail(w, err)
		return
	}
	p, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(p.UpdatedAt))
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuidParam(r, "id")
	if err != nil {
		h.fail(w, err)
		return
	}
	version, ok := httpx.IfMatch(w, r)
	if !ok {
		return
	}
	f, err := decodeFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var p Project
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		p, err = Update(r.Context(), tx, who.UserID, id, graph.UserActor, version, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(p.UpdatedAt))
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuidParam(r, "id")
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return Delete(r.Context(), tx, who.UserID, id, graph.UserActor)
	}); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) board(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuidParam(r, "id")
	if err != nil {
		h.fail(w, err)
		return
	}
	b, err := GetBoard(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (h *Handlers) docs(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuidParam(r, "id")
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := Docs(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type cardBody struct {
	Title     *string         `json:"title"`
	Column    *string         `json:"column"`
	Kind      *string         `json:"kind"`
	ParentID  json.RawMessage `json:"parent_id"`
	Before    *uuid.UUID      `json:"before"`
	OverLimit bool            `json:"over_limit"`
}

func (b cardBody) input() (CardInput, error) {
	in := CardInput{Column: b.Column, Kind: b.Kind, Before: b.Before, OverLimit: b.OverLimit}
	switch string(b.ParentID) {
	case "":
	case "null":
		in.NoParent = true
	default:
		var id uuid.UUID
		if err := json.Unmarshal(b.ParentID, &id); err != nil {
			return in, ValidationError{"parent_id is a card's task id or null."}
		}
		in.ParentID = &id
	}
	return in, nil
}

func (h *Handlers) addCard(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuidParam(r, "id")
	if err != nil {
		h.fail(w, err)
		return
	}
	var body cardBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Title == nil {
		h.fail(w, ValidationError{"A card needs a title."})
		return
	}
	in, err := body.input()
	if err != nil {
		h.fail(w, err)
		return
	}
	var c Card
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		c, err = AddCard(r.Context(), tx, who.UserID, id, graph.UserActor, *body.Title, in)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handlers) placeCard(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuidParam(r, "id")
	if err != nil {
		h.fail(w, err)
		return
	}
	task, err := uuidParam(r, "task")
	if err != nil {
		h.fail(w, err)
		return
	}
	var body cardBody
	if err := httpx.Decode(r, &body); err != nil || body.Title != nil {
		httpx.Problem(w, http.StatusBadRequest, "Send column, kind, parent_id, before, or over_limit.")
		return
	}
	in, err := body.input()
	if err != nil {
		h.fail(w, err)
		return
	}
	var c Card
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		c, err = Place(r.Context(), tx, who.UserID, id, graph.UserActor, task, in)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) removeCard(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	task, err := uuidParam(r, "task")
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return RemoveCard(r.Context(), tx, who.UserID, task)
	}); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
