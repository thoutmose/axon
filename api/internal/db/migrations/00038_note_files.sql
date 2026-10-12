-- +goose Up
-- Pictures and PDFs that notes embed (EP-51): ![](Attachments/x.png) or
-- ![[x.png]]. Stored by vault path so sync writes them next to the notes.
CREATE TABLE note_files (
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path         text NOT NULL CHECK (path ~* '^[^/.][^/]*(/[^/.][^/]*)*\.(png|jpe?g|gif|webp|svg|pdf)$' AND path NOT LIKE '%..%' AND length(path) <= 500),
    content_type text NOT NULL CHECK (content_type IN ('image/png', 'image/jpeg', 'image/gif', 'image/webp', 'image/svg+xml', 'application/pdf')),
    data         bytea NOT NULL CHECK (length(data) <= 10485760),
    hash         text NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, path)
);

-- Paths of deleted files, so sync deletes them on the other side. Kept 30 days.
CREATE TABLE note_file_tombstones (
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path        text NOT NULL,
    deleted_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, path)
);

-- +goose Down
DROP TABLE note_file_tombstones;
DROP TABLE note_files;
