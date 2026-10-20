package progress

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/l10n"
)

// Rule is one way to earn points, shown to the person as is.
type Rule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Points      int    `json:"points"`
}

// Rules are every way points are earned; nothing else gives points.
var Rules = []Rule{
	{"task_done", l10n.Mark("Complete a task"), 10},
	{"task_on_time", l10n.Mark("Complete a task on or before its due date (bonus)"), 5},
	{"habit_kept", l10n.Mark("Check in a habit"), 5},
	{"habit_two_minute", l10n.Mark("Check in a habit's two-minute version"), 3},
	{"focus", l10n.Mark("Every 5 minutes worked in a focus session"), 1},
	{"card_review", l10n.Mark("Review a flashcard"), 1},
	{"note_written", l10n.Mark("Write a note"), 3},
	{"permanent_note", l10n.Mark("Write a permanent (Zettelkasten) note (bonus)"), 2},
	{"skill_learned", l10n.Mark("Mark a roadmap skill learned"), 20},
	{"domain_outstanding", l10n.Mark("Prove a roadmap domain with every bonus item too (outstanding)"), 5},
	{"milestone_done", l10n.Mark("Reach a goal's milestone"), 15},
	{"goal_achieved", l10n.Mark("Achieve a goal"), 50},
	{"project_done", l10n.Mark("Finish a project"), 30},
}

// ledger is every point-earning event: rule, points, when, and the item.
// $1 is the user, $2 and $3 bound the period.
const ledger = `
SELECT t.completed_at, 'task_done', 10, e.id, e.title FROM tasks t JOIN entities e ON e.id = t.id
 WHERE e.user_id = $1 AND t.status = 'done' AND t.completed_at >= $2 AND t.completed_at < $3
UNION ALL
SELECT t.completed_at, 'task_on_time', 5, e.id, e.title FROM tasks t JOIN entities e ON e.id = t.id
 WHERE e.user_id = $1 AND t.status = 'done' AND t.due_on IS NOT NULL
   AND (t.completed_at AT TIME ZONE $4)::date <= t.due_on AND t.completed_at >= $2 AND t.completed_at < $3
UNION ALL
SELECT c.created_at, CASE c.status WHEN 'done' THEN 'habit_kept' ELSE 'habit_two_minute' END,
       CASE c.status WHEN 'done' THEN 5 ELSE 3 END, e.id, e.title
  FROM habit_checkins c JOIN entities e ON e.id = c.habit_id
 WHERE e.user_id = $1 AND c.status IN ('done', 'two_minute') AND c.created_at >= $2 AND c.created_at < $3
UNION ALL
SELECT f.started_at, 'focus', f.worked_minutes / 5, e.id, e.title FROM focus_sessions f JOIN entities e ON e.id = f.id
 WHERE e.user_id = $1 AND f.worked_minutes >= 5 AND f.started_at >= $2 AND f.started_at < $3
UNION ALL
SELECT r.reviewed_at, 'card_review', 1, e.id, e.title FROM card_reviews r JOIN entities e ON e.id = r.card_id
 WHERE e.user_id = $1 AND r.reviewed_at >= $2 AND r.reviewed_at < $3
UNION ALL
SELECT e.created_at, 'note_written', 3, e.id, e.title FROM entities e
 WHERE e.user_id = $1 AND e.type = 'note' AND e.created_at >= $2 AND e.created_at < $3 AND NOT EXISTS (SELECT 1 FROM vault_mirror m WHERE m.note_id = e.id)
UNION ALL
SELECT e.created_at, 'permanent_note', 2, e.id, e.title FROM notes n JOIN entities e ON e.id = n.id
 WHERE e.user_id = $1 AND n.kind = 'permanent' AND e.created_at >= $2 AND e.created_at < $3
UNION ALL
SELECT s.updated_at, 'skill_learned', 20, e.id, e.title FROM skills s JOIN entities e ON e.id = s.id
 WHERE e.user_id = $1 AND s.status = 'done' AND s.updated_at >= $2 AND s.updated_at < $3
UNION ALL
SELECT v.created_at, 'domain_outstanding', 5, e.id, e.title FROM skill_validations v JOIN entities e ON e.id = v.skill_id
 WHERE e.user_id = $1 AND v.mark >= 125 AND v.created_at >= $2 AND v.created_at < $3
UNION ALL
SELECT m.done_at, 'milestone_done', 15, e.id, e.title FROM milestones m JOIN entities e ON e.id = m.id
 WHERE e.user_id = $1 AND m.done_at IS NOT NULL AND m.done_at >= $2 AND m.done_at < $3
UNION ALL
SELECT g.updated_at, 'goal_achieved', 50, e.id, e.title FROM goals g JOIN entities e ON e.id = g.id
 WHERE e.user_id = $1 AND g.status = 'achieved' AND g.updated_at >= $2 AND g.updated_at < $3
UNION ALL
SELECT p.updated_at, 'project_done', 30, e.id, e.title FROM projects p JOIN entities e ON e.id = p.id
 WHERE e.user_id = $1 AND p.status = 'done' AND p.updated_at >= $2 AND p.updated_at < $3`

