// Package goals implements goals with measurable outcomes, milestones, and
// progress computed from linked work with an explanation of how (EP-13).
package goals

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Entity types.
const (
	Type          = "goal"
	MilestoneType = "milestone"
)

// Supports links work to the goal or milestone it moves forward. Only tasks
// count towards progress; linked notes are context.
const Supports = "supports"

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "goals", Export: export})
	graph.RegisterType(graph.Type{Name: MilestoneType, Table: "milestones", Export: exportMilestones})
	graph.RegisterRelation(graph.Relation{
		Name:    Supports,
		Sources: []string{tasks.Type, notes.Type, "habit"}, // habits (EP-12) import goals, not the reverse
		Targets: []string{Type, MilestoneType},
	})
}

var statuses = []string{"active", "achieved", "archived"}

// Errors callers translate to HTTP responses.
var ErrStale = errors.New("stale version")

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Measure is a numeric outcome: from Start to Target, currently Current.
type Measure struct {
	Unit    string  `json:"unit"`
	Start   float64 `json:"start"`
	Target  float64 `json:"target"`
	Current float64 `json:"current"`
}

// Milestone is a step towards a goal.
type Milestone struct {
	ID        uuid.UUID   `json:"id"`
	GoalID    uuid.UUID   `json:"goal_id"`
	Title     string      `json:"title"`
	DueOn     *tasks.Date `json:"due_on"`
	DoneAt    *time.Time  `json:"done_at"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Goal is the API representation.
type Goal struct {
	ID         uuid.UUID   `json:"id"`
	Title      string      `json:"title"`
	Outcome    string      `json:"outcome"`
	Status     string      `json:"status"`
	TargetDate *tasks.Date `json:"target_date"`
	Measure    *Measure    `json:"measure"`
	Progress   Progress    `json:"progress"`
	Milestones []Milestone `json:"milestones"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// Progress is a figure and the method behind it; Explain lists the inputs.
type Progress struct {
	Percent *int   `json:"percent"` // nil when there is nothing to measure yet
	Method  string `json:"method"`  // measure, items, or none
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Summary string `json:"summary"`
}

// Item is one input to an items-based progress figure.
type Item struct {
	ID    uuid.UUID `json:"id"`
	Type  string    `json:"type"` // milestone or task
	Title string    `json:"title"`
	Done  bool      `json:"done"`
	Via   *string   `json:"via"` // the milestone a task supports, if not the goal itself
}

// Explanation is a progress figure with exactly what it was computed from.
type Explanation struct {
	Progress
	Measure *Measure `json:"measure"`
	Items   []Item   `json:"items"`
}

const columns = `g.id, e.title, g.outcome, g.status, g.target_date, g.measure_unit, g.measure_start,
	g.measure_target, g.measure_current, e.created_at, g.updated_at`

func scan(row pgx.Row) (Goal, error) {
	var g Goal
	var target *time.Time
	var unit *string
	var start, tgt, cur *float64
	err := row.Scan(&g.ID, &g.Title, &g.Outcome, &g.Status, &target, &unit, &start, &tgt, &cur, &g.CreatedAt, &g.UpdatedAt)
	if target != nil {
		g.TargetDate = &tasks.Date{Time: *target}
	}
	if unit != nil && start != nil && tgt != nil && cur != nil {
		g.Measure = &Measure{Unit: *unit, Start: *start, Target: *tgt, Current: *cur}
	}
	return g, err
}

// Get returns one goal with its milestones and progress.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Goal, error) {
	g, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM goals g JOIN entities e ON e.id = g.id
		WHERE g.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return g, graph.ErrNotFound
	}
	if err != nil {
		return g, err
	}
	if g.Milestones, err = milestones(ctx, q, userID, id); err != nil {
		return g, err
	}
	ex, err := Explain(ctx, q, userID, g)
	g.Progress = ex.Progress
	return g, err
}

