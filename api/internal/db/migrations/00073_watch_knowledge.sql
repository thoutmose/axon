-- +goose Up
-- The tech watch's knowledge graph read by the person's model (owner
-- request): each item's entities and facts, extracted once from its text.
ALTER TABLE agent_settings ADD COLUMN watch_knowledge boolean NOT NULL DEFAULT false;
ALTER TABLE watch_items ADD COLUMN kg jsonb, ADD COLUMN kg_at timestamptz;
CREATE INDEX watch_items_kg_pending ON watch_items (published_at DESC) WHERE kg_at IS NULL;

-- +goose Down
DROP INDEX watch_items_kg_pending;
ALTER TABLE watch_items DROP COLUMN kg, DROP COLUMN kg_at;
ALTER TABLE agent_settings DROP COLUMN watch_knowledge;
