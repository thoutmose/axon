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
		WHERE e.user_id = $1 AND r.source = 'library' AND e.title = $2)`, userID, d.Title).Scan(&exists); err != nil {
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

	// The habits that drive it.
	for _, h := range p.Habits {
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
	s.Roadmap, err = Get(ctx, tx, userID, rm.ID)
	return s, err
}
