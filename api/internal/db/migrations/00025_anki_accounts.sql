-- +goose Up
-- Anki sync accounts (EP-42): where a person's anki-sync-server is, with the
-- password sealed with AXON_ENCRYPTION_KEY, and how the last sync went.
CREATE TABLE anki_accounts (
    user_id         uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    endpoint        text NOT NULL CHECK (endpoint ~ '^https?://' AND length(endpoint) <= 500),
    username        text NOT NULL CHECK (length(username) BETWEEN 1 AND 200),
    password_sealed bytea NOT NULL,
    auto_sync       boolean NOT NULL DEFAULT true,
    last_sync_at    timestamptz,
    last_status     text CHECK (last_status IN ('ok', 'error')),
    last_message    text NOT NULL DEFAULT '' CHECK (length(last_message) <= 1000),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE anki_accounts;
