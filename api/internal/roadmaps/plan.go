package roadmaps

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/certs"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/para"
	"github.com/thoutmose/axon/api/internal/projects"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// A roadmap with a plan (EP-95): starting it creates everything the plan is
// made of, not only its skills. The goal with its measure, a milestone and
// a project per phase (each phase's exit check on the project's board, as
// issues and subtasks), the 42 cursus as a side project, the habits that
// drive it, the first week's tasks, its areas and resources, and the
// roadmap's own note, all connected.

//go:embed top5-engineer.md
var top5Note string

// Plan is what a roadmap's phases, habits, and first tasks are.
type Plan struct {
	// Version changes when the phases do (EP-146): a roadmap started from
	// an earlier version is outdated, whatever its titles.
	Version      int    `json:"version"`
	Start        string `json:"start"`
	End          string `json:"end"`
	HoursPerWeek string `json:"hours_per_week"`
	Budget       string `json:"budget"`
	NotePath     string `json:"note_path"`
	Goal         struct {
		Title   string `json:"title"`
		Outcome string `json:"outcome"`
	} `json:"goal"`
	Phases    []Phase        `json:"phases"`
	Habits    []PlanHabit    `json:"habits"`
	Tasks     []PlanTask     `json:"tasks"`
	Areas     []PlanArea     `json:"areas"`
	Resources []PlanResource `json:"resources"`
	Rhythm    []PlanDay      `json:"rhythm"`
	Certs     []PlanCert     `json:"certifications"`
}

// PlanCert is an exam the plan takes, by the end of its phase.
type PlanCert struct {
	Name           string `json:"name"`
	Code           string `json:"code"`
	Provider       string `json:"provider"`
	URL            string `json:"url"`
	CostCents      *int64 `json:"cost_cents"`
	ValidityMonths *int   `json:"validity_months"`
	Phase          int    `json:"phase"`
	Skill          string `json:"skill"`
}

// PlanDay is a weekday's study block (1 Monday to 7 Sunday): {M}, {C},
// {L}, and {R} are the phase's tracks ({R} falls back to {L}), {P} its project.
type PlanDay struct {
	Day   int    `json:"day"`
	Hours string `json:"hours"`
	Block string `json:"block"`
}

// Block is one day's study on the calendar.
type Block struct {
	Date  time.Time
	Title string
	Phase int
}

// tracks reads "M = … · C = … · L = …" into its tracks.
func tracks(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, " · ") {
		if k, v, ok := strings.Cut(part, " = "); ok {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	if out["R"] == "" {
		out["R"] = out["L"]
	}
	return out
}

// Blocks are the study blocks from one day to another, the plan moved by
// shift days: each weekday's block with the tracks of its phase.
func (p Plan) Blocks(shift int, from, to time.Time) []Block {
	var out []Block
	for _, ph := range p.Phases {
		start, end := day(ph.Start).AddDate(0, 0, shift), day(ph.End).AddDate(0, 0, shift)
		tr := tracks(ph.Tracks)
		fill := strings.NewReplacer("{M}", tr["M"], "{C}", tr["C"], "{L}", tr["L"], "{R}", tr["R"], "{P}", ph.Project.Slug)
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			wd := int(d.Weekday())
			if wd == 0 {
				wd = 7
			}
			for _, r := range p.Rhythm {
				if r.Day == wd {
					out = append(out, Block{Date: d, Title: fill.Replace(r.Block) + " · " + r.Hours, Phase: ph.N})
				}
			}
		}
	}
	return out
}

// Phase is a stretch of the plan with its milestone, project, and exit check.
type Phase struct {
	N       int      `json:"n"`
	Slug    string   `json:"slug"`
	Title   string   `json:"title"`
	Start   string   `json:"start"`
	End     string   `json:"end"`
	Tracks  string   `json:"tracks"`
	Domains []string `json:"domains"`
	Project struct {
		Slug    string `json:"slug"`
		Title   string `json:"title"`
		Outcome string `json:"outcome"`
	} `json:"project"`
	Exit     []PlanItem `json:"exit"`
	FortyTwo []FortyTwo `json:"fortytwo"`
}