// Entry is one line of the ledger: how points were earned.
type Entry struct {
	At       time.Time `json:"at"`
	Rule     string    `json:"rule"`
	Points   int       `json:"points"`
	EntityID uuid.UUID `json:"entity_id"`
	Title    string    `json:"title"`
}

var epoch = time.Unix(0, 0)

func far() time.Time { return time.Now().AddDate(1, 0, 0) }

// Ledger returns the entries in [from, to), newest first.
func Ledger(ctx context.Context, q db.Querier, userID uuid.UUID, tz string, from, to time.Time, limit int) ([]Entry, error) {
	rows, err := q.Query(ctx, `SELECT * FROM (`+ledger+`) l ORDER BY 1 DESC, 4 LIMIT $5`, userID, from, to, tz, limit)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Entry])
	if out == nil {
		out = []Entry{}
	}
	return out, err
}

// Level is the level for a total: level L starts at 50·L·(L−1) points
// (100 for level 2, 300 for 3, 600 for 4, 1,000 for 5 …).
func Level(xp int) (level, floor, next int) {
	level = 1
	for 50*(level+1)*level <= xp {
		level++
	}
	return level, 50 * level * (level - 1), 50 * (level + 1) * level
}

// Badge is an achievement with the date it was earned.
type Badge struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	EarnedAt    *time.Time `json:"earned_at"`
}

type badgeDef struct {
	id, title, description, rule string
	n                            int // the nth matching ledger entry earns it
}

var badgeDefs = []badgeDef{
	{"first_task", l10n.Mark("First step"), l10n.Mark("Completed your first task."), "task_done", 1},
	{"tasks_100", l10n.Mark("Centurion"), l10n.Mark("Completed 100 tasks."), "task_done", 100},
	{"habit_21", l10n.Mark("It sticks"), l10n.Mark("Kept habits 21 times."), "habit_kept", 21},
	{"focus_10h", l10n.Mark("Deep worker"), l10n.Mark("Focused for 10 hours (120 focus points)."), "focus", 120},
	{"cards_100", l10n.Mark("Remembering"), l10n.Mark("Reviewed 100 flashcards."), "card_review", 100},
	{"zettel_10", l10n.Mark("Slip-box"), l10n.Mark("Wrote 10 permanent notes."), "permanent_note", 10},
	{"skill_first", l10n.Mark("Learner"), l10n.Mark("Learned your first roadmap skill."), "skill_learned", 1},
	{"goal_first", l10n.Mark("Arrived"), l10n.Mark("Achieved your first goal."), "goal_achieved", 1},
}

// Badges computes every badge: count badges from the ledger (focus counts
// points, the rest count entries), and streak badges from the longest run.
func Badges(ctx context.Context, q db.Querier, userID uuid.UUID, tz string, activeDays []time.Time) ([]Badge, error) {
	out := []Badge{}
	for _, b := range badgeDefs {
		var at *time.Time
		var err error
		if b.rule == "focus" {
			err = q.QueryRow(ctx, `SELECT at FROM (SELECT l.started_at AS at, sum(l.pts) OVER (ORDER BY l.started_at) AS running
				FROM (SELECT f.started_at, f.worked_minutes / 5 AS pts FROM focus_sessions f JOIN entities e ON e.id = f.id
				      WHERE e.user_id = $1 AND f.worked_minutes >= 5) l) s WHERE running >= $2 ORDER BY at LIMIT 1`, userID, b.n).Scan(&at)
		} else {
			err = q.QueryRow(ctx, `SELECT at FROM (`+ledger+`) l(at, rule, points, id, title) WHERE rule = $5 ORDER BY at OFFSET $6 LIMIT 1`,
				userID, epoch, far(), tz, b.rule, b.n-1).Scan(&at)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		out = append(out, Badge{b.id, l10n.T(ctx, b.title), l10n.T(ctx, b.description), at})
	}
	for _, s := range []struct {
		id, title string
		n         int
	}{{"streak_7", l10n.Mark("A week"), 7}, {"streak_30", l10n.Mark("A month"), 30}, {"streak_100", l10n.Mark("A hundred days"), 100}} {
		at := longestRunReaches(activeDays, s.n)
		out = append(out, Badge{s.id, l10n.T(ctx, s.title), l10n.T(ctx, "Active %d days in a row.", s.n), at})
	}
	return out, nil
}

// longestRunReaches returns the first day a run of n consecutive active days
// was completed, or nil.
func longestRunReaches(days []time.Time, n int) *time.Time {
	run := 0
	var prev time.Time
	for _, d := range days { // ascending
		if run > 0 && d.Equal(prev.AddDate(0, 0, 1)) {
			run++
		} else {
			run = 1
		}
		prev = d
		if run == n {
			at := d
			return &at
		}
	}
	return nil
}
