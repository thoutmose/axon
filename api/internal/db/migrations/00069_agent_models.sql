-- +goose Up
-- The model each agent uses (EP-151): '*' for all of them, or one agent by
-- name; the provider stays the person's. And, when the person turns it on,
-- videos and episodes summarised in prose by their model, once each.
CREATE TABLE agent_models (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent   text NOT NULL CHECK (agent ~ '^(\*|[a-z][a-z0-9-]{0,59})$'),
    model   text NOT NULL CHECK (length(model) BETWEEN 1 AND 200),
    PRIMARY KEY (user_id, agent)
);
ALTER TABLE agent_settings ADD COLUMN video_summaries boolean NOT NULL DEFAULT false;
ALTER TABLE watch_items ADD COLUMN ai_summary_at timestamptz;
CREATE INDEX watch_items_to_summarise ON watch_items (user_id, published_at DESC) WHERE ai_summary_at IS NULL AND transcript <> '';

-- +goose Down
DROP INDEX watch_items_to_summarise;
ALTER TABLE watch_items DROP COLUMN ai_summary_at;
ALTER TABLE agent_settings DROP COLUMN video_summaries;
DROP TABLE agent_models;
