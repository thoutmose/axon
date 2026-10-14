// Package vault says where a person's Markdown vault lives (EP-69). By
// default it lives nowhere but the database: notes are Markdown already and
// need no files. A person who wants files names a server and a folder; the
// server pairs once with a short-lived code, which becomes a write token for
// `axon notes-sync -watch`, and reports after each sync so Axon can say
// whether the vault is in step. An empty vault starts with the folder
// structure and the templates.
package vault

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
)

// Lifetimes: a pairing code is used within PairTTL, and the token it
// becomes lasts a year (pairing again replaces it).
const (
	PairTTL  = 30 * time.Minute
	TokenTTL = 365 * 24 * time.Hour
	// silentAfter is how long without a report before the vault is shown
	// as out of touch; a watching sync reports at least every few minutes.
	silentAfter = 15 * time.Minute
)

// Errors a person can act on.
var (
	ErrNotFound = errors.New("no vault server")
	ErrBadCode  = errors.New("the pairing code is wrong or has expired")
	ErrInvalid  = errors.New("invalid vault server")
)

// Report is what the server says after a sync.
type Report struct {
	Host      string   `json:"host"`
	Notes     int      `json:"notes"`
	Files     int      `json:"files"`
	Pushed    int      `json:"pushed"`
	Pulled    int      `json:"pulled"`
	Deleted   int      `json:"deleted"`
	Conflicts []string `json:"conflicts"`
	Error     string   `json:"error"`
}

// Source is the vault server of one person.
type Source struct {
	Host           string     `json:"host"`
	Dir            string     `json:"dir"`
	Scaffold       bool       `json:"scaffold"`
	Status         string     `json:"status"` // waiting (not paired yet), connected, silent, failing
	PairExpiresAt  *time.Time `json:"pair_expires_at"`
	PairedAt       *time.Time `json:"paired_at"`
	PairedHost     string     `json:"paired_host"`
	LastSeenAt     *time.Time `json:"last_seen_at"`
	TokenExpiresAt *time.Time `json:"token_expires_at"`
	LastReport     Report     `json:"last_report"`
}

