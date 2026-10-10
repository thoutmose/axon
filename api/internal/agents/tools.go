package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Tool is one capability an agent can be granted. Read tools return data.
// Write tools never change data when the model calls them: the call becomes
// a proposal, and Apply runs only after the user approves it.
type Tool struct {
	Name        string
	Description string
	Properties  map[string]any
	Required    []string
	Write       bool

	// Read returns data for the model (read tools).
	Read func(ctx context.Context, q db.Querier, userID uuid.UUID, in json.RawMessage) (any, error)
	// Summary describes the change for the approval queue (write tools).
	Summary func(ctx context.Context, q db.Querier, userID uuid.UUID, in json.RawMessage) (string, error)
	// Apply makes the change, attributed to the agent (write tools).
	Apply func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, in json.RawMessage) error
}

// toolError is a problem the model can fix (bad id, invalid field). It goes
// back to the model as an error result; anything else stops the run.
type toolError struct{ msg string }

func (e toolError) Error() string { return e.msg }

func toolErrorf(format string, a ...any) error { return toolError{fmt.Sprintf(format, a...)} }

// decode reads a tool input strictly: unknown fields are an error.
func decode(in json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(in))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return toolErrorf("invalid input: %v", err)
	}
	return nil
}

func parseID(s, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return id, toolErrorf("%s is not a valid id", what)
	}
	return id, nil
}

func parseDate(s *string, what string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, *s)
	if err != nil {
		return nil, toolErrorf("%s must be a date like 2026-10-12", what)
	}
	return &t, nil
}

// notFound turns "no such row" into something the model can act on.
func notFound(err error, what string) error {
	if errors.Is(err, graph.ErrNotFound) {
		return toolErrorf("no %s with that id", what)
	}
	var v tasks.ValidationError
	if errors.As(err, &v) {
		return toolError{v.Msg}
	}
	var g goals.ValidationError
	if errors.As(err, &g) {
		return toolError{g.Msg}
	}
	return err
}

var (
	str     = map[string]any{"type": "string"}
	date    = map[string]any{"type": "string", "format": "date", "description": "YYYY-MM-DD. Only when the person gave this date."}
	idField = map[string]any{"type": "string", "description": "An id returned by a list or get tool."}
	reason  = map[string]any{"type": "string", "description": "One sentence the person can check at a glance."}
	status  = map[string]any{"type": "string", "enum": []string{"inbox", "next", "waiting", "someday", "done"}}
)

type taskRow struct {
	ID          uuid.UUID   `json:"id"`
	Title       string      `json:"title"`
	Status      string      `json:"status"`
	Context     *string     `json:"context,omitempty"`
	ScheduledOn *tasks.Date `json:"scheduled_on,omitempty"`
	DueOn       *tasks.Date `json:"due_on,omitempty"`
	Notes       string      `json:"notes,omitempty"`
}

func taskOf(t tasks.Task) taskRow {
	return taskRow{t.ID, t.Title, t.Status, t.Context, t.ScheduledOn, t.DueOn, t.Notes}
}

type taskInput struct {
	ID          string  `json:"id"`
	Title       *string `json:"title"`
	Status      *string `json:"status"`
	Context     *string `json:"context"`
	ScheduledOn *string `json:"scheduled_on"`
	DueOn       *string `json:"due_on"`
	Reason      string  `json:"reason"`
	// Version is added by Axon when the proposal is made: approval fails if
	// the task changed in the meantime.
	Version *time.Time `json:"version,omitempty"`
}

func (in taskInput) fields() (tasks.Fields, error) {
	f := tasks.Fields{Title: in.Title, Status: in.Status, Context: in.Context}
	var err error
	if f.ScheduledOn, err = parseDate(in.ScheduledOn, "scheduled_on"); err != nil {
		return f, err
	}
	f.DueOn, err = parseDate(in.DueOn, "due_on")
	return f, err
}

func changes(in taskInput) string {
	var parts []string
	add := func(name string, v *string) {
		if v != nil {
			parts = append(parts, fmt.Sprintf("%s → %s", name, *v))
		}
	}
	add("title", in.Title)
	add("status", in.Status)
	add("context", in.Context)
	add("do on", in.ScheduledOn)
	add("due", in.DueOn)
	return strings.Join(parts, ", ")
}

