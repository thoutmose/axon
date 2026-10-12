-- +goose Up
-- Repositories the delivery agents may read (EP-26). Read-only unless the
-- person turns on comments; issues they draft land on project_id's board.
CREATE TABLE repo_connections (
    id             uuid PRIMARY KEY,
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    owner          text NOT NULL CHECK (owner ~ '^[A-Za-z0-9-]{1,39}$'),
    repo           text NOT NULL CHECK (repo ~ '^[A-Za-z0-9._-]{1,100}$'),
    token_sealed   bytea,                                   -- sealed with AXON_ENCRYPTION_KEY
    allow_comments boolean NOT NULL DEFAULT false,
    project_id     uuid REFERENCES projects(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, owner, repo)
);

-- +goose Down
DROP TABLE repo_connections;
