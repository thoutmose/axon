-- +goose Up
-- Sign in with another account (EP-72): Google, Microsoft, Apple, Discord,
-- Slack, or any OpenID Connect provider the operator configures. An
-- identity is linked by its owner while signed in (or made with a new
-- account when registration is open); it is never matched by email.
CREATE TABLE auth_identities (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider     text NOT NULL CHECK (length(provider) BETWEEN 1 AND 40),
    subject      text NOT NULL CHECK (length(subject) BETWEEN 1 AND 255),
    email        text NOT NULL DEFAULT '' CHECK (length(email) <= 320),
    name         text NOT NULL DEFAULT '' CHECK (length(name) <= 200),
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    UNIQUE (provider, subject)
);
CREATE INDEX auth_identities_user ON auth_identities (user_id);

-- One sign-in in progress: the state (hashed) is also in a cookie of the
-- browser that started it; the PKCE verifier and the nonce stay here.
CREATE TABLE oauth_states (
    state_hash bytea PRIMARY KEY,
    provider   text NOT NULL,
    intent     text NOT NULL CHECK (intent IN ('signin', 'link')),
    user_id    uuid REFERENCES users(id) ON DELETE CASCADE,
    verifier   text NOT NULL,
    nonce      text NOT NULL,
    next       text NOT NULL DEFAULT '/',
    expires_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE oauth_states;
DROP TABLE auth_identities;
