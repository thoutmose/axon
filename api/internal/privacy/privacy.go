// Package privacy implements consent records, data export, and account erasure.
package privacy

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/audit"
	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Purposes a user can consent to. Everything is off until the user turns it on.
var Purposes = []Purpose{
	{"ai_agents", "Send selected data to the configured AI provider so agents can draft suggestions."},
	{"external_metadata", "Look up films, series, and books with TMDB and Open Library."},
	{"health_data", "Store food, training, and other health data (special-category data under GDPR Art. 9). Turning it off deletes that data."},
	{"health_ai", "Let the Personal Trainer and Nutritionist agents send your food and training logs to the AI provider you set up, to coach you."},
	{"food_lookups", "Search foods in Open Food Facts and USDA FoodData Central; the food name or barcode you type is sent to them."},
	{"ai_embeddings", "Send the text of your notes to the embedding service you chose, to search them by meaning. Turning it off deletes the stored embeddings."},
}

// onWithdraw holds what must happen when a consent is withdrawn, such as
// deleting the data that only that consent allowed to keep.
var onWithdraw = map[string]func(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error{}

// OnWithdraw registers fn to run, in the same transaction, when purpose is withdrawn.
func OnWithdraw(purpose string, fn func(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error) {
	onWithdraw[purpose] = fn
}

// Purpose is one consentable use of data.
type Purpose struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Exporter returns one section of a user's export. Every entity type that
// holds user data registers one, so export stays complete as Axon grows.
type Exporter func(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error)

// Service serves the privacy endpoints.
type Service struct {
	db          *pgxpool.Pool
	log         *slog.Logger
	exporters   map[string]Exporter
	files       map[string]FileExporter
	clearCookie func(http.ResponseWriter)
}

// FileExporter returns files (path → bytes) to include in the export archive
// under a folder named after the section, e.g. notes as Markdown.
type FileExporter func(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string][]byte, error)

// RegisterFiles adds a folder of files to the export archive.
func (s *Service) RegisterFiles(folder string, f FileExporter) { s.files[folder] = f }

// New returns the service with the built-in account sections registered.
// clearCookie signs the browser out after erasure.
func New(pool *pgxpool.Pool, log *slog.Logger, clearCookie func(http.ResponseWriter)) *Service {
	s := &Service{db: pool, log: log, exporters: map[string]Exporter{}, files: map[string]FileExporter{}, clearCookie: clearCookie}
	s.Register("account", exportAccount)
	s.Register("passkeys", exportPasskeys)
	s.Register("sessions", exportSessions)
	s.Register("consents", exportConsents)
	s.Register("audit_events", func(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
		return audit.List(ctx, q, userID, 100000)
	})
	return s
}

// Register adds an export section.
func (s *Service) Register(section string, e Exporter) { s.exporters[section] = e }

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (s *Service) Routes(r chi.Router) {
	r.Get("/v1/me/consents", s.listConsents)
	r.Put("/v1/me/consents/{purpose}", s.setConsent)
	r.Get("/v1/me/export", s.export)
	r.With(auth.NotDemo).Delete("/v1/me", s.erase)
}

func (s *Service) fail(w http.ResponseWriter, err error) {
	s.log.Error("privacy request failed", "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

// --- Consent ---

type consent struct {
	Purpose
	Granted   bool       `json:"granted"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func consents(ctx context.Context, q db.Querier, userID uuid.UUID) ([]consent, error) {
	rows, err := q.Query(ctx, `SELECT purpose, granted, updated_at FROM consents WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	type stored struct {
		Purpose   string
		Granted   bool
		UpdatedAt time.Time
	}
	saved, err := pgx.CollectRows(rows, pgx.RowToStructByPos[stored])
	if err != nil {
		return nil, err
	}
	out := make([]consent, len(Purposes))
	for i, p := range Purposes {
		out[i] = consent{Purpose: p}
		for _, sv := range saved {
			if sv.Purpose == p.ID {
				out[i].Granted, out[i].UpdatedAt = sv.Granted, &sv.UpdatedAt
			}
		}
	}
	return out, nil
}

// Granted reports whether the user has consented to purpose. Features that
// depend on consent must call this before acting.
func Granted(ctx context.Context, q db.Querier, userID uuid.UUID, purpose string) (bool, error) {
	var granted bool
	err := q.QueryRow(ctx, `SELECT granted FROM consents WHERE user_id = $1 AND purpose = $2`, userID, purpose).Scan(&granted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return granted, err
}

func (s *Service) listConsents(w http.ResponseWriter, r *http.Request) {
	id, _ := auth.FromContext(r.Context())
	list, err := consents(r.Context(), s.db, id.UserID)
	if err != nil {
		s.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *Service) setConsent(w http.ResponseWriter, r *http.Request) {
	id, _ := auth.FromContext(r.Context())
	purpose := chi.URLParam(r, "purpose")
	if !slices.ContainsFunc(Purposes, func(p Purpose) bool { return p.ID == purpose }) {
		httpx.Problem(w, http.StatusNotFound, "No such consent purpose.")
		return
	}
	var body struct {
		Granted *bool `json:"granted"`
	}
	if err := httpx.Decode(r, &body); err != nil || body.Granted == nil {
		httpx.Problem(w, http.StatusBadRequest, `Send {"granted": true} or {"granted": false}.`)
		return
	}
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(r.Context(),
			`INSERT INTO consents (user_id, purpose, granted, updated_at) VALUES ($1, $2, $3, now())
			 ON CONFLICT (user_id, purpose) DO UPDATE SET granted = excluded.granted, updated_at = now()`,
			id.UserID, purpose, *body.Granted)
		if err != nil {
			return err
		}
		if fn := onWithdraw[purpose]; fn != nil && !*body.Granted {
			if err := fn(r.Context(), tx, id.UserID); err != nil {
				return err
			}
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.ConsentChanged, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"purpose": purpose, "granted": *body.Granted})
	})
	if err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Export ---

// Collect gathers every registered section inside one read-only transaction,
// so the export is a consistent snapshot.
func (s *Service) Collect(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	out := map[string]any{"format": "axon-export/1", "exported_at": time.Now().UTC()}
	err := pgx.BeginTxFunc(ctx, s.db, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead}, func(tx pgx.Tx) error {
		for name, exporter := range s.exporters {
			section, err := exporter(ctx, tx, userID)
			if err != nil {
				return fmt.Errorf("export %s: %w", name, err)
			}
			out[name] = section
		}
		return nil
	})
	return out, err
}

func (s *Service) export(w http.ResponseWriter, r *http.Request) {
	id, _ := auth.FromContext(r.Context())
	data, err := s.Collect(r.Context(), id.UserID)
	if err != nil {
		s.fail(w, err)
		return
	}
	files := map[string][]byte{}
	for folder, f := range s.files {
		got, err := f(r.Context(), s.db, id.UserID)
		if err != nil {
			s.fail(w, err)
			return
		}
		for p, b := range got {
			files[folder+"/"+p] = b
		}
	}
	if err := audit.Record(r.Context(), s.db, id.UserID, audit.DataExported, httpx.ClientIP(r), r.UserAgent(), nil); err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="axon-export-%s.zip"`, time.Now().UTC().Format("20060102")))
	zw := zip.NewWriter(w)
	if f, err := zw.Create("axon-export.json"); err == nil {
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		_ = enc.Encode(data)
	}
	if f, err := zw.Create("README.md"); err == nil {
		_, _ = f.Write([]byte(readme(data)))
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		if f, err := zw.Create(p); err == nil {
			_, _ = f.Write(files[p])
		}
	}
	if err := zw.Close(); err != nil {
		s.log.Error("export stream failed", "err", err)
	}
}

func readme(data map[string]any) string {
	var b strings.Builder
	b.WriteString("# Your Axon data\n\n")
	fmt.Fprintf(&b, "Exported %s (UTC). The complete, machine-readable data is in `axon-export.json`.\n\n", data["exported_at"].(time.Time).Format(time.RFC3339))
	b.WriteString("| Section | Entries |\n|---|---|\n")
	keys := make([]string, 0, len(data))
	for k := range data {
		if k != "format" && k != "exported_at" {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	for _, k := range keys {
		n := 1
		if raw, err := json.Marshal(data[k]); err == nil {
			var list []any
			if json.Unmarshal(raw, &list) == nil {
				n = len(list)
			}
		}
		fmt.Fprintf(&b, "| %s | %d |\n", k, n)
	}
	return b.String()
}

func exportAccount(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	var a struct {
		ID          uuid.UUID `json:"id"`
		DisplayName string    `json:"display_name"`
		CreatedAt   time.Time `json:"created_at"`
		TOTPEnabled bool      `json:"totp_enabled"`
	}
	err := q.QueryRow(ctx, `SELECT id, display_name, created_at, totp_enabled_at IS NOT NULL FROM users WHERE id = $1`, userID).
		Scan(&a.ID, &a.DisplayName, &a.CreatedAt, &a.TOTPEnabled)
	return a, err
}

func exportPasskeys(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	type passkey struct {
		ID         uuid.UUID  `json:"id"`
		Name       string     `json:"name"`
		CreatedAt  time.Time  `json:"created_at"`
		LastUsedAt *time.Time `json:"last_used_at"`
	}
	rows, err := q.Query(ctx, `SELECT id, name, created_at, last_used_at FROM passkeys WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[passkey])
}

func exportSessions(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	type session struct {
		ID         uuid.UUID `json:"id"`
		UserAgent  string    `json:"user_agent"`
		CreatedAt  time.Time `json:"created_at"`
		LastSeenAt time.Time `json:"last_seen_at"`
		ExpiresAt  time.Time `json:"expires_at"`
	}
	rows, err := q.Query(ctx, `SELECT id, user_agent, created_at, last_seen_at, expires_at FROM sessions WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[session])
}

func exportConsents(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return consents(ctx, q, userID)
}

// --- Erasure ---

// Erase deletes the user; every user-owned table cascades from users (ADR 0005).
func Erase(ctx context.Context, q db.Querier, userID uuid.UUID) error {
	_, err := q.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}

func (s *Service) erase(w http.ResponseWriter, r *http.Request) {
	id, _ := auth.FromContext(r.Context())
	var body struct {
		Confirm string `json:"confirm"`
	}
	if err := httpx.Decode(r, &body); err != nil || body.Confirm != "DELETE" {
		httpx.Problem(w, http.StatusBadRequest, `To delete your account and all its data, send {"confirm": "DELETE"}.`)
		return
	}
	if err := Erase(r.Context(), s.db, id.UserID); err != nil {
		s.fail(w, err)
		return
	}
	// No audit row: it would be erased with the account. The server log keeps
	// only that an erasure happened, without identifying data.
	s.log.Info("account erased")
	s.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
