-- +goose Up
-- Where the Markdown vault lives (EP-106). Without a row, notes live in the
-- database only, which is the default. With one, a server keeps a folder of
-- Markdown files in step with Axon: it pairs once with a short-lived code
-- (stored hashed) that becomes a write token, then reports after each sync.
CREATE TABLE vault_sources (
    user_id         uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    host            text NOT NULL CHECK (length(host) BETWEEN 1 AND 100),
    dir             text NOT NULL CHECK (length(dir) BETWEEN 1 AND 500),
    scaffold        boolean NOT NULL DEFAULT true,
    pair_hash       bytea UNIQUE,
    pair_expires_at timestamptz,
    token_id        uuid REFERENCES api_tokens(id) ON DELETE SET NULL,
    paired_at       timestamptz,
    paired_host     text NOT NULL DEFAULT '' CHECK (length(paired_host) <= 100),
    last_seen_at    timestamptz,
    last_report     jsonb NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT now()
);
-- Every foreign key is indexed (EP-70).
CREATE INDEX vault_sources_token ON vault_sources (token_id);

-- +goose Down
DROP TABLE vault_sources;
