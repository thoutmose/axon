package roadmaps

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/notes"
)

// The roadmap's resources as notes (EP-154): one resource note per book,
// course, site, and practice of the plan, in 3-Resources/<plan>/, saying
// what it is for and where in the plan it serves (the phases that work on
// its domain, their projects and milestones, the certifications of its
// domain), with its link, which notes open in a new tab. Each note is also
// connected to its skill, the goal, and those milestones and projects, so
// the graph and every item's connections show it. A note that exists is
// left as the person made it.

// ResourceNotes writes the notes a started plan's resources lack and
// returns how many it wrote.
func ResourceNotes(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, id uuid.UUID, now time.Time) (int, error) {
	plan, err := GetPlan(ctx, tx, userID, id, now)
	if err != nil {
		return 0, err
	}
	var planID string
	if err := tx.QueryRow(ctx, `SELECT coalesce(plan_id, '') FROM roadmaps WHERE id = $1`, id).Scan(&planID); err != nil {
		return 0, err
	}
	var d *LibraryRoadmap
	for i := range lib.Roadmaps {
		if lib.Roadmaps[i].ID == planID {
			d = &lib.Roadmaps[i]
		}
	}
	if d == nil || !plan.Started {
		return 0, ValidationError{"Only a started library roadmap has resources to write as notes."}
	}
	rm, err := Get(ctx, tx, userID, id)
	if err != nil {
		return 0, err
	}
	skillIDs := map[string]uuid.UUID{}
	for _, s := range rm.Skills {
		skillIDs[s.Title] = s.ID
	}
	describe := map[string]string{}
	for _, s := range d.Skills {
		describe[s.Title] = s.Description
	}
	link := func(from uuid.UUID, rel string, to uuid.UUID) error {
		_, err := graph.CreateLink(ctx, tx, userID, actor, from, rel, to)
		if errors.Is(err, graph.ErrDuplicate) || errors.Is(err, graph.ErrNotFound) {
			return nil
		}
		return err
	}
	written := 0
	for _, r := range d.Resources {
		path := "3-Resources/" + d.ID + "/" + notes.Slug(r.Title) + ".md"
		if _, err := notes.GetByPath(ctx, tx, userID, path); err == nil {
			continue
		} else if !errors.Is(err, graph.ErrNotFound) {
			return written, err
		}
		var phases []PlanPhaseView
		for _, ph := range plan.Phases {
			if slices.Contains(ph.Domains, r.Skill) {
				phases = append(phases, ph)
			}
		}
		n, err := notes.Create(ctx, tx, userID, actor, path, resourceNote(d, r, describe[r.Skill], phases, now))
		if err != nil {
			return written, err
		}
		written++
		targets := []uuid.UUID{id}
		if sk, ok := skillIDs[r.Skill]; ok {
			targets = append(targets, sk)
		}
		if plan.GoalID != nil {
			targets = append(targets, *plan.GoalID)
		}
		for _, ph := range phases {
			if ph.Milestone != nil {
				targets = append(targets, ph.Milestone.ID)
			}
			if ph.Project != nil && ph.Project.ID != nil {
				targets = append(targets, *ph.Project.ID)
			}
		}
		for _, to := range targets {
			if err := link(n.ID, graph.RelatesTo, to); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

// resourceNote is a resource's note, in the vault's resource note shape.
func resourceNote(d *LibraryRoadmap, r Resource, about string, phases []PlanPhaseView, now time.Time) string {
	q := func(s string) string { return strconv.Quote(s) }
	var nums, certs []string
	for _, ph := range phases {
		nums = append(nums, strconv.Itoa(ph.N))
		for _, c := range ph.Certs {
			if c.Skill == r.Skill && !slices.Contains(certs, c.Code) {
				certs = append(certs, c.Code)
			}
		}
	}
	cost := "free"
	if !r.Free {
		cost = "paid"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntype: resource\ntitle: %s\nurl: %s\nkind: %s\nby: %s\ncost: %s\nroadmap: \"[[%s]]\"\ndomain: %s\nphases: [%s]\ncertifications: [%s]\ntags: [resource, %s]\ncreated: %s\n---\n",
		q(r.Title), r.URL, r.Kind, q(r.By), cost, d.ID, q(r.Skill), strings.Join(nums, ", "), strings.Join(certs, ", "),
		notes.Slug(r.Skill), now.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "# %s\n\n> A %s by %s, %s. [Open it ↗](%s)\n\n", r.Title, r.Kind, r.By, cost, r.URL)
	fmt.Fprintf(&b, "## What it is for\n\n**%s** in [[%s]]", r.Skill, d.ID)
	if about != "" {
		fmt.Fprintf(&b, ": %s", about)
	}
	b.WriteString("\n\n## Where it serves in the plan\n\n")
	if len(phases) == 0 {
		b.WriteString("- Alongside every phase.\n")
	}
	for _, ph := range phases {
		fmt.Fprintf(&b, "- %s (%s → %s): project [[%s]], milestone at its end\n", ph.Title, ph.Start, ph.End, projectSlug(d, ph.N))
		for _, c := range ph.Certs {
			if c.Skill == r.Skill {
				fmt.Fprintf(&b, "  - Certification: %s, %s (%s)\n", c.Code, c.Name, c.Provider)
			}
		}
	}
	b.WriteString("\n## Notes while using it\n\n- \n\n## Permanent notes to extract\n\n- [ ] \n")
	return b.String()
}

// projectSlug is the project note of a phase, by its number.
func projectSlug(d *LibraryRoadmap, n int) string {
	for _, ph := range d.Plan.Phases {
		if ph.N == n {
			return ph.Project.Slug
		}
	}
	return ""
}