// PlanItem is one line of an exit check, with its parts as subtasks.
type PlanItem struct {
	Title string   `json:"title"`
	Parts []string `json:"parts"`
}

// FortyTwo is a 42 school project folded into a phase.
type FortyTwo struct {
	Title    string `json:"title"`
	Hours    int    `json:"hours"`
	Optional bool   `json:"optional"`
	Note     string `json:"note"`
}

// PlanHabit is a habit of the plan, after Atomic Habits' four laws.
type PlanHabit struct {
	Title    string `json:"title"`
	Stack    string `json:"stack"`
	Cue      string `json:"cue"`
	Craving  string `json:"craving"`
	Two      string `json:"two"`
	Reward   string `json:"reward"`
	Identity string `json:"identity"`
	Time     string `json:"time"`
	Place    string `json:"place"`
	Days     []int  `json:"days"`
}

// PlanTask is a first task, due Day days after the start.
type PlanTask struct {
	Title      string `json:"title"`
	Day        int    `json:"day"`
	Context    string `json:"context"`
	Project    string `json:"project"`
	Recurrence string `json:"recurrence"`
}

// PlanArea is an area of responsibility the plan belongs to.
type PlanArea struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// PlanResource is a PARA resource the plan keeps at hand.
type PlanResource struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// Started is what starting a plan created.
type Started struct {
	Roadmap  Roadmap
	Goal     uuid.UUID
	Projects map[string]uuid.UUID // by slug
	Habits   []uuid.UUID
	Tasks    []uuid.UUID
}

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(fmt.Sprintf("roadmaps: plan date %q: %v", s, err))
	}
	return t
}

// Shift is how many days the plan moves: none while its first phase is
// still running, so the dates stay the plan's own; after that, the plan
// starts again on the coming Monday.
func (p Plan) Shift(today time.Time) int {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if len(p.Phases) == 0 || !today.After(day(p.Phases[0].End)) {
		return 0
	}
	monday := today.AddDate(0, 0, (8-int(today.Weekday()))%7)
	return int(monday.Sub(day(p.Start)).Hours() / 24)
}

