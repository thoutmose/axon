package agents

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/watch"
)

// The podcast as the hosts' conversation (owner request), written by the
// person's model from the episode's stories (watch.DialoguePrompt). No tool
// is offered, only JSON is read back (watch.ParseDialogue); the call is
// gated and recorded like an agent run, within the monthly budget.

const (
	podcastModel     = "claude-sonnet-5-5" // writes well and fast enough to wait for, as Jarvis
	podcastMaxTokens = 16000
)

// WriteDialogue writes what the hosts say about the episode's stories.
func (r *Runner) WriteDialogue(ctx context.Context, userID uuid.UUID, ep watch.Episode, hosts int) ([]watch.Turn, error) {
	st, err := r.status(ctx, r.db, userID)
	if err != nil {
		return nil, err
	}
	if !st.Available {
		return nil, errors.New("no model available now")
	}
	res, err := r.resolve(ctx, r.db, userID, Skill{Name: "podcast-script", Model: podcastModel})
	if err != nil {
		return nil, err
	}
	input := watch.DialogueInput(ep)
	if worstCaseMicros(res.price, int64(len(input)/3+1500), podcastMaxTokens) > int64(st.BudgetCents)*10_000-st.SpentMicros {
		return nil, errors.New("over the monthly budget")
	}
	minutes := max(3, int(ep.Minutes+0.5))
	params := anthropic.BetaMessageNewParams{
		Model:     res.model,
		MaxTokens: podcastMaxTokens,
		System:    []anthropic.BetaTextBlockParam{{Text: watch.DialoguePrompt(ep, hosts, minutes)}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(input))},
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	resp, err := res.provider.New(cctx, params)
	status, cost := "completed", int64(0)
	var in, out int64
	var turns []watch.Turn
	switch {
	case err != nil:
		status = "failed"
	case resp.StopReason == anthropic.BetaStopReasonRefusal:
		status, err = "refused", watch.ErrNotDialogue
	default:
		cost = costMicros(res.price, resp.Usage)
		in, out = resp.Usage.InputTokens+resp.Usage.CacheReadInputTokens+resp.Usage.CacheCreationInputTokens, resp.Usage.OutputTokens
		var text strings.Builder
		for _, b := range resp.Content {
			text.WriteString(b.Text)
		}
		if turns, err = watch.ParseDialogue(text.String(), ep, hosts); err != nil {
			status = "refused"
		}
	}
	if _, rerr := r.db.Exec(context.WithoutCancel(ctx), `INSERT INTO agent_runs (id, user_id, agent, skill_version, model, status,
		input_tokens, output_tokens, cost_micros, finished_at) VALUES ($1, $2, 'podcast-script', '1.0.0', $3, $4, $5, $6, $7, now())`,
		uuid.Must(uuid.NewV7()), userID, res.name+"/"+res.model, status, in, out, cost); rerr != nil {
		r.log.Error("recording a podcast script failed", "err", rerr)
	}
	return turns, err
}
