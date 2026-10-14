package practice

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

// ErrNotFound means no such problem for this person.
var ErrNotFound = errors.New("not found")

// Platforms, difficulties, and outcomes a problem can have.
var (
	Platforms    = []string{"leetcode", "codeforces", "hackerrank", "advent-of-code", "exercism", "project-euler", "overthewire", "portswigger", "hackthebox", "tryhackme", "kaggle", "other"}
	Difficulties = []string{"easy", "medium", "hard"}
	Outcomes     = []string{"solved", "with_help", "not_solved"}
)

// AttemptOut is an attempt as the API shows it.
type AttemptOut struct {
	ID      uuid.UUID `json:"id"`
	On      string    `json:"on"`
	Outcome string    `json:"outcome"`
	Minutes *int      `json:"minutes"`
}

// Problem is a practice problem with its attempts, oldest first.
type Problem struct {
	ID         uuid.UUID    `json:"id"`
	Title      string       `json:"title"`
	URL        *string      `json:"url"`
	Platform   string       `json:"platform"`
	Difficulty string       `json:"difficulty"`
	Topics     []string     `json:"topics"`
	Language   string       `json:"language"`
	SkillID    *uuid.UUID   `json:"skill_id"`
	CreatedAt  time.Time    `json:"created_at"`
	Attempts   []AttemptOut `json:"attempts"`
	Solved     bool         `json:"solved"` // solved unaided at least once
	DueOn      *string      `json:"due_on"` // the day to re-solve it, if any
}

// Input is a problem and its first (or next) attempt.
type Input struct {
	Title      string     `json:"title"`
	URL        string     `json:"url"`
	Platform   string     `json:"platform"`
	Difficulty string     `json:"difficulty"`
	Topics     []string   `json:"topics"`
	Language   string     `json:"language"`
	SkillID    *uuid.UUID `json:"skill_id"`
	AttemptInput
}

// AttemptInput is an attempt: how it went, how long, and when.
type AttemptInput struct {
	Outcome string `json:"outcome"`
	Minutes *int   `json:"minutes"`
	On      string `json:"on"`
}

func (a *AttemptInput) check(today time.Time) (time.Time, error) {
	if !slices.Contains(Outcomes, a.Outcome) {
		return time.Time{}, ValidationError{"An attempt is solved, with_help, or not_solved."}
	}
	if a.Minutes != nil && (*a.Minutes < 1 || *a.Minutes > 1440) {
		return time.Time{}, ValidationError{"Minutes are 1 to 1,440."}
	}
	if a.On == "" {
		return today, nil
	}
	on, err := time.Parse(time.DateOnly, a.On)
	if err != nil || on.After(today) {
		return time.Time{}, ValidationError{"Log an attempt for today or a day before it."}
	}
	return on, nil
}

func (in *Input) validate() error {
	in.Title = strings.TrimSpace(in.Title)
	in.URL = strings.TrimSpace(in.URL)
	in.Language = strings.ToLower(strings.TrimSpace(in.Language))
	switch {
	case in.Title == "" || utf8.RuneCountInString(in.Title) > 300:
		return ValidationError{"A problem needs a title of 1 to 300 characters."}
	case !slices.Contains(Platforms, in.Platform):
		return ValidationError{"Choose a platform from the list, or other."}
	case !slices.Contains(Difficulties, in.Difficulty):
		return ValidationError{"Difficulty is easy, medium, or hard."}
	case utf8.RuneCountInString(in.Language) > 20:
		return ValidationError{"The language is up to 20 characters."}
	}
	if in.URL != "" {
		u, err := url.Parse(in.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(in.URL) > 2000 {
			return ValidationError{"The link is an http or https address."}
		}
	}
	var topics []string
	for _, t := range in.Topics {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" && !slices.Contains(topics, t) {
			if utf8.RuneCountInString(t) > 40 {
				return ValidationError{"A topic is up to 40 characters."}
			}
			topics = append(topics, t)
		}
	}
	if len(topics) > 10 {
		return ValidationError{"Up to 10 topics."}
	}
	in.Topics = append([]string{}, topics...)
	return nil
}

