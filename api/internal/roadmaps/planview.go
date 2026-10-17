package roadmaps

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/certs"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
)

// The plan on the roadmap's page (EP-142): each phase as the milestone it
// became, with its dates, project, exit check, 42 projects, and the exams
// taken at its end, joined to what the person has done with them. The plan
// is matched to their items by title and code, so a milestone, project, or
// certification they renamed or removed shows as missing, not as an error.

// PlanCertView is a phase's exam and where the person is with it.
type PlanCertView struct {
	PlanCert
	ID       *uuid.UUID `json:"id"`     // their certification, if it is still there
	Status   string     `json:"status"` // planned, booked, passed, …; "" without one
	TargetOn *string    `json:"target_on"`
}

// PlanPhaseView is one phase of a started plan.
type PlanPhaseView struct {
	N         int            `json:"n"`
	Title     string         `json:"title"`
	Start     string         `json:"start"`
	End       string         `json:"end"`
	State     string         `json:"state"` // past, current, or future
	Tracks    string         `json:"tracks"`
	Domains   []string       `json:"domains"`
	Milestone *PlanItemRef   `json:"milestone"`
	Project   *PlanProject   `json:"project"`
	Exit      []PlanItem     `json:"exit"`
	FortyTwo  []FortyTwo     `json:"fortytwo"`
	Certs     []PlanCertView `json:"certifications"`
}

// PlanItemRef is the person's milestone for a phase.
type PlanItemRef struct {
	ID   uuid.UUID `json:"id"`
	Done bool      `json:"done"`
}

// PlanProject is a phase's project and how far its board is.
type PlanProject struct {
	ID      *uuid.UUID `json:"id"` // nil in a preview
	Title   string     `json:"title"`
	Outcome string     `json:"outcome"`
	Status  string     `json:"status"`
	Done    int        `json:"done"`  // cards done
	Total   int        `json:"total"` // cards not cancelled
}

// PlanView is a started plan, or Started false for a roadmap without one.
// A library roadmap's preview is not started but has its phases, at the
// dates starting it today would give (EP-142). Outdated means the roadmap
// was started from an earlier version of the plan: none of its milestones
// is one of the plan's phases; Newer is a roadmap started since from the
// same plan.
type PlanView struct {
	Started  bool            `json:"started"`
	Start    string          `json:"start"`
	End      string          `json:"end"`
	GoalID   *uuid.UUID      `json:"goal_id"`
	Outdated bool            `json:"outdated"`
	Newer    *uuid.UUID      `json:"newer"`
	Phases   []PlanPhaseView `json:"phases"`
}

// phases lays out a plan's phases with its dates moved by shift days.
func (p Plan) phases(shift int, today time.Time) []PlanPhaseView {
	out := []PlanPhaseView{}
	for _, ph := range p.Phases {
		start, end := day(ph.Start).AddDate(0, 0, shift), day(ph.End).AddDate(0, 0, shift)
		v := PlanPhaseView{N: ph.N, Title: ph.Title, Start: start.Format(time.DateOnly), End: end.Format(time.DateOnly),
			State: "current", Tracks: ph.Tracks, Domains: ph.Domains, Exit: ph.Exit, FortyTwo: ph.FortyTwo, Certs: []PlanCertView{}}
		switch {
		case today.After(end):
			v.State = "past"
		case today.Before(start):
			v.State = "future"
		}
		if v.FortyTwo == nil {
			v.FortyTwo = []FortyTwo{}
		}
		v.Project = &PlanProject{Title: ph.Project.Title, Outcome: ph.Project.Outcome}
		for _, c := range p.Certs {
			if c.Phase == ph.N {
				v.Certs = append(v.Certs, PlanCertView{PlanCert: c})
			}
		}
		out = append(out, v)
	}
	return out
}

