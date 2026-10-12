-- +goose Up
-- The vault mirror (EP-48): items kept in step with Markdown notes.

-- One row per mirrored item: which note shows it, the last content Axon
-- wrote, and the free "Notes" section the person keeps in that note.
-- note_id has no foreign key on purpose: when a note is deleted, the
-- mapping is still needed to delete its item.
CREATE TABLE vault_mirror (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key     text NOT NULL CHECK (length(key) <= 100), -- an entity id, or health:food:YYYY-MM-DD, health:workout:<id>
    kind    text NOT NULL,
    note_id uuid NOT NULL UNIQUE,
    hash    text NOT NULL,
    path    text NOT NULL DEFAULT '',
    extra   text NOT NULL DEFAULT '' CHECK (length(extra) <= 100000),
    PRIMARY KEY (user_id, key)
);

-- Who is mirrored, into which folder, and whether health is included.
CREATE TABLE vault_mirror_users (
    user_id   uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    folder    text NOT NULL,
    health    boolean NOT NULL DEFAULT false,
    synced_at timestamptz NOT NULL DEFAULT now()
);

-- Changes that leave no graph event (habit check-ins, health logs).
CREATE TABLE vault_mirror_queue (
    user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key       text NOT NULL CHECK (length(key) <= 100),
    queued_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, key)
);

-- How far the mirror has read the event log.
CREATE TABLE vault_mirror_cursor (
    id       integer PRIMARY KEY CHECK (id = 1),
    event_id uuid NOT NULL
);

-- +goose Down
DROP TABLE vault_mirror_cursor;
DROP TABLE vault_mirror_queue;
DROP TABLE vault_mirror_users;
DROP TABLE vault_mirror;
