// Package ankisync sends a person's flashcards to their anki-sync-server
// (EP-42) through the Anki bridge, a sidecar built on the official Anki
// library (deploy/anki). The bridge imports Axon's cards, updating the same
// Anki notes by stable GUID and never deleting Anki-only cards, then syncs.
package ankisync

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Account is what the person sees: never the password.
type Account struct {
	Endpoint    string     `json:"endpoint"`
	Username    string     `json:"username"`
	AutoSync    bool       `json:"auto_sync"`
	LastSyncAt  *time.Time `json:"last_sync_at"`
	LastStatus  *string    `json:"last_status"`
	LastMessage string     `json:"last_message"`
}

// Result is what one sync did.
type Result struct {
	Added     int `json:"added"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
}

// Service syncs cards; BridgeURL is empty when the server has no bridge.
type Service struct {
	db        *pgxpool.Pool
	log       *slog.Logger
	sealer    *auth.Sealer
	BridgeURL string
	Default   string // default sync server address (ANKI_SYNC_URL)
	client    *http.Client
}

// New returns the service.
func New(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer, bridgeURL, defaultEndpoint string) *Service {
	return &Service{db: pool, log: log, sealer: sealer, BridgeURL: strings.TrimRight(bridgeURL, "/"), Default: defaultEndpoint,
		client: &http.Client{Timeout: 3 * time.Minute}}
}

var errNoAccount = errors.New("no anki account")

func (s *Service) account(ctx context.Context, userID uuid.UUID) (*Account, []byte, error) {
	var a Account
	var sealed []byte
	err := s.db.QueryRow(ctx, `SELECT endpoint, username, auto_sync, last_sync_at, last_status, last_message, password_sealed
		FROM anki_accounts WHERE user_id = $1`, userID).Scan(&a.Endpoint, &a.Username, &a.AutoSync, &a.LastSyncAt, &a.LastStatus, &a.LastMessage, &sealed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	return &a, sealed, err
}

// Sync sends all of the person's cards to their Anki account and records how it went.
func (s *Service) Sync(ctx context.Context, userID uuid.UUID) (Result, error) {
	if s.BridgeURL == "" {
		return Result{}, errors.New("anki sync is not set up on this server")
	}
	a, sealed, err := s.account(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	if a == nil {
		return Result{}, errNoAccount
	}
	password, err := s.sealer.Open(sealed)
	if err != nil {
		return Result{}, err
	}
	res, syncErr := s.send(ctx, userID, a, string(password))
	status, msg := "ok", fmt.Sprintf("%d added, %d updated", res.Added, res.Updated)
	if syncErr != nil {
		status, msg = "error", syncErr.Error()
	}
	if len(msg) > 1000 {
		msg = msg[:1000]
	}
	if _, err := s.db.Exec(ctx, `UPDATE anki_accounts SET last_sync_at = now(), last_status = $2, last_message = $3 WHERE user_id = $1`,
		userID, status, msg); err != nil {
		return res, err
	}
	return res, syncErr
}

func (s *Service) send(ctx context.Context, userID uuid.UUID, a *Account, password string) (Result, error) {
	list, err := cards.List(ctx, s.db, userID, "")
	if err != nil {
		return Result{}, err
	}
	if len(list) == 0 {
		return Result{}, nil
	}
	pkg, err := cards.APKG(ctx, list, func(id uuid.UUID) ([]byte, error) {
		img, _, err := cards.Image(ctx, s.db, userID, id)
		return img, err
	}, time.Now())
	if err != nil {
		return Result{}, err
	}
	body, _ := json.Marshal(map[string]string{"user": userID.String(), "endpoint": a.Endpoint, "username": a.Username,
		"password": password, "apkg": base64.StdEncoding.EncodeToString(pkg)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BridgeURL+"/sync", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, errors.New("the Anki bridge did not answer")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return Result{}, errors.New("Anki: " + summary(e.Error))
	}
	var r Result
	err = json.Unmarshal(raw, &r)
	return r, err
}

// AfterApproved syncs in the background when a Teacher's cards were approved
// and the person turned automatic sync on.
func (s *Service) AfterApproved(userID uuid.UUID, tool string) {
	if tool != "propose_cards" || s.BridgeURL == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		a, _, err := s.account(ctx, userID)
		if err != nil || a == nil || !a.AutoSync {
			return
		}
		if _, err := s.Sync(ctx, userID); err != nil {
			s.log.Warn("automatic anki sync failed", "err", err) // the message is on the account for the person to see
		}
	}()
}

// Routes mounts the endpoints; the caller wraps them in auth.RequireSession.
func (s *Service) Routes(r chi.Router) {
	r.Get("/v1/me/anki", s.get)
	r.With(auth.NotDemo).Put("/v1/me/anki", s.put)
	r.With(auth.NotDemo).Delete("/v1/me/anki", s.delete)
	r.With(auth.NotDemo).Post("/v1/me/anki/sync", s.sync)
}

func (s *Service) fail(w http.ResponseWriter, err error) {
	s.log.Error("anki request failed", "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	a, _, err := s.account(r.Context(), who.UserID)
	if err != nil {
		s.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"available": s.BridgeURL != "" && s.sealer != nil, "default_endpoint": s.Default, "account": a})
}

func (s *Service) put(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Endpoint string  `json:"endpoint"`
		Username string  `json:"username"`
		Password *string `json:"password"` // absent: keep the stored one
		AutoSync *bool   `json:"auto_sync"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Endpoint, body.Username = strings.TrimSpace(body.Endpoint), strings.TrimSpace(body.Username)
	u, err := url.Parse(body.Endpoint)
	switch {
	case s.sealer == nil:
		httpx.Problem(w, http.StatusUnprocessableEntity, "This server cannot store passwords: set AXON_ENCRYPTION_KEY first.")
		return
	case err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(body.Endpoint) > 500:
		httpx.Problem(w, http.StatusUnprocessableEntity, "The sync server address starts with http:// or https://, like https://anki.example/.")
		return
	case body.Username == "" || len(body.Username) > 200:
		httpx.Problem(w, http.StatusUnprocessableEntity, "Enter the user name of your Anki sync account.")
		return
	case body.Password != nil && (*body.Password == "" || len(*body.Password) > 500):
		httpx.Problem(w, http.StatusUnprocessableEntity, "Enter the password of your Anki sync account.")
		return
	}
	auto := true
	if body.AutoSync != nil {
		auto = *body.AutoSync
	}
	var sealed []byte
	if body.Password != nil {
		sealed = s.sealer.Seal([]byte(*body.Password))
	}
	tag, err := s.db.Exec(r.Context(), `INSERT INTO anki_accounts (user_id, endpoint, username, password_sealed, auto_sync)
		SELECT $1, $2, $3, $4, $5 WHERE $4::bytea IS NOT NULL
		ON CONFLICT (user_id) DO UPDATE SET endpoint = $2, username = $3, auto_sync = $5,
			password_sealed = coalesce($4, anki_accounts.password_sealed), updated_at = now()`,
		who.UserID, body.Endpoint, body.Username, sealed, auto)
	if err == nil && tag.RowsAffected() == 0 {
		// No account yet and no password given: it cannot be created.
		var exists bool
		_ = s.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM anki_accounts WHERE user_id = $1)`, who.UserID).Scan(&exists)
		if !exists {
			httpx.Problem(w, http.StatusUnprocessableEntity, "Enter the password of your Anki sync account.")
			return
		}
		_, err = s.db.Exec(r.Context(), `UPDATE anki_accounts SET endpoint = $2, username = $3, auto_sync = $4, updated_at = now() WHERE user_id = $1`,
			who.UserID, body.Endpoint, body.Username, auto)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.get(w, r)
}

func (s *Service) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if _, err := s.db.Exec(r.Context(), `DELETE FROM anki_accounts WHERE user_id = $1`, who.UserID); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) sync(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	res, err := s.Sync(r.Context(), who.UserID)
	switch {
	case errors.Is(err, errNoAccount):
		httpx.Problem(w, http.StatusConflict, "Connect your Anki sync account first.")
	case err != nil && s.BridgeURL == "":
		httpx.Problem(w, http.StatusServiceUnavailable, "Anki sync is not set up on this server.")
	case err != nil:
		httpx.Problem(w, http.StatusBadGateway, err.Error())
	default:
		httpx.JSON(w, http.StatusOK, res)
	}
}

// Export is the account part of a personal data export (never the password).
func (s *Service) Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	var a *Account
	var row Account
	err := q.QueryRow(ctx, `SELECT endpoint, username, auto_sync, last_sync_at, last_status, last_message FROM anki_accounts WHERE user_id = $1`, userID).
		Scan(&row.Endpoint, &row.Username, &row.AutoSync, &row.LastSyncAt, &row.LastStatus, &row.LastMessage)
	if err == nil {
		a = &row
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return a, nil
}

// summary keeps the first line of the Anki library's error, without the
// Unicode isolation marks it wraps details in: the rest is for developers.
func summary(msg string) string {
	first, _, _ := strings.Cut(msg, "\n")
	return strings.TrimSpace(strings.NewReplacer("\u2068", "", "\u2069", "").Replace(first))
}
