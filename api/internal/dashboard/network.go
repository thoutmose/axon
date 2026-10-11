package dashboard

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/roadmaps"
)

// NetNode is a hard or soft skill.
type NetNode struct {
	ID     uuid.UUID `json:"id"`
	Type   string    `json:"type"` // skill
	Title  string    `json:"title"`
	Kind   *string   `json:"kind,omitempty"`   // hard or soft
	Status *string   `json:"status,omitempty"` // todo, learning, done, skipped
	// RoadmapID is the skill's roadmap, so the node can open it.
	RoadmapID *uuid.UUID `json:"roadmap_id,omitempty"`
}

// NetEdge is a prerequisite between two skills.
type NetEdge struct {
	From     uuid.UUID `json:"from"`
	To       uuid.UUID `json:"to"`
	Relation string    `json:"relation"`
}

// maxSkills bounds the network so it stays readable and fast.
const maxSkills = 250

// statusRank orders statuses when several skills become one node.
var statusRank = map[string]int{"done": 3, "learning": 2, "todo": 1}

// skillsNetwork returns the person's hard and soft skills (those being
// learned or learned first) and the prerequisites between them. Only
// skills: resources, goals, projects, and notes are not nodes. Skills from
// roadmaps imported before skills were competencies (one skill per book or
// course) are shown as the competency they teach, merged into one node.
func skillsNetwork(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string]any, error) {
	rows, err := q.Query(ctx, `SELECT s.id, e.title, s.kind, s.status, s.roadmap_id FROM skills s JOIN entities e ON e.id = s.id
		WHERE e.user_id = $1 AND s.status <> 'skipped'
		ORDER BY (s.status = 'todo'), s.updated_at DESC LIMIT $2`, userID, maxSkills)
	if err != nil {
		return nil, err
	}
	type row struct {
		ID        uuid.UUID
		Title     string
		Kind      *string
		Status    *string
		RoadmapID *uuid.UUID
	}
	skills, err := pgx.CollectRows(rows, pgx.RowToStructByPos[row])
	if err != nil {
		return nil, err
	}
	nodes := []NetNode{}
	alias := map[uuid.UUID]uuid.UUID{} // skill id → node id
	byKey := map[string]int{}          // roadmap + competency → node index
	ids := make([]uuid.UUID, 0, len(skills))
	for _, s := range skills {
		ids = append(ids, s.ID)
		title := s.Title
		if c, ok := roadmaps.Competency(s.Title); ok {
			title = c
		}
		key := title
		if s.RoadmapID != nil {
			key = s.RoadmapID.String() + "/" + title
		}
		if i, ok := byKey[key]; ok {
			alias[s.ID] = nodes[i].ID
			if s.Status != nil && (nodes[i].Status == nil || statusRank[*s.Status] > statusRank[*nodes[i].Status]) {
				nodes[i].Status = s.Status
			}
			continue
		}
		byKey[key] = len(nodes)
		alias[s.ID] = s.ID
		nodes = append(nodes, NetNode{ID: s.ID, Type: "skill", Title: title, Kind: s.Kind, Status: s.Status, RoadmapID: s.RoadmapID})
	}
	rows, err = q.Query(ctx, `SELECT source_id, target_id, relation FROM links
		WHERE user_id = $1 AND source_id = ANY ($2) AND target_id = ANY ($2) ORDER BY id`, userID, ids)
	if err != nil {
		return nil, err
	}
	links, err := pgx.CollectRows(rows, pgx.RowToStructByPos[NetEdge])
	if err != nil {
		return nil, err
	}
	edges := []NetEdge{}
	seen := map[[2]uuid.UUID]bool{}
	for _, l := range links {
		from, to := alias[l.From], alias[l.To]
		if from == to || seen[[2]uuid.UUID{from, to}] {
			continue
		}
		seen[[2]uuid.UUID{from, to}] = true
		edges = append(edges, NetEdge{From: from, To: to, Relation: l.Relation})
	}
	return map[string]any{"nodes": nodes, "edges": edges}, nil
}
