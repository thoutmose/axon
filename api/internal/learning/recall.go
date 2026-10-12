package learning

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/knowledge"
	"github.com/thoutmose/axon/api/internal/notes"
)

// Free recall (retrieval practice on a whole note): the person writes
// everything they remember on a blank page, then sees which of the note's
// key ideas they brought back and which they missed. The comparison is
// local NLP: the note's key phrases (TextRank), matched by stem or by a
// thesaurus synonym, so "learn" recalls "learning" and "méthode" recalls
// "technique". Each attempt schedules the next one, further away each time
// the note is recalled well (expanding intervals).

// maxRecall is the longest recall text accepted.
const maxRecall = 20000

// recallKinds are the notes worth recalling: processed knowledge.
var recallKinds = []string{"literature", "permanent"}

// Idea is one of the note's key ideas, and whether the recall found it.
type Idea struct {
	Text     string `json:"text"`
	Recalled bool   `json:"recalled"`
}

// RecallResult is the outcome of one attempt.
type RecallResult struct {
	Score    float64   `json:"score"` // share of key ideas recalled, 0 to 1
	Ideas    []Idea    `json:"ideas"`
	DueOn    string    `json:"due_on"` // the next recall, YYYY-MM-DD
	Interval int       `json:"interval_days"`
	At       time.Time `json:"recalled_at"`
}

// RecallNote is a note due (or suggested) for recall.
type RecallNote struct {
	ID        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	Path      string     `json:"path"`
	Kind      string     `json:"kind"`
	DueOn     *string    `json:"due_on"`     // null: never recalled
	LastScore *float64   `json:"last_score"` // null: never recalled
	LastAt    *time.Time `json:"last_at"`
}

// ideas returns a note's key phrases, at most 12, stemmed in its language.
func ideas(content string) ([]string, [][]string, string) {
	text := knowledge.Clean(content)
	lang := knowledge.Language(text)
	var texts []string
	var stems [][]string
	for _, p := range knowledge.KeyPhrases(text, 12) {
		s, _ := knowledge.TokensIn(p.Text, lang)
		if len(s) > 0 {
			texts, stems = append(texts, p.Text), append(stems, s)
		}
	}
	return texts, stems, lang
}

// Compare scores a recall against a note: an idea counts as recalled when
// at least half of its words (rounded up) appear, as such or as synonyms.
func Compare(content, recall string) (float64, []Idea) {
	texts, stems, lang := ideas(content)
	got, _ := knowledge.TokensIn(knowledge.Clean(recall), lang)
	have := map[string]bool{}
	for _, s := range got {
		have[s] = true
	}
	found := func(stem string) bool {
		if have[stem] {
			return true
		}
		for s := range have {
			if knowledge.Synonyms(stem, s) {
				return true
			}
		}
		return false
	}
	out := make([]Idea, len(texts))
	hit := 0
	for i, words := range stems {
		n := 0
		for _, w := range words {
			if found(w) {
				n++
			}
		}
		out[i] = Idea{Text: texts[i], Recalled: n*2 >= len(words)}
		if out[i].Recalled {
			hit++
		}
	}
	if len(out) == 0 {
		return 0, []Idea{}
	}
	return float64(hit) / float64(len(out)), out
}

// nextInterval spaces recalls: well recalled (80%+), the gap grows about
// 2.5 times (1, 3, 7, 18, 45 days…); half recalled, the same gap again;
// less, back to tomorrow.
func nextInterval(prev int, score float64) int {
	switch {
	case score >= 0.8 && prev == 0:
		return 3
	case score >= 0.8:
		return min(365, int(math.Ceil(float64(prev)*2.5)))
	case score >= 0.5:
		return max(1, prev)
	}
	return 1
}

