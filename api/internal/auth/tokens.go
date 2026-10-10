package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/audit"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Personal access tokens look like "axon_<43 base64url characters>"; the
// prefix makes leaked tokens easy to find with secret scanners.
const tokenPrefix = "axon_"

func (s *Service) identifyToken(r *http.Request, token string) (Identity, error) {
	if !strings.HasPrefix(token, tokenPrefix) {
		return Identity{}, errNotFound
	}
	var id Identity
	var lastUsed *time.Time
	err := s.db.QueryRow(r.Context(),
		`SELECT id, user_id, scope, last_used_at FROM api_tokens WHERE token_hash = $1 AND expires_at > now()`,
		hashToken(token)).Scan(&id.TokenID, &id.UserID, &id.Scope, &lastUsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, errNotFound
	}
	if err != nil {
		return Identity{}, err
	}
	if lastUsed == nil || time.Since(*lastUsed) > 5*time.Minute {
		_, _ = s.db.Exec(r.Context(), `UPDATE api_tokens SET last_used_at = now() WHERE id = $1`, id.TokenID)
	}
	return id, nil
}

type tokenInfo struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Scope      string     `json:"scope"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

func (s *Service) listTokens(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, scope, created_at, last_used_at, expires_at FROM api_tokens
		 WHERE user_id = $1 ORDER BY id LIMIT 100`, id.UserID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[tokenInfo])
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *Service) createToken(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	var body struct {
		Name          string `json:"name"`
		Scope         string `json:"scope"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	name, ok := cleanName(body.Name)
	switch {
	case !ok:
		httpx.Problem(w, http.StatusUnprocessableEntity, "Enter a token name between 1 and 100 characters.")
		return
	case body.Scope != "read" && body.Scope != "write":
		httpx.Problem(w, http.StatusUnprocessableEntity, `Scope must be "read" or "write".`)
		return
	case body.ExpiresInDays < 1 || body.ExpiresInDays > 365:
		httpx.Problem(w, http.StatusUnprocessableEntity, "Tokens expire after 1 to 365 days.")
		return
	}
	raw, _ := newToken()
	token := tokenPrefix + raw
	hash := hashToken(token)
	info := tokenInfo{
		ID: uuid.Must(uuid.NewV7()), Name: name, Scope: body.Scope,
		CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Duration(body.ExpiresInDays) * 24 * time.Hour),
	}
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(r.Context(),
			`INSERT INTO api_tokens (id, user_id, name, token_hash, scope, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			info.ID, id.UserID, info.Name, hash, info.Scope, info.ExpiresAt)
		if err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.TokenCreated, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"name": name, "scope": body.Scope})
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"token": token, "info": info})
}

func (s *Service) revokeToken(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	target, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such token.")
		return
	}
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var name string
		err := tx.QueryRow(r.Context(), `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2 RETURNING name`, target, id.UserID).Scan(&name)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNotFound
		}
		if err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.TokenRevoked, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"name": name})
	})
	switch {
	case errors.Is(err, errNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such token.")
	case err != nil:
		s.fail(w, r, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
