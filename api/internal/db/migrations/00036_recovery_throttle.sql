-- +goose Up
-- Recovery sign-in with an authenticator code locks for 15 minutes after five
-- wrong codes for the account (EP-21 security review).
ALTER TABLE users
    ADD COLUMN recovery_failures integer NOT NULL DEFAULT 0 CHECK (recovery_failures >= 0),
    ADD COLUMN recovery_locked_until timestamptz;

-- +goose Down
ALTER TABLE users DROP COLUMN recovery_failures, DROP COLUMN recovery_locked_until;