// Get returns the person's vault server.
func Get(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (Source, error) {
	var s Source
	err := q.QueryRow(ctx, `SELECT v.host, v.dir, v.scaffold, v.pair_expires_at, v.paired_at, v.paired_host, v.last_seen_at, t.expires_at, v.last_report
		FROM vault_sources v LEFT JOIN api_tokens t ON t.id = v.token_id WHERE v.user_id = $1`, userID).
		Scan(&s.Host, &s.Dir, &s.Scaffold, &s.PairExpiresAt, &s.PairedAt, &s.PairedHost, &s.LastSeenAt, &s.TokenExpiresAt, &s.LastReport)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	if s.LastReport.Conflicts == nil {
		s.LastReport.Conflicts = []string{}
	}
	switch {
	case s.PairedAt == nil || s.TokenExpiresAt == nil || s.TokenExpiresAt.Before(now):
		s.Status = "waiting"
	case s.LastSeenAt == nil || now.Sub(*s.LastSeenAt) > silentAfter:
		s.Status = "silent"
	case s.LastReport.Error != "":
		s.Status = "failing"
	default:
		s.Status = "connected"
	}
	return s, err
}

// clean checks a host name and a folder as a person typed them.
func clean(host, dir string) (string, string, error) {
	host, dir = strings.TrimSpace(host), strings.TrimSpace(dir)
	switch {
	case host == "" || len([]rune(host)) > 100 || strings.IndexFunc(host, unsafeRune) >= 0:
		return "", "", errors.Join(ErrInvalid, errors.New("the server is a name like srv-secondbrain, up to 100 characters, without spaces or quotes"))
	case len([]rune(dir)) > 500 || strings.IndexFunc(dir, unsafeDirRune) >= 0 || !strings.HasPrefix(dir, "/") && !strings.HasPrefix(dir, "~/"),
		strings.Contains("/"+dir+"/", "/../"):
		return "", "", errors.Join(ErrInvalid, errors.New("the folder is a full path like ~/SecondBrain or /srv/vault, up to 500 characters, without quotes or a .. step"))
	}
	return host, dir, nil
}

// The host and folder end up in shell commands Axon shows: no quotes,
// backslashes, dollars, backticks, or control characters.
func unsafeDirRune(r rune) bool {
	return unicode.IsControl(r) || strings.ContainsRune("'\"`$\\;&|<>", r)
}

func unsafeRune(r rune) bool { return unsafeDirRune(r) || unicode.IsSpace(r) }

// Put names the vault server and makes a new pairing code, returned once.
// Changing the server keeps the current token until the new one pairs.
func Put(ctx context.Context, q db.Querier, userID uuid.UUID, host, dir string, scaffold bool, now time.Time) (string, error) {
	host, dir, err := clean(host, dir)
	if err != nil {
		return "", err
	}
	code, hash := newCode()
	_, err = q.Exec(ctx, `INSERT INTO vault_sources (user_id, host, dir, scaffold, pair_hash, pair_expires_at) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET host = $2, dir = $3, scaffold = $4, pair_hash = $5, pair_expires_at = $6`,
		userID, host, dir, scaffold, hash, now.Add(PairTTL))
	return code, err
}

// Remove forgets the vault server and revokes its token. The notes stay.
func Remove(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	var token *uuid.UUID
	err := tx.QueryRow(ctx, `DELETE FROM vault_sources WHERE user_id = $1 RETURNING token_id`, userID).Scan(&token)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil || token == nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1`, *token)
	return err
}

// Paired is what a server receives for a valid code.
type Paired struct {
	Token     string   `json:"token"`
	Dir       string   `json:"dir"`
	Folders   []string `json:"folders"`    // to create in an empty vault
	Scaffold  []string `json:"scaffold"`   // notes added to Axon for an empty vault
	ExpiresAt string   `json:"expires_at"` // when the token stops working
}

// Pair exchanges a code for a write token, once. The previous token of the
// vault server is revoked. When the vault is empty and the person asked
// for it, the folder structure and templates are added to Axon so the
// first sync writes them.
func Pair(ctx context.Context, tx pgx.Tx, code, host string, empty bool, now time.Time, ip, userAgent string) (Paired, error) {
	var out Paired
	var userID uuid.UUID
	var old *uuid.UUID
	var scaffold bool
	err := tx.QueryRow(ctx, `SELECT user_id, token_id, scaffold, dir FROM vault_sources WHERE pair_hash = $1 AND pair_expires_at > $2 FOR UPDATE`,
		hashCode(code), now).Scan(&userID, &old, &scaffold, &out.Dir)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrBadCode
	}
	if err != nil {
		return out, err
	}
	host = strings.TrimSpace(host)
	if len([]rune(host)) > 100 || strings.IndexFunc(host, unsafeRune) >= 0 {
		host = ""
	}
	if old != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1`, *old); err != nil {
			return out, err
		}
	}
	name := "Vault on " + host
	if host == "" {
		name = "Vault server"
	}
	token, info, err := auth.IssueToken(ctx, tx, userID, name, "write", TokenTTL, ip, userAgent)
	if err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `UPDATE vault_sources SET pair_hash = NULL, pair_expires_at = NULL, token_id = $2, paired_at = $3, paired_host = $4
		WHERE user_id = $1`, userID, info.ID, now, host); err != nil {
		return out, err
	}
	out.Token, out.ExpiresAt, out.Folders, out.Scaffold = token, info.ExpiresAt.Format(time.RFC3339), []string{}, []string{}
	if empty && scaffold {
		out.Folders = Folders
		if out.Scaffold, err = Scaffold(ctx, tx, userID); err != nil {
			return out, err
		}
	}
	return out, nil
}

// Heartbeat stores what the vault server reported after a sync.
func Heartbeat(ctx context.Context, q db.Querier, userID uuid.UUID, rep Report, now time.Time) error {
	if len(rep.Conflicts) > 50 {
		rep.Conflicts = rep.Conflicts[:50]
	}
	if len(rep.Error) > 500 {
		rep.Error = rep.Error[:500]
	}
	if len([]rune(rep.Host)) > 100 {
		rep.Host = ""
	}
	tag, err := q.Exec(ctx, `UPDATE vault_sources SET last_seen_at = $2, last_report = $3 WHERE user_id = $1`, userID, now, rep)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// Export is the vault part of a personal data export: where it lives, not the token.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	s, err := Get(ctx, q, userID, time.Now())
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return s, err
}

// Codes are 8 characters from an alphabet without look-alikes, shown as
// ABCD-EFGH: 40 bits, enough for a code that lives 30 minutes, is used once,
// and is tried through the sign-in rate limit.
var codeAlphabet = base32.NewEncoding("ABCDEFGHJKLMNPQRSTUVWXYZ23456789").WithPadding(base32.NoPadding)

func newCode() (string, []byte) {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	c := codeAlphabet.EncodeToString(b)
	code := c[:4] + "-" + c[4:]
	return code, hashCode(code)
}

func hashCode(code string) []byte {
	c := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(code))
	h := sha256.Sum256([]byte(c))
	return h[:]
}