// List returns the user's goals with a status, soonest target first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, status string) ([]Goal, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM goals g JOIN entities e ON e.id = g.id
		WHERE e.user_id = $1 AND g.status = $2 ORDER BY g.target_date NULLS LAST, e.created_at LIMIT 200`, userID, status)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Goal, error) { return scan(row) })
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Milestones, err = milestones(ctx, q, userID, out[i].ID); err != nil {
			return nil, err
		}
		ex, err := Explain(ctx, q, userID, out[i])
		if err != nil {
			return nil, err
		}
		out[i].Progress = ex.Progress
	}
	return out, nil
}

const milestoneColumns = `m.id, m.goal_id, e.title, m.due_on, m.done_at, e.created_at, m.updated_at`

func scanMilestone(row pgx.Row) (Milestone, error) {
	var m Milestone
	var due *time.Time
	err := row.Scan(&m.ID, &m.GoalID, &m.Title, &due, &m.DoneAt, &m.CreatedAt, &m.UpdatedAt)
	if due != nil {
		m.DueOn = &tasks.Date{Time: *due}
	}
	return m, err
}

func milestones(ctx context.Context, q db.Querier, userID, goalID uuid.UUID) ([]Milestone, error) {
	rows, err := q.Query(ctx, `SELECT `+milestoneColumns+` FROM milestones m JOIN entities e ON e.id = m.id
		WHERE m.goal_id = $1 AND e.user_id = $2 ORDER BY m.due_on NULLS LAST, e.created_at`, goalID, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Milestone, error) { return scanMilestone(row) })
	if out == nil {
		out = []Milestone{}
	}
	return out, err
}

// Explain computes a goal's progress. With a measure, progress is how far
// Current has moved from Start towards Target. Otherwise it is the share of
// done items: the goal's milestones and the tasks that support the goal or
// one of its milestones (cancelled tasks are left out).
func Explain(ctx context.Context, q db.Querier, userID uuid.UUID, g Goal) (Explanation, error) {
	ex := Explanation{Measure: g.Measure, Items: []Item{}}
	titles := map[uuid.UUID]string{}
	targets := []uuid.UUID{g.ID}
	for _, m := range g.Milestones {
		ex.Items = append(ex.Items, Item{ID: m.ID, Type: MilestoneType, Title: m.Title, Done: m.DoneAt != nil})
		titles[m.ID] = m.Title
		targets = append(targets, m.ID)
	}
	rows, err := q.Query(ctx, `SELECT DISTINCT ON (e.id) e.id, e.title, t.status = 'done', l.target_id
		FROM links l JOIN entities e ON e.id = l.source_id JOIN tasks t ON t.id = e.id
		WHERE l.user_id = $1 AND l.relation = $2 AND l.target_id = ANY($3) AND t.status <> 'cancelled'
		ORDER BY e.id, l.target_id = $4 DESC LIMIT 1000`, userID, Supports, targets, g.ID)
	if err != nil {
		return ex, err
	}
	defer rows.Close()
	for rows.Next() {
		var it Item
		var target uuid.UUID
		if err := rows.Scan(&it.ID, &it.Title, &it.Done, &target); err != nil {
			return ex, err
		}
		it.Type = tasks.Type
		if target != g.ID {
			via := titles[target]
			it.Via = &via
		}
		ex.Items = append(ex.Items, it)
	}
	if err := rows.Err(); err != nil {
		return ex, err
	}
	for _, it := range ex.Items {
		ex.Total++
		if it.Done {
			ex.Done++
		}
	}
	ex.Progress = compute(g.Measure, ex.Done, ex.Total)
	return ex, nil
}

// compute turns the inputs into a figure and a sentence saying how.
func compute(m *Measure, done, total int) Progress {
	p := Progress{Done: done, Total: total}
	switch {
	case m != nil:
		pct := int(math.Floor(math.Max(0, math.Min(1, (m.Current-m.Start)/(m.Target-m.Start))) * 100))
		p.Method, p.Percent = "measure", &pct
		p.Summary = fmt.Sprintf("%s of %s %s (from %s)", num(m.Current), num(m.Target), m.Unit, num(m.Start))
	case total > 0:
		pct := done * 100 / total
		p.Method, p.Percent = "items", &pct
		p.Summary = fmt.Sprintf("%d of %d milestones and linked tasks done", done, total)
	default:
		p.Method = "none"
		p.Summary = "Add a measure, a milestone, or a linked task to track progress."
	}
	return p
}

func num(f float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

// Fields are the editable parts of a goal; nil means "not provided".
type Fields struct {
	Title      *string
	Outcome    *string
	Status     *string
	TargetDate *time.Time
	Measure    *Measure

	ClearTargetDate, ClearMeasure bool
}

func (f *Fields) validate() error {
	if f.Title != nil {
		*f.Title = strings.TrimSpace(*f.Title)
		if *f.Title == "" || len([]rune(*f.Title)) > 500 {
			return ValidationError{"A goal needs a title of 1 to 500 characters."}
		}
	}
	if f.Outcome != nil && len(*f.Outcome) > 20000 {
		return ValidationError{"The outcome is limited to 20,000 characters."}
	}
	if f.Status != nil && !contains(statuses, *f.Status) {
		return ValidationError{"Status must be active, achieved, or archived."}
	}
	if m := f.Measure; m != nil {
		m.Unit = strings.TrimSpace(m.Unit)
		finite := !math.IsNaN(m.Start+m.Target+m.Current) && !math.IsInf(m.Start+m.Target+m.Current, 0)
		switch {
		case m.Unit == "" || len([]rune(m.Unit)) > 40:
			return ValidationError{"A measure needs a unit of 1 to 40 characters, like km or books."}
		case !finite:
			return ValidationError{"Measure values must be numbers."}
		case m.Start == m.Target:
			return ValidationError{"A measure's target must differ from its start."}
		}
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Create adds a goal. Dates and measures are only ever what the user sent.
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f Fields) (Goal, error) {
	if f.Title == nil {
		return Goal{}, ValidationError{"A goal needs a title."}
	}
	if err := f.validate(); err != nil {
		return Goal{}, err
	}
	g := Goal{Title: *f.Title, Status: "active", Measure: f.Measure}
	if f.Outcome != nil {
		g.Outcome = *f.Outcome
	}
	if f.Status != nil {
		g.Status = *f.Status
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, g.Title, g.Outcome)
	if err != nil {
		return Goal{}, err
	}
	unit, start, target, current := measureCols(g.Measure)
	_, err = tx.Exec(ctx, `INSERT INTO goals (id, outcome, status, target_date, measure_unit, measure_start, measure_target, measure_current)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, g.Outcome, g.Status, f.TargetDate, unit, start, target, current)
	if err != nil {
		return Goal{}, err
	}
	return Get(ctx, tx, userID, id)
}