// Log records an attempt at a problem, creating the problem the first time
// (a problem with the same link is the same problem). It reports whether
// the problem is new.
func Log(ctx context.Context, tx pgx.Tx, userID uuid.UUID, in Input, today time.Time) (Problem, bool, error) {
	if err := in.validate(); err != nil {
		return Problem{}, false, err
	}
	on, err := in.check(today)
	if err != nil {
		return Problem{}, false, err
	}
	if in.SkillID != nil {
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM entities WHERE id = $1 AND user_id = $2 AND type = 'skill')`, in.SkillID, userID).Scan(&ok); err != nil {
			return Problem{}, false, err
		}
		if !ok {
			return Problem{}, false, ValidationError{"No such skill."}
		}
	}
	var id uuid.UUID
	created := false
	if in.URL != "" {
		err := tx.QueryRow(ctx, `SELECT id FROM practice_problems WHERE user_id = $1 AND url = $2`, userID, in.URL).Scan(&id)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return Problem{}, false, err
		}
	}
	if id == uuid.Nil {
		id, created = uuid.Must(uuid.NewV7()), true
		var u *string
		if in.URL != "" {
			u = &in.URL
		}
		if _, err := tx.Exec(ctx, `INSERT INTO practice_problems (id, user_id, title, url, platform, difficulty, topics, language, skill_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, id, userID, in.Title, u, in.Platform, in.Difficulty, in.Topics, in.Language, in.SkillID); err != nil {
			return Problem{}, false, err
		}
	}
	if err := addAttempt(ctx, tx, id, on, in.AttemptInput); err != nil {
		return Problem{}, false, err
	}
	p, err := Get(ctx, tx, userID, id)
	return p, created, err
}

func addAttempt(ctx context.Context, tx pgx.Tx, problem uuid.UUID, on time.Time, a AttemptInput) error {
	_, err := tx.Exec(ctx, `INSERT INTO practice_attempts (id, problem_id, tried_on, outcome, minutes) VALUES ($1, $2, $3, $4, $5)`,
		uuid.Must(uuid.NewV7()), problem, on.Format(time.DateOnly), a.Outcome, a.Minutes)
	return err
}

// AddAttempt logs another attempt at one of the person's problems.
func AddAttempt(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, a AttemptInput, today time.Time) (Problem, error) {
	on, err := a.check(today)
	if err != nil {
		return Problem{}, err
	}
	if _, err := Get(ctx, tx, userID, id); err != nil {
		return Problem{}, err
	}
	if err := addAttempt(ctx, tx, id, on, a); err != nil {
		return Problem{}, err
	}
	return Get(ctx, tx, userID, id)
}

