package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// ceremony is a WebAuthn challenge waiting for the browser's response.
type ceremony struct {
	UserID      *uuid.UUID
	DisplayName string // register: the new account's name; add_passkey: the passkey's name
	Session     webauthn.SessionData
}

// saveCeremony stores the challenge server-side and points a short-lived cookie at it.
func (s *Service) saveCeremony(ctx context.Context, w http.ResponseWriter, kind string, c ceremony) error {
	id := uuid.Must(uuid.NewV7())
	data, err := json.Marshal(c.Session)
	if err != nil {
		return err
	}
	expires := time.Now().Add(ceremonyTTL)
	// Opportunistic cleanup keeps the table tiny without a background job.
	if _, err := s.db.Exec(ctx, `DELETE FROM webauthn_ceremonies WHERE expires_at < now()`); err != nil {
		return err
	}
	_, err = s.db.Exec(ctx,
		`INSERT INTO webauthn_ceremonies (id, user_id, kind, data, display_name, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, c.UserID, kind, data, c.DisplayName, expires)
	if err != nil {
		return err
	}
	s.setCookie(w, ceremonyCookie, id.String(), expires)
	return nil
}

// takeCeremony loads and deletes the caller's ceremony, so each challenge is used once.
func (s *Service) takeCeremony(ctx context.Context, q db.Querier, w http.ResponseWriter, r *http.Request, kind string) (ceremony, error) {
	s.clearCookie(w, ceremonyCookie)
	cookie, err := r.Cookie(ceremonyCookie)
	if err != nil {
		return ceremony{}, errNotFound
	}
	id, err := uuid.Parse(cookie.Value)
	if err != nil {
		return ceremony{}, errNotFound
	}
	var c ceremony
	var data []byte
	var name *string
	err = q.QueryRow(ctx,
		`DELETE FROM webauthn_ceremonies WHERE id = $1 AND kind = $2 AND expires_at > now()
		 RETURNING user_id, data, display_name`, id, kind).Scan(&c.UserID, &data, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return ceremony{}, errNotFound
	}
	if err != nil {
		return ceremony{}, err
	}
	if name != nil {
		c.DisplayName = *name
	}
	return c, json.Unmarshal(data, &c.Session)
}

// Recovery codes: 80 random bits each, shown once, stored as SHA-256 hashes.
const recoveryCodeCount = 10

var codeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func newRecoveryCode() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	s := strings.ToLower(codeEncoding.EncodeToString(b)) // 16 characters
	return s[0:4] + "-" + s[4:8] + "-" + s[8:12] + "-" + s[12:16]
}

func hashRecoveryCode(code string) []byte {
	normalized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '2' && r <= '7':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		}
		return -1
	}, code)
	h := sha256.Sum256([]byte(normalized))
	return h[:]
}

// issueRecoveryCodes replaces the user's recovery codes and returns the new ones.
func issueRecoveryCodes(ctx context.Context, q db.Querier, userID uuid.UUID) ([]string, error) {
	if _, err := q.Exec(ctx, `DELETE FROM recovery_codes WHERE user_id = $1`, userID); err != nil {
		return nil, err
	}
	codes := make([]string, recoveryCodeCount)
	for i := range codes {
		codes[i] = newRecoveryCode()
		_, err := q.Exec(ctx, `INSERT INTO recovery_codes (id, user_id, code_hash) VALUES ($1, $2, $3)`,
			uuid.Must(uuid.NewV7()), userID, hashRecoveryCode(codes[i]))
		if err != nil {
			return nil, err
		}
	}
	return codes, nil
}
