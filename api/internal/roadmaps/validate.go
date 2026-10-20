package roadmaps

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
)

// A domain is proven, not ticked (EP-181, after École 42's defenses): the
// person fills in the skill's evaluation sheet, each item of its proof
// checked with where the evidence is, explains it in their own words, and
// gets a mark: 100 when every mandatory item holds, up to 125 with the
// bonus items, 0 when one is missing, to try again. A pass marks the skill
// done, reaches its "<skill> proven" milestone, and counts it in the goals
// it supports that are measured in domains proven.

// ValidationItem is one line of the sheet.
type ValidationItem struct {
	Text     string `json:"text"`
	Kind     string `json:"kind"` // mandatory or bonus
	OK       bool   `json:"ok"`
	Evidence string `json:"evidence"`
}

// Validation is one attempt at the sheet.
type Validation struct {
	ID          uuid.UUID        `json:"id"`
	SkillID     uuid.UUID        `json:"skill_id"`
	Attempt     int              `json:"attempt"`
	Items       []ValidationItem `json:"items"`
	Explanation string           `json:"explanation"`
	Mark        int              `json:"mark"` // 0, or 100 to 125
	Flag        string           `json:"flag"` // ok, outstanding, or incomplete
	CreatedAt   time.Time        `json:"created_at"`
}

// Sheet is what the person fills in: the items of the last attempt, to check
// again, or the skill's own proof criteria, and the attempts so far.
type Sheet struct {
	SkillID   uuid.UUID        `json:"skill_id"`
	Title     string           `json:"title"`
	Status    string           `json:"status"`
	Items     []ValidationItem `json:"items"`
	Attempts  []Validation     `json:"attempts"`
	Milestone *uuid.UUID       `json:"milestone"` // the "<skill> proven" milestone the plan made, if any
}

const (
	maxSheetItems = 30
	provenUnit    = "domains proven"
	bonusPoints   = 25
)

var proofRe = regexp.MustCompile(`(?i)\bproof:\s*`)

// Criteria reads a skill's proof criteria from its description: what follows
// "Proof:", one item per comma or semicolon outside parentheses, the final
// period dropped. Nil when the description names no proof.
func Criteria(description string) []string {
	loc := proofRe.FindStringIndex(description)
	if loc == nil {
		return nil
	}
	rest := strings.TrimSuffix(strings.TrimSpace(description[loc[1]:]), ".")
	var out []string
	depth, start := 0, 0
	for i, r := range rest {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',', ';':
			if depth == 0 {
				if p := strings.TrimSpace(rest[start:i]); p != "" {
					out = append(out, p)
				}
				start = i + 1
			}
		}
	}
	if p := strings.TrimSpace(rest[start:]); p != "" {
		out = append(out, p)
	}
	return out
}

// checkItems cleans a sheet's items and refuses what cannot be kept.
func checkItems(items []ValidationItem, explanation string) error {
	if len(items) == 0 || len(items) > maxSheetItems {
		return ValidationError{"An evaluation sheet has 1 to 30 items."}
	}
	mandatory := 0
	for i := range items {
		it := &items[i]
		it.Text = strings.TrimSpace(it.Text)
		it.Evidence = strings.TrimSpace(it.Evidence)
		if it.Text == "" || len([]rune(it.Text)) > 300 {
			return ValidationError{"Each item is 1 to 300 characters."}
		}
		if len([]rune(it.Evidence)) > 500 {
			return ValidationError{"The evidence for an item is up to 500 characters."}
		}
		switch it.Kind {
		case "", "mandatory":
			it.Kind = "mandatory"
			mandatory++
		case "bonus":
		default:
			return ValidationError{"An item is mandatory or a bonus."}
		}
	}
	if mandatory == 0 {
		return ValidationError{"At least one item is mandatory."}
	}
	if len([]rune(explanation)) > 5000 {
		return ValidationError{"The explanation is up to 5,000 characters."}
	}
	return nil
}

// grade marks an attempt: 0 and incomplete when a mandatory item is missing;
// else 100 plus the bonus items' share of 25, outstanding when all hold.
func grade(items []ValidationItem) (mark int, flag string) {
	bonus, bonusOK := 0, 0
	for _, it := range items {
		if it.Kind == "bonus" {
			bonus++
			if it.OK {
				bonusOK++
			}
			continue
		}
		if !it.OK {
			return 0, "incomplete"
		}
	}
	mark = 100
	if bonus > 0 {
		mark += (bonusPoints*bonusOK + bonus/2) / bonus
	}
	if mark >= 100+bonusPoints {
		return mark, "outstanding"
	}
	return mark, "ok"
}

const validationColumns = `id, skill_id, attempt, items, explanation, mark, flag, created_at`

func scanValidation(r pgx.Row) (Validation, error) {
	var v Validation
	err := r.Scan(&v.ID, &v.SkillID, &v.Attempt, &v.Items, &v.Explanation, &v.Mark, &v.Flag, &v.CreatedAt)
	if v.Items == nil {
		v.Items = []ValidationItem{}
	}
	return v, err
}

