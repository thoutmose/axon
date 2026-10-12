-- +goose Up
-- Task reminders at a time of day (EP-68): sent once, at remind_at.
ALTER TABLE tasks ADD COLUMN remind_at timestamptz, ADD COLUMN reminded_at timestamptz;
CREATE INDEX tasks_remind ON tasks (remind_at) WHERE remind_at IS NOT NULL AND reminded_at IS NULL;

-- +goose Down
DROP INDEX tasks_remind;
ALTER TABLE tasks DROP COLUMN reminded_at, DROP COLUMN remind_at;
