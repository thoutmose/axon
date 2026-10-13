// Package journal keeps a daily entry (EP-91): mood, energy, and stress on
// a five-point scale, hours slept, up to three things the person is
// grateful for, and a note. Mood and sleep are health data under the GDPR:
// nothing is kept without the health_data consent, and withdrawing it
// deletes the journal along with the health logs.
package journal

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/privacy"
)

// Consent is the purpose the journal is kept under.
const Consent = "health_data"

func init() {
	privacy.OnWithdraw(Consent, DeleteAll)
}

// OnChange, when set, hears which day changed, in the same transaction:
// "health:journal:YYYY-MM-DD", or "health:*" when the journal was deleted
// (the vault mirror then rewrites or removes its daily notes).
var OnChange func(ctx context.Context, q db.Querier, userID uuid.UUID, key string) error

func changed(ctx context.Context, q db.Querier, userID uuid.UUID, key string) error {
	if OnChange == nil {
		return nil
	}
	return OnChange(ctx, q, userID, key)
}

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// ErrNotFound means no entry that day.
var ErrNotFound = errors.New("not found")

// Fields are what a person writes for a day; nil means not given.
type Fields struct {
	Mood       *int     `json:"mood"`
	Energy     *int     `json:"energy"`
	Stress     *int     `json:"stress"`
	SleepHours *float64 `json:"sleep_hours"`
	Gratitude  []string `json:"gratitude"`
	Note       string   `json:"note"`
}

// Entry is one day of the journal.
type Entry struct {
	Day string `json:"day"`
	Fields
	UpdatedAt time.Time `json:"updated_at"`
}

func scale(v *int) bool { return v == nil || (*v >= 1 && *v <= 5) }

func (f *Fields) check() error {
	var thanks []string
	for _, g := range f.Gratitude {
		if g = strings.TrimSpace(g); g != "" {
			thanks = append(thanks, g)
		}
	}
	f.Gratitude, f.Note = thanks, strings.TrimSpace(f.Note)
	if f.Gratitude == nil {
		f.Gratitude = []string{}
	}
	switch {
	case !scale(f.Mood) || !scale(f.Energy) || !scale(f.Stress):
		return ValidationError{"Mood, energy, and stress are 1 to 5."}
	case f.SleepHours != nil && (*f.SleepHours < 0 || *f.SleepHours > 24):
		return ValidationError{"Sleep is 0 to 24 hours."}
	case len(f.Gratitude) > 3:
		return ValidationError{"Up to three things you are grateful for."}
	case utf8.RuneCountInString(f.Note) > 10_000:
		return ValidationError{"The note is up to 10,000 characters."}
	case f.Mood == nil && f.Energy == nil && f.Stress == nil && f.SleepHours == nil && len(f.Gratitude) == 0 && f.Note == "":
		return ValidationError{"Write something about the day: a mood, your sleep, a thank-you, or a note."}
	}
	for _, g := range f.Gratitude {
		if utf8.RuneCountInString(g) > 300 {
			return ValidationError{"Each thing you are grateful for is up to 300 characters."}
		}
	}
	return nil
}

const columns = `day::text, mood, energy, stress, sleep_hours::float8, gratitude, note, updated_at`

func scan(row pgx.Row) (Entry, error) {
	var e Entry
	err := row.Scan(&e.Day, &e.Mood, &e.Energy, &e.Stress, &e.SleepHours, &e.Gratitude, &e.Note, &e.UpdatedAt)
	return e, err
}

// Put writes the entry for a day, replacing what was there.
func Put(ctx context.Context, q db.Querier, userID uuid.UUID, day, today time.Time, f Fields) (Entry, error) {
	// Calendar dates, compared as such: the person's today is local midnight.
	if day.Format(time.DateOnly) > today.Format(time.DateOnly) || day.Year() < 2000 {
		return Entry{}, ValidationError{"Write about today or a day before it."}
	}
	if err := f.check(); err != nil {
		return Entry{}, err
	}
	e, err := scan(q.QueryRow(ctx, `INSERT INTO journal_entries (user_id, day, mood, energy, stress, sleep_hours, gratitude, note, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (user_id, day) DO UPDATE SET mood = $3, energy = $4, stress = $5, sleep_hours = $6, gratitude = $7, note = $8, updated_at = now()
		RETURNING `+columns, userID, day.Format(time.DateOnly), f.Mood, f.Energy, f.Stress, f.SleepHours, f.Gratitude, f.Note))
	if err != nil {
		return e, err
	}
	return e, changed(ctx, q, userID, "health:journal:"+e.Day)
}

// Get returns the entry for a day.
func Get(ctx context.Context, q db.Querier, userID uuid.UUID, day string) (Entry, error) {
	e, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM journal_entries WHERE user_id = $1 AND day = $2`, userID, day))
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

// List returns the entries from one day to another, newest first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, from, to string) ([]Entry, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM journal_entries WHERE user_id = $1 AND day BETWEEN $2 AND $3
		ORDER BY day DESC LIMIT 400`, userID, from, to)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Entry, error) { return scan(r) })
	if out == nil {
		out = []Entry{}
	}
	return out, err
}

// Delete removes the entry for a day.
func Delete(ctx context.Context, q db.Querier, userID uuid.UUID, day string) error {
	tag, err := q.Exec(ctx, `DELETE FROM journal_entries WHERE user_id = $1 AND day = $2`, userID, day)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return changed(ctx, q, userID, "health:journal:"+day)
}

// DeleteAll removes the person's whole journal (the consent was withdrawn).
func DeleteAll(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM journal_entries WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return changed(ctx, tx, userID, "health:*")
}

// Export returns the person's journal for their export.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return List(ctx, q, userID, "0001-01-01", "9999-12-31")
}