// ErrStarted means the person already started this roadmap.
var ErrStarted = ValidationError{"You have already started this roadmap: open it from your roadmaps."}

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
	at := func(s string) *time.Time { t := day(s).AddDate(0, 0, shift); return &t }
	str := func(s string) *string { return &s }
	s := Started{Projects: map[string]uuid.UUID{}}
	rm, err := importLibrary(ctx, tx, userID, actor, d)
	if err != nil {
		return s, err
	}
	if _, err := tx.Exec(ctx, `UPDATE roadmaps SET plan_id = $2, plan_start = $3, plan_version = $4 WHERE id = $1`, rm.ID, d.ID, at(p.Start), p.Version); err != nil {
		return s, err
	}
	link := func(from uuid.UUID, rel string, to uuid.UUID) error {
		_, err := graph.CreateLink(ctx, tx, userID, actor, from, rel, to)
		if errors.Is(err, graph.ErrDuplicate) {
			return nil
		}
		return err
	}

	// The roadmap's own note, unless the vault already has it.
	note, err := notes.GetByPath(ctx, tx, userID, p.NotePath)
	if errors.Is(err, graph.ErrNotFound) {
		note, err = notes.Create(ctx, tx, userID, actor, p.NotePath, top5Note)
	}
	if err != nil {
		return s, err
	}
	if err := link(note.ID, graph.RelatesTo, rm.ID); err != nil {
		return s, err
	}

	// Areas, reused when the person already has one of the same name.
	areas, err := para.Areas.List(ctx, tx, userID, "active")
	if err != nil {
		return s, err
	}
	var mainArea uuid.UUID
	for i, a := range p.Areas {
		id := uuid.Nil
		for _, have := range areas {
			if strings.EqualFold(have.Title, a.Title) {
				id = have.ID
			}
		}
		if id == uuid.Nil {
			item, err := para.Areas.Create(ctx, tx, userID, actor, para.Fields{Title: str(a.Title), Description: str(a.Description)})
			if err != nil {
				return s, err
			}
			id = item.ID
		}
		if i == 0 {
			mainArea = id
		}
		if err := link(rm.ID, para.InArea, id); err != nil {
			return s, err
		}
	}

	// The goal: every domain proven, measured in domains.
	domains := 0
	for _, sk := range d.Skills {
		if sk.Kind != "soft" {
			domains++
		}
	}
	g, err := goals.Create(ctx, tx, userID, actor, goals.Fields{Title: &p.Goal.Title, Outcome: &p.Goal.Outcome, TargetDate: at(p.End),
		Measure: &goals.Measure{Unit: "domains proven", Target: float64(domains)}})
	if err != nil {
		return s, err
	}
	s.Goal = g.ID
	for _, l := range []struct {
		from uuid.UUID
		rel  string
		to   uuid.UUID
	}{{rm.ID, goals.Supports, g.ID}, {g.ID, para.InArea, mainArea}, {note.ID, graph.RelatesTo, g.ID}} {
		if l.to != uuid.Nil {
			if err := link(l.from, l.rel, l.to); err != nil {
				return s, err
			}
		}
	}

	skills := map[string]uuid.UUID{}
	for _, sk := range rm.Skills {
		skills[sk.Title] = sk.ID
	}
	// A card on a board: a task with its due date, placed as epic, issue, or subtask.
	card := func(project uuid.UUID, title, column, kind string, parent *uuid.UUID, due *time.Time, notes string) (uuid.UUID, error) {
		status := map[string]string{"backlog": "someday", "todo": "next"}[column]
		t, err := tasks.Create(ctx, tx, userID, actor, tasks.Fields{Title: &title, Status: &status, DueOn: due, Notes: &notes})
		if err != nil {
			return uuid.Nil, err
		}
		_, err = projects.Place(ctx, tx, userID, project, actor, t.ID, projects.CardInput{Column: &column, Kind: &kind, ParentID: parent})
		return t.ID, err
	}

	// A milestone and a project per phase; the exit check is on the board.
	today := day(p.Start).AddDate(0, 0, shift)
	var side uuid.UUID
	for _, ph := range p.Phases {
		end := at(ph.End)
		current := ph.N == p.Phases[0].N
		m, err := goals.AddMilestone(ctx, tx, userID, g.ID, actor, goals.MilestoneFields{Title: &ph.Title, DueOn: end})
		if err != nil {
			return s, err
		}
		status, column := "on_hold", "backlog"
		if current {
			status, column = "active", "todo"
		}
		desc := fmt.Sprintf("%s\n\nDeliverable of %s, %s → %s.\n\nTracks: %s", ph.Project.Outcome, ph.Title,
			at(ph.Start).Format(time.DateOnly), end.Format(time.DateOnly), ph.Tracks)
		pr, err := projects.Create(ctx, tx, userID, actor, projects.Fields{Title: &ph.Project.Title, Description: &desc,
			Status: &status, DocsFolder: str("1-Projects/" + ph.Project.Slug), WIPLimit: new(3)})
		if err != nil {
			return s, err
		}
		s.Projects[ph.Project.Slug] = pr.ID
		if err := projectNote(ctx, tx, userID, actor, pr.ID, ph, at(ph.Start), end, current); err != nil {
			return s, err
		}
		if err := link(pr.ID, goals.Supports, g.ID); err != nil {
			return s, err
		}
		if err := link(pr.ID, graph.RelatesTo, m.ID); err != nil {
			return s, err
		}
		for _, dom := range ph.Domains {
			if err := link(skills[dom], graph.RelatesTo, m.ID); err != nil {
				return s, err
			}
		}
		epic, err := card(pr.ID, fmt.Sprintf("Phase %d exit check", ph.N), column, "epic", nil, end, "")
		if err != nil {
			return s, err
		}
		for _, it := range ph.Exit {
			issue, err := card(pr.ID, it.Title, column, "issue", &epic, nil, "")
			if err != nil {
				return s, err
			}
			for _, part := range it.Parts {
				if _, err := card(pr.ID, part, column, "subtask", &issue, nil, ""); err != nil {
					return s, err
				}
			}
		}

		// The 42 projects of the phase, on the side project's board.
		if len(ph.FortyTwo) == 0 {
			continue
		}
		if side == uuid.Nil {
			pr, err := projects.Create(ctx, tx, userID, actor, projects.Fields{Title: str("42 cursus (self-taught)"),
				Description: str("The 42 school's projects, done self-taught and part-time alongside the plan: the real cursus is free but on-site and full-time, which does not fit 22 hours a week. Optional projects are for when you are ahead. Subjects and testers are in the resource “42 subjects and testers”; never copy solutions."),
				DocsFolder:  str("1-Projects/42-cursus"), WIPLimit: new(2)})
			if err != nil {
				return s, err
			}
			side = pr.ID
			s.Projects["42-cursus"] = side
			if err := link(side, goals.Supports, g.ID); err != nil {
				return s, err
			}
		}
		group, err := card(side, fmt.Sprintf("Phase %d", ph.N), column, "epic", nil, end, ph.Title)
		if err != nil {
			return s, err
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
			if _, err := card(side, title, col, "issue", &group, nil, f.Note); err != nil {
				return s, err
			}
		}
	}

	// The habits that drive it, and the exams, unless the person already
	// has them (an earlier version of the plan, or their own).
	have := map[string]bool{}
	hs, err := habits.List(ctx, tx, userID, "active", now)
	if err != nil {
		return s, err
	}
	for _, h := range hs {
		have["habit "+strings.ToLower(h.Title)] = true
	}
	cs, err := certs.List(ctx, tx, userID)
	if err != nil {
		return s, err
	}
	for _, c := range cs {
		have["cert "+c.Code] = true
	}
	for _, h := range p.Habits {
		if have["habit "+strings.ToLower(h.Title)] {
			continue
		}
		habit, err := habits.Create(ctx, tx, userID, actor, habits.Fields{Title: str(h.Title), StackAfter: str(h.Stack), Cue: str(h.Cue),
			Craving: str(h.Craving), TwoMinute: str(h.Two), Reward: str(h.Reward), Identity: str(h.Identity),
			IntentionTime: str(h.Time), IntentionPlace: str(h.Place), Days: h.Days}, now)
		if err != nil {
			return s, err
		}
		s.Habits = append(s.Habits, habit.ID)
		if err := link(habit.ID, goals.Supports, g.ID); err != nil {
			return s, err
		}
	}

	// The first week's tasks; a project's go on its board.
	for _, t := range p.Tasks {
		due := today.AddDate(0, 0, t.Day)
		f := tasks.Fields{Title: str(t.Title), Status: str("next"), DueOn: &due}
		if t.Context != "" {
			f.Context = str(t.Context)
		}
		if t.Recurrence != "" {
			f.Recurrence = str(t.Recurrence)
		}
		task, err := tasks.Create(ctx, tx, userID, actor, f)
		if err != nil {
			return s, err
		}
		s.Tasks = append(s.Tasks, task.ID)
		if pr, ok := s.Projects[t.Project]; ok {
			if _, err := projects.Place(ctx, tx, userID, pr, actor, task.ID, projects.CardInput{Column: str("todo")}); err != nil {
				return s, err
			}
		} else if err := link(task.ID, goals.Supports, g.ID); err != nil {
			return s, err
		}
	}

	// The exams, planned for the end of their phase (EP-104).
	for _, c := range p.Certs {
		if have["cert "+c.Code] {
			continue
		}
		var target string
		for _, ph := range p.Phases {
			if ph.N == c.Phase {
				target = at(ph.End).Format(time.DateOnly)
			}
		}
		f := certs.Fields{Name: c.Name, Code: c.Code, Provider: c.Provider, Status: "planned", TargetOn: &target,
			CostCents: c.CostCents, Currency: "USD", ValidityMonths: c.ValidityMonths}
		if c.URL != "" {
			f.URL = str(c.URL)
		}
		if id, ok := skills[c.Skill]; ok {
			f.SkillID = &id
		}
		if _, err := certs.Create(ctx, tx, userID, f); err != nil {
			return s, fmt.Errorf("certification %s: %w", c.Code, err)
		}
	}

	// Resources kept at hand.
	for _, r := range p.Resources {
		item, err := para.Resources.Create(ctx, tx, userID, actor, para.Fields{Title: str(r.Title), Description: str(r.Description), URL: str(r.URL)})
		if err != nil {
			return s, err
		}
		if err := link(item.ID, graph.RelatesTo, rm.ID); err != nil {
			return s, err
		}
	}
	// Every resource of the plan as a note, connected to where it serves (EP-154).
	if _, err := ResourceNotes(ctx, tx, userID, actor, rm.ID, now); err != nil {
		return s, err
	}
	s.Roadmap, err = Get(ctx, tx, userID, rm.ID)
	return s, err
}

