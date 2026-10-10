-- +goose Up
-- Version for optimistic concurrency (ETag / If-Match) on the account.
ALTER TABLE users ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
-- Last accepted TOTP time step, so each code works only once (ASVS 2.8.4).
ALTER TABLE users ADD COLUMN totp_last_step bigint NOT NULL DEFAULT 0;

-- Stored responses for POST requests that carried an Idempotency-Key.
CREATE TABLE idempotency_keys (
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key          text NOT NULL CHECK (length(key) BETWEEN 1 AND 200),
    request_hash bytea NOT NULL,
    status       int,           -- NULL while the first request is still running
    body         bytea,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, key)
);
CREATE INDEX idempotency_keys_created ON idempotency_keys (created_at);

-- Personal access tokens for scripts and Markdown sync.
CREATE TABLE api_tokens (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    token_hash    bytea NOT NULL UNIQUE,
    scope         text NOT NULL CHECK (scope IN ('read', 'write')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_used_at  timestamptz,
    expires_at    timestamptz NOT NULL
);
CREATE INDEX api_tokens_user ON api_tokens (user_id);

-- +goose Down
DROP TABLE api_tokens;
DROP TABLE idempotency_keys;
ALTER TABLE users DROP COLUMN totp_last_step;
ALTER TABLE users DROP COLUMN updated_at;
