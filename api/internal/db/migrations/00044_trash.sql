-- +goose Up
-- Trash (EP-66): what a deletion removes is kept for 30 days as the rows
-- it removed, so it can be put back. Deletions made in one transaction
-- (a goal and its milestones) share a batch and come back together.
CREATE TABLE trash (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    batch       bigint NOT NULL,          -- the deleting transaction
    seq         bigserial,                -- order of deletion within it
    entity_id   uuid NOT NULL,
    entity_type text NOT NULL,
    title       text NOT NULL,
    rows        jsonb NOT NULL,           -- [{"table": …, "row": {…}}], parents first
    deleted_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX trash_user ON trash (user_id, deleted_at DESC);
CREATE INDEX trash_batch ON trash (user_id, batch);

-- +goose Down
DROP TABLE trash;
