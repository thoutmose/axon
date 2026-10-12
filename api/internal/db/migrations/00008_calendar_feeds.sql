-- +goose Up
-- Read-only iCal feeds. The URL carries a random token; only its SHA-256 is
-- stored, and deleting the row revokes the feed.
CREATE TABLE calendar_feeds (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    token_hash   bytea NOT NULL UNIQUE,
    types        text[] NOT NULL CHECK (cardinality(types) > 0 AND types <@ ARRAY['task', 'goal', 'milestone']),
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz
);
CREATE INDEX calendar_feeds_user ON calendar_feeds (user_id);

-- +goose Down
DROP TABLE calendar_feeds;
