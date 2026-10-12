package roadmaps

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/graph"
)

// The curated library (EP-27): Axon-authored roadmaps that list widely
// recommended books, papers, courses, and channels per domain. Only titles,
// authors, and links are stored, so nothing is redistributed.

//go:embed library.json
var libraryJSON []byte

// Resource is one entry in the curated library.
type Resource struct {
	Title    string   `json:"title"`
	Kind     string   `json:"kind"` // book, course, video, channel, paper, site, practice
	By       string   `json:"by"`
	Free     bool     `json:"free"`
	URL      string   `json:"url"`
	Section  string   `json:"section"`
	Requires []string `json:"requires,omitempty"`
	Skill    string   `json:"skill"` // the skill this resource teaches
}

// LibrarySkill is a competency in a curated roadmap; resources teach it.
type LibrarySkill struct {
	Title       string   `json:"title"`
	Section     string   `json:"section"`
	Description string   `json:"description"`
	Requires    []string `json:"requires,omitempty"` // earlier skills
}

// LibraryRoadmap is a domain of the curated library.
type LibraryRoadmap struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Skills      []LibrarySkill `json:"skills"`
	Resources   []Resource     `json:"resources"`
}

type library struct {
	Attribution string           `json:"attribution"`
	Roadmaps    []LibraryRoadmap `json:"roadmaps"`
}

var lib = func() library {
	var l library
	if err := json.Unmarshal(libraryJSON, &l); err != nil {
		panic(fmt.Sprintf("roadmaps: library.json: %v", err))
	}
	return l
}()

// Library returns the curated roadmaps.
func Library() []LibraryRoadmap { return lib.Roadmaps }

// Source is something that can be imported as a roadmap.
type Source struct {
	ID          string  `json:"id"`
	Kind        string  `json:"kind"` // curriculum or library
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Attribution string  `json:"attribution"`
	LicenseURL  *string `json:"license_url"`
	Count       int     `json:"count"`  // courses or resources
	Skills      int     `json:"skills"` // skills they are grouped into
}

// LibraryMatch is a resource found by SearchLibrary, with its domain.
type LibraryMatch struct {
	Resource
	Domain string `json:"domain"`
}

// SearchLibrary finds curated resources whose title, author, section, or
// domain contains every word of the query (case-insensitive).
func SearchLibrary(query, kind string, limit int) []LibraryMatch {
	words := strings.Fields(strings.ToLower(query))
	var out []LibraryMatch
	seen := map[string]bool{}
	for _, d := range lib.Roadmaps {
		for _, r := range d.Resources {
			if (kind != "" && r.Kind != kind) || seen[r.URL] {
				continue
			}
			hay := strings.ToLower(strings.Join([]string{r.Title, r.By, r.Section, d.ID, d.Title, d.Description}, " "))
			ok := true
			for _, w := range words {
				if !strings.Contains(hay, w) {
					ok = false
					break
				}
			}
			if ok {
				seen[r.URL] = true
				out = append(out, LibraryMatch{r, d.ID})
				if len(out) == limit {
					return out
				}
			}
		}
	}
	return out
}

func importLibrary(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, d LibraryRoadmap) (Roadmap, error) {
	id, err := create(ctx, tx, userID, actor, d.Title, d.Description, "library", lib.Attribution, nil)
	if err != nil {
		return Roadmap{}, err
	}
	kind := "hard"
	if d.ID == "soft-skills" {
		kind = "soft"
	}
	byskill := map[string][]SkillResource{}
	for _, r := range d.Resources {
		byskill[r.Skill] = append(byskill[r.Skill], SkillResource{Title: r.Title, Kind: r.Kind, By: r.By, Free: r.Free, URL: r.URL})
	}
	if err := addSkills(ctx, tx, userID, id, actor, d.Skills, byskill, kind); err != nil {
		return Roadmap{}, err
	}
	return Get(ctx, tx, userID, id)
}