// PreviewPlan is a library roadmap's plan as starting it today would lay
// it out, so its milestones can be read before it is started.
func PreviewPlan(ctx context.Context, q db.Querier, userID uuid.UUID, source string, now time.Time) (PlanView, error) {
	today, _, err := clock.Today(ctx, q, userID, now)
	if err != nil {
		return PlanView{}, err
	}
	for _, d := range lib.Roadmaps {
		if d.ID == source && d.Plan != nil {
			shift := d.Plan.Shift(today)
			return PlanView{Start: day(d.Plan.Start).AddDate(0, 0, shift).Format(time.DateOnly),
				End: day(d.Plan.End).AddDate(0, 0, shift).Format(time.DateOnly), Phases: d.Plan.phases(shift, today)}, nil
		}
	}
	return PlanView{}, pgx.ErrNoRows
}

// GetPlan returns the plan of a roadmap with the person's progress on it.
func GetPlan(ctx context.Context, q db.Querier, userID, id uuid.UUID, now time.Time) (PlanView, error) {
	var planID *string
	var planStart *time.Time
	if err := q.QueryRow(ctx, `SELECT r.plan_id, r.plan_start FROM roadmaps r JOIN entities e ON e.id = r.id
		WHERE r.id = $1 AND e.user_id = $2`, id, userID).Scan(&planID, &planStart); err != nil {
		return PlanView{}, err
	}
	out := PlanView{Phases: []PlanPhaseView{}}
	var p *Plan
	for _, d := range lib.Roadmaps {
		if planID != nil && d.ID == *planID {
			p = d.Plan
		}
	}
	if p == nil || planStart == nil {
		return out, nil
	}
	shift := int(planStart.Sub(day(p.Start)).Hours() / 24)
	at := func(s string) time.Time { return day(s).AddDate(0, 0, shift) }
	today, _, err := clock.Today(ctx, q, userID, now)
	if err != nil {
		return out, err
	}
	out.Started, out.Start, out.End = true, at(p.Start).Format(time.DateOnly), at(p.End).Format(time.DateOnly)
	if err := q.QueryRow(ctx, `SELECT r.id FROM roadmaps r JOIN entities e ON e.id = r.id
		WHERE e.user_id = $1 AND r.plan_id = $2 AND r.id <> $3
		  AND e.created_at > (SELECT created_at FROM entities WHERE id = $3)
		ORDER BY e.created_at DESC LIMIT 1`, userID, *planID, id).Scan(&out.Newer); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}

	// The goal the roadmap supports, its milestones by title, and the
	// projects supporting it by title.
	var goal *uuid.UUID
	if err := q.QueryRow(ctx, `SELECT l.target_id FROM links l JOIN goals g ON g.id = l.target_id
		WHERE l.user_id = $1 AND l.source_id = $2 AND l.relation = 'supports' LIMIT 1`, userID, id).Scan(&goal); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	out.GoalID = goal
	milestones := map[string]PlanItemRef{}
	projects := map[string]PlanProject{}
	if goal != nil {
		rows, err := q.Query(ctx, `SELECT e.title, m.id, m.done_at IS NOT NULL FROM milestones m JOIN entities e ON e.id = m.id
			WHERE m.goal_id = $1 AND e.user_id = $2`, *goal, userID)
		if err != nil {
			return out, err
		}
		var title string
		var ref PlanItemRef
		if _, err := pgx.ForEachRow(rows, []any{&title, &ref.ID, &ref.Done}, func() error {
			milestones[title] = ref
			return nil
		}); err != nil {
			return out, err
		}
		rows, err = q.Query(ctx, `SELECT e.id, e.title, p.status,
			  (SELECT count(*) FROM project_cards c JOIN tasks t ON t.id = c.task_id WHERE c.project_id = p.id AND t.status = 'done'),
			  (SELECT count(*) FROM project_cards c JOIN tasks t ON t.id = c.task_id WHERE c.project_id = p.id AND t.status <> 'cancelled')
			FROM links l JOIN projects p ON p.id = l.source_id JOIN entities e ON e.id = p.id
			WHERE l.user_id = $1 AND l.target_id = $2 AND l.relation = 'supports'`, userID, *goal)
		if err != nil {
			return out, err
		}
		var pr PlanProject
		if _, err := pgx.ForEachRow(rows, []any{&pr.ID, &pr.Title, &pr.Status, &pr.Done, &pr.Total}, func() error {
			projects[pr.Title] = pr
			return nil
		}); err != nil {
			return out, err
		}
	}
	mine, err := certs.List(ctx, q, userID)
	if err != nil {
		return out, err
	}

	matched := 0
	for _, v := range p.phases(shift, today) {
		v.Project = nil
		if m, ok := milestones[v.Title]; ok {
			v.Milestone = &m
			matched++
		}
		if pr, ok := projects[p.Phases[v.N-1].Project.Title]; ok {
			pr.Outcome = p.Phases[v.N-1].Project.Outcome
			v.Project = &pr
		}
		for i, c := range v.Certs {
			for _, m := range mine {
				if m.Code == c.Code {
					v.Certs[i].ID, v.Certs[i].Status, v.Certs[i].TargetOn = &m.ID, m.Status, m.TargetOn
				}
			}
		}
		out.Phases = append(out.Phases, v)
	}
	out.Outdated = len(milestones) > 0 && matched == 0
	return out, nil
}

