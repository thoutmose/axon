package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/audit"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Sign in with another account (EP-72), with the authorization code flow:
// PKCE (S256), a state bound to the browser by a cookie, and for OpenID
// Connect a nonce. The ID token comes straight from the provider's token
// endpoint over TLS, so its issuer is the server Axon talked to (OpenID
// Connect Core 3.1.3.7); Axon still checks iss, aud, exp, and the nonce.
// An identity is linked by its owner while signed in, never by email.

const (
	oauthCookie = "axon_oauth"
	oauthTTL    = 10 * time.Minute
)

// identity is who the provider says signed in.
type identity struct {
	Subject, Email, Name string
}

func (s *Service) provider(id string) (Provider, bool) {
	for _, p := range s.providers {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}

func (s *Service) callbackURL(p Provider) string {
	return s.base + "/api/v1/auth/oauth/" + p.ID + "/callback"
}

func random(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// safeNext keeps redirects on this site.
func safeNext(n string) string {
	if !strings.HasPrefix(n, "/") || strings.HasPrefix(n, "//") || strings.HasPrefix(n, "/\\") || len(n) > 500 {
		return "/"
	}
	return n
}

// providersList is public: the sign-in page shows a button for each.
func (s *Service) providersList(w http.ResponseWriter, _ *http.Request) {
	out := make([]map[string]string, 0, len(s.providers))
	for _, p := range s.providers {
		out = append(out, map[string]string{"id": p.ID, "name": p.Name})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

// oauthStart sends the browser to the provider.
func (s *Service) oauthStart(w http.ResponseWriter, r *http.Request) {
	p, ok := s.provider(chi.URLParam(r, "provider"))
	if !ok {
		httpx.Problem(w, http.StatusNotFound, "This Axon does not offer that sign-in.")
		return
	}
	intent := r.URL.Query().Get("intent")
	var userID *uuid.UUID
	switch intent {
	case "", "signin":
		intent = "signin"
	case "link":
		id, err := s.identify(r)
		if err != nil || id.Scope != "session" {
			httpx.Problem(w, http.StatusUnauthorized, "Sign in to link another account.")
			return
		}
		if id.Demo {
			httpx.Problem(w, http.StatusForbidden, "The demo account cannot do this. Create your own account to use it.")
			return
		}
		userID = &id.UserID
	default:
		httpx.Problem(w, http.StatusBadRequest, "intent is signin or link.")
		return
	}
	state, verifier, nonce := random(32), random(48), random(24)
	stateHash := sha256.Sum256([]byte(state))
	if _, err := s.db.Exec(r.Context(), `DELETE FROM oauth_states WHERE expires_at < now()`); err != nil {
		s.fail(w, r, err)
		return
	}
	if _, err := s.db.Exec(r.Context(), `INSERT INTO oauth_states (state_hash, provider, intent, user_id, verifier, nonce, next, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, stateHash[:], p.ID, intent, userID, verifier, nonce, safeNext(r.URL.Query().Get("next")), time.Now().Add(oauthTTL)); err != nil {
		s.fail(w, r, err)
		return
	}
	s.setCookie(w, oauthCookie, state, time.Now().Add(oauthTTL))
	challenge := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {p.ClientID},
		"redirect_uri":          {s.callbackURL(p)},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	if len(p.Scopes) > 0 {
		q.Set("scope", strings.Join(p.Scopes, " "))
	}
	if p.UserURL == "" {
		q.Set("nonce", nonce)
	}
	if p.ID == "apple" {
		q.Set("response_mode", "query") // no scopes: Apple answers with a plain redirect
	}
	if p.ID == "google" || p.ID == "microsoft" {
		q.Set("prompt", "select_account")
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, p.AuthURL+"?"+q.Encode(), http.StatusFound)
}

var errOAuth = errors.New("sign-in with the provider failed")

// oauthCallback finishes: checks the state, exchanges the code, and signs
// in, links, or creates the account.
func (s *Service) oauthCallback(w http.ResponseWriter, r *http.Request) {
	fail := func(reason string, err error) {
		if err != nil {
			s.log.Info("provider sign-in refused", "reason", reason, "err", err)
		}
		s.clearCookie(w, oauthCookie)
		http.Redirect(w, r, "/signin?error="+url.QueryEscape(reason), http.StatusFound)
	}
	p, ok := s.provider(chi.URLParam(r, "provider"))
	if !ok {
		fail("provider", nil)
		return
	}
	q := r.URL.Query()
	if q.Get("error") != "" {
		fail("cancelled", nil)
		return
	}
	c, err := r.Cookie(oauthCookie)
	state := q.Get("state")
	if err != nil || state == "" || c.Value != state {
		fail("state", err)
		return
	}
	stateHash := sha256.Sum256([]byte(state))
	var intent, verifier, nonce, next string
	var linkUser *uuid.UUID
	err = s.db.QueryRow(r.Context(), `DELETE FROM oauth_states WHERE state_hash = $1 AND provider = $2 AND expires_at > now()
		RETURNING intent, user_id, verifier, nonce, next`, stateHash[:], p.ID).Scan(&intent, &linkUser, &verifier, &nonce, &next)
	if err != nil {
		fail("expired", err)
		return
	}
	id, err := s.exchange(r.Context(), p, q.Get("code"), verifier, nonce)
	if err != nil {
		fail("provider", err)
		return
	}
	s.clearCookie(w, oauthCookie)
	if intent == "link" {
		s.link(w, r, p, *linkUser, id)
		return
	}
	s.signInWith(w, r, p, id, next)
}

func (s *Service) link(w http.ResponseWriter, r *http.Request, p Provider, userID uuid.UUID, id identity) {
	var owner uuid.UUID
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(r.Context(), `INSERT INTO auth_identities (id, user_id, provider, subject, email, name) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (provider, subject) DO UPDATE SET email = EXCLUDED.email, name = EXCLUDED.name RETURNING user_id`,
			uuid.Must(uuid.NewV7()), userID, p.ID, id.Subject, truncate(id.Email, 320), truncate(id.Name, 200)).Scan(&owner)
		if err != nil || owner != userID {
			return err
		}
		return audit.Record(r.Context(), tx, userID, audit.IdentityLinked, httpx.ClientIP(r), r.UserAgent(), map[string]any{"provider": p.ID})
	})
	switch {
	case err != nil:
		s.fail(w, r, err)
	case owner != userID:
		http.Redirect(w, r, "/account/security?error=taken", http.StatusFound)
	default:
		http.Redirect(w, r, "/account/security?linked="+p.ID, http.StatusFound)
	}
}

func (s *Service) signInWith(w http.ResponseWriter, r *http.Request, p Provider, id identity, next string) {
	created := false
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var userID uuid.UUID
		err := tx.QueryRow(r.Context(), `UPDATE auth_identities SET last_used_at = now() WHERE provider = $1 AND subject = $2 RETURNING user_id`,
			p.ID, id.Subject).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			// A new person: an account, when this Axon takes new accounts.
			if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(7364001)`); err != nil {
				return err
			}
			if open, err := s.registrationOpen(r.Context(), tx); err != nil || !open {
				return errors.Join(err, errRegistrationClosed)
			}
			name, ok := cleanName(id.Name)
			if !ok {
				name = p.Name + " user"
			}
			userID = uuid.Must(uuid.NewV7())
			if _, err := tx.Exec(r.Context(), `INSERT INTO users (id, display_name) VALUES ($1, $2)`, userID, name); err != nil {
				return err
			}
			if _, err := tx.Exec(r.Context(), `INSERT INTO auth_identities (id, user_id, provider, subject, email, name, last_used_at) VALUES ($1, $2, $3, $4, $5, $6, now())`,
				uuid.Must(uuid.NewV7()), userID, p.ID, id.Subject, truncate(id.Email, 320), truncate(id.Name, 200)); err != nil {
				return err
			}
			if err := audit.Record(r.Context(), tx, userID, audit.AccountCreated, httpx.ClientIP(r), r.UserAgent(), map[string]any{"provider": p.ID}); err != nil {
				return err
			}
			created = true
		} else if err != nil {
			return err
		}
		if err := audit.Record(r.Context(), tx, userID, audit.SignInProvider, httpx.ClientIP(r), r.UserAgent(), map[string]any{"provider": p.ID}); err != nil {
			return err
		}
		return s.startSession(r.Context(), tx, w, r, userID)
	})
	switch {
	case errors.Is(err, errRegistrationClosed):
		http.Redirect(w, r, "/signin?error=not_linked", http.StatusFound)
	case err != nil:
		s.fail(w, r, err)
	case created:
		http.Redirect(w, r, "/account/security?welcome="+p.ID, http.StatusFound)
	default:
		http.Redirect(w, r, next, http.StatusFound)
	}
}

// exchange trades the code for tokens and reads who signed in.
func (s *Service) exchange(ctx context.Context, p Provider, code, verifier, nonce string) (identity, error) {
	if code == "" {
		return identity{}, errors.Join(errOAuth, errors.New("no code"))
	}
	secret, err := p.clientSecret(time.Now())
	if err != nil {
		return identity{}, err
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.callbackURL(p)},
		"client_id":     {p.ClientID},
		"client_secret": {secret},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return identity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	var tok struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		Error       string `json:"error"`
	}
	if err := getJSON(req, &tok); err != nil {
		return identity{}, err
	}
	if tok.Error != "" {
		return identity{}, errors.Join(errOAuth, errors.New(tok.Error))
	}
	if p.UserURL != "" { // OAuth 2: ask who this is
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserURL, nil)
		if err != nil {
			return identity{}, err
		}
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		var u struct {
			ID         string `json:"id"`
			Username   string `json:"username"`
			GlobalName string `json:"global_name"`
		}
		if err := getJSON(req, &u); err != nil {
			return identity{}, err
		}
		if u.ID == "" {
			return identity{}, errors.Join(errOAuth, errors.New("no user id"))
		}
		name := u.GlobalName
		if name == "" {
			name = u.Username
		}
		return identity{Subject: u.ID, Name: name}, nil
	}
	claims, err := idTokenClaims(tok.IDToken)
	if err != nil {
		return identity{}, err
	}
	return claims.check(p, nonce, time.Now())
}

