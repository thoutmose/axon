-- +goose Up
CREATE TABLE users (
    id               uuid PRIMARY KEY,
    display_name     text NOT NULL CHECK (length(display_name) BETWEEN 1 AND 100),
    totp_secret      bytea,          -- AES-256-GCM sealed; NULL when TOTP is off
    totp_enabled_at  timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE passkeys (
    id             uuid PRIMARY KEY,
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id  bytea NOT NULL UNIQUE,
    credential     jsonb NOT NULL,   -- webauthn.Credential (public key, sign count, flags)
    name           text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    created_at     timestamptz NOT NULL DEFAULT now(),
    last_used_at   timestamptz
);
CREATE INDEX passkeys_user_id ON passkeys (user_id);

CREATE TABLE sessions (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    bytea NOT NULL UNIQUE,
    user_agent    text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL
);
CREATE INDEX sessions_user_id ON sessions (user_id);

CREATE TABLE recovery_codes (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  bytea NOT NULL UNIQUE,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX recovery_codes_user_id ON recovery_codes (user_id);

-- In-flight WebAuthn ceremonies (challenge state), referenced by a short-lived cookie.
CREATE TABLE webauthn_ceremonies (
    id          uuid PRIMARY KEY,
    user_id     uuid REFERENCES users(id) ON DELETE CASCADE,
    kind        text NOT NULL CHECK (kind IN ('register', 'login', 'add_passkey')),
    data        jsonb NOT NULL,
    display_name text,
    expires_at  timestamptz NOT NULL
);

CREATE TABLE audit_events (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action      text NOT NULL,
    ip          text NOT NULL DEFAULT '',
    user_agent  text NOT NULL DEFAULT '',
    details     jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_user_created ON audit_events (user_id, created_at DESC);

-- Audit events are append-only; rows only disappear with their user (erasure).
-- +goose StatementBegin
CREATE FUNCTION audit_events_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only';
END $$;
-- +goose StatementEnd
CREATE TRIGGER audit_events_no_update BEFORE UPDATE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_append_only();

CREATE TABLE consents (
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     text NOT NULL,
    granted     boolean NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, purpose)
);

-- +goose Down
DROP TABLE consents;
DROP TRIGGER audit_events_no_update ON audit_events;
DROP FUNCTION audit_events_append_only();
DROP TABLE audit_events;
DROP TABLE webauthn_ceremonies;
DROP TABLE recovery_codes;
DROP TABLE sessions;
DROP TABLE passkeys;
DROP TABLE users;
