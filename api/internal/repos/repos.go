// Package repos connects GitHub repositories for the delivery agents
// (EP-26). Access is read-only: the client has no way to push, approve, or
// merge. The single write is a plain pull-request comment, only when the
// person turned comments on for that repository and approved the comment.
package repos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// BaseURL is GitHub's API; tests point it at a fake server.
var BaseURL = "https://api.github.com"

// Errors callers translate.
var (
	ErrCommentsOff = errors.New("comments are turned off for this repository")
	ErrGitHub      = errors.New("GitHub did not answer as expected")
)

// Connection is a repository the person connected.
type Connection struct {
	ID            uuid.UUID  `json:"id"`
	Owner         string     `json:"owner"`
	Repo          string     `json:"repo"`
	HasToken      bool       `json:"has_token"`
	AllowComments bool       `json:"allow_comments"`
	ProjectID     *uuid.UUID `json:"project_id"`
	CreatedAt     time.Time  `json:"created_at"`
}

const columns = `id, owner, repo, token_sealed IS NOT NULL, allow_comments, project_id, created_at`

func scan(row pgx.Row) (Connection, error) {
	var c Connection
	err := row.Scan(&c.ID, &c.Owner, &c.Repo, &c.HasToken, &c.AllowComments, &c.ProjectID, &c.CreatedAt)
	return c, err
}

// List returns the user's connections.
func List(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Connection, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM repo_connections WHERE user_id = $1 ORDER BY owner, repo LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Connection, error) { return scan(r) })
	if out == nil {
		out = []Connection{}
	}
	return out, err
}

// Get returns one of the user's connections.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Connection, error) {
	c, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM repo_connections WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, graph.ErrNotFound
	}
	return c, err
}

// Client reads one repository with the connection's token (if any).
type Client struct {
	conn  Connection
	token string
	http  *http.Client
}

// Open returns a client for one of the user's connections.
func Open(ctx context.Context, q db.Querier, sealer *auth.Sealer, userID, id uuid.UUID) (*Client, error) {
	c, err := Get(ctx, q, userID, id)
	if err != nil {
		return nil, err
	}
	var sealed []byte
	if err := q.QueryRow(ctx, `SELECT token_sealed FROM repo_connections WHERE id = $1`, id).Scan(&sealed); err != nil {
		return nil, err
	}
	cl := &Client{conn: c, http: &http.Client{Timeout: 20 * time.Second}}
	if len(sealed) > 0 {
		if sealer == nil {
			return nil, errors.New("cannot open the stored token without AXON_ENCRYPTION_KEY")
		}
		b, err := sealer.Open(sealed)
		if err != nil {
			return nil, err
		}
		cl.token = string(b)
	}
	return cl, nil
}

// Connection describes the repository this client reads.
func (c *Client) Connection() Connection { return c.conn }

func (c *Client) do(ctx context.Context, method, path, accept string, body any, limit int64, out any) ([]byte, error) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, BaseURL+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHub, err)
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: HTTP %d", ErrGitHub, res.StatusCode)
	}
	if int64(len(b)) > limit {
		b = b[:limit]
	}
	if out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			return nil, fmt.Errorf("%w: unreadable reply", ErrGitHub)
		}
	}
	return b, nil
}

func (c *Client) repoPath() string {
	return "/repos/" + url.PathEscape(c.conn.Owner) + "/" + url.PathEscape(c.conn.Repo)
}

// Tree lists file paths on the default branch (up to 2,000), optionally under a prefix.
func (c *Client) Tree(ctx context.Context, prefix string) ([]string, error) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if _, err := c.do(ctx, http.MethodGet, c.repoPath(), "application/vnd.github+json", nil, 1<<20, &repo); err != nil {
		return nil, err
	}
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if _, err := c.do(ctx, http.MethodGet, c.repoPath()+"/git/trees/"+url.PathEscape(repo.DefaultBranch)+"?recursive=1",
		"application/vnd.github+json", nil, 8<<20, &tree); err != nil {
		return nil, err
	}
	out := []string{}
	for _, t := range tree.Tree {
		if t.Type == "blob" && strings.HasPrefix(t.Path, prefix) && len(out) < 2000 {
			out = append(out, t.Path)
		}
	}
	return out, nil
}

// File returns up to 100 KB of a file on the default branch.
func (c *Client) File(ctx context.Context, path string) (string, error) {
	if strings.Contains(path, "..") || strings.HasPrefix(path, "/") || path == "" {
		return "", errors.New("give a path inside the repository")
	}
	parts := strings.Split(path, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	b, err := c.do(ctx, http.MethodGet, c.repoPath()+"/contents/"+strings.Join(parts, "/"), "application/vnd.github.raw", nil, 100<<10, nil)
	return string(b), err
}

// PullRequest is an open pull request.
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

// PullRequests lists up to 30 open pull requests.
func (c *Client) PullRequests(ctx context.Context) ([]PullRequest, error) {
	var raw []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if _, err := c.do(ctx, http.MethodGet, c.repoPath()+"/pulls?state=open&per_page=30", "application/vnd.github+json", nil, 2<<20, &raw); err != nil {
		return nil, err
	}
	out := []PullRequest{}
	for _, p := range raw {
		out = append(out, PullRequest{p.Number, p.Title, p.User.Login})
	}
	return out, nil
}

// Diff returns up to 200 KB of a pull request's unified diff.
func (c *Client) Diff(ctx context.Context, number int) (string, error) {
	b, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/pulls/%d", c.repoPath(), number), "application/vnd.github.diff", nil, 200<<10, nil)
	return string(b), err
}

// Comment posts a plain comment on a pull request. It is the client's only
// write, and it is refused unless comments are on for this repository.
// There is deliberately no review, approval, or merge call.
func (c *Client) Comment(ctx context.Context, number int, body string) error {
	if !c.conn.AllowComments {
		return ErrCommentsOff
	}
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/issues/%d/comments", c.repoPath(), number), "application/vnd.github+json",
		map[string]string{"body": body}, 1<<20, nil)
	return err
}

type sealerKey struct{}

// WithSealer gives code that opens connections (agent tools) the sealer.
func WithSealer(ctx context.Context, s *auth.Sealer) context.Context {
	return context.WithValue(ctx, sealerKey{}, s)
}

// SealerFrom returns the sealer set by WithSealer, or nil.
func SealerFrom(ctx context.Context) *auth.Sealer {
	s, _ := ctx.Value(sealerKey{}).(*auth.Sealer)
	return s
}
