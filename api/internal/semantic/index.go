package semantic

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/knowledge"

	"github.com/thoutmose/axon/api/internal/telemetry"
)

// batch is how many notes go to the service in one request, and perStep
// how many a person's notes are embedded per step, so a big vault is
// indexed over a few minutes instead of in one burst.
const (
	batch   = 32
	perStep = 256
	maxText = 8000
)

// Indexer keeps every person's vectors in step with their notes.
type Indexer struct {
	db     *pgxpool.Pool
	log    *slog.Logger
	sealer *auth.Sealer
}

// NewIndexer returns an indexer; call Run in a goroutine, or Step in tests.
func NewIndexer(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer) *Indexer {
	return &Indexer{db: pool, log: log, sealer: sealer}
}

// Run indexes every minute until ctx ends.
func (x *Indexer) Run(ctx context.Context) {
	for {
		done := telemetry.Time("semantic-index")
		err := x.Step(ctx)
		done(err)
		if err != nil && ctx.Err() == nil {
			x.log.Error("semantic index step failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}

// Step embeds the notes that are new or changed since they were embedded,
// for everyone with a service and the consent. Item notes the vault mirror
// keeps are left out, as from search.
func (x *Indexer) Step(ctx context.Context) error {
	rows, err := x.db.Query(ctx, `SELECT s.user_id FROM embedding_settings s JOIN consents c ON c.user_id = s.user_id
		JOIN users u ON u.id = s.user_id
		WHERE c.purpose = $1 AND c.granted AND NOT u.demo`, Purpose)
	if err != nil {
		return err
	}
	users, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	for _, u := range users {
		if err := x.person(ctx, u); err != nil && ctx.Err() == nil {
			x.log.Info("embedding failed", "user", u, "err", err)
			_, _ = x.db.Exec(ctx, `UPDATE embedding_settings SET error = $2 WHERE user_id = $1`, u, err.Error())
		}
	}
	return ctx.Err()
}

func (x *Indexer) person(ctx context.Context, userID uuid.UUID) error {
	s, err := Load(ctx, x.db, x.sealer, userID)
	if err != nil || s == nil {
		return err
	}
	type note struct {
		id      uuid.UUID
		title   string
		content string
	}
	rows, err := x.db.Query(ctx, `SELECT n.id, e.title, n.content FROM notes n JOIN entities e ON e.id = n.id
		LEFT JOIN note_embeddings v ON v.note_id = n.id AND v.model = $2
		WHERE n.user_id = $1 AND (v.note_id IS NULL OR v.hash <> encode(sha256(convert_to(n.content, 'UTF8')), 'hex'))
		AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = n.id)
		ORDER BY n.updated_at DESC LIMIT $3`, userID, s.modelKey(), perStep)
	if err != nil {
		return err
	}
	notes, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (note, error) {
		var n note
		err := r.Scan(&n.id, &n.title, &n.content)
		return n, err
	})
	if err != nil {
		return err
	}
	for start := 0; start < len(notes); start += batch {
		part := notes[start:min(start+batch, len(notes))]
		texts := make([]string, len(part))
		for i, n := range part {
			texts[i] = Text(n.title, n.content)
		}
		vecs, err := s.Embed(ctx, texts, "document")
		if err != nil {
			return err
		}
		for i, n := range part {
			if _, err := x.db.Exec(ctx, `INSERT INTO note_embeddings (note_id, user_id, model, hash, embedding)
				VALUES ($1, $2, $3, encode(sha256(convert_to($4::text, 'UTF8')), 'hex'), $5::vector)
				ON CONFLICT (note_id) DO UPDATE SET model = $3, hash = excluded.hash, embedding = $5::vector, updated_at = now()`,
				n.id, userID, s.modelKey(), n.content, literal(vecs[i])); err != nil {
				return err
			}
		}
	}
	if len(notes) > 0 {
		_, err = x.db.Exec(ctx, `UPDATE embedding_settings SET error = '' WHERE user_id = $1`, userID)
	}
	return err
}

// Text is what is embedded for a note: its title and its words, without
// Markdown, cut to a size every service accepts.
func Text(title, content string) string {
	t := title + "\n\n" + strings.Join(strings.Fields(knowledge.Clean(content)), " ")
	if r := []rune(t); len(r) > maxText {
		t = string(r[:maxText])
	}
	return t
}
