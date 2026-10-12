package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/projects"
	"github.com/thoutmose/axon/api/internal/repos"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Repository tools for the delivery agents (EP-26). All of them read, except
// draft_issue (a card on the person's own board) and post_review_comment
// (a plain comment). Both are proposals the person approves. No tool can
// push, approve, or merge: that capability does not exist in Axon.

func openRepo(ctx context.Context, q db.Querier, userID uuid.UUID, id string) (*repos.Client, error) {
	rid, err := parseID(id, "repo_id")
	if err != nil {
		return nil, err
	}
	c, err := repos.Open(ctx, q, repos.SealerFrom(ctx), userID, rid)
	if errors.Is(err, graph.ErrNotFound) {
		return nil, toolErrorf("no connected repository with that id; call list_repos")
	}
	return c, err
}

// gitHubError lets the model see that GitHub failed, without its body.
func gitHubError(err error) error {
	if errors.Is(err, repos.ErrGitHub) {
		return toolErrorf("GitHub did not answer (%s); the path or number may be wrong", strings.TrimPrefix(err.Error(), repos.ErrGitHub.Error()+": "))
	}
	return err
}

var repoID = map[string]any{"type": "string", "description": "A repository id from list_repos."}

type draftInput struct {
	RepoID  string `json:"repo_id"`
	Title   string `json:"title"`
	Details string `json:"details"`
	Kind    string `json:"kind"`
	Reason  string `json:"reason"`
}

type commentInput struct {
	RepoID string `json:"repo_id"`
	Number int    `json:"number"`
	Body   string `json:"body"`
	Reason string `json:"reason"`
}

func (in commentInput) check() error {
	switch {
	case in.Number <= 0:
		return toolErrorf("number is the pull request number")
	case strings.TrimSpace(in.Body) == "" || len(in.Body) > 8000:
		return toolErrorf("a comment is 1 to 8,000 characters")
	}
	return nil
}

