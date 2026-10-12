// Package projects runs projects on a Kanban board built on tasks: epics,
// issues, and sub-tasks are tasks with a card on one project's board (EP-10).
package projects

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Type is the graph entity type of a project.
const Type = "project"

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "projects", Export: export})
}

// Errors callers translate to HTTP responses.
var (
	ErrStale     = errors.New("stale version")
	ErrOverLimit = errors.New("over the WIP limit")
)

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

var (
	statuses = []string{"active", "on_hold", "done", "archived"}
	columns  = []struct{ ID, Label string }{{"backlog", "Backlog"}, {"todo", "To do"}, {"doing", "Doing"}, {"done", "Done"}}
	kinds    = []string{"epic", "issue", "subtask"}
)

// Project is the API representation.
type Project struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	WIPLimit    *int      `json:"wip_limit"`
	DocsFolder  string    `json:"docs_folder"`
	Done        int       `json:"done"`  // cards whose task is done
	Total       int       `json:"total"` // all cards
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const projectColumns = `p.id, e.title, p.description, p.status, p.wip_limit, p.docs_folder,
	(SELECT count(*) FROM project_cards c JOIN tasks t ON t.id = c.task_id WHERE c.project_id = p.id AND t.status = 'done'),
	(SELECT count(*) FROM project_cards c WHERE c.project_id = p.id),
	e.created_at, p.updated_at`

