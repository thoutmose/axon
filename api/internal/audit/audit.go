// Package audit records security-relevant events that the account owner can review.
package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Actions recorded in the audit log.
const (
	AccountCreated      = "account.created"
	SignInPasskey       = "signin.passkey"
	SignInRecoveryCode  = "signin.recovery_code"
	RecoveryFailed      = "signin.recovery_failed"
	SignOut             = "signout"
	SessionRevoked      = "session.revoked"
	SessionsRevokedAll  = "sessions.revoked_all"
	PasskeyAdded        = "passkey.added"
	PasskeyRemoved      = "passkey.removed"
	RecoveryCodesIssued = "recovery_codes.issued"
	TOTPEnabled         = "totp.enabled"
	TOTPDisabled        = "totp.disabled"
	ConsentChanged      = "consent.changed"
	DataExported        = "data.exported"
	AccountUpdated      = "account.updated"
	TokenCreated        = "token.created"
	TokenRevoked        = "token.revoked"
	CalendarFeedCreated = "calendar_feed.created"
	CalendarFeedRevoked = "calendar_feed.revoked"
)

// Event is one audit log entry.
type Event struct {
	ID        uuid.UUID      `json:"id"`
	Action    string         `json:"action"`
	IP        string         `json:"ip"`
	UserAgent string         `json:"user_agent"`
	Details   map[string]any `json:"details"`
	CreatedAt time.Time      `json:"created_at"`
}

// Record appends an event. Pass the caller's transaction so the event and the
// change it describes commit together.
func Record(ctx context.Context, q db.Querier, userID uuid.UUID, action, ip, userAgent string, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	_, err := q.Exec(ctx,
		`INSERT INTO audit_events (id, user_id, action, ip, user_agent, details) VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.Must(uuid.NewV7()), userID, action, ip, truncate(userAgent, 300), details)
	return err
}

// List returns the user's most recent events, newest first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, limit int) ([]Event, error) {
	rows, err := q.Query(ctx,
		`SELECT id, action, ip, user_agent, details, created_at FROM audit_events
		 WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Event, error) {
		var e Event
		err := row.Scan(&e.ID, &e.Action, &e.IP, &e.UserAgent, &e.Details, &e.CreatedAt)
		return e, err
	})
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
