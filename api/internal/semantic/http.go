package semantic

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves search by meaning.
type Handlers struct {
	db     *pgxpool.Pool
	log    *slog.Logger
	sealer *auth.Sealer
}

// NewHandlers returns the endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer) *Handlers {
	return &Handlers{db: pool, log: log, sealer: sealer}
}

// Routes mounts them; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/me/embeddings", h.status)
	r.Put("/v1/me/embeddings", h.save)
	r.Delete("/v1/me/embeddings", h.remove)
	r.Get("/v1/notes/semantic", h.search)
	r.Get("/v1/notes/{id}/similar", h.similar)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	default:
		h.log.Error("semantic request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// Status says what is set up and how far indexing has come.
type Status struct {
	Settings *Settings `json:"settings"`
	Consent  bool      `json:"consent"`
	Indexed  int       `json:"indexed"`
	Notes    int       `json:"notes"`
}

func (h *Handlers) status(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	s, consent, err := Ready(r.Context(), h.db, h.sealer, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := Status{Settings: s, Consent: consent}
	model := ""
	if s != nil {
		model = s.modelKey()
	}
	if err := h.db.QueryRow(r.Context(), `SELECT
		(SELECT count(*) FROM note_embeddings WHERE user_id = $1 AND model = $2),
		(SELECT count(*) FROM notes n WHERE n.user_id = $1 AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = n.id))`,
		who.UserID, model).Scan(&out.Indexed, &out.Notes); err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handlers) save(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var in Input
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return Save(r.Context(), tx, h.sealer, who.UserID, in) }); err != nil {
		h.fail(w, err)
		return
	}
	h.status(w, r)
}

func (h *Handlers) remove(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return Remove(r.Context(), tx, who.UserID) }); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Hit is a note found by meaning.
type Hit struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Path  string    `json:"path"`
	Score float64   `json:"score"` // cosine similarity, 1 is the same meaning
}

const nearest = `SELECT n.id, e.title, n.path, 1 - (v.embedding <=> $2::vector) FROM note_embeddings v
	JOIN notes n ON n.id = v.note_id JOIN entities e ON e.id = n.id
	WHERE v.user_id = $1 AND v.model = $3 AND n.id <> coalesce($5, '00000000-0000-0000-0000-000000000000'::uuid)
	ORDER BY v.embedding <=> $2::vector LIMIT $4`

func (h *Handlers) ready(w http.ResponseWriter, r *http.Request) (*Settings, bool) {
	who, _ := auth.FromContext(r.Context())
	s, consent, err := Ready(r.Context(), h.db, h.sealer, who.UserID)
	switch {
	case err != nil:
		h.fail(w, err)
	case s == nil:
		httpx.Problem(w, http.StatusConflict, "Search by meaning is off: choose an embedding service in Preferences.")
	case !consent:
		httpx.Problem(w, http.StatusForbidden, "Search by meaning needs your consent to send notes to the embedding service (Privacy).")
	default:
		return s, true
	}
	return nil, false
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" || len([]rune(q)) > 500 {
		httpx.Problem(w, http.StatusBadRequest, "Send ?q= with up to 500 characters.")
		return
	}
	s, ok := h.ready(w, r)
	if !ok {
		return
	}
	vecs, err := s.Embed(r.Context(), []string{q}, "query")
	if err != nil {
		httpx.Problem(w, http.StatusBadGateway, "The embedding service did not answer: "+err.Error()+".")
		return
	}
	h.hits(w, r, who.UserID, literal(vecs[0]), s.modelKey(), nil)
}

func (h *Handlers) similar(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such note.")
		return
	}
	s, ok := h.ready(w, r)
	if !ok {
		return
	}
	var vec string
	err = h.db.QueryRow(r.Context(), `SELECT embedding::text FROM note_embeddings WHERE note_id = $1 AND user_id = $2 AND model = $3`,
		id, who.UserID, s.modelKey()).Scan(&vec)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, map[string]any{"items": []Hit{}, "indexed": false})
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.hits(w, r, who.UserID, vec, s.modelKey(), &id)
}

func (h *Handlers) hits(w http.ResponseWriter, r *http.Request, userID uuid.UUID, vec, model string, except *uuid.UUID) {
	rows, err := h.db.Query(r.Context(), nearest, userID, vec, model, 10, except)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Hit])
	if err != nil {
		h.fail(w, err)
		return
	}
	if items == nil {
		items = []Hit{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "indexed": true})
}
