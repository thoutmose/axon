package agents

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// AI usage (EP-152): the tokens and cost of every run, the meal photos, the
// video summaries, and Jarvis, from their records: this month, all along,
// month by month, and agent by agent, with what compacting tool results
// saved, measured in characters.

// Tally is tokens and cost over some runs.
type Tally struct {
	Runs         int64 `json:"runs"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CostMicros   int64 `json:"cost_micros"`
	ToolRaw      int64 `json:"tool_chars_raw"`  // tool results before compaction
	ToolSent     int64 `json:"tool_chars_sent"` // and as sent
}

// AgentTally is one agent's tally.
type AgentTally struct {
	Agent string `json:"agent"`
	Tally
}

// MonthTally is one month's tally.
type MonthTally struct {
	Month string `json:"month"` // 2026-10
	Tally
}

// Usage is what Preferences shows.
type Usage struct {
	ThisMonth Tally        `json:"this_month"`
	AllTime   Tally        `json:"all_time"`
	ByAgent   []AgentTally `json:"by_agent"`
	ByMonth   []MonthTally `json:"by_month"` // the last twelve, oldest first
}

const tallyColumns = `count(id), coalesce(sum(input_tokens), 0)::bigint, coalesce(sum(output_tokens), 0)::bigint,
	coalesce(sum(cost_micros), 0)::bigint, coalesce(sum(tool_chars_raw), 0)::bigint, coalesce(sum(tool_chars_sent), 0)::bigint`

func scanTally(row pgx.Row, extra ...any) (Tally, error) {
	var t Tally
	err := row.Scan(append(extra, &t.Runs, &t.InputTokens, &t.OutputTokens, &t.CostMicros, &t.ToolRaw, &t.ToolSent)...)
	return t, err
}

// GetUsage adds up a person's runs.
func GetUsage(ctx context.Context, q db.Querier, userID uuid.UUID) (Usage, error) {
	u := Usage{ByAgent: []AgentTally{}, ByMonth: []MonthTally{}}
	var err error
	if u.AllTime, err = scanTally(q.QueryRow(ctx, `SELECT `+tallyColumns+` FROM agent_runs WHERE user_id = $1`, userID)); err != nil {
		return u, err
	}
	if u.ThisMonth, err = scanTally(q.QueryRow(ctx, `SELECT `+tallyColumns+` FROM agent_runs WHERE user_id = $1
		AND started_at >= date_trunc('month', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'`, userID)); err != nil {
		return u, err
	}
	rows, err := q.Query(ctx, `SELECT agent, `+tallyColumns+` FROM agent_runs WHERE user_id = $1
		GROUP BY agent ORDER BY sum(input_tokens + output_tokens) DESC, agent`, userID)
	if err != nil {
		return u, err
	}
	for rows.Next() {
		var a AgentTally
		if a.Tally, err = scanTally(rows, &a.Agent); err != nil {
			rows.Close()
			return u, err
		}
		u.ByAgent = append(u.ByAgent, a)
	}
	if err := rows.Err(); err != nil {
		return u, err
	}
	rows, err = q.Query(ctx, `SELECT to_char(m, 'YYYY-MM'), `+tallyColumns+`
		FROM generate_series(date_trunc('month', now() AT TIME ZONE 'UTC') - interval '11 months', date_trunc('month', now() AT TIME ZONE 'UTC'), interval '1 month') m
		LEFT JOIN agent_runs r ON r.user_id = $1 AND r.started_at >= m AT TIME ZONE 'UTC' AND r.started_at < (m + interval '1 month') AT TIME ZONE 'UTC'
		GROUP BY m ORDER BY m`, userID)
	if err != nil {
		return u, err
	}
	for rows.Next() {
		var m MonthTally
		if m.Tally, err = scanTally(rows, &m.Month); err != nil {
			rows.Close()
			return u, err
		}
		u.ByMonth = append(u.ByMonth, m)
	}
	return u, rows.Err()
}

func (r *Runner) usage(w http.ResponseWriter, req *http.Request) {
	who, _ := auth.FromContext(req.Context())
	u, err := GetUsage(req.Context(), r.db, who.UserID)
	if err != nil {
		r.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}
