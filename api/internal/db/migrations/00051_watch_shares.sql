-- +goose Up
-- Read-only links to a selection of the tech watch (EP-75). The URL carries
-- a random token; only its SHA-256 is stored. A link expires, and deleting
-- the row revokes it at once.
CREATE TABLE watch_shares (
    id             uuid PRIMARY KEY,
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name           text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    token_hash     bytea NOT NULL UNIQUE,
    query          text NOT NULL CHECK (length(query) <= 2000), -- the filter, as a query string
    expires_at     timestamptz NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    last_viewed_at timestamptz,
    views          integer NOT NULL DEFAULT 0
);
CREATE INDEX watch_shares_user ON watch_shares (user_id);

-- +goose Down
DROP TABLE watch_shares;