// addSkills creates curated skills in order with their resources and
// prerequisites. A skill links to its first resource.
func addSkills(ctx context.Context, tx pgx.Tx, userID, roadmapID uuid.UUID, actor graph.Actor, skills []LibrarySkill, resources map[string][]SkillResource, kind string) error {
	ids := map[string]uuid.UUID{}
	for i, s := range skills {
		rs, pos := resources[s.Title], i+1
		f := SkillFields{Title: &s.Title, Description: &s.Description, Section: &s.Section, Kind: &kind, Position: &pos, Resources: rs}
		if len(rs) > 0 {
			f.URL = &rs[0].URL
		}
		sid, err := addSkill(ctx, tx, userID, roadmapID, actor, f)
		if err != nil {
			return err
		}
		ids[s.Title] = sid
		for _, req := range s.Requires {
			if to, ok := ids[req]; ok {
				if _, err := graph.CreateLink(ctx, tx, userID, actor, sid, Requires, to); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// DraftSkill is one skill of a roadmap the Tutor agent drafted.
type DraftSkill struct {
	Title       string   `json:"title"`
	Section     string   `json:"section"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Requires    []string `json:"requires"`
}

// CreateDraft creates an approved Tutor roadmap. Every skill must cite a
// source link; prerequisites refer to earlier skills by title.
func CreateDraft(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, title, desc string, skills []DraftSkill) (Roadmap, error) {
	if err := (Fields{Title: &title, Description: &desc}).validate(); err != nil {
		return Roadmap{}, err
	}
	if len(skills) == 0 || len(skills) > 150 {
		return Roadmap{}, ValidationError{"A drafted roadmap has 1 to 150 skills."}
	}
	id, err := create(ctx, tx, userID, actor, strings.TrimSpace(title), strings.TrimSpace(desc), "tutor",
		"Drafted by the Tutor agent and approved by you. Each skill links to the source it was based on.", nil)
	if err != nil {
		return Roadmap{}, err
	}
	ids := map[string]uuid.UUID{}
	for i, s := range skills {
		if s.URL == "" {
			return Roadmap{}, ValidationError{fmt.Sprintf("Skill %q needs a source link.", s.Title)}
		}
		f := SkillFields{Title: &s.Title, Description: &s.Description, URL: &s.URL, Section: &s.Section}
		if err := f.validate(); err != nil {
			return Roadmap{}, err
		}
		pos := i + 1
		f.Position = &pos
		if _, dup := ids[strings.TrimSpace(s.Title)]; dup {
			return Roadmap{}, ValidationError{fmt.Sprintf("Skill %q appears twice.", s.Title)}
		}
		sid, err := addSkill(ctx, tx, userID, id, actor, f)
		if err != nil {
			return Roadmap{}, err
		}
		for _, req := range s.Requires {
			to, ok := ids[strings.TrimSpace(req)]
			if !ok {
				return Roadmap{}, ValidationError{fmt.Sprintf("Skill %q needs %q, which must be an earlier skill.", s.Title, req)}
			}
			if _, err := graph.CreateLink(ctx, tx, userID, actor, sid, Requires, to); err != nil {
				return Roadmap{}, err
			}
		}
		ids[strings.TrimSpace(s.Title)] = sid
	}
	return Get(ctx, tx, userID, id)
}

// catalogue returns the skills and resources an imported roadmap came from:
// OSSU, or the library domain whose resources match most of its skill titles.
func catalogue(r Roadmap) ([]LibrarySkill, map[string][]SkillResource, string, bool) {
	if r.Source == "ossu" {
		var c curriculum
		if err := json.Unmarshal(ossuJSON, &c); err != nil {
			return nil, nil, "", false
		}
		byskill := map[string][]SkillResource{}
		for _, course := range c.Courses {
			note := strings.Trim(strings.Join([]string{course.Duration, course.Effort}, " · "), " ·")
			byskill[course.Skill] = append(byskill[course.Skill], SkillResource{Title: course.Title, Kind: "course", By: "OSSU", Free: true, URL: course.URL, Note: note})
		}
		return c.Skills, byskill, "hard", true
	}
	best, score := -1, 0
	for i, d := range lib.Roadmaps {
		titles := map[string]bool{} // an older import names resources, a newer one skills
		for _, res := range d.Resources {
			titles[res.Title] = true
		}
		for _, sk := range d.Skills {
			titles[sk.Title] = true
		}
		n := 0
		for _, s := range r.Skills {
			if titles[s.Title] {
				n++
			}
		}
		if n > score {
			best, score = i, n
		}
	}
	if best < 0 || score*2 < len(r.Skills) {
		return nil, nil, "", false
	}
	d := lib.Roadmaps[best]
	byskill := map[string][]SkillResource{}
	for _, res := range d.Resources {
		byskill[res.Skill] = append(byskill[res.Skill], SkillResource{Title: res.Title, Kind: res.Kind, By: res.By, Free: res.Free, URL: res.URL})
	}
	kind := "hard"
	if d.ID == "soft-skills" {
		kind = "soft"
	}
	return d.Skills, byskill, kind, true
}

// combined is the status of a skill grouped from several: done when every
// counted part is done, learning once any part is started.
func combined(parts []Skill) string {
	done, started, skipped := 0, 0, 0
	for _, p := range parts {
		switch p.Status {
		case "done":
			done++
		case "learning":
			started++
		case "skipped":
			skipped++
		}
	}
	switch {
	case skipped == len(parts):
		return "skipped"
	case done > 0 && done+skipped == len(parts):
		return "done"
	case done+started > 0:
		return "learning"
	}
	return "todo"
}

// Regroup turns a roadmap imported before skills were competencies (one
// skill per course or book) into skills with their resources. Progress is
// combined, connections move to the skill that absorbs them, and skills the
// person added by hand are kept. Running it twice changes nothing.
func Regroup(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) (Roadmap, error) {
	r, err := Get(ctx, tx, userID, id)
	if err != nil {
		return r, err
	}
	if r.Source != "ossu" && r.Source != "library" {
		return r, ValidationError{"Only roadmaps imported from OSSU or the library can be grouped into skills."}
	}
	skills, resources, kind, ok := catalogue(r)
	if !ok {
		return r, ValidationError{"This roadmap no longer matches a library domain; import it again instead."}
	}
	owner := map[string]string{} // resource title -> skill title
	for skill, rs := range resources {
		for _, res := range rs {
			owner[res.Title] = skill
		}
	}
	parts := map[string][]Skill{}
	for _, s := range r.Skills {
		if skill, ok := owner[s.Title]; ok && len(s.Resources) == 0 {
			parts[skill] = append(parts[skill], s)
		}
	}
	if len(parts) == 0 {
		return r, nil // already grouped
	}
	grouped := map[uuid.UUID]bool{}
	for _, group := range parts {
		for _, s := range group {
			grouped[s.ID] = true
		}
	}
	for _, e := range r.Edges {
		if !grouped[e.From] && !grouped[e.To] {
			continue // a prerequisite the person added between their own skills
		}
		if err := graph.DeleteLink(ctx, tx, userID, actor, e.LinkID); err != nil {
			return r, err
		}
	}
	existing := map[string]uuid.UUID{}
	for _, s := range r.Skills {
		if len(s.Resources) > 0 {
			existing[s.Title] = s.ID
		}
	}
	ids := map[string]uuid.UUID{}
	for i, ls := range skills {
		rs, pos := resources[ls.Title], i+1
		group := parts[ls.Title]
		var sid uuid.UUID
		switch {
		case existing[ls.Title] != uuid.Nil:
			sid = existing[ls.Title]
		case len(group) == 0:
			todo := "todo"
			f := SkillFields{Title: &ls.Title, Description: &ls.Description, Section: &ls.Section, Kind: &kind, Status: &todo, Position: &pos, Resources: rs}
			if len(rs) > 0 {
				f.URL = &rs[0].URL
			}
			if sid, err = addSkill(ctx, tx, userID, id, actor, f); err != nil {
				return r, err
			}
		default:
			sid = group[0].ID
			for _, old := range group[1:] {
				if err := moveLinks(ctx, tx, userID, actor, old.ID, sid); err != nil {
					return r, err
				}
				if err := graph.DeleteEntity(ctx, tx, userID, old.ID, actor); err != nil {
					return r, err
				}
			}
			var url *string
			if len(rs) > 0 {
				url = &rs[0].URL
			}
			if err := graph.UpdateEntity(ctx, tx, userID, sid, actor, ls.Title, ls.Description, []string{"title", "description", "resources"}); err != nil {
				return r, err
			}
			if _, err := tx.Exec(ctx, `UPDATE skills SET description = $2, url = $3, section = $4, status = $5, kind = $6, position = $7,
				resources = $8, updated_at = clock_timestamp() WHERE id = $1`, sid, ls.Description, url, ls.Section, combined(group), kind, pos, rs); err != nil {
				return r, err
			}
		}
		ids[ls.Title] = sid
		for _, req := range ls.Requires {
			if to, ok := ids[req]; ok {
				if _, err := graph.CreateLink(ctx, tx, userID, actor, sid, Requires, to); err != nil && !errors.Is(err, graph.ErrDuplicate) {
					return r, err
				}
			}
		}
	}
	// Skills added by hand go after the curated ones, in their old order.
	if _, err := tx.Exec(ctx, `UPDATE skills SET position = position + $2 WHERE roadmap_id = $1 AND resources = '[]'::jsonb`, id, len(skills)); err != nil {
		return r, err
	}
	return Get(ctx, tx, userID, id)
}

// moveLinks recreates from's connections (other than prerequisites) on to.
func moveLinks(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, from, to uuid.UUID) error {
	rows, err := tx.Query(ctx, `SELECT source_id, relation, target_id FROM links
		WHERE user_id = $1 AND (source_id = $2 OR target_id = $2) AND relation <> $3`, userID, from, Requires)
	if err != nil {
		return err
	}
	type link struct {
		Source   uuid.UUID
		Relation string
		Target   uuid.UUID
	}
	links, err := pgx.CollectRows(rows, pgx.RowToStructByPos[link])
	if err != nil {
		return err
	}
	for _, l := range links {
		if l.Source == from {
			l.Source = to
		} else {
			l.Target = to
		}
		if _, err := graph.CreateLink(ctx, tx, userID, actor, l.Source, l.Relation, l.Target); err != nil &&
			!errors.Is(err, graph.ErrDuplicate) && !errors.Is(err, graph.ErrRelationRejected) {
			return err
		}
	}
	return nil
}

// competencyOf maps a curated resource or OSSU course title to the skill it
// teaches. Roadmaps imported before skills were competencies (ADR 0020) have
// one skill per resource; this lets views show them as the skill instead.
var competencyOf = func() map[string]string {
	m := map[string]string{}
	for _, d := range lib.Roadmaps {
		for _, r := range d.Resources {
			if r.Skill != "" {
				m[strings.ToLower(r.Title)] = r.Skill
			}
		}
	}
	var c curriculum
	if json.Unmarshal(ossuJSON, &c) == nil {
		for _, course := range c.Courses {
			if course.Skill != "" {
				m[strings.ToLower(course.Title)] = course.Skill
			}
		}
	}
	return m
}()

// Competency returns the skill a curated resource or course teaches, if title is one.
func Competency(title string) (string, bool) {
	s, ok := competencyOf[strings.ToLower(strings.TrimSpace(title))]
	return s, ok
}