func getJSON(req *http.Request, out any) error {
	res, err := OAuthClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode/100 != 2 && res.StatusCode != http.StatusBadRequest {
		return errors.Join(errOAuth, fmt.Errorf("%s answered %s", req.URL.Host, res.Status))
	}
	return json.Unmarshal(b, out)
}

// claims are the parts of an ID token Axon reads.
type claims struct {
	Iss   string          `json:"iss"`
	Sub   string          `json:"sub"`
	Aud   json.RawMessage `json:"aud"` // a string or a list
	Azp   string          `json:"azp"`
	Exp   int64           `json:"exp"`
	Nonce string          `json:"nonce"`
	Email string          `json:"email"`
	Name  string          `json:"name"`
}

func idTokenClaims(jwt string) (claims, error) {
	var c claims
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return c, errors.Join(errOAuth, errors.New("no ID token"))
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return c, errors.Join(errOAuth, err)
	}
	return c, json.Unmarshal(b, &c)
}

func (c claims) audiences() []string {
	var one string
	if json.Unmarshal(c.Aud, &one) == nil {
		return []string{one}
	}
	var many []string
	_ = json.Unmarshal(c.Aud, &many)
	return many
}

// check validates an ID token's claims (OpenID Connect Core 3.1.3.7).
func (c claims) check(p Provider, nonce string, now time.Time) (identity, error) {
	aud := c.audiences()
	inAud := false
	for _, a := range aud {
		inAud = inAud || a == p.ClientID
	}
	switch {
	case !p.issuerOK(c.Iss):
		return identity{}, errors.Join(errOAuth, fmt.Errorf("unexpected issuer %q", c.Iss))
	case !inAud || (len(aud) > 1 && c.Azp != p.ClientID):
		return identity{}, errors.Join(errOAuth, errors.New("the token is for another client"))
	case c.Exp < now.Unix():
		return identity{}, errors.Join(errOAuth, errors.New("the token has expired"))
	case p.ID != "apple" && c.Nonce != nonce, p.ID == "apple" && c.Nonce != "" && c.Nonce != nonce:
		return identity{}, errors.Join(errOAuth, errors.New("the nonce does not match"))
	case c.Sub == "" || len(c.Sub) > 255:
		return identity{}, errors.Join(errOAuth, errors.New("no subject"))
	}
	return identity{Subject: c.Sub, Email: c.Email, Name: c.Name}, nil
}

