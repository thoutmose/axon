package agents

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// The model each agent uses (EP-151). The provider is the person's (or the
// server's); a model chosen for all agents ('*') replaces the provider's,
// and one chosen for an agent replaces both. Prices follow the model, the
// dearest known one when it is not known, so budgets err on the safe side.

// AllAgents names the model every agent uses unless it has its own.
const AllAgents = "*"

// Helpers that call a model outside an agent run, chosen like agents.
var helpers = []string{"food-photo", "video-summary", Jarvis}

// ModelChoice is one agent and the model it would use.
type ModelChoice struct {
	Agent   string `json:"agent"`
	Title   string `json:"title"`
	Default string `json:"default"` // the model without a choice of the person's
	Model   string `json:"model"`   // the person's choice, "" for none
}

// Models is what Preferences shows: the choice for all agents and each one.
type Models struct {
	All    string        `json:"all"`
	Agents []ModelChoice `json:"agents"`
}

func (r *Runner) models(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string]string, error) {
	rows, err := q.Query(ctx, `SELECT agent, model FROM agent_models WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	var agent, model string
	_, err = pgx.ForEachRow(rows, []any{&agent, &model}, func() error {
		out[agent] = model
		return nil
	})
	return out, err
}

// chosenModel is the person's model for an agent: its own, or all agents'.
func (r *Runner) chosenModel(ctx context.Context, q db.Querier, userID uuid.UUID, agent string) (string, error) {
	var model string
	err := q.QueryRow(ctx, `SELECT model FROM agent_models WHERE user_id = $1 AND agent = ANY($2)
		ORDER BY agent = '*' LIMIT 1`, userID, []string{agent, AllAgents}).Scan(&model)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return model, err
}

func (r *Runner) known(agent string) bool {
	_, ok := r.Skills[agent]
	return ok || agent == AllAgents || slices.Contains(helpers, agent)
}

// ListModels returns the person's choices with each agent's default.
func (r *Runner) ListModels(ctx context.Context, q db.Querier, userID uuid.UUID) (Models, error) {
	chosen, err := r.models(ctx, q, userID)
	if err != nil {
		return Models{}, err
	}
	s, _, err := r.providerSettings(ctx, q, userID)
	if err != nil {
		return Models{}, err
	}
	out := Models{All: chosen[AllAgents], Agents: []ModelChoice{}}
	names := make([]string, 0, len(r.Skills))
	for n := range r.Skills {
		names = append(names, n)
	}
	slices.Sort(names)
	defaults := map[string]string{"food-photo": photoModel, "video-summary": summaryModel, "jarvis": jarvisModel}
	titles := map[string]string{"food-photo": "Meal photos", "video-summary": "Video summaries", "jarvis": "Jarvis"}
	for _, h := range helpers {
		if _, ok := r.Skills[h]; !ok { // Jarvis is a skill too
			names = append(names, h)
		}
	}
	for _, n := range names {
		def, title := defaults[n], titles[n]
		if sk, ok := r.Skills[n]; ok {
			def, title = sk.Model, sk.Title
		}
		if s != nil { // the person's provider has one model for all
			def = s.Model
		}
		if out.All != "" {
			def = out.All
		}
		out.Agents = append(out.Agents, ModelChoice{Agent: n, Title: title, Default: def, Model: chosen[n]})
	}
	return out, nil
}

// SetModels replaces the person's choices; an empty model removes one.
func (r *Runner) SetModels(ctx context.Context, tx pgx.Tx, userID uuid.UUID, in map[string]string) error {
	for agent, model := range in {
		model = strings.TrimSpace(model)
		switch {
		case !r.known(agent):
			return InputError{"There is no agent called " + agent + "."}
		case model != "" && !validModel(model):
			return InputError{"Enter the model name your provider uses, like claude-sonnet-5-5 or llama3.1:8b."}
		}
		in[agent] = model
	}
	if _, err := tx.Exec(ctx, `DELETE FROM agent_models WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for agent, model := range in {
		if model == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO agent_models (user_id, agent, model) VALUES ($1, $2, $3)`, userID, agent, model); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) getModels(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	m, err := r.ListModels(req.Context(), r.db, who.UserID)
	if err != nil {
		r.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

func (r *Runner) putModels(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	var body struct {
		Models map[string]string `json:"models"`
	}
	if err := httpx.Decode(req, &body); err != nil || body.Models == nil || len(body.Models) > 60 {
		httpx.Problem(w, http.StatusBadRequest, "Send the models as {\"models\": {\"*\": \"…\", \"tutor\": \"…\"}}.")
		return
	}
	err := pgx.BeginFunc(req.Context(), r.db, func(tx pgx.Tx) error {
		return r.SetModels(req.Context(), tx, who.UserID, body.Models)
	})
	var ie InputError
	if errors.As(err, &ie) {
		httpx.Problem(w, http.StatusUnprocessableEntity, ie.Msg)
		return
	}
	if err != nil {
		r.fail(w, err)
		return
	}
	r.getModels(w, req)
}