var registry = map[string]Tool{
	"list_tasks": {
		Name:        "list_tasks",
		Description: "List the person's open tasks (up to 50), soonest date first. Optionally filter by status.",
		Properties: map[string]any{
			"status": map[string]any{"type": "string", "enum": []string{"inbox", "next", "waiting", "someday"}},
			"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 50},
		},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				Status *string `json:"status"`
				Limit  int     `json:"limit"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			if in.Limit <= 0 || in.Limit > 50 {
				in.Limit = 50
			}
			rows, err := q.Query(ctx, `SELECT t.id FROM tasks t JOIN entities e ON e.id = t.id
				WHERE e.user_id = $1 AND t.status NOT IN ('done', 'cancelled') AND ($2::text IS NULL OR t.status = $2)
				ORDER BY coalesce(t.due_on, t.scheduled_on, 'infinity'::date), e.created_at LIMIT $3`, userID, in.Status, in.Limit)
			if err != nil {
				return nil, err
			}
			ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
			if err != nil {
				return nil, err
			}
			out := []taskRow{}
			for _, id := range ids {
				t, err := tasks.Get(ctx, q, userID, id)
				if err != nil {
					return nil, err
				}
				row := taskOf(t)
				row.Notes = "" // get_task returns notes
				out = append(out, row)
			}
			return out, nil
		},
	},
	"get_task": {
		Name:        "get_task",
		Description: "Read one task, including its notes.",
		Properties:  map[string]any{"id": idField},
		Required:    []string{"id"},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				ID string `json:"id"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			id, err := parseID(in.ID, "id")
			if err != nil {
				return nil, err
			}
			t, err := tasks.Get(ctx, q, userID, id)
			if err != nil {
				return nil, notFound(err, "task")
			}
			return taskOf(t), nil
		},
	},
	"update_task": {
		Name:        "update_task",
		Description: "Propose changes to one task. Nothing changes until the person approves. Include only the fields to change.",
		Properties: map[string]any{
			"id": idField, "title": str, "status": status,
			"context":      map[string]any{"type": "string", "description": "Like @home or @phone."},
			"scheduled_on": date, "due_on": date, "reason": reason,
		},
		Required: []string{"id", "reason"},
		Write:    true,
		Summary: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (string, error) {
			var in taskInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", err
			}
			id, _ := uuid.Parse(in.ID)
			t, err := tasks.Get(ctx, q, userID, id)
			if err != nil {
				return "", notFound(err, "task")
			}
			return fmt.Sprintf("Change task “%s”: %s", t.Title, changes(in)), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in taskInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			id, err := parseID(in.ID, "id")
			if err != nil {
				return err
			}
			f, err := in.fields()
			if err != nil {
				return err
			}
			if changes(in) == "" {
				return toolErrorf("include at least one field to change")
			}
			cur, err := tasks.Get(ctx, tx, userID, id)
			if err != nil {
				return notFound(err, "task")
			}
			version := cur.UpdatedAt
			if in.Version != nil {
				version = *in.Version
			}
			_, err = tasks.Update(ctx, tx, userID, id, actor, version, f)
			if errors.Is(err, tasks.ErrStale) {
				return toolErrorf("the task changed after this was proposed")
			}
			return notFound(err, "task")
		},
	},
	"create_task": {
		Name:        "create_task",
		Description: "Propose a new task. Nothing is created until the person approves.",
		Properties: map[string]any{
			"title": str, "status": status,
			"context":      map[string]any{"type": "string", "description": "Like @home or @phone."},
			"scheduled_on": date, "due_on": date, "reason": reason,
		},
		Required: []string{"title", "reason"},
		Write:    true,
		Summary: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (string, error) {
			var in taskInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", err
			}
			title := ""
			if in.Title != nil {
				title = *in.Title
			}
			in.Title = nil
			s := fmt.Sprintf("Create task “%s”", title)
			if c := changes(in); c != "" {
				s += ": " + c
			}
			return s, nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in taskInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			if in.ID != "" {
				return toolErrorf("create_task takes no id")
			}
			f, err := in.fields()
			if err != nil {
				return err
			}
			_, err = tasks.Create(ctx, tx, userID, actor, f)
			return notFound(err, "task")
		},
	},
	"list_goals": {
		Name:        "list_goals",
		Description: "List the person's active goals with their milestones and progress.",
		Properties:  map[string]any{},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			if err := decode(raw, &struct{}{}); err != nil {
				return nil, err
			}
			list, err := goals.List(ctx, q, userID, "active")
			if err != nil {
				return nil, err
			}
			out := []map[string]any{}
			for _, g := range list {
				out = append(out, goalOf(g))
			}
			return out, nil
		},
	},
	"get_goal": {
		Name:        "get_goal",
		Description: "Read one goal: outcome, measure, milestones, and the tasks linked to it.",
		Properties:  map[string]any{"id": idField},
		Required:    []string{"id"},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				ID string `json:"id"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			id, err := parseID(in.ID, "id")
			if err != nil {
				return nil, err
			}
			g, err := goals.Get(ctx, q, userID, id)
			if err != nil {
				return nil, notFound(err, "goal")
			}
			ex, err := goals.Explain(ctx, q, userID, g)
			if err != nil {
				return nil, err
			}
			out := goalOf(g)
			out["outcome"] = g.Outcome
			out["progress_inputs"] = ex.Items
			return out, nil
		},
	},
	"add_milestone": {
		Name:        "add_milestone",
		Description: "Propose a milestone for a goal. Nothing is added until the person approves. Set due_on only if the person gave a date.",
		Properties:  map[string]any{"goal_id": idField, "title": str, "due_on": date, "reason": reason},
		Required:    []string{"goal_id", "title", "reason"},
		Write:       true,
		Summary: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (string, error) {
			var in milestoneInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", err
			}
			id, _ := uuid.Parse(in.GoalID)
			g, err := goals.Get(ctx, q, userID, id)
			if err != nil {
				return "", notFound(err, "goal")
			}
			s := fmt.Sprintf("Add milestone “%s” to goal “%s”", in.Title, g.Title)
			if in.DueOn != nil {
				s += ", due " + *in.DueOn
			}
			return s, nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in milestoneInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			id, err := parseID(in.GoalID, "goal_id")
			if err != nil {
				return err
			}
			due, err := parseDate(in.DueOn, "due_on")
			if err != nil {
				return err
			}
			_, err = goals.AddMilestone(ctx, tx, userID, id, actor, goals.MilestoneFields{Title: &in.Title, DueOn: due})
			return notFound(err, "goal")
		},
	},
	"link_task_to_goal": {
		Name:        "link_task_to_goal",
		Description: "Propose linking a task to a goal it moves forward. Nothing changes until the person approves.",
		Properties:  map[string]any{"task_id": idField, "goal_id": idField, "reason": reason},
		Required:    []string{"task_id", "goal_id", "reason"},
		Write:       true,
		Summary: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (string, error) {
			var in linkInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", err
			}
			tid, _ := uuid.Parse(in.TaskID)
			gid, _ := uuid.Parse(in.GoalID)
			t, err := tasks.Get(ctx, q, userID, tid)
			if err != nil {
				return "", notFound(err, "task")
			}
			g, err := goals.Get(ctx, q, userID, gid)
			if err != nil {
				return "", notFound(err, "goal")
			}
			return fmt.Sprintf("Link task “%s” to goal “%s”", t.Title, g.Title), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in linkInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			tid, err := parseID(in.TaskID, "task_id")
			if err != nil {
				return err
			}
			gid, err := parseID(in.GoalID, "goal_id")
			if err != nil {
				return err
			}
			if _, err := tasks.Get(ctx, tx, userID, tid); err != nil {
				return notFound(err, "task")
			}
			if _, err := goals.Get(ctx, tx, userID, gid); err != nil {
				return notFound(err, "goal")
			}
			_, err = graph.CreateLink(ctx, tx, userID, actor, tid, goals.Supports, gid)
			if errors.Is(err, graph.ErrDuplicate) {
				return toolErrorf("that task is already linked to the goal")
			}
			return err
		},
	},
	"remember": {
		Name:        "remember",
		Description: "Propose remembering a stable preference the person stated, for future runs. Needs approval; the person can delete it any time.",
		Properties:  map[string]any{"fact": map[string]any{"type": "string", "maxLength": 500}, "reason": reason},
		Required:    []string{"fact"},
		Write:       true,
		Summary: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (string, error) {
			var in memoryInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", err
			}
			return fmt.Sprintf("Remember: “%s”", in.Fact), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in memoryInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			in.Fact = strings.TrimSpace(in.Fact)
			if in.Fact == "" || len(in.Fact) > 500 {
				return toolErrorf("a memory is 1 to 500 characters")
			}
			_, err := tx.Exec(ctx, `INSERT INTO agent_memories (id, user_id, agent, content) VALUES ($1, $2, $3, $4)`,
				uuid.Must(uuid.NewV7()), userID, strings.TrimPrefix(string(actor), "agent:"), in.Fact)
			return err
		},
	},
}

type milestoneInput struct {
	GoalID string  `json:"goal_id"`
	Title  string  `json:"title"`
	DueOn  *string `json:"due_on"`
	Reason string  `json:"reason"`
}

type linkInput struct {
	TaskID string `json:"task_id"`
	GoalID string `json:"goal_id"`
	Reason string `json:"reason"`
}

type memoryInput struct {
	Fact   string `json:"fact"`
	Reason string `json:"reason"`
}

func goalOf(g goals.Goal) map[string]any {
	ms := []map[string]any{}
	for _, m := range g.Milestones {
		ms = append(ms, map[string]any{"id": m.ID, "title": m.Title, "due_on": m.DueOn, "done": m.DoneAt != nil})
	}
	return map[string]any{
		"id": g.ID, "title": g.Title, "target_date": g.TargetDate, "measure": g.Measure,
		"progress": g.Progress.Summary, "milestones": ms,
	}
}

// inputReason extracts the "reason" every write tool carries.
func inputReason(raw json.RawMessage) string {
	var in struct {
		Reason string `json:"reason"`
	}
	_ = json.Unmarshal(raw, &in)
	if len(in.Reason) > 1000 {
		return in.Reason[:1000]
	}
	return in.Reason
}
