package repos

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers manages repository connections. They hold tokens, so the caller
// wraps them in auth.RequireSession: no script can add or read them.
type Handlers struct {
	db     *pgxpool.Pool
	log    *slog.Logger
	sealer *auth.Sealer
}

// NewHandlers returns the connection endpoints; sealer may be nil (then
// only public repositories, without a token, can be connected).
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer) *Handlers {
	return &Handlers{db: pool, log: log, sealer: sealer}
}

// Routes mounts the endpoints.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/repos", h.list)
	r.With(auth.NotDemo).Post("/v1/repos", h.create)
	r.With(auth.NotDemo).Patch("/v1/repos/{id}", h.update)
	r.Delete("/v1/repos/{id}", h.delete)
}

var (
	ownerRE = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}$`)
	repoRE  = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
)

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, graph.ErrNotFound) {
		httpx.Problem(w, http.StatusNotFound, "No such repository connection.")
		return
	}
	h.log.Error("repos request failed", "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := List(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "can_store_tokens": h.sealer != nil})
}

// projectOK checks a project id belongs to the user.
func (h *Handlers) projectOK(r *http.Request, userID uuid.UUID, id uuid.UUID) bool {
	var ok bool
	_ = h.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM projects p JOIN entities e ON e.id = p.id WHERE p.id = $1 AND e.user_id = $2)`, id, userID).Scan(&ok)
	return ok
}

func (h *Handlers) seal(token string) ([]byte, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, true
	}
	if h.sealer == nil || len(token) > 500 {
		return nil, false
	}
	return h.sealer.Seal([]byte(token)), true
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Owner         string     `json:"owner"`
		Repo          string     `json:"repo"`
		Token         string     `json:"token"`
		AllowComments bool       `json:"allow_comments"`
		ProjectID     *uuid.UUID `json:"project_id"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if !ownerRE.MatchString(body.Owner) || !repoRE.MatchString(body.Repo) {
		httpx.Problem(w, http.StatusUnprocessableEntity, "Give the repository as owner and name, like thoutmose and axon.")
		return
	}
	if body.ProjectID != nil && !h.projectOK(r, who.UserID, *body.ProjectID) {
		httpx.Problem(w, http.StatusUnprocessableEntity, "Choose one of your projects for drafted issues.")
		return
	}
	sealed, ok := h.seal(body.Token)
	if !ok {
		httpx.Problem(w, http.StatusUnprocessableEntity, "This server cannot store tokens without AXON_ENCRYPTION_KEY, or the token is too long.")
		return
	}
	id := uuid.Must(uuid.NewV7())
	_, err := h.db.Exec(r.Context(), `INSERT INTO repo_connections (id, user_id, owner, repo, token_sealed, allow_comments, project_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, who.UserID, body.Owner, body.Repo, sealed, body.AllowComments, body.ProjectID)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		httpx.Problem(w, http.StatusConflict, "That repository is already connected.")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	c, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, graph.ErrNotFound)
		return
	}
	cur, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	var raw map[string]json.RawMessage
	if err := httpx.Decode(r, &raw); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	allow, project := cur.AllowComments, cur.ProjectID
	var token *string
	for k, v := range raw {
		var err error
		switch k {
		case "allow_comments":
			err = json.Unmarshal(v, &allow)
		case "token":
			err = json.Unmarshal(v, &token)
		case "project_id":
			project = nil
			if string(v) != "null" {
				var p uuid.UUID
				if err = json.Unmarshal(v, &p); err == nil {
					if !h.projectOK(r, who.UserID, p) {
						httpx.Problem(w, http.StatusUnprocessableEntity, "Choose one of your projects for drafted issues.")
						return
					}
					project = &p
				}
			}
		default:
			httpx.Problem(w, http.StatusBadRequest, "Send allow_comments, token, or project_id.")
			return
		}
		if err != nil {
			httpx.Problem(w, http.StatusBadRequest, k+" has the wrong type.")
			return
		}
	}
	if token != nil {
		sealed, ok := h.seal(*token)
		if !ok {
			httpx.Problem(w, http.StatusUnprocessableEntity, "This server cannot store tokens without AXON_ENCRYPTION_KEY, or the token is too long.")
			return
		}
		if _, err := h.db.Exec(r.Context(), `UPDATE repo_connections SET token_sealed = $2 WHERE id = $1`, id, sealed); err != nil {
			h.fail(w, err)
			return
		}
	}
	if _, err := h.db.Exec(r.Context(), `UPDATE repo_connections SET allow_comments = $2, project_id = $3 WHERE id = $1`, id, allow, project); err != nil {
		h.fail(w, err)
		return
	}
	c, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.fail(w, graph.ErrNotFound)
		return
	}
	tag, err := h.db.Exec(r.Context(), `DELETE FROM repo_connections WHERE id = $1 AND user_id = $2`, id, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		h.fail(w, graph.ErrNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
