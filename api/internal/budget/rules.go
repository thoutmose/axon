package budget

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Rule categorises or tags a transaction whose payee or note matches.
type Rule struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Field      string     `json:"field"`    // payee or note
	Operator   string     `json:"operator"` // contains, equals, or starts_with
	Value      string     `json:"value"`
	CategoryID *uuid.UUID `json:"category_id"`
	Tag        string     `json:"tag"`
	Position   int        `json:"position"`
	Active     bool       `json:"active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Matches reports whether the rule applies to a transaction (case-insensitive).
func (r Rule) Matches(payee, note string) bool {
	hay := payee
	if r.Field == "note" {
		hay = note
	}
	hay, v := strings.ToLower(strings.TrimSpace(hay)), strings.ToLower(strings.TrimSpace(r.Value))
	switch r.Operator {
	case "equals":
		return hay == v
	case "starts_with":
		return strings.HasPrefix(hay, v)
	default:
		return strings.Contains(hay, v)
	}
}

const ruleColumns = `id, name, field, operator, value, category_id, tag, position, active, created_at, updated_at`

func scanRule(r pgx.Row) (Rule, error) {
	var x Rule
	err := r.Scan(&x.ID, &x.Name, &x.Field, &x.Operator, &x.Value, &x.CategoryID, &x.Tag, &x.Position, &x.Active, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

// Rules lists the user's rules in the order they are tried.
func Rules(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Rule, error) {
	rows, err := q.Query(ctx, `SELECT `+ruleColumns+` FROM budget_rules WHERE user_id = $1 ORDER BY position, created_at`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Rule, error) { return scanRule(r) })
	if out == nil {
		out = []Rule{}
	}
	return out, err
}

// applyRules fills the category (first matching rule whose category fits
// the kind) and adds tags, for a transaction without a category. It never
// overrides what the person chose.
func applyRules(ctx context.Context, q db.Querier, userID uuid.UUID, t *Transaction) error {
	if t.Kind == "transfer" {
		return nil
	}
	rules, err := Rules(ctx, q, userID)
	if err != nil {
		return err
	}
	kinds := map[uuid.UUID]string{}
	if len(rules) > 0 {
		cats, err := Categories(ctx, q, userID)
		if err != nil {
			return err
		}
		for _, c := range cats {
			kinds[c.ID] = c.Kind
		}
	}
	for _, r := range rules {
		if !r.Active || !r.Matches(t.Payee, t.Note) {
			continue
		}
		if r.CategoryID != nil && t.CategoryID == nil && kinds[*r.CategoryID] == t.Kind {
			id := *r.CategoryID
			t.CategoryID = &id
		}
		if r.Tag != "" && !slices.Contains(t.Tags, r.Tag) && len(t.Tags) < 20 {
			t.Tags = append(t.Tags, r.Tag)
		}
	}
	return nil
}

// ApplyRules runs the rules over uncategorised transactions. Running it
// again changes nothing.
func ApplyRules(ctx context.Context, q db.Querier, userID uuid.UUID) (int, error) {
	rows, err := q.Query(ctx, `SELECT `+transactionColumns+` FROM budget_transactions
		WHERE user_id = $1 AND category_id IS NULL AND kind <> 'transfer' LIMIT 10000`, userID)
	if err != nil {
		return 0, err
	}
	txs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Transaction, error) { return scanTransaction(r) })
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, t := range txs {
		before := len(t.Tags)
		if err := applyRules(ctx, q, userID, &t); err != nil {
			return changed, err
		}
		if t.CategoryID == nil && len(t.Tags) == before {
			continue
		}
		if _, err := q.Exec(ctx, `UPDATE budget_transactions SET category_id = $2, tags = $3, updated_at = now() WHERE id = $1`,
			t.ID, t.CategoryID, t.Tags); err != nil {
			return changed, err
		}
		changed++
	}
	return changed, nil
}

// RuleFields are the editable parts; nil means "not provided".
type RuleFields struct {
	Name, Field, Operator, Value, Tag *string
	CategoryID                        *uuid.UUID
	ClearCategory                     bool
	Position                          *int
	Active                            *bool
}

func (f *RuleFields) apply(r *Rule) {
	set := func(dst *string, src *string) {
		if src != nil {
			*dst = strings.TrimSpace(*src)
		}
	}
	set(&r.Name, f.Name)
	set(&r.Field, f.Field)
	set(&r.Operator, f.Operator)
	set(&r.Value, f.Value)
	if f.Tag != nil {
		r.Tag = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(*f.Tag), "#"))
	}
	if f.CategoryID != nil || f.ClearCategory {
		r.CategoryID = f.CategoryID
	}
	if f.Position != nil {
		r.Position = *f.Position
	}
	if f.Active != nil {
		r.Active = *f.Active
	}
}

func checkRule(ctx context.Context, q db.Querier, userID uuid.UUID, r Rule) error {
	switch {
	case r.Name == "" || len([]rune(r.Name)) > 100:
		return ValidationError{"A rule needs a name of 1 to 100 characters."}
	case r.Field != "payee" && r.Field != "note":
		return ValidationError{"A rule looks at the payee or the note."}
	case !slices.Contains([]string{"contains", "equals", "starts_with"}, r.Operator):
		return ValidationError{"A rule's operator is contains, equals, or starts_with."}
	case r.Value == "" || len([]rune(r.Value)) > 200:
		return ValidationError{"A rule needs text to match, up to 200 characters."}
	case r.Tag != "" && !tagRe.MatchString(r.Tag):
		return ValidationError{"A tag is up to 40 letters, digits, - or _."}
	case r.CategoryID == nil && r.Tag == "":
		return ValidationError{"A rule sets a category, a tag, or both."}
	}
	if r.CategoryID != nil {
		if _, err := GetCategory(ctx, q, userID, *r.CategoryID); errors.Is(err, ErrNotFound) {
			return ValidationError{"No such category."}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func getRule(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Rule, error) {
	r, err := scanRule(q.QueryRow(ctx, `SELECT `+ruleColumns+` FROM budget_rules WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// CreateRule adds a rule, tried after the existing ones.