func init() {
	registry["list_repos"] = Tool{
		Name:        "list_repos",
		Description: "List the repositories the person connected, with their ids and whether comments are allowed.",
		Properties:  map[string]any{},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			if err := decode(raw, &struct{}{}); err != nil {
				return nil, err
			}
			return repos.List(ctx, q, userID)
		},
	}
	registry["repo_tree"] = Tool{
		Name:        "repo_tree",
		Description: "List file paths on the default branch (up to 2,000), optionally under a folder prefix.",
		Properties:  map[string]any{"repo_id": repoID, "prefix": map[string]any{"type": "string"}},
		Required:    []string{"repo_id"},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				RepoID string `json:"repo_id"`
				Prefix string `json:"prefix"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			c, err := openRepo(ctx, q, userID, in.RepoID)
			if err != nil {
				return nil, err
			}
			paths, err := c.Tree(ctx, in.Prefix)
			return paths, gitHubError(err)
		},
	}
	registry["read_repo_file"] = Tool{
		Name:        "read_repo_file",
		Description: "Read a file on the default branch (first 100 KB).",
		Properties:  map[string]any{"repo_id": repoID, "path": map[string]any{"type": "string"}},
		Required:    []string{"repo_id", "path"},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				RepoID string `json:"repo_id"`
				Path   string `json:"path"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			c, err := openRepo(ctx, q, userID, in.RepoID)
			if err != nil {
				return nil, err
			}
			text, err := c.File(ctx, in.Path)
			return text, gitHubError(err)
		},
	}
	registry["list_pull_requests"] = Tool{
		Name:        "list_pull_requests",
		Description: "List open pull requests (up to 30).",
		Properties:  map[string]any{"repo_id": repoID},
		Required:    []string{"repo_id"},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				RepoID string `json:"repo_id"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			c, err := openRepo(ctx, q, userID, in.RepoID)
			if err != nil {
				return nil, err
			}
			prs, err := c.PullRequests(ctx)
			return prs, gitHubError(err)
		},
	}
	registry["read_pull_request_diff"] = Tool{
		Name:        "read_pull_request_diff",
		Description: "Read a pull request's unified diff (first 200 KB).",
		Properties:  map[string]any{"repo_id": repoID, "number": map[string]any{"type": "integer", "minimum": 1}},
		Required:    []string{"repo_id", "number"},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				RepoID string `json:"repo_id"`
				Number int    `json:"number"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			c, err := openRepo(ctx, q, userID, in.RepoID)
			if err != nil {
				return nil, err
			}
			diff, err := c.Diff(ctx, in.Number)
			return diff, gitHubError(err)
		},
	}
	registry["draft_issue"] = Tool{
		Name:        "draft_issue",
		Description: "Propose an issue for the repository's project board. Nothing is created until the person approves.",
		Properties: map[string]any{
			"repo_id": repoID, "title": str,
			"details": map[string]any{"type": "string", "description": "What and why, with the files involved."},
			"kind":    map[string]any{"type": "string", "enum": []string{"issue", "epic"}},
			"reason":  reason,
		},
		Required: []string{"repo_id", "title", "details", "reason"},
		Write:    true,
		Summary: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (string, error) {
			var in draftInput
			_ = json.Unmarshal(raw, &in)
			kind := in.Kind
			if kind == "" {
				kind = "issue"
			}
			return fmt.Sprintf("Add %s “%s” to the backlog", kind, in.Title), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in draftInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			c, err := openRepo(ctx, tx, userID, in.RepoID)
			if err != nil {
				return err
			}
			project := c.Connection().ProjectID
			if project == nil {
				return toolErrorf("connect this repository to a project first, so issues have a board")
			}
			if in.Kind == "" {
				in.Kind = "issue"
			}
			if in.Kind != "issue" && in.Kind != "epic" {
				return toolErrorf("kind is issue or epic")
			}
			status := "someday"
			notes := strings.TrimSpace(in.Details) + fmt.Sprintf("\n\nRepository: %s/%s", c.Connection().Owner, c.Connection().Repo)
			t, err := tasks.Create(ctx, tx, userID, actor, tasks.Fields{Title: &in.Title, Notes: &notes, Status: &status})
			if err != nil {
				return notFound(err, "task")
			}
			col := "backlog"
			_, err = projects.Place(ctx, tx, userID, *project, actor, t.ID, projects.CardInput{Column: &col, Kind: &in.Kind})
			var v projects.ValidationError
			if errors.As(err, &v) {
				return toolError{v.Msg}
			}
			return err
		},
	}
	registry["post_review_comment"] = Tool{
		Name:        "post_review_comment",
		Description: "Propose a comment on a pull request. It is posted only after the person approves, and only if they allowed comments on this repository. Comments cannot approve or merge.",
		Properties: map[string]any{
			"repo_id": repoID, "number": map[string]any{"type": "integer", "minimum": 1},
			"body":   map[string]any{"type": "string", "description": "Markdown review: what to change and why, with file and line references."},
			"reason": reason,
		},
		Required: []string{"repo_id", "number", "body"},
		Write:    true,
		Summary: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (string, error) {
			var in commentInput
			_ = json.Unmarshal(raw, &in)
			c, err := openRepo(ctx, q, userID, in.RepoID)
			if err != nil {
				return "", err
			}
			body := in.Body
			if len([]rune(body)) > 600 {
				body = string([]rune(body)[:600]) + "…"
			}
			return fmt.Sprintf("Comment on %s/%s#%d:\n\n%s", c.Connection().Owner, c.Connection().Repo, in.Number, body), nil
		},
		Check: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) error {
			var in commentInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			if err := in.check(); err != nil {
				return err
			}
			c, err := openRepo(ctx, q, userID, in.RepoID)
			if err != nil {
				return err
			}
			if !c.Connection().AllowComments {
				return toolErrorf("comments are turned off for this repository; put your review in your reply instead")
			}
			return nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, _ graph.Actor, raw json.RawMessage) error {
			var in commentInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			if err := in.check(); err != nil {
				return err
			}
			c, err := openRepo(ctx, tx, userID, in.RepoID)
			if err != nil {
				return err
			}
			body := strings.TrimSpace(in.Body) + "\n\n<sub>Drafted by Axon's Code Review agent and approved by the repository owner.</sub>"
			err = c.Comment(ctx, in.Number, body)
			switch {
			case errors.Is(err, repos.ErrCommentsOff):
				return toolErrorf("comments are turned off for this repository")
			case errors.Is(err, repos.ErrGitHub):
				return toolErrorf("GitHub refused the comment (%v)", err)
			}
			return err
		},
	}
}
