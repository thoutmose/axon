// Package certs keeps the person's certifications (EP-104): the exams they
// plan, book, pass, or fail, what they cost, and when they expire.
package certs

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// ErrNotFound means no such certification for this person.
var ErrNotFound = errors.New("not found")

// Statuses a certification can be in.
var Statuses = []string{"planned", "booked", "passed", "failed"}

// Fields are what a person writes about a certification.
type Fields struct {
	Name           string     `json:"name"`
	Provider       string     `json:"provider"`
	Code           string     `json:"code"`
	URL            *string    `json:"url"`
	Status         string     `json:"status"`
	TargetOn       *string    `json:"target_on"`
	ExamOn         *string    `json:"exam_on"`
	PassedOn       *string    `json:"passed_on"`
	ExpiresOn      *string    `json:"expires_on"`
	ValidityMonths *int       `json:"validity_months"`
	Score          string     `json:"score"`
	CostCents      *int64     `json:"cost_cents"`
	Currency       string     `json:"currency"`
	CredentialURL  *string    `json:"credential_url"`
	Notes          string     `json:"notes"`
	SkillID        *uuid.UUID `json:"skill_id"`
}

// Cert is a stored certification.
type Cert struct {
	ID uuid.UUID `json:"id"`
	Fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// badLink is true when a link is given and is not an http or https address.
func badLink(p *string) bool {
	if p == nil || *p == "" {
		return false
	}
	u, err := url.Parse(*p)
	return err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(*p) > 2000
}

func date(p *string) (*time.Time, error) {
	if p == nil || *p == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, *p)
	if err != nil {
		return nil, ValidationError{"Dates are YYYY-MM-DD."}
	}
	return &t, nil
}

// check validates the fields and works out the expiry of a pass.
func (f *Fields) check() error {
	f.Name, f.Provider, f.Code, f.Score = strings.TrimSpace(f.Name), strings.TrimSpace(f.Provider), strings.TrimSpace(f.Code), strings.TrimSpace(f.Score)
	f.Currency = strings.ToUpper(strings.TrimSpace(f.Currency))
	if f.Currency == "" {
		f.Currency = "USD"
	}
	switch {
	case f.Name == "" || utf8.RuneCountInString(f.Name) > 200:
		return ValidationError{"A certification needs a name of 1 to 200 characters."}
	case utf8.RuneCountInString(f.Provider) > 100 || utf8.RuneCountInString(f.Code) > 40 || utf8.RuneCountInString(f.Score) > 40:
		return ValidationError{"Provider, code, and score are up to 100, 40, and 40 characters."}
	case !slices.Contains(Statuses, f.Status):
		return ValidationError{"Status is planned, booked, passed, or failed."}
	case f.CostCents != nil && *f.CostCents < 0:
		return ValidationError{"The cost cannot be negative."}
	case len(f.Currency) != 3 || strings.ToUpper(f.Currency) != f.Currency:
		return ValidationError{"The currency is a three-letter code, like USD or EUR."}
	case f.ValidityMonths != nil && (*f.ValidityMonths < 1 || *f.ValidityMonths > 240):
		return ValidationError{"Validity is 1 to 240 months."}
	case utf8.RuneCountInString(f.Notes) > 5000:
		return ValidationError{"Notes are limited to 5,000 characters."}
	}
	if badLink(f.URL) {
		return ValidationError{"The link is an http or https address."}
	}
	if badLink(f.CredentialURL) {
		return ValidationError{"The credential link is an http or https address."}
	}
	var dates [4]*time.Time
	for i, p := range []*string{f.TargetOn, f.ExamOn, f.PassedOn, f.ExpiresOn} {
		d, err := date(p)
		if err != nil {
			return err
		}
		dates[i] = d
	}
	exam, passed := dates[1], dates[2]
	switch f.Status {
	case "booked", "failed":
		if exam == nil {
			return ValidationError{"A booked or failed exam has its date."}
		}
	case "passed":
		if passed == nil {
			if exam == nil {
				return ValidationError{"A passed certification has the day it was passed."}
			}
			s := exam.Format(time.DateOnly)
			f.PassedOn, passed = &s, exam
		}
		if dates[3] == nil && f.ValidityMonths != nil {
			s := passed.AddDate(0, *f.ValidityMonths, 0).Format(time.DateOnly)
			f.ExpiresOn = &s
		}
	}
	return nil
}