func measureCols(m *Measure) (unit *string, start, target, current *float64) {
	if m == nil {
		return nil, nil, nil, nil
	}
	return &m.Unit, &m.Start, &m.Target, &m.Current
}

// Update applies f if the goal's version still equals version. Archiving is
// a status change: milestones, links, and history stay.
func Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, version time.Time, f Fields) (Goal, error) {
	if err := f.validate(); err != nil {
		return Goal{}, err
	}
	cur, err := Get(ctx, tx, userID, id)
	if err != nil {
		return Goal{}, err
	}
	if !cur.UpdatedAt.Equal(version) {
		return Goal{}, ErrStale
	}
	next := cur
	var changed []string
	set := func(name string, cond bool, apply func()) {
		if cond {
			apply()
			changed = append(changed, name)
		}
	}
	set("title", f.Title != nil && *f.Title != cur.Title, func() { next.Title = *f.Title })
	set("outcome", f.Outcome != nil && *f.Outcome != cur.Outcome, func() { next.Outcome = *f.Outcome })
	set("status", f.Status != nil && *f.Status != cur.Status, func() { next.Status = *f.Status })
	set("target_date", f.TargetDate != nil || (f.ClearTargetDate && cur.TargetDate != nil), func() {
		next.TargetDate = nil
		if f.TargetDate != nil {
			next.TargetDate = &tasks.Date{Time: *f.TargetDate}
		}
	})
	set("measure", f.Measure != nil || (f.ClearMeasure && cur.Measure != nil), func() { next.Measure = f.Measure })
	if len(changed) == 0 {
		return cur, nil
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, next.Title, next.Outcome, changed); err != nil {
		return Goal{}, err
	}
	var target *time.Time
	if next.TargetDate != nil {
		target = &next.TargetDate.Time
	}
	unit, start, tgt, current := measureCols(next.Measure)
	_, err = tx.Exec(ctx, `UPDATE goals SET outcome = $2, status = $3, target_date = $4, measure_unit = $5, measure_start = $6,
		measure_target = $7, measure_current = $8, updated_at = clock_timestamp() WHERE id = $1`,
		id, next.Outcome, next.Status, target, unit, start, tgt, current)
	if err != nil {
		return Goal{}, err
	}
	return Get(ctx, tx, userID, id)
}

