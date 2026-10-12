-- +goose Up
-- Agents (EP-19). Agents never write user data directly: every change they
-- want is a proposal the user approves. Run records hold no prompt or reply
-- text, only what is needed to audit cost and behaviour.

-- Monthly spending cap; agents also need the ai_agents consent.
CREATE TABLE agent_settings (
    user_id              uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    monthly_budget_cents integer NOT NULL DEFAULT 200 CHECK (monthly_budget_cents BETWEEN 0 AND 100000),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE agent_runs (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent         text NOT NULL,
    skill_version text NOT NULL,
    model         text NOT NULL,
    status        text NOT NULL DEFAULT 'running'
                  CHECK (status IN ('running', 'completed', 'budget_exhausted', 'step_limit', 'refused', 'failed')),
    tools_called  text[] NOT NULL DEFAULT '{}',  -- tool names and their outcome, never inputs
    proposals     integer NOT NULL DEFAULT 0,
    input_tokens  bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    cost_micros   bigint NOT NULL DEFAULT 0,     -- US dollars × 1,000,000
    error         text CHECK (length(error) <= 200),  -- a fixed reason code, never provider text
    started_at    timestamptz NOT NULL DEFAULT now(),
    finished_at   timestamptz
);
CREATE INDEX agent_runs_user_started ON agent_runs (user_id, started_at DESC);

CREATE TABLE agent_proposals (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    run_id      uuid NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    agent       text NOT NULL,
    tool        text NOT NULL,
    input       jsonb NOT NULL,                  -- the validated tool input, applied on approval
    summary     text NOT NULL CHECK (length(summary) <= 2000),  -- preview shown to the user
    reason      text NOT NULL DEFAULT '' CHECK (length(reason) <= 1000),
    status      text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'failed')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    decided_at  timestamptz
);
CREATE INDEX agent_proposals_user_status ON agent_proposals (user_id, status, created_at);

-- What agents may remember about the user: written only by approved
-- proposals, readable and deletable by the user.
CREATE TABLE agent_memories (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent      text NOT NULL,
    content    text NOT NULL CHECK (length(content) BETWEEN 1 AND 500),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX agent_memories_user ON agent_memories (user_id, agent);

-- +goose Down
DROP TABLE agent_memories;
DROP TABLE agent_proposals;
DROP TABLE agent_runs;
DROP TABLE agent_settings;
