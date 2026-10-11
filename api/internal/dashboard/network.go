package dashboard

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// NetNode is a skill, or a goal, project, or note connected to one.
type NetNode struct {
	ID     uuid.UUID `json:"id"`
	Type   string    `json:"type"` // skill, goal, project, note
	Title  string    `json:"title"`
	Kind   *string   `json:"kind,omitempty"`   // skills: hard or soft
	Status *string   `json:"status,omitempty"` // skills: todo, learning, done, skipped
	// RoadmapID is the skill's roadmap, so the node can open it.
	RoadmapID *uuid.UUID `json:"roadmap_id,omitempty"`
}

// NetEdge is a prerequisite or a link.
type NetEdge struct {
	From     uuid.UUID `json:"from"`
	To       uuid.UUID `json:"to"`
	Relation string    `json:"relation"`
}

// maxSkills bounds the network so it stays readable and fast.
const maxSkills = 250

// skillsNetwork returns the skills being learned or learned (and, when there
// are few, those still to learn), the goals, projects, and notes linked to
// them, and every prerequisite and link between these nodes.
func skillsNetwork(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string]any, error) {
	rows, err := q.Query(ctx, `SELECT s.id, 'skill', e.title, s.kind, s.status, s.roadmap_id FROM skills s JOIN entities e ON e.id = s.id
		WHERE e.user_id = $1 AND s.status <> 'skipped'
		ORDER BY (s.status = 'todo'), s.updated_at DESC LIMIT $2`, userID, maxSkills)
	if err != nil {
		return nil, err
	}
	nodes, err := pgx.CollectRows(rows, pgx.RowToStructByPos[NetNode])
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	rows, err = q.Query(ctx, `SELECT DISTINCT o.id, o.type, o.title, NULL::text, NULL::text, NULL::uuid
		FROM links l JOIN entities o ON o.id = CASE WHEN l.source_id = ANY ($2) THEN l.target_id ELSE l.source_id END
		WHERE l.user_id = $1 AND (l.source_id = ANY ($2) OR l.target_id = ANY ($2)) AND o.type IN ('goal', 'project', 'note')
		LIMIT 200`, userID, ids)
	if err != nil {
		return nil, err
	}
	others, err := pgx.CollectRows(rows, pgx.RowToStructByPos[NetNode])
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, others...)
	all := make([]uuid.UUID, len(nodes))
	for i, n := range nodes {
		all[i] = n.ID
	}
	rows, err = q.Query(ctx, `SELECT source_id, target_id, relation FROM links
		WHERE user_id = $1 AND source_id = ANY ($2) AND target_id = ANY ($2) ORDER BY id`, userID, all)
	if err != nil {
		return nil, err
	}
	edges, err := pgx.CollectRows(rows, pgx.RowToStructByPos[NetEdge])
	if err != nil {
		return nil, err
	}
	if nodes == nil {
		nodes = []NetNode{}
	}
	if edges == nil {
		edges = []NetEdge{}
	}
	return map[string]any{"nodes": nodes, "edges": edges}, nil
}