func CreateRule(ctx context.Context, q db.Querier, userID uuid.UUID, f RuleFields) (Rule, error) {
	r := Rule{Field: "payee", Operator: "contains", Active: true}
	if err := q.QueryRow(ctx, `SELECT coalesce(max(position), 0) + 1 FROM budget_rules WHERE user_id = $1`, userID).Scan(&r.Position); err != nil {
		return r, err
	}
	f.apply(&r)
	if err := checkRule(ctx, q, userID, r); err != nil {
		return r, err
	}
	r.ID = uuid.Must(uuid.NewV7())
	if _, err := q.Exec(ctx, `INSERT INTO budget_rules (id, user_id, name, field, operator, value, category_id, tag, position, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`, r.ID, userID, r.Name, r.Field, r.Operator, r.Value, r.CategoryID, r.Tag, r.Position, r.Active); err != nil {
		return r, err
	}
	return getRule(ctx, q, userID, r.ID)
}

// UpdateRule changes a rule.
func UpdateRule(ctx context.Context, q db.Querier, userID, id uuid.UUID, f RuleFields) (Rule, error) {
	r, err := getRule(ctx, q, userID, id)
	if err != nil {
		return r, err
	}
	f.apply(&r)
	if err := checkRule(ctx, q, userID, r); err != nil {
		return r, err
	}
	if _, err := q.Exec(ctx, `UPDATE budget_rules SET name = $3, field = $4, operator = $5, value = $6, category_id = $7, tag = $8,
		position = $9, active = $10, updated_at = now() WHERE id = $1 AND user_id = $2`,
		id, userID, r.Name, r.Field, r.Operator, r.Value, r.CategoryID, r.Tag, r.Position, r.Active); err != nil {
		return r, err
	}
	return getRule(ctx, q, userID, id)
}

// DeleteRule removes a rule; transactions it already changed stay as they are.
func DeleteRule(ctx context.Context, q db.Querier, userID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM budget_rules WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
