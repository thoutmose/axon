package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/privacy"
	"github.com/thoutmose/axon/api/internal/textx"
)

// Jarvis (EP-153) is the master agent of the side panel. Its one tool asks
// another agent: that agent runs inside Jarvis's run, on its own provider
// model and tools, within what is left of the run's budget, and its
// proposals belong to the run, so the panel shows them for approval. Only
// Jarvis may ask, so agents cannot call each other in a loop.

// Jarvis is the master agent's name.
const Jarvis = "jarvis"

type askInput struct {
	Agent       string `json:"agent"`
	Instruction string `json:"instruction"`
}

func init() {
	registry["ask_agent"] = Tool{
		Name: "ask_agent",
		Description: "Ask one of Axon's agents to do something for the person, with a complete instruction. " +
			"It answers in text; any change it wants is a proposal the person approves. The agents are listed in your instructions.",
		Properties: map[string]any{
			"agent":       map[string]any{"type": "string", "description": "The agent's name, such as media or task-manager."},
			"instruction": map[string]any{"type": "string", "maxLength": 4000},
		},
		Required: []string{"agent", "instruction"},
	}
}

// agentsList tells Jarvis which agents it can ask, and what each does.
func (r *Runner) agentsList() string {
	names := make([]string, 0, len(r.Skills))
	for n := range r.Skills {
		if n != Jarvis {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	var b strings.Builder
	b.WriteString("Agents you can ask (name: what it does):\n")
	for _, n := range names {
		fmt.Fprintf(&b, "- %s: %s\n", n, r.Skills[n].Description)
	}
	return b.String()
}

// ask runs another agent for Jarvis and returns its answer as data.
func (r *Runner) ask(ctx context.Context, userID uuid.UUID, rn *run, raw json.RawMessage) (string, bool, error) {
	var in askInput
	if err := json.Unmarshal(raw, &in); err != nil || strings.TrimSpace(in.Instruction) == "" {
		return "Give ask_agent an agent and an instruction.", true, nil
	}
	sub, ok := r.Skills[in.Agent]
	if !ok || in.Agent == Jarvis {
		return fmt.Sprintf("There is no agent called %q to ask; use one from your list.", textx.Cut(in.Agent, 40)), true, nil
	}
	for _, c := range sub.Consents {
		granted, err := privacy.Granted(ctx, r.db, userID, c)
		if err != nil {
			return "", false, err
		}
		if !granted {
			return fmt.Sprintf("The %s agent needs the person's consent (%s) first: they can give it in Privacy.", sub.Title, c), true, nil
		}
	}
	res, err := r.resolve(ctx, r.db, userID, sub)
	if err != nil {
		return fmt.Sprintf("The %s agent is not available now.", sub.Title), true, nil
	}
	// The child shares the run: its tokens, cost, and proposals add up.
	child := &run{id: rn.id, status: "running", budgetMicros: rn.budgetMicros, cost: rn.cost, input: rn.input, output: rn.output,
		toolRaw: rn.toolRaw, toolSent: rn.toolSent, proposals: rn.proposals}
	err = r.loop(ctx, userID, sub, res, textx.Cut(in.Instruction, 4000), nil, child)
	rn.cost, rn.input, rn.output, rn.toolRaw, rn.toolSent = child.cost, child.input, child.output, child.toolRaw, child.toolSent
	made := child.proposals - rn.proposals
	rn.proposals = child.proposals
	rn.tools = append(rn.tools, "ask_agent:"+sub.Name)
	for _, t := range child.tools {
		rn.tools = append(rn.tools, sub.Name+"/"+t)
	}
	switch {
	case err != nil && ctx.Err() != nil:
		return "", false, err
	case err != nil:
		r.log.Warn("an agent asked by Jarvis failed", "agent", sub.Name, "err", err)
		return fmt.Sprintf("The %s agent failed; try again later.", sub.Title), true, nil
	case child.status == "budget_exhausted":
		return fmt.Sprintf("The %s agent stopped: this month's AI budget is spent.", sub.Title), true, nil
	}
	return fmt.Sprintf("<data>\nThe %s agent answered:\n%s\n\nIt made %d proposal(s), waiting for the person's approval.\n</data>",
		sub.Title, child.reply, made), false, nil
}