const columns = `id, name, provider, code, url, status, target_on::text, exam_on::text, passed_on::text, expires_on::text,
	validity_months, score, cost_cents, currency, credential_url, notes, skill_id, created_at, updated_at`

func scan(row pgx.Row) (Cert, error) {
	var c Cert
	err := row.Scan(&c.ID, &c.Name, &c.Provider, &c.Code, &c.URL, &c.Status, &c.TargetOn, &c.ExamOn, &c.PassedOn, &c.ExpiresOn,
		&c.ValidityMonths, &c.Score, &c.CostCents, &c.Currency, &c.CredentialURL, &c.Notes, &c.SkillID, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func nullable(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

func ownSkill(ctx context.Context, q db.Querier, userID uuid.UUID, id *uuid.UUID) error {
	if id == nil {
		return nil
	}
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM entities WHERE id = $1 AND user_id = $2 AND type = 'skill')`, id, userID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ValidationError{"No such skill."}
	}
	return nil
}

// Create adds a certification.
func Create(ctx context.Context, q db.Querier, userID uuid.UUID, f Fields) (Cert, error) {
	if err := f.check(); err != nil {
		return Cert{}, err
	}
	if err := ownSkill(ctx, q, userID, f.SkillID); err != nil {
		return Cert{}, err
	}
	return scan(q.QueryRow(ctx, `INSERT INTO certifications (id, user_id, name, provider, code, url, status, target_on, exam_on, passed_on,
		expires_on, validity_months, score, cost_cents, currency, credential_url, notes, skill_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18) RETURNING `+columns,
		uuid.Must(uuid.NewV7()), userID, f.Name, f.Provider, f.Code, nullable(f.URL), f.Status, nullable(f.TargetOn), nullable(f.ExamOn),
		nullable(f.PassedOn), nullable(f.ExpiresOn), f.ValidityMonths, f.Score, f.CostCents, f.Currency, nullable(f.CredentialURL), f.Notes, f.SkillID))
}

// Replace rewrites one of the person's certifications.
func Replace(ctx context.Context, q db.Querier, userID, id uuid.UUID, f Fields) (Cert, error) {
	if err := f.check(); err != nil {
		return Cert{}, err
	}
	if err := ownSkill(ctx, q, userID, f.SkillID); err != nil {
		return Cert{}, err
	}
	c, err := scan(q.QueryRow(ctx, `UPDATE certifications SET name = $3, provider = $4, code = $5, url = $6, status = $7, target_on = $8,
		exam_on = $9, passed_on = $10, expires_on = $11, validity_months = $12, score = $13, cost_cents = $14, currency = $15,
		credential_url = $16, notes = $17, skill_id = $18, updated_at = now() WHERE id = $1 AND user_id = $2 RETURNING `+columns,
		id, userID, f.Name, f.Provider, f.Code, nullable(f.URL), f.Status, nullable(f.TargetOn), nullable(f.ExamOn), nullable(f.PassedOn),
		nullable(f.ExpiresOn), f.ValidityMonths, f.Score, f.CostCents, f.Currency, nullable(f.CredentialURL), f.Notes, f.SkillID))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// Delete removes one of the person's certifications.
func Delete(ctx context.Context, q db.Querier, userID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM certifications WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// List returns the person's certifications: upcoming exams first, then by target.
func List(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Cert, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM certifications WHERE user_id = $1
		ORDER BY coalesce(exam_on, target_on, passed_on) NULLS LAST, name LIMIT 500`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Cert, error) { return scan(r) })
	if out == nil {
		out = []Cert{}
	}
	return out, err
}

// Export is the certifications part of a personal data export.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return List(ctx, q, userID)
}
