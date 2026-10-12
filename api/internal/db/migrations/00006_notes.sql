-- +goose Up
-- Markdown notes. content is the whole file, front matter included, so
-- export and sync reproduce it byte for byte; the other columns are derived
-- from it for listing and search.
CREATE TABLE notes (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    -- Duplicated from entities so a path is unique per user (ADR 0006 allows
    -- owner columns in domain tables when a constraint needs them).
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path        text NOT NULL CHECK (path ~ '^[^/].*\.md$' AND path NOT LIKE '%..%' AND length(path) <= 500),
    content     text NOT NULL CHECK (length(content) <= 1000000),
    kind        text NOT NULL DEFAULT 'note',   -- front matter "type": fleeting, literature, permanent, project, …
    para        text NOT NULL CHECK (para IN ('inbox', 'projects', 'areas', 'resources', 'archive', 'other')),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, path)
);
CREATE INDEX notes_user_para ON notes (user_id, para, path);
CREATE INDEX notes_user_updated ON notes (user_id, updated_at, id);

-- Paths of deleted notes, so sync can delete the file on the other side. Kept 30 days.
CREATE TABLE note_tombstones (
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path        text NOT NULL,
    deleted_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, path)
);

-- +goose Down
DROP TABLE note_tombstones;
DROP TABLE notes;
