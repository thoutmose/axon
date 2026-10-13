package agents

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/l10n"
	"github.com/thoutmose/axon/api/internal/repos"
)

// Routes mounts what scripts may also use; the caller wraps them in auth.Require.
func (r *Runner) Routes(m chi.Router) {
	m.Get("/v1/agents", r.list)
	m.Post("/v1/agents/{name}/runs", r.start)
	m.Get("/v1/agent-runs", r.runs)
	m.Get("/v1/agent-runs/{id}", r.getRun)
	m.Post("/v1/agent-runs/{id}/continue", r.continueRun)
	m.Get("/v1/proposals", r.proposals)
	m.Get("/v1/agent-memories", r.memories)
}

// UploadRoutes mounts the meal photo estimate, which needs a larger body
// limit than JSON (EP-89).
func (r *Runner) UploadRoutes(m chi.Router) {
	m.Post("/v1/health/food/estimate", r.estimate)
}

// SessionRoutes mount decisions only a person at a browser may take:
// approving changes, budgets, and forgetting. The caller wraps them in
// auth.RequireSession, so no token (or agent) can approve a proposal.
func (r *Runner) SessionRoutes(m chi.Router) {
	m.Put("/v1/me/agent-settings", r.settings)
	m.Get("/v1/me/agent-provider", r.getProvider)
	m.With(auth.NotDemo).Put("/v1/me/agent-provider", r.putProvider)
	m.With(auth.NotDemo).Delete("/v1/me/agent-provider", r.deleteProvider)
	m.With(auth.NotDemo).Post("/v1/me/agent-provider/test", r.testProvider)
	m.Post("/v1/proposals/{id}/approve", r.approve)
	m.Post("/v1/proposals/{id}/reject", r.reject)
	m.Delete("/v1/agent-memories/{id}", r.forget)
}

func (r *Runner) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnknown):
		httpx.Problem(w, http.StatusNotFound, "No such agent.")
	case errors.Is(err, ErrUnavailable):
		httpx.Problem(w, http.StatusServiceUnavailable, "Agents are not set up on this server. Everything else works without them.")
	case errors.As(err, new(NeedsConsent)):
		var nc NeedsConsent
		errors.As(err, &nc)
		httpx.Problem(w, http.StatusForbidden, "This agent needs your “"+nc.Purpose+"” consent. Turn it on under Account → Privacy, where it says what is shared.")
	case errors.Is(err, ErrNoConsent):
		httpx.Problem(w, http.StatusForbidden, "Agents are off. Turn on “AI agents” under Account → Privacy to use them.")
	case errors.Is(err, ErrNoBudget):
		httpx.Problem(w, http.StatusPaymentRequired, "This month's agent budget is used up. Raise it in agent settings or wait until next month.")
	case errors.Is(err, ErrExpired):
		httpx.Problem(w, http.StatusGone, "This conversation is no longer held (replies are kept for an hour). Start a new run.")
	case errors.Is(err, ErrBusy):
		httpx.Problem(w, http.StatusConflict, "An agent is already working for you. Wait for it to finish.")
	default:
		r.log.Error("agents request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

type toolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Write       bool   `json:"write"`
}

