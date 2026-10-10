// Package auth implements accounts, passkey sign-in, recovery codes, TOTP,
// and server-side sessions.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/db"
)

// Session and ceremony lifetimes.
const (
	SessionIdle     = 7 * 24 * time.Hour
	SessionAbsolute = 30 * 24 * time.Hour
	ceremonyTTL     = 5 * time.Minute
)

// Config comes from the environment (see .env.example).
type Config struct {
	BaseURL          string // AXON_BASE_URL, e.g. https://srv-prod.tail781d53.ts.net
	RPID             string // WEBAUTHN_RP_ID
	EncryptionKey    string // AXON_ENCRYPTION_KEY; TOTP is unavailable without it
	OpenRegistration bool   // AXON_REGISTRATION=open; otherwise only the first account can register
}

// Service holds the auth dependencies.
type Service struct {
	db           *pgxpool.Pool
	wa           *webauthn.WebAuthn
	sealer       *Sealer // nil when no encryption key is configured
	log          *slog.Logger
	origin       string
	secureCookie bool
	openReg      bool
	signInLimit  *limiter
}

// New validates cfg and builds the service.
func New(cfg Config, pool *pgxpool.Pool, log *slog.Logger) (*Service, error) {
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("AXON_BASE_URL must be an absolute URL, got %q", cfg.BaseURL)
	}
	origin := base.Scheme + "://" + base.Host
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: "Axon",
		RPOrigins:     []string{origin},
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn config: %w", err)
	}
	s := &Service{
		db:           pool,
		wa:           wa,
		log:          log,
		origin:       origin,
		secureCookie: base.Scheme == "https",
		openReg:      cfg.OpenRegistration,
		signInLimit:  newLimiter(6*time.Second, 10), // 10 at once, then one every 6 s
	}
	if cfg.EncryptionKey != "" {
		if s.sealer, err = NewSealer(cfg.EncryptionKey); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// user adapts an account and its passkeys to webauthn.User.
type user struct {
	ID          uuid.UUID
	DisplayName string
	credentials []webauthn.Credential
}

func (u *user) WebAuthnID() []byte                         { return u.ID[:] }
func (u *user) WebAuthnName() string                       { return u.DisplayName }
func (u *user) WebAuthnDisplayName() string                { return u.DisplayName }
func (u *user) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

var errNotFound = errors.New("not found")

func loadUser(ctx context.Context, q db.Querier, id uuid.UUID) (*user, error) {
	u := &user{ID: id}
	err := q.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, id).Scan(&u.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT credential FROM passkeys WHERE user_id = $1`, id)
	if err != nil {
		return nil, err
	}
	u.credentials, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (webauthn.Credential, error) {
		var raw []byte
		var c webauthn.Credential
		if err := row.Scan(&raw); err != nil {
			return c, err
		}
		return c, json.Unmarshal(raw, &c)
	})
	return u, err
}

func (s *Service) registrationOpen(ctx context.Context, q db.Querier) (bool, error) {
	if s.openReg {
		return true, nil
	}
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users)`).Scan(&exists)
	return !exists, err
}

// creationOptions asks for a discoverable passkey (usernameless sign-in).
func creationOptions(exclude []webauthn.Credential) []webauthn.RegistrationOption {
	descriptors := make([]protocol.CredentialDescriptor, len(exclude))
	for i := range exclude {
		descriptors[i] = exclude[i].Descriptor()
	}
	return []webauthn.RegistrationOption{
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationPreferred,
		}),
		webauthn.WithExclusions(descriptors),
	}
}