// Delete removes a goal, its milestones, and their links.
func Delete(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	g, err := Get(ctx, tx, userID, id)
	if err != nil {
		return err
	}
	for _, m := range g.Milestones {
		if err := graph.DeleteEntity(ctx, tx, userID, m.ID, actor); err != nil {
			return err
		}
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

// MilestoneFields are the editable parts of a milestone.
type MilestoneFields struct {
	Title    *string
	DueOn    *time.Time
	ClearDue bool
	Done     *bool
}

func (f *MilestoneFields) validate() error {
	if f.Title != nil {
		*f.Title = strings.TrimSpace(*f.Title)
		if *f.Title == "" || len([]rune(*f.Title)) > 500 {
			return ValidationError{"A milestone needs a title of 1 to 500 characters."}
		}
	}
	return nil
}

// GetMilestone returns one of the user's milestones.
func GetMilestone(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Milestone, error) {
	m, err := scanMilestone(q.QueryRow(ctx, `SELECT `+milestoneColumns+` FROM milestones m JOIN entities e ON e.id = m.id
		WHERE m.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return m, graph.ErrNotFound
	}
	return m, err
}

// AddMilestone adds a milestone to one of the user's goals.
func AddMilestone(ctx context.Context, tx pgx.Tx, userID, goalID uuid.UUID, actor graph.Actor, f MilestoneFields) (Milestone, error) {
	if f.Title == nil {
		return Milestone{}, ValidationError{"A milestone needs a title."}
	}
	if err := f.validate(); err != nil {
		return Milestone{}, err
	}
	if _, err := graph.GetEntity(ctx, tx, userID, goalID); err != nil {
		return Milestone{}, err
	}
	var isGoal bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM goals WHERE id = $1)`, goalID).Scan(&isGoal); err != nil {
		return Milestone{}, err
	}
	if !isGoal {
		return Milestone{}, graph.ErrNotFound
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, MilestoneType, *f.Title, "")
	if err != nil {
		return Milestone{}, err
	}
	var done *time.Time
	if f.Done != nil && *f.Done {
		now := time.Now()
		done = &now
	}
	if _, err := tx.Exec(ctx, `INSERT INTO milestones (id, goal_id, due_on, done_at) VALUES ($1, $2, $3, $4)`, id, goalID, f.DueOn, done); err != nil {
		return Milestone{}, err
	}
	return GetMilestone(ctx, tx, userID, id)
}

// UpdateMilestone changes a milestone if its version still equals version.
func UpdateMilestone(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, version time.Time, f MilestoneFields) (Milestone, error) {
	if err := f.validate(); err != nil {
		return Milestone{}, err
	}
	cur, err := GetMilestone(ctx, tx, userID, id)
	if err != nil {
		return Milestone{}, err
	}
	if !cur.UpdatedAt.Equal(version) {
		return Milestone{}, ErrStale
	}
	next := cur
	var changed []string
	if f.Title != nil && *f.Title != cur.Title {
		next.Title = *f.Title
		changed = append(changed, "title")
	}
	if f.DueOn != nil || (f.ClearDue && cur.DueOn != nil) {
		next.DueOn = nil
		if f.DueOn != nil {
			next.DueOn = &tasks.Date{Time: *f.DueOn}
		}
		changed = append(changed, "due_on")
	}
	if f.Done != nil && *f.Done != (cur.DoneAt != nil) {
		next.DoneAt = nil
		if *f.Done {
			now := time.Now()
			next.DoneAt = &now
		}
		changed = append(changed, "done")
	}
	if len(changed) == 0 {
		return cur, nil
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, next.Title, "", changed); err != nil {
		return Milestone{}, err
	}
	var due *time.Time
	if next.DueOn != nil {
		due = &next.DueOn.Time
	}
	if _, err := tx.Exec(ctx, `UPDATE milestones SET due_on = $2, done_at = $3, updated_at = clock_timestamp() WHERE id = $1`,
		id, due, next.DoneAt); err != nil {
		return Milestone{}, err
	}
	return GetMilestone(ctx, tx, userID, id)
}

// DeleteMilestone removes a milestone and its links.
func DeleteMilestone(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	if _, err := GetMilestone(ctx, tx, userID, id); err != nil {
		return err
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM goals g JOIN entities e ON e.id = g.id WHERE e.user_id = $1 ORDER BY g.id`, userID)
	if err != nil {
		return nil, err
	}
	type row struct {
		ID         uuid.UUID   `json:"id"`
		Title      string      `json:"title"`
		Outcome    string      `json:"outcome"`
		Status     string      `json:"status"`
		TargetDate *tasks.Date `json:"target_date"`
		Measure    *Measure    `json:"measure"`
		CreatedAt  time.Time   `json:"created_at"`
		UpdatedAt  time.Time   `json:"updated_at"`
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		g, err := scan(r)
		return row{g.ID, g.Title, g.Outcome, g.Status, g.TargetDate, g.Measure, g.CreatedAt, g.UpdatedAt}, err
	})
}

func exportMilestones(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+milestoneColumns+` FROM milestones m JOIN entities e ON e.id = m.id WHERE e.user_id = $1 ORDER BY m.id`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Milestone, error) { return scanMilestone(row) })
}
