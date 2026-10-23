package roadmaps

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/certs"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/para"
	"github.com/thoutmose/axon/api/internal/projects"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// StartPlan imports a roadmap with a plan and creates the plan around it,
// its dates moved by shift days. The first phase is the one under way.
func StartPlan(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, d LibraryRoadmap, shift int, now time.Time) (Started, error) {
	p := *d.Plan
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM roadmaps r JOIN entities e ON e.id = r.id
		WHERE e.user_id = $1 AND r.plan_id = $2 AND r.plan_version = $3)`, userID, d.ID, p.Version).Scan(&exists); err != nil {
		return Started{}, err
	}
	if exists {
		return Started{}, ErrStarted
	}
	r := &planRun{ctx: ctx, tx: tx, userID: userID, actor: actor, d: d, p: p, shift: shift, now: now,
		s: Started{Projects: map[string]uuid.UUID{}}}
	for _, step := range []func() error{r.importRoadmap, r.addNote, r.addAreas, r.addGoal, r.addPhases,
		r.addHabitsAndStages, r.addFirstTasks, r.addExams, r.addPortfolio, r.addResources} {
		if err := step(); err != nil {
			return r.s, err
		}
	}
	var err error
	r.s.Roadmap, err = Get(ctx, tx, userID, r.rm.ID)
	return r.s, err
}

// planRun is what the steps of StartPlan share.
type planRun struct {
	ctx    context.Context
	tx     pgx.Tx
	userID uuid.UUID
	actor  graph.Actor
	d      LibraryRoadmap
	p      Plan
	shift  int
	now    time.Time

	s       Started
	rm      Roadmap
	note    notes.Note
	areaIDs map[string]uuid.UUID
	goal    uuid.UUID
	skills  map[string]uuid.UUID
	side    uuid.UUID       // the 42 cursus project, once created
	have    map[string]bool // certifications the person already has
}

func str(s string) *string { return &s }

// at is a date of the plan, moved by the shift.
func (r *planRun) at(s string) *time.Time { t := day(s).AddDate(0, 0, r.shift); return &t }

func (r *planRun) link(from uuid.UUID, rel string, to uuid.UUID) error {
	return planLink(r.ctx, r.tx, r.userID, r.actor, from, rel, to)
}

// card is a card on a board: a task with its due date, placed as epic, issue, or subtask.
func (r *planRun) card(project uuid.UUID, title, column, kind string, parent *uuid.UUID, due *time.Time, notes string) (uuid.UUID, error) {
	status := map[string]string{"backlog": "someday", "todo": "next"}[column]
	t, err := tasks.Create(r.ctx, r.tx, r.userID, r.actor, tasks.Fields{Title: &title, Status: &status, DueOn: due, Notes: &notes})
	if err != nil {
		return uuid.Nil, err
	}
	_, err = projects.Place(r.ctx, r.tx, r.userID, project, r.actor, t.ID, projects.CardInput{Column: &column, Kind: &kind, ParentID: parent})
	return t.ID, err
}

func (r *planRun) importRoadmap() error {
	rm, err := importLibrary(r.ctx, r.tx, r.userID, r.actor, r.d)
	if err != nil {
		return err
	}
	r.rm = rm
	_, err = r.tx.Exec(r.ctx, `UPDATE roadmaps SET plan_id = $2, plan_start = $3, plan_version = $4 WHERE id = $1`, rm.ID, r.d.ID, r.at(r.p.Start), r.p.Version)
	return err
}

// addNote adds the roadmap's own note, unless the vault already has it.
func (r *planRun) addNote() error {
	note, err := notes.GetByPath(r.ctx, r.tx, r.userID, r.p.NotePath)
	if errors.Is(err, graph.ErrNotFound) {
		note, err = notes.Create(r.ctx, r.tx, r.userID, r.actor, r.p.NotePath, top5Note)
	}
	if err != nil {
		return err
	}
	r.note = note
	return r.link(note.ID, graph.RelatesTo, r.rm.ID)
}

func (r *planRun) addAreas() error {
	var err error
	r.areaIDs, _, err = planAreas(r.ctx, r.tx, r.userID, r.actor, r.p, r.rm.ID)
	return err
}

// addGoal adds the goal: every domain proven, measured in domains.
func (r *planRun) addGoal() error {
	mainArea := r.areaIDs[strings.ToLower(r.p.Areas[0].Title)]
	domains := 0
	for _, sk := range r.d.Skills {
		if sk.Kind != "soft" {
			domains++
		}
	}
	p := r.p
	g, err := goals.Create(r.ctx, r.tx, r.userID, r.actor, goals.Fields{Title: &p.Goal.Title, Outcome: &p.Goal.Outcome, TargetDate: r.at(p.End),
		Measure: &goals.Measure{Unit: "domains proven", Target: float64(domains)}})
	if err != nil {
		return err
	}
	r.s.Goal = g.ID
	r.goal = g.ID
	for _, l := range []struct {
		from uuid.UUID
		rel  string
		to   uuid.UUID
	}{{r.rm.ID, goals.Supports, g.ID}, {g.ID, para.InArea, mainArea}, {r.note.ID, graph.RelatesTo, g.ID}} {
		if l.to != uuid.Nil {
			if err := r.link(l.from, l.rel, l.to); err != nil {
				return err
			}
		}
	}
	r.skills = map[string]uuid.UUID{}
	for _, sk := range r.rm.Skills {
		r.skills[sk.Title] = sk.ID
	}
	return nil
}

// addPhases adds a milestone and a project per phase; the exit check is on the board.
func (r *planRun) addPhases() error {
	for _, ph := range r.p.Phases {
		end := r.at(ph.End)
		current := ph.N == r.p.Phases[0].N
		m, err := goals.AddMilestone(r.ctx, r.tx, r.userID, r.goal, r.actor, goals.MilestoneFields{Title: &ph.Title, DueOn: end})
		if err != nil {
			return err
		}
		status, column := "on_hold", "backlog"
		if current {
			status, column = "active", "todo"
		}
		desc := fmt.Sprintf("%s\n\nDeliverable of %s, %s → %s.\n\nTracks: %s", ph.Project.Outcome, ph.Title,
			r.at(ph.Start).Format(time.DateOnly), end.Format(time.DateOnly), ph.Tracks)
		pr, err := projects.Create(r.ctx, r.tx, r.userID, r.actor, projects.Fields{Title: &ph.Project.Title, Description: &desc,
			Status: &status, DocsFolder: str("1-Projects/" + ph.Project.Slug), WIPLimit: new(3)})
		if err != nil {
			return err
		}
		r.s.Projects[ph.Project.Slug] = pr.ID
		if err := projectNote(r.ctx, r.tx, r.userID, r.actor, pr.ID, ph, r.at(ph.Start), end, current); err != nil {
			return err
		}
		if err := r.link(pr.ID, goals.Supports, r.goal); err != nil {
			return err
		}
		if err := r.link(pr.ID, graph.RelatesTo, m.ID); err != nil {
			return err
		}
		for _, dom := range ph.Domains {
			if err := r.link(r.skills[dom], graph.RelatesTo, m.ID); err != nil {
				return err
			}
		}
		if err := r.exitCheck(pr.ID, ph, column, end); err != nil {
			return err
		}
		if err := r.fortyTwo(ph, column, end); err != nil {
			return err
		}
	}
	return nil
}

// exitCheck puts the phase's exit check on its project's board.
func (r *planRun) exitCheck(project uuid.UUID, ph Phase, column string, end *time.Time) error {
	epic, err := r.card(project, fmt.Sprintf("Phase %d exit check", ph.N), column, "epic", nil, end, "")
	if err != nil {
		return err
	}
	for _, it := range ph.Exit {
		issue, err := r.card(project, it.Title, column, "issue", &epic, nil, "")
		if err != nil {
			return err
		}
		for _, part := range it.Parts {
			if _, err := r.card(project, part, column, "subtask", &issue, nil, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// fortyTwo puts the 42 projects of the phase on the side project's board.
func (r *planRun) fortyTwo(ph Phase, column string, end *time.Time) error {
	if len(ph.FortyTwo) == 0 {
		return nil
	}
	if r.side == uuid.Nil {
		pr, err := projects.Create(r.ctx, r.tx, r.userID, r.actor, projects.Fields{Title: str("42 cursus (self-taught)"),
			Description: str("The 42 school's projects, done self-taught and part-time alongside the plan: the real cursus is free but on-site and full-time, which does not fit 22 hours a week. Optional projects are for when you are ahead. Subjects and testers are in the resource “42 subjects and testers”; never copy solutions."),
			DocsFolder:  str("1-Projects/42-cursus"), WIPLimit: new(2)})
		if err != nil {
			return err
		}
		r.side = pr.ID
		r.s.Projects["42-cursus"] = r.side
		if err := r.link(r.side, goals.Supports, r.goal); err != nil {
			return err
		}
	}
	group, err := r.card(r.side, fmt.Sprintf("Phase %d", ph.N), column, "epic", nil, end, ph.Title)
	if err != nil {
		return err
	}
	for _, f := range ph.FortyTwo {
		title := fmt.Sprintf("%s (~%d h)", f.Title, f.Hours)
		if f.Optional {
			title += ", optional"
		}
		col := column
		if f.Optional {
			col = "backlog"
		}
		if _, err := r.card(r.side, title, col, "issue", &group, nil, f.Note); err != nil {
			return err
		}
	}
	return nil
}

// addHabitsAndStages adds the habits that drive the plan and a goal per group
// of domains, unless the person already has them (an earlier version of the
// plan, or their own).
func (r *planRun) addHabitsAndStages() error {
	r.have = map[string]bool{}
	cs, err := certs.List(r.ctx, r.tx, r.userID)
	if err != nil {
		return err
	}
	for _, c := range cs {
		r.have["cert "+c.Code] = true
	}
	if r.s.Habits, err = planHabits(r.ctx, r.tx, r.userID, r.actor, r.p, r.areaIDs, r.goal, r.now); err != nil {
		return err
	}
	_, err = planStages(r.ctx, r.tx, r.userID, r.actor, r.d, r.areaIDs, r.skills, r.goal, r.shift)
	return err
}

// addFirstTasks adds the first week's tasks; a project's go on its board.
func (r *planRun) addFirstTasks() error {
	today := day(r.p.Start).AddDate(0, 0, r.shift)
	for _, t := range r.p.Tasks {
		due := today.AddDate(0, 0, t.Day)
		f := tasks.Fields{Title: str(t.Title), Status: str("next"), DueOn: &due}
		if t.Context != "" {
			f.Context = str(t.Context)
		}
		if t.Recurrence != "" {
			f.Recurrence = str(t.Recurrence)
		}
		task, err := tasks.Create(r.ctx, r.tx, r.userID, r.actor, f)
		if err != nil {
			return err
		}
		r.s.Tasks = append(r.s.Tasks, task.ID)
		if pr, ok := r.s.Projects[t.Project]; ok {
			if _, err := projects.Place(r.ctx, r.tx, r.userID, pr, r.actor, task.ID, projects.CardInput{Column: str("todo")}); err != nil {
				return err
			}
		} else if err := r.link(task.ID, goals.Supports, r.goal); err != nil {
			return err
		}
	}
	return nil
}

// addExams adds the exams, planned for the end of their phase (EP-104).
func (r *planRun) addExams() error {
	for _, c := range r.p.Certs {
		if r.have["cert "+c.Code] {
			continue
		}
		var target string
		for _, ph := range r.p.Phases {
			if ph.N == c.Phase {
				target = r.at(ph.End).Format(time.DateOnly)
			}
		}
		f := certs.Fields{Name: c.Name, Code: c.Code, Provider: c.Provider, Status: "planned", TargetOn: &target,
			CostCents: c.CostCents, Currency: "USD", ValidityMonths: c.ValidityMonths, Labels: c.Labels}
		if c.URL != "" {
			f.URL = str(c.URL)
		}
		if id, ok := r.skills[c.Skill]; ok {
			f.SkillID = &id
		}
		if _, err := certs.Create(r.ctx, r.tx, r.userID, f); err != nil {
			return fmt.Errorf("certification %s: %w", c.Code, err)
		}
	}
	return nil
}

// addPortfolio adds the portfolio projects, each shipped by the end of its phase.
func (r *planRun) addPortfolio() error {
	p := r.p
	for _, pp := range p.Projects {
		i := slices.IndexFunc(p.Phases, func(ph Phase) bool { return ph.N == pp.Phase })
		if i < 0 {
			return fmt.Errorf("portfolio project %s: no phase %d in the plan", pp.Slug, pp.Phase)
		}
		ph := p.Phases[i]
		end := r.at(ph.End)
		status := "on_hold"
		if pp.Phase == p.Phases[0].N {
			status = "active"
		}
		desc := fmt.Sprintf("%s\n\nPortfolio project of %s, %s → %s. Domains: %s.", pp.Outcome, ph.Title,
			r.at(ph.Start).Format(time.DateOnly), end.Format(time.DateOnly), strings.Join(pp.Domains, ", "))
		pr, err := projects.Create(r.ctx, r.tx, r.userID, r.actor, projects.Fields{Title: &pp.Title, Description: &desc,
			Status: &status, DocsFolder: str("1-Projects/" + pp.Slug), WIPLimit: new(3)})
		if err != nil {
			return err
		}
		r.s.Projects[pp.Slug] = pr.ID
		if err := r.link(pr.ID, goals.Supports, r.goal); err != nil {
			return err
		}
		if area, ok := r.areaIDs[strings.ToLower(pp.Area)]; ok {
			if err := r.link(pr.ID, para.InArea, area); err != nil {
				return err
			}
		}
		for _, dom := range pp.Domains {
			if id, ok := r.skills[dom]; ok {
				if err := r.link(id, graph.RelatesTo, pr.ID); err != nil {
					return err
				}
			}
		}
		if err := backlogCard(r.ctx, r.tx, r.userID, r.actor, pr.ID, "Ship "+pp.Title, pp.Outcome, end); err != nil {
			return err
		}
	}
	return nil
}

// addResources keeps resources at hand, then adds every resource of the plan
// as a note, connected to where it serves (EP-154).
func (r *planRun) addResources() error {
	for _, res := range r.p.Resources {
		item, err := para.Resources.Create(r.ctx, r.tx, r.userID, r.actor, para.Fields{Title: str(res.Title), Description: str(res.Description), URL: str(res.URL)})
		if err != nil {
			return err
		}
		if err := r.link(item.ID, graph.RelatesTo, r.rm.ID); err != nil {
			return err
		}
	}
	_, err := ResourceNotes(r.ctx, r.tx, r.userID, r.actor, r.rm.ID, r.now)
	return err
}
