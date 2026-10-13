package roadmaps

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// The skills assessment (EP-41): before a roadmap, the person describes their
// situation and rates themselves in the domains they care about, and answers
// three quick check questions per domain. Each domain gets an explained level
// from 0 to 5, used to place them on library roadmaps and read by the Tutor.

// AssessmentType is the entity type of an assessment.
const AssessmentType = "assessment"

func init() {
	graph.RegisterType(graph.Type{Name: AssessmentType, Table: "assessments", Export: exportAssessments})
}

//go:embed assessment.json
var assessmentJSON []byte

// Question is one check question; Answer is never sent to the browser.
type Question struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Choices  []string `json:"choices"`
	Answer   int      `json:"answer,omitempty"`
}

// QuestionDomain groups a domain's questions.
type QuestionDomain struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}

var bank = func() []QuestionDomain {
	var b struct {
		Domains []QuestionDomain `json:"domains"`
	}
	if err := json.Unmarshal(assessmentJSON, &b); err != nil {
		panic(fmt.Sprintf("roadmaps: assessment.json: %v", err))
	}
	return b.Domains
}()

// Questions returns the bank without the answers.
func Questions() []QuestionDomain {
	out := make([]QuestionDomain, len(bank))
	for i, d := range bank {
		qs := make([]Question, len(d.Questions))
		for j, q := range d.Questions {
			qs[j] = Question{ID: q.ID, Question: q.Question, Choices: q.Choices}
		}
		out[i] = QuestionDomain{d.ID, d.Title, qs}
	}
	return out
}

// Profile is the person's situation.
type Profile struct {
	Role         string   `json:"role"`
	Experience   string   `json:"experience"`
	Goals        string   `json:"goals"`
	HoursPerWeek int      `json:"hours_per_week"`
	Formats      []string `json:"formats"`
}

// AssessmentInput is what the person answers.
type AssessmentInput struct {
	Profile   Profile        `json:"profile"`
	Interests []string       `json:"interests"`
	Ratings   map[string]int `json:"ratings"`
	Answers   map[string]int `json:"answers"`
}

// DomainResult is the level in one domain, with the reason for it.
type DomainResult struct {
	Domain  string `json:"domain"`
	Title   string `json:"title"`
	Rating  int    `json:"rating"`
	Correct int    `json:"correct"`
	Total   int    `json:"total"`
	Level   int    `json:"level"`
	Reason  string `json:"reason"`
}

// Assessment is a stored, scored assessment.
type Assessment struct {
	ID        uuid.UUID      `json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	Profile   Profile        `json:"profile"`
	Results   []DomainResult `json:"results"`
	Suggested []string       `json:"suggested"` // library domains to start with
}

var formats = []string{"book", "course", "video", "practice", "paper"}

func (in *AssessmentInput) validate() error {
	p := &in.Profile
	p.Role, p.Experience, p.Goals = strings.TrimSpace(p.Role), strings.TrimSpace(p.Experience), strings.TrimSpace(p.Goals)
	switch {
	case len([]rune(p.Role)) > 200 || len([]rune(p.Experience)) > 200:
		return ValidationError{"Role and experience are up to 200 characters each."}
	case len([]rune(p.Goals)) > 2000:
		return ValidationError{"Goals are up to 2,000 characters."}
	case p.HoursPerWeek < 0 || p.HoursPerWeek > 80:
		return ValidationError{"Hours per week are between 0 and 80."}
	case len(in.Interests) == 0:
		return ValidationError{"Choose at least one domain you want to learn."}
	}
	for _, f := range p.Formats {
		if !slices.Contains(formats, f) {
			return ValidationError{"Formats are book, course, video, practice, or paper."}
		}
	}
	seen := map[string]bool{}
	for _, d := range in.Interests {
		if domain(d) == nil {
			return ValidationError{fmt.Sprintf("Unknown domain %q.", d)}
		}
		if seen[d] {
			return ValidationError{fmt.Sprintf("Domain %q is listed twice.", d)}
		}
		seen[d] = true
		if r, ok := in.Ratings[d]; !ok || r < 0 || r > 5 {
			return ValidationError{fmt.Sprintf("Rate %q from 0 to 5.", d)}
		}
	}
	return nil
}

func domain(id string) *QuestionDomain {
	for i := range bank {
		if bank[i].ID == id {
			return &bank[i]
		}
	}
	return nil
}

// score turns answers into explained levels: the average of the self-rating
// and the check score on the same 0–5 scale, rounded.
func score(in AssessmentInput) ([]DomainResult, []string) {
	var results []DomainResult
	for _, id := range in.Interests {
		d := domain(id)
		r := DomainResult{Domain: id, Title: d.Title, Rating: in.Ratings[id], Total: len(d.Questions)}
		for _, q := range d.Questions {
			if a, ok := in.Answers[q.ID]; ok && a == q.Answer {
				r.Correct++
			}
		}
		checks := 5 * float64(r.Correct) / float64(max(1, r.Total))
		r.Level = int(math.Round((float64(r.Rating) + checks) / 2))
		r.Reason = fmt.Sprintf("You rated it %d/5 and answered %d of %d checks correctly, so your level is %d of 5.", r.Rating, r.Correct, r.Total, r.Level)
		results = append(results, r)
	}
	order := slices.Clone(results)
	sort.SliceStable(order, func(i, j int) bool { return order[i].Level < order[j].Level })
	var suggested []string
	for _, r := range order {
		if r.Level < 5 && len(suggested) < 3 {
			suggested = append(suggested, r.Domain)
		}
	}
	if suggested == nil {
		suggested = []string{}
	}
	return results, suggested
}

type stored struct {
	Results   []DomainResult `json:"results"`
	Suggested []string       `json:"suggested"`
}

// CreateAssessment scores and stores an assessment.
func CreateAssessment(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, in AssessmentInput, now time.Time) (Assessment, error) {
	if err := in.validate(); err != nil {
		return Assessment{}, err
	}
	if in.Profile.Formats == nil {
		in.Profile.Formats = []string{}
	}
	results, suggested := score(in)
	id, err := graph.CreateEntity(ctx, tx, userID, actor, AssessmentType, "Skills assessment "+now.Format(time.DateOnly), in.Profile.Goals)
	if err != nil {
		return Assessment{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO assessments (id, profile, results) VALUES ($1, $2, $3)`,
		id, in.Profile, stored{results, suggested}); err != nil {
		return Assessment{}, err
	}
	return GetAssessment(ctx, tx, userID, id)
}

