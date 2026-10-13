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
// one per section, since a radar needs three spokes. Skipped skills are not
// counted, as in a roadmap's progress.

// maxSpokes keeps a radar readable: the smallest areas fold into "Other".
const maxSpokes = 9

// Spoke is one area: its skills done, being learnt, and in all. By section,
// Label is the section and Roadmap says whose; by roadmap, both are its title.
type Spoke struct {
	Label    string `json:"label"`
	Roadmap  string `json:"roadmap"`
	Done     int    `json:"done"`
	Learning int    `json:"learning"`
	Total    int    `json:"total"`
}

// RadarKind is the spokes of one kind of skill, and what a spoke stands for.
type RadarKind struct {
	By     string  `json:"by"` // roadmap or section
	Spokes []Spoke `json:"spokes"`
}

// Radar is the hard and the soft skills.
type Radar struct {
	Hard RadarKind `json:"hard"`
	Soft RadarKind `json:"soft"`
}

// skillCount is how many skills of a roadmap's section have a status.
type skillCount struct {
	Kind, RoadmapID, Roadmap, Section, Status string
	N                                         int
}

// GetRadar counts the person's skills by kind, roadmap, section, and status.
func GetRadar(ctx context.Context, q db.Querier, userID uuid.UUID) (Radar, error) {
	rows, err := q.Query(ctx, `SELECT s.kind, r.id::text, e.title, s.section, s.status, count(*)::int
		FROM skills s JOIN roadmaps r ON r.id = s.roadmap_id JOIN entities e ON e.id = r.id
		WHERE e.user_id = $1 AND s.status <> 'skipped'
		GROUP BY s.kind, r.id, e.title, s.section, s.status`, userID)
	if err != nil {
		return Radar{}, err
	}
	counts, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (c skillCount, err error) {
		err = r.Scan(&c.Kind, &c.RoadmapID, &c.Roadmap, &c.Section, &c.Status, &c.N)
		return c, err
	})
	if err != nil {
		return Radar{}, err
	}
	return radar(counts), nil
}

func radar(counts []skillCount) Radar {
	var hard, soft []skillCount
	for _, c := range counts {
		if c.Kind == "soft" {
			soft = append(soft, c)
		} else {
			hard = append(hard, c)
		}
	}
	return Radar{Hard: radarKind(hard), Soft: radarKind(soft)}
}

// sectionNumber is the "2 · " that orders library sections.
var sectionNumber = regexp.MustCompile(`^\d+\s*·\s*`)

func radarKind(counts []skillCount) RadarKind {
	// Order by roadmap, then section, so spokes keep their place as they fill.
	slices.SortStableFunc(counts, func(a, b skillCount) int {
		return cmp.Or(cmp.Compare(a.Roadmap, b.Roadmap), cmp.Compare(a.RoadmapID, b.RoadmapID), cmp.Compare(a.Section, b.Section))
	})
	roadmaps := map[string]bool{}
	for _, c := range counts {
		roadmaps[c.RoadmapID] = true
	}
	by := "roadmap"
	if len(roadmaps) < 3 {
		by = "section"
	}
	var spokes []Spoke
	var keys []string
	for _, c := range counts {
		key, label := c.RoadmapID, c.Roadmap
		if by == "section" {
			key += "\x00" + c.Section
			if section := sectionNumber.ReplaceAllString(c.Section, ""); section != "" {
				label = section
			}
		}
		if len(keys) == 0 || keys[len(keys)-1] != key {
			keys = append(keys, key)
			spokes = append(spokes, Spoke{Label: label, Roadmap: c.Roadmap})
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
