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
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Invitations (EP-120). Registration is closed after the first account
// unless the operator opens it; the owner, that first account, can invite
// one person at a time with a link. The link carries 256 random bits; only
// their SHA-256 is stored. It is used once, inside the transaction that
// creates the account, and expires.

// Invite is an invitation as its creator sees it.
type Invite struct {
	ID         uuid.UUID  `json:"id"`
	Note       string     `json:"note"`
	Household  bool       `json:"household"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UsedAt     *time.Time `json:"used_at"`
	UsedByName *string    `json:"used_by_name"`
	URL        string     `json:"url,omitempty"` // only when it is created
}

func hashInvite(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// isOwner reports whether the user is the first account of this Axon.
func isOwner(ctx context.Context, q db.Querier, userID uuid.UUID) (bool, error) {
	var first uuid.UUID
	err := q.QueryRow(ctx, `SELECT id FROM users WHERE NOT demo ORDER BY created_at, id LIMIT 1`).Scan(&first)
	return err == nil && first == userID, err
}

// pendingInvite is a usable invitation for a token: unused and not expired.
func pendingInvite(ctx context.Context, q db.Querier, token string) (inviter string, household bool, err error) {
	err = q.QueryRow(ctx, `SELECT u.display_name, i.household FROM invites i JOIN users u ON u.id = i.created_by
		WHERE i.token_hash = $1 AND i.used_at IS NULL AND i.expires_at > now()`, hashInvite(token)).Scan(&inviter, &household)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, errNotFound
	}
	return inviter, household, err
}

// useInvite spends an invitation for a new account, once, and brings the
// person into the inviter's household when the invitation says so.
func useInvite(ctx context.Context, tx pgx.Tx, tokenHash []byte, newUser uuid.UUID) error {
	var inviter uuid.UUID
	var household bool
	err := tx.QueryRow(ctx, `UPDATE invites SET used_at = now(), used_by = $2
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now() RETURNING created_by, household`,
		tokenHash, newUser).Scan(&inviter, &household)
	if errors.Is(err, pgx.ErrNoRows) {
		return errRegistrationClosed
	}
	if err != nil || !household {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO household_members (household_id, user_id)
		SELECT household_id, $2 FROM household_members WHERE user_id = $1`, inviter, newUser)
	return err
}

func (s *Service) inviteOwner(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	who, _ := FromContext(r.Context())
	if who.Demo { // the demo shows its sample invitations; creating one is refused on the route
		return who.UserID, true
	}
	owner, err := isOwner(r.Context(), s.db, who.UserID)
	if err != nil {
		s.fail(w, r, err)
		return who.UserID, false
	}
	if !owner {
		httpx.Problem(w, http.StatusForbidden, "Only the owner of this Axon can invite people.")
		return who.UserID, false
	}
	return who.UserID, true
}

func (s *Service) createInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := s.inviteOwner(w, r)
	if !ok {
		return
	}
	var in struct {
		Note      string `json:"note"`
		Household bool   `json:"household"`
		Days      *int   `json:"days"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	days := 7
	if in.Days != nil {
		days = *in.Days
	}
	in.Note = strings.TrimSpace(in.Note)
	switch {
	case days < 1 || days > 30:
		httpx.Problem(w, http.StatusUnprocessableEntity, "An invitation lasts 1 to 30 days.")
		return
	case utf8.RuneCountInString(in.Note) > 200:
		httpx.Problem(w, http.StatusUnprocessableEntity, "The note is up to 200 characters.")
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		s.fail(w, r, err)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	inv := Invite{ID: uuid.Must(uuid.NewV7()), Note: in.Note, Household: in.Household, ExpiresAt: time.Now().AddDate(0, 0, days)}
	if err := s.db.QueryRow(r.Context(), `INSERT INTO invites (id, created_by, token_hash, note, household, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`, inv.ID, user, hashInvite(token), inv.Note, inv.Household, inv.ExpiresAt).
		Scan(&inv.CreatedAt); err != nil {
		s.fail(w, r, err)
		return
	}
	inv.URL = s.base + "/register?invite=" + token
	httpx.JSON(w, http.StatusCreated, inv)
}

func (s *Service) listInvites(w http.ResponseWriter, r *http.Request) {
	user, ok := s.inviteOwner(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT i.id, i.note, i.household, i.expires_at, i.created_at, i.used_at, u.display_name
		FROM invites i LEFT JOIN users u ON u.id = i.used_by WHERE i.created_by = $1 ORDER BY i.created_at DESC LIMIT 100`, user)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Invite, error) {
		var i Invite
		return i, row.Scan(&i.ID, &i.Note, &i.Household, &i.ExpiresAt, &i.CreatedAt, &i.UsedAt, &i.UsedByName)
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []Invite{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

// revokeInvite deletes an unused invitation; a used one stays as a record.
func (s *Service) revokeInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := s.inviteOwner(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such invitation, or it was already used.")
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM invites WHERE id = $1 AND created_by = $2 AND used_at IS NULL`, id, user)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Problem(w, http.StatusNotFound, "No such invitation, or it was already used.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// inviteInfo tells someone with a link who invites them.
func (s *Service) inviteInfo(w http.ResponseWriter, r *http.Request) {
	inviter, household, err := pendingInvite(r.Context(), s.db, r.URL.Query().Get("token"))
	if errors.Is(err, errNotFound) {
		httpx.Problem(w, http.StatusNotFound, "This invitation does not exist, was used, or has expired.")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"inviter": inviter, "household": household})
}