// Linked accounts, in Account → Security.

type identityInfo struct {
	ID         uuid.UUID  `json:"id"`
	Provider   string     `json:"provider"`
	Email      string     `json:"email"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

func (s *Service) listIdentities(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	rows, err := s.db.Query(r.Context(), `SELECT id, provider, email, name, created_at, last_used_at FROM auth_identities WHERE user_id = $1 ORDER BY created_at`, id.UserID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[identityInfo])
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if list == nil {
		list = []identityInfo{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// unlinkIdentity removes a linked account, unless it is the last way in.
func (s *Service) unlinkIdentity(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	target, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such linked account.")
		return
	}
	errLast := errors.New("last way in")
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var provider string
		err := tx.QueryRow(r.Context(), `DELETE FROM auth_identities WHERE id = $1 AND user_id = $2 RETURNING provider`, target, id.UserID).Scan(&provider)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNotFound
		}
		if err != nil {
			return err
		}
		var left bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM passkeys WHERE user_id = $1) OR EXISTS (SELECT 1 FROM auth_identities WHERE user_id = $1)`, id.UserID).Scan(&left); err != nil {
			return err
		}
		if !left {
			return errLast
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.IdentityUnlinked, httpx.ClientIP(r), r.UserAgent(), map[string]any{"provider": provider})
	})
	switch {
	case errors.Is(err, errNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such linked account.")
	case errors.Is(err, errLast):
		httpx.Problem(w, http.StatusConflict, "This is your only way to sign in: add a passkey first.")
	case err != nil:
		s.fail(w, r, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// ExportIdentities is the linked accounts part of a personal data export.
func ExportIdentities(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT id, provider, email, name, created_at, last_used_at FROM auth_identities WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[identityInfo])
}
