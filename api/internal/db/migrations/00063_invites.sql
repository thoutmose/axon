-- +goose Up
-- Invitations (EP-120): the owner of an Axon whose registration is closed
-- invites one person with a link. Only the token's SHA-256 is stored; a
-- link is used once, expires, and can bring the person into the owner's
-- household.
CREATE TABLE invites (
    id         uuid PRIMARY KEY,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    note       text NOT NULL DEFAULT '' CHECK (length(note) <= 200),
    household  boolean NOT NULL DEFAULT false,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    used_at    timestamptz,
    used_by    uuid REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX invites_created_by ON invites (created_by);
CREATE INDEX invites_used_by ON invites (used_by);

-- +goose Down
DROP TABLE invites;
