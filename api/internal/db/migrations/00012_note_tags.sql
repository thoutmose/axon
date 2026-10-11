-- +goose Up
-- Tags parsed from a note's front matter (tags:) and body (#tag), for the
-- notes workspace (EP-25). NULL means not parsed yet: "axon migrate up"
-- fills existing notes after this migration.
ALTER TABLE notes ADD COLUMN tags text[];
CREATE INDEX notes_tags ON notes USING gin (tags);

-- +goose Down
DROP INDEX notes_tags;
ALTER TABLE notes DROP COLUMN tags;
