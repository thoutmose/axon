package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

const (
	sessionCookie  = "axon_session"
	ceremonyCookie = "axon_ceremony"
)

// Identity is the signed-in caller, available to handlers behind Require.
// Exactly one of SessionID (browser) and TokenID (personal access token) is set.
type Identity struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	TokenID   uuid.UUID
	Scope     string // "session", or the token's "read" / "write"
}

type ctxKey struct{}

// FromContext returns the caller's identity; ok is false for anonymous requests.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}

func newToken() (token string, hash []byte) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(token))
	return token, h[:]
}

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// startSession creates a session in q (the caller's transaction) and sets its
// cookie. A new token is issued on every sign-in, so a pre-set cookie can
// never become an authenticated session (no session fixation).
func (s *Service) startSession(ctx context.Context, q db.Querier, w http.ResponseWriter, r *http.Request, userID uuid.UUID) error {
	token, hash := newToken()
	expires := time.Now().Add(SessionAbsolute)
	_, err := q.Exec(ctx,
		`INSERT INTO sessions (id, user_id, token_hash, user_agent, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		uuid.Must(uuid.NewV7()), userID, hash, truncate(r.UserAgent(), 300), expires)
	if err != nil {
		return err
	}
	s.setCookie(w, sessionCookie, token, expires)
	return nil
}

func (s *Service) setCookie(w http.ResponseWriter, name, value string, expires time.Time) {
	// Secure follows AXON_BASE_URL: on for HTTPS deployments, off only for
	// plain-HTTP localhost development, where browsers would drop the cookie.
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // see comment above
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie signs the browser out (used after account erasure).
func (s *Service) ClearSessionCookie(w http.ResponseWriter) { s.clearCookie(w, sessionCookie) }

func (s *Service) clearCookie(w http.ResponseWriter, name string) {
	s.setCookie(w, name, "", time.Unix(0, 0))
}

// bearer returns the token from an "Authorization: Bearer" header.
func bearer(r *http.Request) (string, bool) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return token, ok
}

// identify resolves a personal access token or, without one, the session
// cookie. A request with an Authorization header never falls back to the cookie.
func (s *Service) identify(r *http.Request) (Identity, error) {
	if token, ok := bearer(r); ok {
		return s.identifyToken(r, token)
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return Identity{}, errNotFound
	}
	var id Identity
	var lastSeen time.Time
	err = s.db.QueryRow(r.Context(),
		`SELECT id, user_id, last_seen_at FROM sessions WHERE token_hash = $1 AND expires_at > now()`,
		hashToken(c.Value)).Scan(&id.SessionID, &id.UserID, &lastSeen)
	if errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, errNotFound
	}
	if err != nil {
		return Identity{}, err
	}
	if time.Since(lastSeen) > SessionIdle {
		_, _ = s.db.Exec(r.Context(), `DELETE FROM sessions WHERE id = $1`, id.SessionID)
		return Identity{}, errNotFound
	}
	if time.Since(lastSeen) > 5*time.Minute { // avoid a write on every request
		_, _ = s.db.Exec(r.Context(), `UPDATE sessions SET last_seen_at = now() WHERE id = $1`, id.SessionID)
	}
	id.Scope = "session"
	return id, nil
}

// Require accepts a browser session or a personal access token; read-only
// tokens may only make safe (GET/HEAD) requests.
func (s *Service) Require(next http.Handler) http.Handler {
	return s.require(next, false)
}

// RequireSession accepts browser sessions only. Account security, export, and
// erasure must not be reachable with a token that a script could leak.
func (s *Service) RequireSession(next http.Handler) http.Handler {
	return s.require(next, true)
}

func (s *Service) require(next http.Handler, sessionOnly bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := s.identify(r)
		if err != nil {
			if !errors.Is(err, errNotFound) {
				s.log.Error("session lookup failed", "err", err)
				httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
				return
			}
			if _, isToken := bearer(r); isToken {
				httpx.Problem(w, http.StatusUnauthorized, "The access token is invalid, expired, or revoked.")
				return
			}
			s.clearCookie(w, sessionCookie)
			httpx.Problem(w, http.StatusUnauthorized, "Sign in to continue.")
			return
		}
		if sessionOnly && id.Scope != "session" {
			httpx.Problem(w, http.StatusForbidden, "Access tokens cannot manage the account. Use the web app.")
			return
		}
		if id.Scope == "read" && r.Method != http.MethodGet && r.Method != http.MethodHead {
			httpx.Problem(w, http.StatusForbidden, "This access token is read-only.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}

// SameOrigin blocks cross-site state-changing requests (CSRF). Browsers send
// Origin on every non-GET fetch; requests without it must at least declare
// themselves same-origin through Sec-Fetch-Site. Token requests are exempt:
// a cross-site page cannot attach an Authorization header (Axon allows no
// CORS), and identify never falls back to the cookie when one is present.
func (s *Service) SameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, isToken := bearer(r)
		switch {
		case isToken, r.Method == http.MethodGet, r.Method == http.MethodHead, r.Method == http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		ok := origin == s.origin || (origin == "" && r.Header.Get("Sec-Fetch-Site") == "same-origin")
		if !ok {
			httpx.Problem(w, http.StatusForbidden, "Cross-site requests are not allowed.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