const assessmentColumns = `a.id, a.created_at, a.profile, a.results`

func scanAssessment(row pgx.Row) (Assessment, error) {
	var a Assessment
	var s stored
	if err := row.Scan(&a.ID, &a.CreatedAt, &a.Profile, &s); err != nil {
		return a, err
	}
	a.Results, a.Suggested = s.Results, s.Suggested
	return a, nil
}

// GetAssessment returns one of the person's assessments.
func GetAssessment(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Assessment, error) {
	a, err := scanAssessment(q.QueryRow(ctx, `SELECT `+assessmentColumns+` FROM assessments a JOIN entities e ON e.id = a.id
		WHERE a.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return a, graph.ErrNotFound
	}
	return a, err
}

// ListAssessments returns the person's assessments, newest first.
func ListAssessments(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Assessment, error) {
	rows, err := q.Query(ctx, `SELECT `+assessmentColumns+` FROM assessments a JOIN entities e ON e.id = a.id
		WHERE e.user_id = $1 ORDER BY a.created_at DESC, a.id DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Assessment, error) { return scanAssessment(r) })
	if out == nil {
		out = []Assessment{}
	}
	return out, err
}

// DeleteAssessment removes an assessment.
func DeleteAssessment(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	if _, err := GetAssessment(ctx, tx, userID, id); err != nil {
		return err
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

// latestResult finds the newest assessment that covers a domain.
func latestResult(ctx context.Context, q db.Querier, userID uuid.UUID, domainID string) (*DomainResult, time.Time, error) {
	all, err := ListAssessments(ctx, q, userID)
	if err != nil {
		return nil, time.Time{}, err
	}
	for _, a := range all {
		for _, r := range a.Results {
			if r.Domain == domainID {
				return &r, a.CreatedAt, nil
			}
		}
	}
	return nil, time.Time{}, nil
}

// ImportPlaced starts a library roadmap and places the person on it with
// their latest assessments: a skill whose domain they reached level 3 of 5
// in starts as being learnt. None starts done, since each needs its proof.
func ImportPlaced(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, source string, now time.Time) (Roadmap, error) {
	r, err := Import(ctx, tx, userID, actor, source, now)
	if err != nil {
		return r, err
	}
	domain := map[string]string{}
	for _, d := range lib.Roadmaps {
		if d.ID == source {
			for _, sk := range d.Skills {
				domain[sk.Title] = sk.Assess
			}
		}
	}
	assessed, placed := false, []string{}
	learning := "learning"
	for _, sk := range r.Skills {
		if domain[sk.Title] == "" {
			continue
		}
		res, _, err := latestResult(ctx, tx, userID, domain[sk.Title])
		if err != nil {
			return r, err
		}
		if res == nil {
			continue
		}
		assessed = true
		if res.Level >= 3 {
			if _, err := UpdateSkill(ctx, tx, userID, sk.ID, actor, SkillFields{Status: &learning}); err != nil {
				return r, err
			}
			placed = append(placed, sk.Title)
		}
	}
	if !assessed {
		return Roadmap{}, ValidationError{"No assessment covers this roadmap yet. Take the skills assessment, or start it without placement."}
	}
	note := "Placed by your skills assessment: nothing reached level 3 of 5 yet, so every skill starts from the beginning."
	if len(placed) > 0 {
		note = fmt.Sprintf("Placed by your skills assessment: %s at level 3 of 5 or more, so being learnt; each is done once its proof is.", quoteList(placed))
	}
	desc := strings.TrimSpace(r.Description + "\n\n" + note)
	return Update(ctx, tx, userID, r.ID, actor, Fields{Description: &desc})
}

func quoteList(items []string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = "“" + s + "”"
	}
	if len(q) == 1 {
		return q[0] + " is"
	}
	return strings.Join(q, " and ") + " are"
}

func exportAssessments(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return ListAssessments(ctx, q, userID)
}