// Recall records an attempt at a literature or permanent note.
func Recall(ctx context.Context, tx pgx.Tx, userID, noteID uuid.UUID, text string, now time.Time) (RecallResult, error) {
	if strings.TrimSpace(text) == "" || len(text) > maxRecall {
		return RecallResult{}, ValidationError{"Write what you remember, up to 20,000 characters."}
	}
	n, err := notes.Get(ctx, tx, userID, noteID)
	if err != nil {
		return RecallResult{}, err
	}
	if n.Kind != "literature" && n.Kind != "permanent" {
		return RecallResult{}, ValidationError{"Free recall is for literature and permanent notes: turn this note into one first."}
	}
	score, found := Compare(*n.Content, text)
	if len(found) == 0 {
		return RecallResult{}, ValidationError{"This note is too short to recall: write a few sentences in it first."}
	}
	loc := zone(ctx, tx, userID)
	prev := 0
	var lastDue, lastAt time.Time
	err = tx.QueryRow(ctx, `SELECT due_on, recalled_at FROM note_recalls WHERE user_id = $1 AND note_id = $2 ORDER BY recalled_at DESC LIMIT 1`,
		userID, noteID).Scan(&lastDue, &lastAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return RecallResult{}, err
	default:
		prev = max(1, int(lastDue.Sub(day(lastAt, loc)).Hours()/24))
	}
	interval := nextInterval(prev, score)
	due := day(now, loc).AddDate(0, 0, interval)
	if _, err := tx.Exec(ctx, `INSERT INTO note_recalls (id, user_id, note_id, score, recalled_at, due_on) VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.Must(uuid.NewV7()), userID, noteID, score, now, due); err != nil {
		return RecallResult{}, err
	}
	return RecallResult{Score: score, Ideas: found, DueOn: due.Format(time.DateOnly), Interval: interval, At: now}, nil
}

// zone is the person's time zone: days are theirs, not the server's.
func zone(ctx context.Context, q db.Querier, userID uuid.UUID) *time.Location {
	var tz string
	if err := q.QueryRow(ctx, `SELECT timezone FROM users WHERE id = $1`, userID).Scan(&tz); err == nil {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return time.UTC
}

// day is the calendar day of t in loc, as a UTC midnight (how dates are stored).
func day(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// RecallDue lists notes whose next recall has come, oldest first, and up
// to suggest study notes never recalled, written at least a day ago (a
// first recall after a gap is what makes it work).
func RecallDue(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time, suggest int) (due, fresh []RecallNote, err error) {
	rows, err := q.Query(ctx, `SELECT n.id, e.title, n.path, n.kind, to_char(r.due_on, 'YYYY-MM-DD'), r.score::float8, r.recalled_at
		FROM notes n JOIN entities e ON e.id = n.id
		JOIN LATERAL (SELECT due_on, score, recalled_at FROM note_recalls WHERE note_id = n.id ORDER BY recalled_at DESC LIMIT 1) r ON true
		WHERE n.user_id = $1 AND n.kind = ANY($2) AND r.due_on <= $3 ORDER BY r.due_on, e.title LIMIT 50`, userID, recallKinds, day(now, zone(ctx, q, userID)))
	if err != nil {
		return nil, nil, err
	}
	if due, err = pgx.CollectRows(rows, pgx.RowToStructByPos[RecallNote]); err != nil {
		return nil, nil, err
	}
	rows, err = q.Query(ctx, `SELECT n.id, e.title, n.path, n.kind, NULL::text, NULL::float8, NULL::timestamptz
		FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND n.kind = ANY($2) AND e.created_at < $3
		  AND NOT EXISTS (SELECT 1 FROM note_recalls r WHERE r.note_id = n.id)
		  AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = n.id)
		ORDER BY e.created_at, n.id LIMIT $4`, userID, recallKinds, now.Add(-24*time.Hour), suggest)
	if err != nil {
		return nil, nil, err
	}
	if fresh, err = pgx.CollectRows(rows, pgx.RowToStructByPos[RecallNote]); err != nil {
		return nil, nil, err
	}
	if due == nil {
		due = []RecallNote{}
	}
	if fresh == nil {
		fresh = []RecallNote{}
	}
	return due, fresh, nil
}

// RecallHistory is a note's last attempts, newest first.
type RecallHistory struct {
	Score float64   `json:"score"`
	At    time.Time `json:"recalled_at"`
	DueOn string    `json:"due_on"`
}

// History returns up to 20 attempts at a note.
func History(ctx context.Context, q db.Querier, userID, noteID uuid.UUID) ([]RecallHistory, error) {
	if _, err := notes.Get(ctx, q, userID, noteID); err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT score::float8, recalled_at, to_char(due_on, 'YYYY-MM-DD') FROM note_recalls
		WHERE user_id = $1 AND note_id = $2 ORDER BY recalled_at DESC LIMIT 20`, userID, noteID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[RecallHistory])
	if out == nil {
		out = []RecallHistory{}
	}
	return out, err
}