func scan(row pgx.Row) (Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.WIPLimit, &p.DocsFolder, &p.Done, &p.Total, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// Get returns one of the user's projects.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Project, error) {
	p, err := scan(q.QueryRow(ctx, `SELECT `+projectColumns+` FROM projects p JOIN entities e ON e.id = p.id
		WHERE p.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, graph.ErrNotFound
	}
	return p, err
}

// List returns the user's projects with a status, newest first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, status string) ([]Project, error) {
	rows, err := q.Query(ctx, `SELECT `+projectColumns+` FROM projects p JOIN entities e ON e.id = p.id
		WHERE e.user_id = $1 AND p.status = $2 ORDER BY e.created_at DESC LIMIT 200`, userID, status)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Project, error) { return scan(r) })
	if out == nil {
		out = []Project{}
	}
	return out, err
}

// Fields are the editable parts of a project; nil means "not provided".
type Fields struct {
	Title, Description, Status, DocsFolder *string
	WIPLimit                               *int
	ClearWIPLimit                          bool
}

func (f *Fields) validate() error {
	if f.Title != nil {
		*f.Title = strings.TrimSpace(*f.Title)
		if *f.Title == "" || len([]rune(*f.Title)) > 500 {
			return ValidationError{"A project needs a title of 1 to 500 characters."}
		}
	}
	if f.Description != nil && len(*f.Description) > 20000 {
		return ValidationError{"The description is limited to 20,000 characters."}
	}
	if f.Status != nil && !slices.Contains(statuses, *f.Status) {
		return ValidationError{"Status must be active, on_hold, done, or archived."}
	}
	if f.WIPLimit != nil && (*f.WIPLimit < 1 || *f.WIPLimit > 50) {
		return ValidationError{"A WIP limit is between 1 and 50 cards."}
	}
	if f.DocsFolder != nil {
		d := strings.Trim(strings.TrimSpace(*f.DocsFolder), "/")
		if d == "" || len(d) > 300 || strings.Contains(d, "..") || strings.HasPrefix(d, ".") {
			return ValidationError{"The docs folder is a relative folder such as 1-Projects/axon."}
		}
		f.DocsFolder = &d
	}
	return nil
}

func slug(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127 && r != ' ':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "project"
	}
	return s
}

// Create adds a project; its docs live in 1-Projects/<slug> unless given.
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f Fields) (Project, error) {
	if f.Title == nil {
		return Project{}, ValidationError{"A project needs a title."}
	}
	if err := f.validate(); err != nil {
		return Project{}, err
	}
	docs := "1-Projects/" + slug(*f.Title)
	if f.DocsFolder != nil {
		docs = *f.DocsFolder
	}
	desc, status := "", "active"
	if f.Description != nil {
		desc = *f.Description
	}
	if f.Status != nil {
		status = *f.Status
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, *f.Title, desc)
	if err != nil {
		return Project{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO projects (id, description, status, wip_limit, docs_folder) VALUES ($1, $2, $3, $4, $5)`,
		id, desc, status, f.WIPLimit, docs); err != nil {
		return Project{}, err
	}
	return Get(ctx, tx, userID, id)
}

// Update applies f if the project's version still equals version.
func Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, version time.Time, f Fields) (Project, error) {
	if err := f.validate(); err != nil {
		return Project{}, err
	}
	cur, err := Get(ctx, tx, userID, id)
	if err != nil {
		return Project{}, err
	}
	if !cur.UpdatedAt.Equal(version) {
		return Project{}, ErrStale
	}
	next := cur
	var changed []string
	set := func(name string, p *string, dst *string) {
		if p != nil && *p != *dst {
			*dst = *p
			changed = append(changed, name)
		}
	}
	set("title", f.Title, &next.Title)
	set("description", f.Description, &next.Description)
	set("status", f.Status, &next.Status)
	set("docs_folder", f.DocsFolder, &next.DocsFolder)
	if f.WIPLimit != nil || (f.ClearWIPLimit && cur.WIPLimit != nil) {
		next.WIPLimit = f.WIPLimit
		changed = append(changed, "wip_limit")
	}
	if len(changed) == 0 {
		return cur, nil
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, next.Title, next.Description, changed); err != nil {
		return Project{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE projects SET description = $2, status = $3, wip_limit = $4, docs_folder = $5,
		updated_at = clock_timestamp() WHERE id = $1`, id, next.Description, next.Status, next.WIPLimit, next.DocsFolder); err != nil {
		return Project{}, err
	}
	return Get(ctx, tx, userID, id)
}

// Delete removes a project and its board; the tasks themselves stay.
func Delete(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	if _, err := Get(ctx, tx, userID, id); err != nil {
		return err
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

// Card is a task on a board.
type Card struct {
	TaskID   uuid.UUID   `json:"task_id"`
	Title    string      `json:"title"`
	Status   string      `json:"status"`
	DueOn    *tasks.Date `json:"due_on"`
	Column   string      `json:"column"`
	Kind     string      `json:"kind"`
	ParentID *uuid.UUID  `json:"parent_id"`
	Done     int         `json:"children_done"`
	Total    int         `json:"children"`
}

// Column is one lane of the board.
type Column struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Limit *int   `json:"limit"`
	Cards []Card `json:"cards"`
}

// Board is a project's Kanban board. A card whose task is done always shows
// in Done, wherever it was, so the board and task lists never disagree.
type Board struct {
	Project Project  `json:"project"`
	Columns []Column `json:"columns"`
}

// GetBoard returns the board, cards in column order.
func GetBoard(ctx context.Context, q db.Querier, userID, projectID uuid.UUID) (Board, error) {
	p, err := Get(ctx, q, userID, projectID)
	if err != nil {
		return Board{}, err
	}
	rows, err := q.Query(ctx, `SELECT c.task_id, e.title, t.status, t.due_on,
			CASE WHEN t.status IN ('done', 'cancelled') THEN 'done' ELSE c.col END, c.kind, c.parent_id,
			(SELECT count(*) FROM project_cards k JOIN tasks kt ON kt.id = k.task_id WHERE k.parent_id = c.task_id AND kt.status = 'done'),
			(SELECT count(*) FROM project_cards k WHERE k.parent_id = c.task_id)
		FROM project_cards c JOIN tasks t ON t.id = c.task_id JOIN entities e ON e.id = c.task_id
		WHERE c.project_id = $1 AND e.user_id = $2
		ORDER BY c.position, e.created_at LIMIT 2000`, projectID, userID)
	if err != nil {
		return Board{}, err
	}
	cards, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Card, error) {
		var c Card
		var due *time.Time
		err := r.Scan(&c.TaskID, &c.Title, &c.Status, &due, &c.Column, &c.Kind, &c.ParentID, &c.Done, &c.Total)
		if due != nil {
			c.DueOn = &tasks.Date{Time: *due}
		}
		return c, err
	})
	if err != nil {
		return Board{}, err
	}
	b := Board{Project: p}
	for _, col := range columns {
		c := Column{ID: col.ID, Label: col.Label, Cards: []Card{}}
		if col.ID == "doing" {
			c.Limit = p.WIPLimit
		}
		for _, card := range cards {
			if card.Column == col.ID {
				c.Cards = append(c.Cards, card)
			}
		}
		b.Columns = append(b.Columns, c)
	}
	return b, nil
}

// CardInput places a task on a board.
type CardInput struct {
	Column    *string
	Kind      *string
	ParentID  *uuid.UUID
	NoParent  bool
	Before    *uuid.UUID // place before this card; nil: at the end
	OverLimit bool       // the person confirmed going over the WIP limit
}

type cardRow struct {
	project uuid.UUID
	col     string
	kind    string
	parent  *uuid.UUID
}

func getCard(ctx context.Context, q db.Querier, userID, taskID uuid.UUID) (cardRow, error) {
	var c cardRow
	err := q.QueryRow(ctx, `SELECT c.project_id, c.col, c.kind, c.parent_id FROM project_cards c JOIN entities e ON e.id = c.task_id
		WHERE c.task_id = $1 AND e.user_id = $2`, taskID, userID).Scan(&c.project, &c.col, &c.kind, &c.parent)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, graph.ErrNotFound
	}
	return c, err
}

// AddCard creates a task and puts it on the board.
func AddCard(ctx context.Context, tx pgx.Tx, userID, projectID uuid.UUID, actor graph.Actor, title string, in CardInput) (Card, error) {
	if _, err := Get(ctx, tx, userID, projectID); err != nil {
		return Card{}, err
	}
	col := "todo"
	if in.Column != nil {
		col = *in.Column
	}
	status := map[string]string{"backlog": "someday", "todo": "next", "doing": "next", "done": "done"}[col]
	if status == "" {
		return Card{}, ValidationError{"Column must be backlog, todo, doing, or done."}
	}
	t, err := tasks.Create(ctx, tx, userID, actor, tasks.Fields{Title: &title, Status: &status})
	if err != nil {
		var v tasks.ValidationError
		if errors.As(err, &v) {
			return Card{}, ValidationError{v.Msg}
		}
		return Card{}, err
	}
	return place(ctx, tx, userID, projectID, actor, t.ID, in, true)
}

// Place puts an existing task on a board, or moves or re-parents its card.
func Place(ctx context.Context, tx pgx.Tx, userID, projectID uuid.UUID, actor graph.Actor, taskID uuid.UUID, in CardInput) (Card, error) {
	if _, err := Get(ctx, tx, userID, projectID); err != nil {
		return Card{}, err
	}
	if _, err := tasks.Get(ctx, tx, userID, taskID); err != nil {
		return Card{}, err
	}
	_, err := getCard(ctx, tx, userID, taskID)
	isNew := errors.Is(err, graph.ErrNotFound)
	if err != nil && !isNew {
		return Card{}, err
	}
	return place(ctx, tx, userID, projectID, actor, taskID, in, isNew)
}

func place(ctx context.Context, tx pgx.Tx, userID, projectID uuid.UUID, actor graph.Actor, taskID uuid.UUID, in CardInput, isNew bool) (Card, error) {
	cur := cardRow{project: projectID, col: "todo", kind: "issue"}
	if !isNew {
		var err error
		if cur, err = getCard(ctx, tx, userID, taskID); err != nil {
			return Card{}, err
		}
	}
	t, err := tasks.Get(ctx, tx, userID, taskID)
	if err != nil {
		return Card{}, err
	}
	shown := cur.col
	if t.Status == "done" || t.Status == "cancelled" {
		shown = "done"
	}
	next := cur
	next.project = projectID
	if in.Column != nil {
		if !slices.ContainsFunc(columns, func(c struct{ ID, Label string }) bool { return c.ID == *in.Column }) {
			return Card{}, ValidationError{"Column must be backlog, todo, doing, or done."}
		}
		next.col = *in.Column
	}
	if in.Kind != nil {
		if !slices.Contains(kinds, *in.Kind) {
			return Card{}, ValidationError{"Kind must be epic, issue, or subtask."}
		}
		next.kind = *in.Kind
	}
	if in.NoParent {
		next.parent = nil
	} else if in.ParentID != nil {
		next.parent = in.ParentID
	}
	if err := checkParent(ctx, tx, userID, projectID, taskID, next); err != nil {
		return Card{}, err
	}

	// WIP limit: entering Doing past the limit needs the person's confirmation.
	if next.col == "doing" && shown != "doing" && !in.OverLimit {
		p, err := Get(ctx, tx, userID, projectID)
		if err != nil {
			return Card{}, err
		}
		if p.WIPLimit != nil {
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM project_cards c JOIN tasks t ON t.id = c.task_id
				WHERE c.project_id = $1 AND c.col = 'doing' AND t.status NOT IN ('done', 'cancelled') AND c.task_id <> $2`,
				projectID, taskID).Scan(&n); err != nil {
				return Card{}, err
			}
			if n >= *p.WIPLimit {
				return Card{}, fmt.Errorf("%w: %d of %d", ErrOverLimit, n, *p.WIPLimit)
			}
		}
	}

	pos, err := position(ctx, tx, projectID, next.col, taskID, in.Before)
	if err != nil {
		return Card{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_cards (task_id, project_id, col, position, kind, parent_id) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (task_id) DO UPDATE SET project_id = $2, col = $3, position = $4, kind = $5, parent_id = $6`,
		taskID, projectID, next.col, pos, next.kind, next.parent); err != nil {
		return Card{}, err
	}

	// Done on the board and done in the task list are the same thing.
	finished := t.Status == "done" || t.Status == "cancelled"
	var status *string
	switch {
	case next.col == "done" && !finished:
		s := "done"
		status = &s
	case next.col != "done" && finished && in.Column != nil:
		s := "next"
		status = &s
	}
	if status != nil {
		if _, err := tasks.Update(ctx, tx, userID, taskID, actor, t.UpdatedAt, tasks.Fields{Status: status}); err != nil {
			return Card{}, err
		}
	}
	b, err := GetBoard(ctx, tx, userID, projectID)
	if err != nil {
		return Card{}, err
	}
	for _, c := range b.Columns {
		for _, card := range c.Cards {
			if card.TaskID == taskID {
				return card, nil
			}
		}
	}
	return Card{}, graph.ErrNotFound
}

// checkParent keeps the hierarchy simple: issues belong to epics, sub-tasks
// to issues or epics, epics to nothing, all on the same board.
func checkParent(ctx context.Context, q db.Querier, userID, projectID, taskID uuid.UUID, c cardRow) error {
	if c.parent == nil {
		if c.kind == "subtask" {
			return ValidationError{"A sub-task belongs to an issue or an epic."}
		}
		return nil
	}
	if c.kind == "epic" {
		return ValidationError{"An epic cannot have a parent."}
	}
	if *c.parent == taskID {
		return ValidationError{"A card cannot be its own parent."}
	}
	parent, err := getCard(ctx, q, userID, *c.parent)
	if errors.Is(err, graph.ErrNotFound) || err == nil && parent.project != projectID {
		return ValidationError{"The parent must be a card on the same board."}
	}
	if err != nil {
		return err
	}
	if c.kind == "issue" && parent.kind != "epic" || c.kind == "subtask" && parent.kind == "subtask" {
		return ValidationError{"Issues belong to epics; sub-tasks belong to issues or epics."}
	}
	return nil
}

// position places a card before another, or at the end of the column.
func position(ctx context.Context, q db.Querier, projectID uuid.UUID, col string, taskID uuid.UUID, before *uuid.UUID) (float64, error) {
	if before == nil || *before == taskID {
		var end float64
		err := q.QueryRow(ctx, `SELECT coalesce(max(position), 0) + 1 FROM project_cards WHERE project_id = $1 AND col = $2 AND task_id <> $3`,
			projectID, col, taskID).Scan(&end)
		return end, err
	}
	var at float64
	if err := q.QueryRow(ctx, `SELECT position FROM project_cards WHERE task_id = $1 AND project_id = $2`, *before, projectID).Scan(&at); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ValidationError{"The card to place before is not on this board."}
		}
		return 0, err
	}
	var prev *float64
	if err := q.QueryRow(ctx, `SELECT max(position) FROM project_cards WHERE project_id = $1 AND col = $2 AND position < $3 AND task_id <> $4`,
		projectID, col, at, taskID).Scan(&prev); err != nil {
		return 0, err
	}
	if prev == nil {
		return at - 1, nil
	}
	return (*prev + at) / 2, nil
}

// RemoveCard takes a task off its board; the task stays.
func RemoveCard(ctx context.Context, tx pgx.Tx, userID, taskID uuid.UUID) error {
	if _, err := getCard(ctx, tx, userID, taskID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM project_cards WHERE task_id = $1`, taskID)
	return err
}

