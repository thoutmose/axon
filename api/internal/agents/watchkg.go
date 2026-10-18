package agents

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/telemetry"
	"github.com/thoutmose/axon/api/internal/watch"
)

// The tech watch's knowledge graph read by the person's model (owner
// request): for people who turn it on, each tech watch item goes to their
// model once, which names its entities and the facts its text states
// (watch.KnowledgePrompt). The text is data, never instructions: no tool is
// offered, only JSON is read back, and a fact must quote the item
// (watch.ParseItemKG). Each call is gated and recorded like an agent run,
// within the monthly budget; general news stays out.

const (
	knowledgeModel     = "claude-haiku-5-5"
	knowledgeMaxTokens = 8000  // 40 entities and 40 facts, each quoting its sentence
	knowledgeChars     = 16000 // of an item's text: an article's start says what it states
	knowledgePerStep   = 10
)

// ExtractKnowledge reads up to n items of the people who asked for it, each
// person's newest first and in turn, so one whose budget or provider is not
// available now never holds back the others.
func (r *Runner) ExtractKnowledge(ctx context.Context, n int) error {
	rows, err := r.db.Query(ctx, `SELECT id, user_id, title, summary, words FROM (
		SELECT i.id, i.user_id, i.title, i.summary, left(i.content, $2) AS words, i.published_at,
			row_number() OVER (PARTITION BY i.user_id ORDER BY i.published_at DESC) AS rank
		FROM watch_items i JOIN agent_settings s ON s.user_id = i.user_id JOIN users u ON u.id = i.user_id
		JOIN watch_sources src ON src.id = i.source_id
		WHERE s.watch_knowledge AND NOT u.demo AND i.kg_at IS NULL AND NOT src.news AND i.outlier = '') t
		WHERE rank <= $1 ORDER BY rank, published_at DESC`, n, knowledgeChars)
	if err != nil {
		return err
	}
	type item struct {
		ID, User              uuid.UUID
		Title, Summary, Words string
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[item])
	if err != nil {
		return err
	}
	waiting := map[uuid.UUID]bool{}
	for _, it := range list {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if n == 0 {
			break
		}
		if waiting[it.User] {
			continue
		}
		article := it.Title + ".\n\n" + it.Summary + "\n\n" + it.Words
		tried, err := r.extractOne(ctx, it.ID, it.User, article)
		if err != nil {
			r.log.Warn("watch knowledge failed", "item", it.ID, "err", err)
		}
		if !tried || err != nil {
			waiting[it.User] = true // their other items wait for a later step
			continue
		}
		n--
	}
	return nil
}

// extractOne reads one item; tried is false when the person's budget,
// consent, or provider is not available now, and the item waits.
func (r *Runner) extractOne(ctx context.Context, id, userID uuid.UUID, article string) (tried bool, err error) {
	st, err := r.status(ctx, r.db, userID)
	if err != nil || !st.Available {
		return false, err
	}
	res, err := r.resolve(ctx, r.db, userID, Skill{Name: "watch-knowledge", Model: knowledgeModel})
	if err != nil {
		return false, err
	}
	if worstCaseMicros(res.price, int64(len(article)/3+1200), knowledgeMaxTokens) > int64(st.BudgetCents)*10_000-st.SpentMicros {
		return false, nil
	}
	params := anthropic.BetaMessageNewParams{
		Model:     res.model,
		MaxTokens: knowledgeMaxTokens,
		System:    []anthropic.BetaTextBlockParam{{Text: watch.KnowledgePrompt()}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("Article:\n" + article))},
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	resp, err := res.provider.New(cctx, params)
	status, cost := "completed", int64(0)
	var in, out int64
	var kg watch.ItemKG
	switch {
	case err != nil:
		status = "failed"
	case resp.StopReason == anthropic.BetaStopReasonRefusal:
		status, err = "refused", watch.ErrNotKnowledge
	case resp.StopReason == anthropic.BetaStopReasonMaxTokens: // cut off mid-JSON: not read again, nor charged twice
		status, err = "step_limit", watch.ErrNotKnowledge
		cost = costMicros(res.price, resp.Usage)
		in, out = resp.Usage.InputTokens+resp.Usage.CacheReadInputTokens+resp.Usage.CacheCreationInputTokens, resp.Usage.OutputTokens
	default:
		cost = costMicros(res.price, resp.Usage)
		in, out = resp.Usage.InputTokens+resp.Usage.CacheReadInputTokens+resp.Usage.CacheCreationInputTokens, resp.Usage.OutputTokens
		var text strings.Builder
		for _, b := range resp.Content {
			text.WriteString(b.Text)
		}
		if kg, err = watch.ParseItemKG(text.String(), article); err != nil {
			status = "refused"
		}
	}
	if _, rerr := r.db.Exec(context.WithoutCancel(ctx), `INSERT INTO agent_runs (id, user_id, agent, skill_version, model, status,
		input_tokens, output_tokens, cost_micros, finished_at) VALUES ($1, $2, 'watch-knowledge', '1.0.0', $3, $4, $5, $6, $7, now())`,
		uuid.Must(uuid.NewV7()), userID, res.name+"/"+res.model, status, in, out, cost); rerr != nil {
		r.log.Error("recording a watch knowledge run failed", "err", rerr)
	}
	switch {
	case errors.Is(err, watch.ErrNotKnowledge): // read once: the graph's own reading stays
		_, err = r.db.Exec(ctx, `UPDATE watch_items SET kg_at = now() WHERE id = $1 AND user_id = $2`, id, userID)
		return true, err
	case err != nil: // the provider failed: tried again on a later step
		return true, err
	}
	raw, err := json.Marshal(kg)
	if err != nil {
		return true, err
	}
	_, err = r.db.Exec(ctx, `UPDATE watch_items SET kg = $3, kg_at = now() WHERE id = $1 AND user_id = $2`, id, userID, raw)
	return true, err
}

// RunWatchKnowledge reads a few items every two minutes.
func (r *Runner) RunWatchKnowledge(ctx context.Context) {
	for {
		done := telemetry.Time("watch-knowledge")
		err := r.ExtractKnowledge(ctx, knowledgePerStep)
		done(err)
		if err != nil && ctx.Err() == nil {
			r.log.Error("watch knowledge failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Minute):
		}
	}
}