// projectNote writes a phase project's note in its docs folder, in the
// vault's project template, unless the vault already has one.
func projectNote(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, project uuid.UUID, ph Phase, start, end *time.Time, current bool) error {
	path := "1-Projects/" + ph.Project.Slug + "/" + ph.Project.Slug + ".md"
	n, err := notes.GetByPath(ctx, tx, userID, path)
	if errors.Is(err, graph.ErrNotFound) {
		status := "someday"
		if current {
			status = "active"
		}
		n, err = notes.Create(ctx, tx, userID, actor, path, fmt.Sprintf(`---
type: project
title: %s
created: %s
status: %s
deadline: %s
area: "[[engineering-mastery]]"
goal: "[[top5-engineer-2029]]"
roadmap: "[[top5-engineer]]"
---
> PARA **Project**: a short-term effort with a clear outcome and a deadline.
> Deliverable of milestone [[%s]].

## Outcome (what does "done" look like?)

%s

## Next actions

Tasks live on the project's board in Axon: the phase's exit check, its issues, and their subtasks.

## Resources & notes

- Tracks: %s
- Links in [[top5-engineer]]

## Log

- %s created
`, ph.Project.Title, start.Format(time.DateOnly), status, end.Format(time.DateOnly), ph.Slug, ph.Project.Outcome, ph.Tracks, start.Format(time.DateOnly)))
	}
	if err != nil {
		return err
	}
	_, err = graph.CreateLink(ctx, tx, userID, actor, n.ID, graph.RelatesTo, project)
	if errors.Is(err, graph.ErrDuplicate) {
		return nil
	}
	return err
}

