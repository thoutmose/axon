package roadmaps

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves roadmaps and skills.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the roadmap endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/roadmaps", h.list)
	r.Post("/v1/roadmaps", h.create)
	r.Get("/v1/roadmaps/sources", h.sources)
	r.Post("/v1/roadmaps/import", h.importRoadmap)
	r.Get("/v1/roadmaps/{id}", h.get)
	r.Patch("/v1/roadmaps/{id}", h.update)
	r.Delete("/v1/roadmaps/{id}", h.delete)
	r.Post("/v1/roadmaps/{id}/skills", h.addSkill)
	r.Patch("/v1/skills/{id}", h.updateSkill)
	r.Delete("/v1/skills/{id}", h.deleteSkill)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such roadmap or skill.")
	case errors.Is(err, graph.ErrDuplicate):
		httpx.Problem(w, http.StatusConflict, "That prerequisite already exists.")
	default:
		h.log.Error("roadmaps request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func id(r *http.Request) (uuid.UUID, error) {
	v, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return v, graph.ErrNotFound
	}
	return v, nil
}

// tx runs f in a transaction and writes its result.
func (h *Handlers) tx(w http.ResponseWriter, r *http.Request, status int, f func(tx pgx.Tx) (any, error)) {
	var out any
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		out, err = f(tx)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	if out == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	httpx.JSON(w, status, out)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := List(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) sources(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"items": Sources()})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var f struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	h.tx(w, r, http.StatusCreated, func(tx pgx.Tx) (any, error) {
		return Create(r.Context(), tx, who.UserID, graph.UserActor, Fields{f.Title, f.Description})
	})
}

func (h *Handlers) importRoadmap(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Source string `json:"source"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	h.tx(w, r, http.StatusCreated, func(tx pgx.Tx) (any, error) {
		return Import(r.Context(), tx, who.UserID, graph.UserActor, body.Source)
	})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	rm, err := Get(r.Context(), h.db, who.UserID, rid)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, rm)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var f struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	h.tx(w, r, http.StatusOK, func(tx pgx.Tx) (any, error) {
		return Update(r.Context(), tx, who.UserID, rid, graph.UserActor, Fields{f.Title, f.Description})
	})
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.tx(w, r, 0, func(tx pgx.Tx) (any, error) {
		return nil, Delete(r.Context(), tx, who.UserID, rid, graph.UserActor)
	})
}

type skillBody struct {
	Title       *string     `json:"title"`
	Description *string     `json:"description"`
	URL         *string     `json:"url"`
	Section     *string     `json:"section"`
	Status      *string     `json:"status"`
	Kind        *string     `json:"kind"`
	Position    *int        `json:"position"`
	Requires    []uuid.UUID `json:"requires"`
}

func (b skillBody) fields() SkillFields {
	return SkillFields(b)
}

func (h *Handlers) addSkill(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var b skillBody
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	h.tx(w, r, http.StatusCreated, func(tx pgx.Tx) (any, error) {
		return AddSkill(r.Context(), tx, who.UserID, rid, graph.UserActor, b.fields())
	})
}

func (h *Handlers) updateSkill(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	sid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var raw map[string]json.RawMessage
	if err := httpx.Decode(r, &raw); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := raw["requires"]; ok {
		httpx.Problem(w, http.StatusBadRequest, "Change prerequisites with /v1/links (relation requires).")
		return
	}
	b, _ := json.Marshal(raw)
	var body skillBody
	_ = json.Unmarshal(b, &body)
	h.tx(w, r, http.StatusOK, func(tx pgx.Tx) (any, error) {
		return UpdateSkill(r.Context(), tx, who.UserID, sid, graph.UserActor, body.fields())
	})
}

func (h *Handlers) deleteSkill(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	sid, err := id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.tx(w, r, 0, func(tx pgx.Tx) (any, error) {
		return nil, DeleteSkill(r.Context(), tx, who.UserID, sid, graph.UserActor)
	})
}
