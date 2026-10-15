-- +goose Up
-- A task checked off stays in its list, crossed out, until the next day
-- (EP-116): the list it was in when it was done, so it shows there and
-- Undo puts it back. Empty for tasks done before, and once reopened.
ALTER TABLE tasks ADD COLUMN done_from text CHECK (done_from IN ('inbox', 'next', 'doing', 'waiting', 'someday'));

-- +goose Down
ALTER TABLE tasks DROP COLUMN done_from;