// Delete removes one of the person's problems with its attempts.
func Delete(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID) error {
	tag, err := tx.Exec(ctx, `DELETE FROM practice_problems WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

const columns = `p.id, p.title, p.url, p.platform, p.difficulty, p.topics, p.language, p.skill_id, p.created_at,
	coalesce((SELECT json_agg(json_build_object('id', a.id, 'on', a.tried_on, 'outcome', a.outcome, 'minutes', a.minutes)
		ORDER BY a.tried_on, a.created_at) FROM practice_attempts a WHERE a.problem_id = p.id), '[]')`

func scan(row pgx.Row) (Problem, error) {
	var p Problem
	err := row.Scan(&p.ID, &p.Title, &p.URL, &p.Platform, &p.Difficulty, &p.Topics, &p.Language, &p.SkillID, &p.CreatedAt, &p.Attempts)
	if err != nil {
		return p, err
	}
	var as []Attempt
	for _, a := range p.Attempts {
		on, _ := time.Parse(time.DateOnly, a.On)
		as = append(as, Attempt{On: on, Outcome: a.Outcome})
		p.Solved = p.Solved || a.Outcome == "solved"
	}
	if n := Next(p.Difficulty, as); n != nil {
		s := n.Format(time.DateOnly)
		p.DueOn = &s
	}
	return p, nil
}

// Get returns one of the person's problems.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Problem, error) {
	p, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM practice_problems p WHERE p.id = $1 AND p.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

// List returns the person's problems, most recently tried first; with due,
// only those to re-solve by today.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, today time.Time, due bool) ([]Problem, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM practice_problems p WHERE p.user_id = $1
		ORDER BY (SELECT max(tried_on) FROM practice_attempts a WHERE a.problem_id = p.id) DESC NULLS LAST, p.id DESC LIMIT 1000`, userID)
	if err != nil {
		return nil, err
	}
	all, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Problem, error) { return scan(r) })
	if err != nil {
		return nil, err
	}
	out := []Problem{}
	for _, p := range all {
		if !due || (p.DueOn != nil && *p.DueOn <= today.Format(time.DateOnly)) {
			out = append(out, p)
		}
	}
	return out, nil
}

// Count is a label with how many problems are solved and in all.
type Count struct {
	Topic  string `json:"topic,omitempty"`
	Solved int    `json:"solved"`
	Total  int    `json:"total"`
}

// Week is how many problems were first solved unaided in a week (Monday).
type Week struct {
	Week   string `json:"week"`
	Solved int    `json:"solved"`
}

// Stats sums up the log.
type Stats struct {
	Problems     int            `json:"problems"`
	Solved       int            `json:"solved"`
	Due          int            `json:"due"`
	ByDifficulty map[string]int `json:"by_difficulty"` // solved
	ByPlatform   map[string]int `json:"by_platform"`   // solved
	Topics       []Count        `json:"topics"`
	Weeks        []Week         `json:"weeks"` // the last 12, oldest first
	Minutes      int            `json:"minutes"`
}

// Summarise works the stats out from the problems.
func Summarise(problems []Problem, today time.Time) Stats {
	s := Stats{ByDifficulty: map[string]int{"easy": 0, "medium": 0, "hard": 0}, ByPlatform: map[string]int{}, Topics: []Count{}}
	monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	start := monday.AddDate(0, 0, -7*11)
	for i := range 12 {
		s.Weeks = append(s.Weeks, Week{Week: start.AddDate(0, 0, 7*i).Format(time.DateOnly)})
	}
	topics := map[string]*Count{}
	for _, p := range problems {
		s.Problems++
		if p.DueOn != nil && *p.DueOn <= today.Format(time.DateOnly) {
			s.Due++
		}
		for _, a := range p.Attempts {
			if a.Minutes != nil {
				s.Minutes += *a.Minutes
			}
		}
		for _, t := range p.Topics {
			if topics[t] == nil {
				topics[t] = &Count{Topic: t}
			}
			topics[t].Total++
			if p.Solved {
				topics[t].Solved++
			}
		}
		if !p.Solved {
			continue
		}
		s.Solved++
		s.ByDifficulty[p.Difficulty]++
		s.ByPlatform[p.Platform]++
		for _, a := range p.Attempts {
			if a.Outcome != "solved" {
				continue
			}
			on, _ := time.Parse(time.DateOnly, a.On)
			if w := int(on.Sub(start).Hours() / 24 / 7); !on.Before(start) && w < 12 {
				s.Weeks[w].Solved++
			}
			break // the first unaided solve
		}
	}
	for _, c := range topics {
		s.Topics = append(s.Topics, *c)
	}
	slices.SortFunc(s.Topics, func(a, b Count) int {
		if a.Total != b.Total {
			return b.Total - a.Total
		}
		return strings.Compare(a.Topic, b.Topic)
	})
	return s
}

// Export is the practice part of a personal data export.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return List(ctx, q, userID, time.Now(), false)
}