// ScheduledBlock is a study block of one of the person's roadmaps.
type ScheduledBlock struct {
	Block
	RoadmapID uuid.UUID
}

// Schedule is the study blocks of the person's started plans in a range.
func Schedule(ctx context.Context, q db.Querier, userID uuid.UUID, from, to time.Time) ([]ScheduledBlock, error) {
	// A plan started again (a newer version of it) replaces the earlier one.
	rows, err := q.Query(ctx, `SELECT DISTINCT ON (r.plan_id) r.id, r.plan_id, r.plan_start FROM roadmaps r JOIN entities e ON e.id = r.id
		WHERE e.user_id = $1 AND r.plan_id IS NOT NULL ORDER BY r.plan_id, e.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	type started struct {
		ID    uuid.UUID
		Plan  string
		Start time.Time
	}
	plans, err := pgx.CollectRows(rows, pgx.RowToStructByPos[started])
	if err != nil {
		return nil, err
	}
	var out []ScheduledBlock
	for _, sp := range plans {
		for _, d := range lib.Roadmaps {
			if d.ID != sp.Plan || d.Plan == nil {
				continue
			}
			shift := int(sp.Start.Sub(day(d.Plan.Start)).Hours() / 24)
			for _, b := range d.Plan.Blocks(shift, from, to) {
				out = append(out, ScheduledBlock{b, sp.ID})
			}
		}
	}
	return out, nil
}
