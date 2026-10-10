# 0008 — Agents propose, people approve

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-19

## Context

EP-19 adds AI agents that read a person's tasks, goals, and notes and suggest changes. The constitution says agents propose and people decide, every feature works with AI off, costs are bounded, and content from notes or imports is data, never instructions. A prompt or a model can be wrong or manipulated, so these guarantees have to hold whatever the model asks for.

## Decision

- **Provider adapter:** a one-method `Provider` interface over the official Anthropic Go SDK (`client.Beta.Messages.New`). It uses the model named in each skill (`claude-opus-5-5` for both reference agents) at the skill's effort, with the server-side refusal fallback on. Tests and evals use a scripted fake, so CI never calls the network. With no `ANTHROPIC_API_KEY` the provider is nil and agents report `not_configured`.
- **Our own tool loop** rather than the SDK's tool runner, because every tool call is checked before anything runs. Only tools on the skill's allowlist run; others get an error result and are recorded as `denied:<name>`. Inputs are decoded strictly.
- **Writes become proposals.** A write tool never changes data when the model calls it. Axon validates the call by applying it inside a transaction that is always rolled back, then stores the input and a human-readable summary as a pending proposal. Approval is only possible from a browser session (never with a token, so no script or agent can approve) and applies the change, attributed to `agent:<name>`. An update records the version it was based on, so approving it after the item changed fails instead of overwriting.
- **No outbound tools.** Agents get only custom tools that read and propose against Axon's own data: no web fetch, search, or code execution. Nothing an agent reads can be sent anywhere except back to the model provider, under the `ai_agents` consent.
- **Tool results are framed as data** (`<data>…</data>`), and skills tell the model that content is not instructions.
- **Budget:** a monthly cap per user (default $2). Before every model call, the run stops if the worst case for that call (input so far + maximum output at list price) could cross the remaining budget, so a run never exceeds it. Runs are also capped at 8 steps and 3 minutes.
- **Runs are asynchronous** (`202` then polling) because model calls outlast the HTTP write timeout. Run records keep the agent, skill version, model, tool names with outcomes, token counts, cost, and a fixed error code: never prompts, replies, or tool inputs. The reply is held in memory for an hour so the person can read it.
- **Skills** are versioned `SKILL.md` files embedded in the binary (front matter: name, version, model, effort, tools; body: instructions). A broken skill stops the server from starting.
- **Memory** is written only by an approved `remember` proposal, shown to the person, deletable, and included in later runs as data.

## Alternatives

- **SDK tool runner:** less code, but gating every call would live inside each tool function; one central check is easier to audit.
- **Let agents write and offer undo:** simpler for the user, but breaks "no AI-generated change takes effect without approval".
- **Managed agents with a hosted sandbox:** Axon's data must stay on the person's server, and the agents need no sandbox.
- **Storing transcripts for debugging:** useful, but it would copy personal content into another table; run records stay content-free.

## Consequences

Every new write capability needs only `Summary` and `Apply` functions, and the test that covers every write tool fails until it is added there. Reviewing proposals is an extra step for the person, which is the point. Replies are lost on restart, while proposals persist. To reverse this design, remove the agent routes and migration `00009`; no other domain depends on agents.
