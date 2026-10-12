package household

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the household.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the household endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/household", h.get)
	r.Post("/v1/household", h.create)
	r.Patch("/v1/household", h.rename)
	r.Delete("/v1/household/me", h.leave)
	r.Post("/v1/household/invites", h.invite)
	r.Post("/v1/household/join", h.join)
	r.Delete("/v1/household/members/{id}", h.remove)
	r.Post("/v1/household/lists", h.createList)
	r.Patch("/v1/household/lists/{id}", h.renameList)
	r.Delete("/v1/household/lists/{id}", h.deleteList)
	r.Post("/v1/household/lists/{id}/items", h.addItem)
	r.Patch("/v1/household/items/{id}", h.updateItem)
	r.Delete("/v1/household/items/{id}", h.deleteItem)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, ErrNone):
		httpx.Problem(w, http.StatusNotFound, "You are not in a household. Start one, or join with a code from a member.")
	case errors.Is(err, ErrAlready):
		httpx.Problem(w, http.StatusConflict, "You are already in a household; leave it first.")
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "Not found in your household.")
	case errors.Is(err, ErrNotOwner):
		httpx.Problem(w, http.StatusForbidden, "Only the household's owner can remove members.")
	case errors.Is(err, ErrCode):
		httpx.Problem(w, http.StatusUnprocessableEntity, "That invite code is not valid, was used, or has expired. Ask for a new one.")
	default:
		h.log.Error("household request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func (h *Handlers) show(w http.ResponseWriter, r *http.Request, status int) {
	who, _ := auth.FromContext(r.Context())
	hh, err := Get(r.Context(), h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, status, hh)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) { h.show(w, r, http.StatusOK) }

type nameBody struct {
	Name string `json:"name"`
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b nameBody
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return Create(r.Context(), tx, who.UserID, b.Name) }); err != nil {
		h.fail(w, err)
		return
	}
	h.show(w, r, http.StatusCreated)
}

func (h *Handlers) rename(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b nameBody
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := Rename(r.Context(), h.db, who.UserID, b.Name); err != nil {
		h.fail(w, err)
		return
	}
	h.show(w, r, http.StatusOK)
}

func (h *Handlers) leave(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return Leave(r.Context(), tx, who.UserID) }); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) invite(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	code, expires, err := Invite(r.Context(), h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"code": code, "expires_at": expires})
}

func (h *Handlers) join(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		Code string `json:"code"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return Join(r.Context(), tx, who.UserID, b.Code, h.now()) }); err != nil {
		h.fail(w, err)
		return
	}
	h.show(w, r, http.StatusOK)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id, ErrNotFound
	}
	return id, nil
}

func (h *Handlers) remove(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	member, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return Remove(r.Context(), tx, who.UserID, member) }); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) createList(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		Title string `json:"title"`
		Kind  string `json:"kind"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := CreateList(r.Context(), h.db, who.UserID, b.Title, b.Kind); err != nil {
		h.fail(w, err)
		return
	}
	h.show(w, r, http.StatusCreated)
}

func (h *Handlers) renameList(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var b struct {
		Title string `json:"title"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := RenameList(r.Context(), h.db, who.UserID, id, b.Title); err != nil {
		h.fail(w, err)
		return
	}
	h.show(w, r, http.StatusOK)
}

func (h *Handlers) deleteList(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := DeleteList(r.Context(), h.db, who.UserID, id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) addItem(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var f ItemFields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := AddItem(r.Context(), h.db, who.UserID, id, f); err != nil {
		h.fail(w, err)
		return
	}
	h.show(w, r, http.StatusCreated)
}

func (h *Handlers) updateItem(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var f ItemFields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := UpdateItem(r.Context(), h.db, who.UserID, id, f, h.now()); err != nil {
		h.fail(w, err)
		return
	}
	h.show(w, r, http.StatusOK)
}

func (h *Handlers) deleteItem(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := DeleteItem(r.Context(), h.db, who.UserID, id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
