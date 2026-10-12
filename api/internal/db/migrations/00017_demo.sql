-- +goose Up
-- The demo account (EP-28): at most one, reset by `axon demo reset`. It does
-- not count as an account for the "only the first account may register" rule.
ALTER TABLE users ADD COLUMN demo boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX users_one_demo ON users (demo) WHERE demo;

-- +goose Down
DELETE FROM users WHERE demo;
ALTER TABLE users DROP COLUMN demo;
