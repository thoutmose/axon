package roadmaps

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
)

// Study hours against the plan (EP-105): focus time spent on a roadmap,
// counted from the sessions that were for it, one of its skills, its goal
// or milestones, or the projects, habits, and tasks of its plan. A session
// for nothing in particular is not counted: Axon cannot tell what it was for.

// Minutes is focus time on one thing.
type Minutes struct {
	ID      uuid.UUID `json:"id"`
	Title   string    `json:"title"`
	Minutes int       `json:"minutes"`
}

// StudyDay is a day of this week: the plan's block and the time spent.
type StudyDay struct {
	Date    string `json:"date"`
	Block   string `json:"block"`
	Minutes int    `json:"minutes"`
}

// StudyWeek is the focus time of a week starting on Monday.
type StudyWeek struct {
	Week    string `json:"week"`
	Minutes int    `json:"minutes"`
}

// Hours is a roadmap's study time against its plan.
type Hours struct {
	TargetHours *float64    `json:"target_hours"` // a week, from the plan
	Weeks       []StudyWeek `json:"weeks"`        // the last 12, oldest first
	Days        []StudyDay  `json:"days"`         // this week, Monday first
	Skills      []Minutes   `json:"skills"`
	Projects    []Minutes   `json:"projects"`
}

// targetHours reads the low end of a plan's "22-26".
func targetHours(s string) *float64 {
	lo, _, _ := strings.Cut(s, "-")
	f, err := strconv.ParseFloat(strings.TrimSpace(lo), 64)
	if err != nil || f <= 0 {
		return nil
	}
	return &f
}

// StudyHours works a roadmap's study time out over the last 12 weeks.
func StudyHours(ctx context.Context, q db.Querier, userID, id uuid.UUID, now time.Time) (Hours, error) {
	var planID *string
	var planStart *time.Time
	if err := q.QueryRow(ctx, `SELECT r.plan_id, r.plan_start FROM roadmaps r JOIN entities e ON e.id = r.id
		WHERE r.id = $1 AND e.user_id = $2`, id, userID).Scan(&planID, &planStart); err != nil {
		return Hours{}, err
	}
	today, loc, err := clock.Today(ctx, q, userID, now)
	if err != nil {
		return Hours{}, err
	}
	monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	since := monday.AddDate(0, 0, -7*11)
	rows, err := q.Query(ctx, `
		WITH goals AS (SELECT target_id AS id FROM links WHERE user_id = $1 AND source_id = $2 AND relation = 'supports'),
		plan AS (
			SELECT id FROM skills WHERE roadmap_id = $2
			UNION SELECT $2::uuid
			UNION SELECT id FROM goals
			UNION SELECT m.id FROM milestones m WHERE m.goal_id IN (SELECT id FROM goals)
			UNION SELECT l.source_id FROM links l WHERE l.user_id = $1 AND l.relation = 'supports' AND l.target_id IN (SELECT id FROM goals)
		),
		boards AS (SELECT pc.task_id, pc.project_id FROM project_cards pc WHERE pc.project_id IN (SELECT id FROM plan))
		SELECT (s.started_at AT TIME ZONE $4)::date, s.worked_minutes, t.id, t.type, t.title,
		       coalesce(b.project_id, CASE WHEN t.type = 'project' THEN t.id END),
		       coalesce((SELECT title FROM entities WHERE id = b.project_id), CASE WHEN t.type = 'project' THEN t.title END)
		FROM focus_sessions s JOIN entities e ON e.id = s.id
		JOIN links l ON l.source_id = s.id AND l.relation = 'focused_on'
		JOIN entities t ON t.id = l.target_id
		LEFT JOIN boards b ON b.task_id = t.id
		WHERE e.user_id = $1 AND s.started_at >= $3 AND s.worked_minutes > 0
		  AND (t.id IN (SELECT id FROM plan) OR b.task_id IS NOT NULL)`,
		userID, id, since.Add(-24*time.Hour), loc.String())
	if err != nil {
		return Hours{}, err
	}
	type row struct {
		Day          time.Time
		Minutes      int
		TargetID     uuid.UUID
		TargetType   string
		TargetTitle  string
		ProjectID    *uuid.UUID
		ProjectTitle *string
	}
	sessions, err := pgx.CollectRows(rows, pgx.RowToStructByPos[row])
	if err != nil {
		return Hours{}, err
	}

	h := Hours{Weeks: make([]StudyWeek, 12), Days: make([]StudyDay, 7), Skills: []Minutes{}, Projects: []Minutes{}}
	for i := range h.Weeks {
		h.Weeks[i].Week = since.AddDate(0, 0, 7*i).Format(time.DateOnly)
	}
	for i := range h.Days {
		h.Days[i].Date = monday.AddDate(0, 0, i).Format(time.DateOnly)
	}
	skills, projects := map[uuid.UUID]*Minutes{}, map[uuid.UUID]*Minutes{}
	add := func(m map[uuid.UUID]*Minutes, id uuid.UUID, title string, n int) {
		if m[id] == nil {
			m[id] = &Minutes{ID: id, Title: title}
		}
		m[id].Minutes += n
	}
	for _, s := range sessions {
		if s.Day.Before(since) {
			continue
		}
		h.Weeks[int(s.Day.Sub(since).Hours()/24)/7].Minutes += s.Minutes
		if d := int(s.Day.Sub(monday).Hours() / 24); d >= 0 && d < 7 {
			h.Days[d].Minutes += s.Minutes
		}
		if s.TargetType == SkillType {
			add(skills, s.TargetID, s.TargetTitle, s.Minutes)
		}
		if s.ProjectID != nil {
			add(projects, *s.ProjectID, *s.ProjectTitle, s.Minutes)
		}
	}
	for _, m := range skills {
		h.Skills = append(h.Skills, *m)
	}
	for _, m := range projects {
		h.Projects = append(h.Projects, *m)
	}
	for _, l := range [][]Minutes{h.Skills, h.Projects} {
		sort.Slice(l, func(a, b int) bool { return l[a].Minutes > l[b].Minutes })
	}

	// The plan's blocks for this week, and its weekly target.
	if planID != nil && planStart != nil {
		for _, d := range lib.Roadmaps {
			if d.ID != *planID || d.Plan == nil {
				continue
			}
			h.TargetHours = targetHours(d.Plan.HoursPerWeek)
			shift := int(planStart.Sub(day(d.Plan.Start)).Hours() / 24)
			for _, b := range d.Plan.Blocks(shift, monday, monday.AddDate(0, 0, 6)) {
				h.Days[int(b.Date.Sub(monday).Hours()/24)].Block = b.Title
			}
		}
	}
	return h, nil
}
