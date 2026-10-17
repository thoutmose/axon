package roadmaps

import (
	"cmp"
	"context"
	"regexp"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// The skills radar (EP-82): how much of each area of hard skills and of soft
// skills is acquired. One spoke per roadmap; under three roadmaps of a kind,
// one per section, since a radar needs three spokes, except for a roadmap's
// plan, whose skills are domains (maths, Rust, Kubernetes…): one spoke per
// domain, with the level the assessment last gave it (EP-163). Skipped skills
// are not counted, as in a roadmap's progress.

// maxSpokes keeps a radar readable: the smallest areas fold into "Other".
const maxSpokes = 9

// Spoke is one area: its skills done, being learnt, and in all. By section
// or domain, Label is the section or skill and Roadmap says whose; by
// roadmap, both are its title. Level is the assessed level, 0 to 5, of a
// domain the person has assessed.
type Spoke struct {
	Label    string `json:"label"`
	Roadmap  string `json:"roadmap"`
	Done     int    `json:"done"`
	Learning int    `json:"learning"`
	Total    int    `json:"total"`
	Level    *int   `json:"level,omitempty"`
}

// RadarKind is the spokes of one kind of skill, and what a spoke stands for.
type RadarKind struct {
	By     string  `json:"by"` // roadmap, section, or domain
	Spokes []Spoke `json:"spokes"`
}

// Radar is the hard and the soft skills.
type Radar struct {
	Hard RadarKind `json:"hard"`
	Soft RadarKind `json:"soft"`
}

// skillCount is how many skills of a roadmap's section have a status. Skill
// and Plan are set for a plan's skill, counted one by one.
type skillCount struct {
	Kind, RoadmapID, Roadmap, Section, Status string
	N                                         int
	Skill, Plan                               string
}

// GetRadar counts the person's skills by kind, roadmap, section, and status.
func GetRadar(ctx context.Context, q db.Querier, userID uuid.UUID) (Radar, error) {
	rows, err := q.Query(ctx, `SELECT s.kind, r.id::text, e.title, s.section, s.status, count(*)::int,
			CASE WHEN r.plan_id IS NULL THEN '' ELSE se.title END, coalesce(r.plan_id, '')
		FROM skills s JOIN roadmaps r ON r.id = s.roadmap_id JOIN entities e ON e.id = r.id JOIN entities se ON se.id = s.id
		WHERE e.user_id = $1 AND s.status <> 'skipped'
		GROUP BY s.kind, r.id, e.title, s.section, s.status, 7, 8`, userID)
	if err != nil {
		return Radar{}, err
	}
	counts, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (c skillCount, err error) {
		err = r.Scan(&c.Kind, &c.RoadmapID, &c.Roadmap, &c.Section, &c.Status, &c.N, &c.Skill, &c.Plan)
		return c, err
	})
	if err != nil {
		return Radar{}, err
	}
	levels, err := assessedLevels(ctx, q, userID)
	if err != nil {
		return Radar{}, err
	}
	return radar(counts, levels), nil
}

// assessedLevels is the latest level of each assessed domain.
func assessedLevels(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string]int, error) {
	all, err := ListAssessments(ctx, q, userID) // newest first
	if err != nil {
		return nil, err
	}
	levels := map[string]int{}
	for _, a := range all {
		for _, r := range a.Results {
			if _, seen := levels[r.Domain]; !seen {
				levels[r.Domain] = r.Level
			}
		}
	}
	return levels, nil
}

// assessDomain is the assessment domain of a plan's skill, from the library.
func assessDomain(plan, skill string) string {
	for _, d := range lib.Roadmaps {
		if d.ID == plan {
			for _, sk := range d.Skills {
				if sk.Title == skill {
					return sk.Assess
				}
			}
		}
	}
	return ""
}

func radar(counts []skillCount, levels map[string]int) Radar {
	var hard, soft []skillCount
	for _, c := range counts {
		if c.Kind == "soft" {
			soft = append(soft, c)
		} else {
			hard = append(hard, c)
		}
	}
	return Radar{Hard: radarKind(hard, levels), Soft: radarKind(soft, levels)}
}

// sectionNumber is the "2 · " that orders library sections.
var sectionNumber = regexp.MustCompile(`^\d+\s*·\s*`)

func radarKind(counts []skillCount, levels map[string]int) RadarKind {
	// Order by roadmap, then section, so spokes keep their place as they fill.
	slices.SortStableFunc(counts, func(a, b skillCount) int {
		return cmp.Or(cmp.Compare(a.Roadmap, b.Roadmap), cmp.Compare(a.RoadmapID, b.RoadmapID), cmp.Compare(a.Section, b.Section), cmp.Compare(a.Skill, b.Skill))
	})
	roadmaps := map[string]bool{}
	for _, c := range counts {
		roadmaps[c.RoadmapID] = true
	}
	by := "roadmap"
	if len(roadmaps) < 3 {
		by = "section"
		if len(counts) > 0 {
			by = "domain"
		}
		for _, c := range counts {
			if c.Plan == "" {
				by = "section"
			}
		}
	}
	var spokes []Spoke
	var keys []string
	for _, c := range counts {
		key, label := c.RoadmapID, c.Roadmap
		if by == "domain" {
			key += "\x00" + c.Skill
			label = c.Skill
		}
		if by == "section" {
			key += "\x00" + c.Section
			if section := sectionNumber.ReplaceAllString(c.Section, ""); section != "" {
				label = section
			}
		}
		if len(keys) == 0 || keys[len(keys)-1] != key {
			keys = append(keys, key)
			spokes = append(spokes, Spoke{Label: label, Roadmap: c.Roadmap})
			if by == "domain" {
				if l, ok := levels[assessDomain(c.Plan, c.Skill)]; ok {
					spokes[len(spokes)-1].Level = &l
				}
			}
		}
		s := &spokes[len(spokes)-1]
		s.Total += c.N
		switch c.Status {
		case "done":
			s.Done += c.N
		case "learning":
			s.Learning += c.N
		}
	}
	if by == "domain" { // every domain keeps its spoke
		return RadarKind{By: by, Spokes: append([]Spoke{}, spokes...)}
	}
	return RadarKind{By: by, Spokes: fold(spokes)}
}

// fold keeps the largest spokes in their order and adds up the rest as Other.
func fold(spokes []Spoke) []Spoke {
	if len(spokes) <= maxSpokes {
		return append([]Spoke{}, spokes...)
	}
	bySize := make([]int, len(spokes))
	for i := range bySize {
		bySize[i] = i
	}
	slices.SortStableFunc(bySize, func(a, b int) int { return cmp.Compare(spokes[b].Total, spokes[a].Total) })
	keep := make([]bool, len(spokes))
	for _, i := range bySize[:maxSpokes-1] {
		keep[i] = true
	}
	out := make([]Spoke, 0, maxSpokes)
	other := Spoke{Label: "Other"}
	for i, s := range spokes {
		if keep[i] {
			out = append(out, s)
			continue
		}
		other.Done += s.Done
		other.Learning += s.Learning
		other.Total += s.Total
	}
	return append(out, other)
}
