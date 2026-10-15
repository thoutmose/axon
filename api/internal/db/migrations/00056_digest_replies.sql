-- +goose Up
-- Answering the digest (EP-108): the numbers of the latest digest or list
-- a person received, so "done 2" means the same item everywhere, and what
-- the last chat command did, so "undo" can take it back.
CREATE TABLE digest_items (
    user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    n         integer NOT NULL CHECK (n BETWEEN 1 AND 100),
    entity_id uuid NOT NULL,
    type      text NOT NULL CHECK (type IN ('task', 'habit', 'milestone')),
    title     text NOT NULL CHECK (length(title) <= 500),
    sent_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, n)
);

ALTER TABLE notify_channels ADD COLUMN last_action jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE notify_channels DROP COLUMN last_action;
DROP TABLE digest_items;
