-- +goose Up
-- Tutor and Resource agents (EP-27): roadmaps imported from Axon's curated
-- library or drafted by the Tutor agent, and runs that continue a
-- conversation.
ALTER TABLE roadmaps DROP CONSTRAINT roadmaps_source_check;
ALTER TABLE roadmaps ADD CONSTRAINT roadmaps_source_check CHECK (source IN ('manual', 'ossu', 'library', 'tutor'));
ALTER TABLE agent_runs ADD COLUMN parent_run_id uuid REFERENCES agent_runs(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE agent_runs DROP COLUMN parent_run_id;
UPDATE roadmaps SET source = 'manual' WHERE source IN ('library', 'tutor');
ALTER TABLE roadmaps DROP CONSTRAINT roadmaps_source_check;
ALTER TABLE roadmaps ADD CONSTRAINT roadmaps_source_check CHECK (source IN ('manual', 'ossu'));
