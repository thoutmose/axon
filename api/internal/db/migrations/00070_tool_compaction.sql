-- +goose Up
-- What agents' tool results weighed before and after Axon compacted them
-- (EP-152), in characters, so the saving is measured, not guessed.
ALTER TABLE agent_runs ADD COLUMN tool_chars_raw bigint NOT NULL DEFAULT 0 CHECK (tool_chars_raw >= 0),
    ADD COLUMN tool_chars_sent bigint NOT NULL DEFAULT 0 CHECK (tool_chars_sent >= 0);

-- +goose Down
ALTER TABLE agent_runs DROP COLUMN tool_chars_raw, DROP COLUMN tool_chars_sent;