func (r *Runner) list(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	st, err := r.status(req.Context(), r.db, who.UserID)
	if err != nil {
		r.fail(w, err)
		return
	}
	type agentInfo struct {
		Skill
		Tools []toolInfo `json:"tools"`
	}
	names := make([]string, 0, len(r.Skills))
	for n := range r.Skills {
		names = append(names, n)
	}
	slices.Sort(names)
	agents := []agentInfo{}
	for _, n := range names {
		s := r.Skills[n].In(l10n.Lang(req.Context()))
		a := agentInfo{Skill: s, Tools: []toolInfo{}}
		for _, t := range s.Tools {
			a.Tools = append(a.Tools, toolInfo{t, registry[t].Description, registry[t].Write})
		}
		agents = append(agents, a)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": st, "agents": agents})
}

func (r *Runner) settings(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	var body struct {
		MonthlyBudgetCents int `json:"monthly_budget_cents"`
	}
	if err := httpx.Decode(req, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var ie InputError
	if err := SetBudget(req.Context(), r.db, who.UserID, body.MonthlyBudgetCents); errors.As(err, &ie) {
		httpx.Problem(w, http.StatusUnprocessableEntity, ie.Msg)
		return
	} else if err != nil {
		r.fail(w, err)
		return
	}
	st, err := r.status(req.Context(), r.db, who.UserID)
	if err != nil {
		r.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

// instruction reads and checks the person's message to an agent.
func instruction(w http.ResponseWriter, req *http.Request) (string, bool) {
	var body struct {
		Instruction string `json:"instruction"`
	}
	if err := httpx.Decode(req, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return "", false
	}
	body.Instruction = strings.TrimSpace(body.Instruction)
	if body.Instruction == "" || utf8.RuneCountInString(body.Instruction) > 4000 {
		httpx.Problem(w, http.StatusUnprocessableEntity, "Tell the agent what to do in 1 to 4,000 characters.")
		return "", false
	}
	return body.Instruction, true
}

func (r *Runner) start(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	text, ok := instruction(w, req)
	if !ok {
		return
	}
	r.started(w, req, who.UserID)(r.Start(req.Context(), who.UserID, chi.URLParam(req, "name"), text))
}

func (r *Runner) continueRun(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	parent, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such run.")
		return
	}
	text, ok := instruction(w, req)
	if !ok {
		return
	}
	if _, err := r.record(req, who.UserID, parent); errors.Is(err, graph.ErrNotFound) {
		httpx.Problem(w, http.StatusNotFound, "No such run.")
		return
	} else if err != nil {
		r.fail(w, err)
		return
	}
	r.started(w, req, who.UserID)(r.Continue(req.Context(), who.UserID, parent, text))
}

// started answers a request that started a run.
func (r *Runner) started(w http.ResponseWriter, req *http.Request, userID uuid.UUID) func(uuid.UUID, error) {
	return func(id uuid.UUID, err error) {
		if err != nil {
			r.fail(w, err)
			return
		}
		rec, err := r.record(req, userID, id)
		if err != nil {
			r.fail(w, err)
			return
		}
		httpx.JSON(w, http.StatusAccepted, rec)
	}
}

// RunRecord is the audit record of a run: no prompt or reply text.
type RunRecord struct {
	ID           uuid.UUID  `json:"id"`
	Agent        string     `json:"agent"`
	SkillVersion string     `json:"skill_version"`
	Model        string     `json:"model"`
	Status       string     `json:"status"`
	ToolsCalled  []string   `json:"tools_called"`
	Proposals    int        `json:"proposals"`
	InputTokens  int64      `json:"input_tokens"`
	OutputTokens int64      `json:"output_tokens"`
	CostMicros   int64      `json:"cost_micros"`
	Error        *string    `json:"error"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	ParentRunID  *uuid.UUID `json:"parent_run_id"`
}

const runColumns = `id, agent, skill_version, model, status, tools_called, proposals, input_tokens, output_tokens, cost_micros, error, started_at, finished_at, parent_run_id`

func (r *Runner) record(req *http.Request, userID, id uuid.UUID) (RunRecord, error) {
	rows, err := r.db.Query(req.Context(), `SELECT `+runColumns+` FROM agent_runs WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return RunRecord{}, err
	}
	rec, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[RunRecord])
	if errors.Is(err, pgx.ErrNoRows) {
		return rec, graph.ErrNotFound
	}
	return rec, err
}

func (r *Runner) runs(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	rows, err := r.db.Query(req.Context(), `SELECT `+runColumns+` FROM agent_runs WHERE user_id = $1 ORDER BY started_at DESC LIMIT 50`, who.UserID)
	if err != nil {
		r.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[RunRecord])
	if err != nil {
		r.fail(w, err)
		return
	}
	if items == nil {
		items = []RunRecord{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (r *Runner) getRun(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	id, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such run.")
		return
	}
	rec, err := r.record(req, who.UserID, id)
	if errors.Is(err, graph.ErrNotFound) {
		httpx.Problem(w, http.StatusNotFound, "No such run.")
		return
	}
	if err != nil {
		r.fail(w, err)
		return
	}
	props, err := r.listProposals(req, who.UserID, "", &id)
	if err != nil {
		r.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"run": rec, "reply": r.replies.get(id, who.UserID), "proposals": props,
		"can_continue": r.replies.conversation(id, who.UserID) != nil && rec.Status == "completed"})
}

// Proposal is a change an agent wants, waiting for the user's decision.
type Proposal struct {
	ID        uuid.UUID       `json:"id"`
	RunID     uuid.UUID       `json:"run_id"`
	Agent     string          `json:"agent"`
	Tool      string          `json:"tool"`
	Input     json.RawMessage `json:"input"`
	Summary   string          `json:"summary"`
	Reason    string          `json:"reason"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	DecidedAt *time.Time      `json:"decided_at"`
}

const proposalColumns = `id, run_id, agent, tool, input, summary, reason, status, created_at, decided_at`

func (r *Runner) listProposals(req *http.Request, userID uuid.UUID, status string, runID *uuid.UUID) ([]Proposal, error) {
	var st *string
	if status != "" {
		st = &status
	}
	rows, err := r.db.Query(req.Context(), `SELECT `+proposalColumns+` FROM agent_proposals
		WHERE user_id = $1 AND ($2::text IS NULL OR status = $2) AND ($3::uuid IS NULL OR run_id = $3)
		ORDER BY created_at DESC LIMIT 100`, userID, st, runID)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Proposal])
	if items == nil {
		items = []Proposal{}
	}
	return items, err
}

func (r *Runner) proposals(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	status := req.URL.Query().Get("status")
	if status != "" && !slices.Contains([]string{"pending", "approved", "rejected", "failed"}, status) {
		httpx.Problem(w, http.StatusBadRequest, "?status= is pending, approved, rejected, or failed.")
		return
	}
	items, err := r.listProposals(req, who.UserID, status, nil)
	if err != nil {
		r.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

var errDecided = errors.New("already decided")

// decide locks a pending proposal and runs f on it in one transaction.
func (r *Runner) decide(w http.ResponseWriter, req *http.Request, f func(tx pgx.Tx, p Proposal) (string, error)) {
	who, _ := auth.FromContext(req.Context())
	id, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such proposal.")
		return
	}
	var out Proposal
	var applyErr error
	err = pgx.BeginFunc(req.Context(), r.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(req.Context(), `SELECT `+proposalColumns+` FROM agent_proposals WHERE id = $1 AND user_id = $2 FOR UPDATE`, id, who.UserID)
		if err != nil {
			return err
		}
		p, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Proposal])
		if errors.Is(err, pgx.ErrNoRows) {
			return graph.ErrNotFound
		}
		if err != nil {
			return err
		}
		if p.Status != "pending" {
			return errDecided
		}
		// Apply in a savepoint: if the change fails, the proposal is still marked failed.
		sp, err := tx.Begin(req.Context())
		if err != nil {
			return err
		}
		status, err := f(sp, p)
		var te toolError
		if errors.As(err, &te) {
			_ = sp.Rollback(req.Context())
			status, applyErr = "failed", te
		} else if err != nil {
			return err
		} else if err := sp.Commit(req.Context()); err != nil {
			return err
		}
		rows, err = tx.Query(req.Context(), `UPDATE agent_proposals SET status = $2, decided_at = now() WHERE id = $1 RETURNING `+proposalColumns, id, status)
		if err != nil {
			return err
		}
		out, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Proposal])
		return err
	})
	switch {
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such proposal.")
	case errors.Is(err, errDecided):
		httpx.Problem(w, http.StatusConflict, "This proposal was already decided.")
	case err != nil:
		r.fail(w, err)
	case applyErr != nil:
		httpx.Problem(w, http.StatusUnprocessableEntity, "Could not apply this proposal: "+applyErr.Error()+". It was marked failed; nothing changed.")
	default:
		if out.Status == "approved" && r.OnApproved != nil {
			r.OnApproved(who.UserID, out.Tool)
		}
		httpx.JSON(w, http.StatusOK, out)
	}
}

func (r *Runner) approve(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	r.decide(w, req, func(tx pgx.Tx, p Proposal) (string, error) {
		t, ok := registry[p.Tool]
		if !ok || !t.Write {
			return "", toolErrorf("this kind of change is no longer supported")
		}
		ctx := repos.WithSealer(req.Context(), r.sealer)
		return "approved", t.Apply(ctx, tx, who.UserID, graph.Actor("agent:"+p.Agent), p.Input)
	})
}

func (r *Runner) reject(w http.ResponseWriter, req *http.Request) {
	r.decide(w, req, func(pgx.Tx, Proposal) (string, error) { return "rejected", nil })
}

type memory struct {
	ID        uuid.UUID `json:"id"`
	Agent     string    `json:"agent"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *Runner) memories(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	rows, err := r.db.Query(req.Context(), `SELECT id, agent, content, created_at FROM agent_memories WHERE user_id = $1 ORDER BY agent, created_at LIMIT 200`, who.UserID)
	if err != nil {
		r.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[memory])
	if err != nil {
		r.fail(w, err)
		return
	}
	if items == nil {
		items = []memory{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (r *Runner) forget(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	id, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such memory.")
		return
	}
	tag, err := r.db.Exec(req.Context(), `DELETE FROM agent_memories WHERE id = $1 AND user_id = $2`, id, who.UserID)
	if err != nil {
		r.fail(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Problem(w, http.StatusNotFound, "No such memory.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Export returns the user's run records, proposals, and memories.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+runColumns+` FROM agent_runs WHERE user_id = $1 ORDER BY started_at`, userID)
	if err != nil {
		return nil, err
	}
	runs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[RunRecord])
	if err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT `+proposalColumns+` FROM agent_proposals WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	props, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Proposal])
	if err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT id, agent, content, created_at FROM agent_memories WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	mems, err := pgx.CollectRows(rows, pgx.RowToStructByPos[memory])
	if err != nil {
		return nil, err
	}
	return map[string]any{"runs": runs, "proposals": props, "memories": mems}, nil
}

// WriteTools names every tool that changes data, for tests that must cover them all.
func WriteTools() []string {
	var out []string
	for name, t := range registry {
		if t.Write {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

func (r *Runner) getProvider(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	s, _, err := r.providerSettings(req.Context(), r.db, who.UserID)
	if err != nil {
		r.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"settings": s, "server_default": r.provider != nil, "can_store_keys": r.sealer != nil, "providers": providerNames(),
	})
}

func (r *Runner) putProvider(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	var in ProviderInput
	if err := httpx.Decode(req, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var ie InputError
	if err := r.SetProvider(req.Context(), r.db, who.UserID, in); errors.As(err, &ie) {
		httpx.Problem(w, http.StatusUnprocessableEntity, ie.Msg)
		return
	} else if err != nil {
		r.fail(w, err)
		return
	}
	r.getProvider(w, req)
}

func (r *Runner) deleteProvider(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	if _, err := r.db.Exec(req.Context(), `DELETE FROM agent_providers WHERE user_id = $1`, who.UserID); err != nil {
		r.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testProvider sends one tiny request with no personal data.
func (r *Runner) testProvider(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	res, err := r.resolve(req.Context(), r.db, who.UserID, Skill{Model: "claude-opus-5-5"})
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "detail": "No provider is set up yet."})
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 12*time.Second)
	defer cancel()
	_, err = res.provider.New(ctx, testParams(res.model))
	if err != nil {
		r.log.Info("agent provider test failed", "provider", res.name, "err", err)
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "detail": "The provider did not answer: check the model name, key, and address."})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "detail": "Connected to " + res.name + " (" + res.model + ")."})
}