// PlanToday is what the plan asks of today, for Today's page (EP-145): the
// phase under way, today's study block, the phase's project and the next
// exam, from the person's latest started plan.
type PlanToday struct {
	RoadmapID uuid.UUID      `json:"roadmap_id"`
	Title     string         `json:"title"`
	Start     string         `json:"start"`
	Phases    int            `json:"phases"`
	Reached   int            `json:"reached"` // milestones reached
	Phase     *PlanPhaseView `json:"phase"`   // under way, or the first one before the start
	Week      int            `json:"week"`    // of the phase, from 1; 0 before it
	Weeks     int            `json:"weeks"`
	DaysLeft  int            `json:"days_left"` // to the phase's milestone
	Block     *string        `json:"block"`     // today's study block
	NextCert  *PlanCertView  `json:"next_certification"`
}

// TodayPlan returns today's part of the latest started plan, or nil.
func TodayPlan(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (*PlanToday, error) {
	var id uuid.UUID
	var title, planID string
	var planStart time.Time
	err := q.QueryRow(ctx, `SELECT r.id, e.title, r.plan_id, r.plan_start FROM roadmaps r JOIN entities e ON e.id = r.id
		WHERE e.user_id = $1 AND r.plan_id IS NOT NULL AND r.plan_start IS NOT NULL
		ORDER BY e.created_at DESC LIMIT 1`, userID).Scan(&id, &title, &planID, &planStart)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p, err := GetPlan(ctx, q, userID, id, now)
	if err != nil || !p.Started || len(p.Phases) == 0 {
		return nil, err
	}
	today, _, err := clock.Today(ctx, q, userID, now)
	if err != nil {
		return nil, err
	}
	out := &PlanToday{RoadmapID: id, Title: title, Start: p.Start, Phases: len(p.Phases)}
	for i := range p.Phases {
		ph := &p.Phases[i]
		if ph.Milestone != nil && ph.Milestone.Done {
			out.Reached++
		}
		if out.Phase == nil && ph.State != "past" {
			out.Phase = ph
		}
	}
	if out.Phase == nil {
		return out, nil // the plan is over
	}
	start, end := day(out.Phase.Start), day(out.Phase.End)
	out.Weeks = int(end.Sub(start).Hours()/24)/7 + 1
	if !today.Before(start) {
		out.Week = int(today.Sub(start).Hours()/24)/7 + 1
	}
	out.DaysLeft = int(end.Sub(today).Hours() / 24)
	if out.Phase.State == "current" {
		for _, d := range lib.Roadmaps {
			if d.ID == planID && d.Plan != nil {
				shift := int(planStart.Sub(day(d.Plan.Start)).Hours() / 24)
				for _, b := range d.Plan.Blocks(shift, today, today) {
					out.Block = &b.Title
				}
			}
		}
	}
	for _, ph := range p.Phases[out.Phase.N-1:] {
		for _, c := range ph.Certs {
			if c.Status != "passed" && out.NextCert == nil {
				c := c
				out.NextCert = &c
			}
		}
	}
	return out, nil
}