// Doc is a note in the project's docs folder.
type Doc struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Docs lists the notes under the project's docs folder.
func Docs(ctx context.Context, q db.Querier, userID, projectID uuid.UUID) ([]Doc, error) {
	p, err := Get(ctx, q, userID, projectID)
	if err != nil {
		return nil, err
	}
	prefix := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(p.DocsFolder) + "/%"
	rows, err := q.Query(ctx, `SELECT n.id, e.title, n.path, n.updated_at FROM notes n JOIN entities e ON e.id = n.id
		WHERE n.user_id = $1 AND n.path LIKE $2 ORDER BY n.path LIMIT 200`, userID, prefix)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Doc])
	if out == nil {
		out = []Doc{}
	}
	return out, err
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+projectColumns+` FROM projects p JOIN entities e ON e.id = p.id WHERE e.user_id = $1 ORDER BY p.id`, userID)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Project, error) { return scan(r) })
	if err != nil {
		return nil, err
	}
	type withBoard struct {
		Project
		Cards []map[string]any `json:"cards"`
	}
	out := []withBoard{}
	for _, p := range list {
		rows, err := q.Query(ctx, `SELECT task_id, col, position, kind, parent_id FROM project_cards WHERE project_id = $1 ORDER BY col, position`, p.ID)
		if err != nil {
			return nil, err
		}
		cards := []map[string]any{}
		for rows.Next() {
			var id uuid.UUID
			var col, kind string
			var pos float64
			var parent *uuid.UUID
			if err := rows.Scan(&id, &col, &pos, &kind, &parent); err != nil {
				rows.Close()
				return nil, err
			}
			cards = append(cards, map[string]any{"task_id": id, "column": col, "position": pos, "kind": kind, "parent_id": parent})
		}
		rows.Close()
		out = append(out, withBoard{Project: p, Cards: cards})
	}
	return out, nil
}
