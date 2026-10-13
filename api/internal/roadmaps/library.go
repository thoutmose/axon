package roadmaps

import (
	"context"
	_ "embed"
	"encoding/json"
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
	Kind        string   `json:"kind,omitempty"`     // soft, or hard when empty
	Assess      string   `json:"assess,omitempty"`   // the skills assessment's domain that places it
}

// LibraryRoadmap is a domain of the curated library.
type LibraryRoadmap struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Skills      []LibrarySkill `json:"skills"`
	Resources   []Resource     `json:"resources"`
	Plan        *Plan          `json:"plan,omitempty"`
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
	byskill := map[string][]SkillResource{}
	for _, r := range d.Resources {
		byskill[r.Skill] = append(byskill[r.Skill], SkillResource{Title: r.Title, Kind: r.Kind, By: r.By, Free: r.Free, URL: r.URL})
	}
	if err := addSkills(ctx, tx, userID, id, actor, d.Skills, byskill, "hard"); err != nil {
		return Roadmap{}, err
	}
	return Get(ctx, tx, userID, id)
}

// addSkills creates curated skills in order with their resources and
// prerequisites, of the given kind unless a skill says its own. A skill
// links to its first resource.
func addSkills(ctx context.Context, tx pgx.Tx, userID, roadmapID uuid.UUID, actor graph.Actor, skills []LibrarySkill, resources map[string][]SkillResource, kind string) error {
	ids := map[string]uuid.UUID{}
	for i, s := range skills {
		rs, pos, k := resources[s.Title], i+1, kind
		if s.Kind != "" {
			k = s.Kind
		}
		f := SkillFields{Title: &s.Title, Description: &s.Description, Section: &s.Section, Kind: &k, Position: &pos, Resources: rs}
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

// competencyOf maps a curated resource title to the skill it
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
	return m
}()

// Competency returns the skill a curated resource teaches, if title is one.
func Competency(title string) (string, bool) {
	s, ok := competencyOf[strings.ToLower(strings.TrimSpace(title))]
	return s, ok
}