func attempts(ctx context.Context, q db.Querier, userID, skillID uuid.UUID) ([]Validation, error) {
	rows, err := q.Query(ctx, `SELECT `+validationColumns+` FROM skill_validations WHERE user_id = $1 AND skill_id = $2
		ORDER BY attempt DESC`, userID, skillID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Validation, error) { return scanValidation(r) })
	if out == nil {
		out = []Validation{}
	}
	return out, err
}

// provenMilestone finds the "<skill> proven" milestone related to a skill.
func provenMilestone(ctx context.Context, q db.Querier, userID uuid.UUID, s Skill, openOnly bool) (*uuid.UUID, error) {
	var id *uuid.UUID
	err := q.QueryRow(ctx, `SELECT m.id FROM links l JOIN milestones m ON m.id = l.target_id JOIN entities e ON e.id = m.id
		WHERE l.user_id = $1 AND l.source_id = $2 AND l.relation = 'relates_to' AND lower(e.title) = lower($3)
		  AND ($4 = false OR m.done_at IS NULL) LIMIT 1`, userID, s.ID, s.Title+" proven", openOnly).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return id, err
}

// GetSheet returns a skill's evaluation sheet.
func GetSheet(ctx context.Context, q db.Querier, userID, skillID uuid.UUID) (Sheet, error) {
	s, _, err := GetSkill(ctx, q, userID, skillID)
	if err != nil {
		return Sheet{}, err
	}
	out := Sheet{SkillID: s.ID, Title: s.Title, Status: s.Status, Items: []ValidationItem{}}
	if out.Attempts, err = attempts(ctx, q, userID, skillID); err != nil {
		return out, err
	}
	switch {
	case len(out.Attempts) > 0:
		// A failed attempt is checked again from the start; a passed one is read as it was.
		for _, it := range out.Attempts[0].Items {
			if out.Attempts[0].Flag == "incomplete" {
				it.OK = false
			}
			out.Items = append(out.Items, it)
		}
	default:
		for _, c := range Criteria(s.Description) {
			out.Items = append(out.Items, ValidationItem{Text: c, Kind: "mandatory"})
		}
	}
	out.Milestone, err = provenMilestone(ctx, q, userID, s, false)
	return out, err
}

// Validate keeps an attempt and, when it passes, marks the skill done,
// reaches its milestone, and counts it in the goals it supports.
func Validate(ctx context.Context, tx pgx.Tx, userID, skillID uuid.UUID, actor graph.Actor, items []ValidationItem, explanation string) (Validation, error) {
	explanation = strings.TrimSpace(explanation)
	if err := checkItems(items, explanation); err != nil {
		return Validation{}, err
	}
	s, _, err := GetSkill(ctx, tx, userID, skillID)
	if err != nil {
		return Validation{}, err
	}
	mark, flag := grade(items)
	if flag != "incomplete" && explanation == "" {
		return Validation{}, ValidationError{"Explain it in your own words before validating: what it is and why it works, as if to a beginner."}
	}
	v := Validation{ID: uuid.Must(uuid.NewV7()), SkillID: s.ID, Items: items, Explanation: explanation, Mark: mark, Flag: flag}
	if err := tx.QueryRow(ctx, `INSERT INTO skill_validations (id, user_id, skill_id, attempt, items, explanation, mark, flag)
		VALUES ($1, $2, $3, (SELECT coalesce(max(attempt), 0) + 1 FROM skill_validations WHERE skill_id = $3), $4, $5, $6, $7)
		RETURNING attempt, created_at`, v.ID, userID, s.ID, items, explanation, mark, flag).Scan(&v.Attempt, &v.CreatedAt); err != nil {
		return v, err
	}
	if flag == "incomplete" {
		return v, nil
	}
	if s.Status != "done" {
		done := "done"
		if _, err := UpdateSkill(ctx, tx, userID, s.ID, actor, SkillFields{Status: &done}); err != nil {
			return v, err
		}
	}
	mid, err := provenMilestone(ctx, tx, userID, s, true)
	if err != nil {
		return v, err
	}
	if mid != nil {
		cur, err := goals.GetMilestone(ctx, tx, userID, *mid)
		if err != nil {
			return v, err
		}
		yes := true
		if _, err := goals.UpdateMilestone(ctx, tx, userID, *mid, actor, cur.UpdatedAt, goals.MilestoneFields{Done: &yes}); err != nil {
			return v, err
		}
	}
	// The goals it supports that are measured in domains proven count their
	// skills done.
	_, err = tx.Exec(ctx, `UPDATE goals g SET measure_current = (SELECT count(*) FROM links l JOIN skills s ON s.id = l.source_id
			WHERE l.user_id = $1 AND l.target_id = g.id AND l.relation = 'supports' AND s.status = 'done'),
			updated_at = clock_timestamp()
		WHERE g.measure_unit = $2
		  AND g.id IN (SELECT l.target_id FROM links l WHERE l.user_id = $1 AND l.source_id = $3 AND l.relation = 'supports')`,
		userID, provenUnit, s.ID)
	return v, err
}

// ExportValidations is the evaluation sheets part of a personal data export.
func ExportValidations(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+validationColumns+` FROM skill_validations WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Validation, error) { return scanValidation(r) })
	if out == nil {
		out = []Validation{}
	}
	return out, err
}
