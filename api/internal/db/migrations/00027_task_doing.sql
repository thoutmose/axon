-- +goose Up
-- A "doing" status for tasks in progress, so the task Kanban board has an
-- In progress column (owner request, EP-42).
ALTER TABLE tasks DROP CONSTRAINT tasks_status_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_status_check
    CHECK (status IN ('inbox', 'next', 'doing', 'waiting', 'someday', 'done', 'cancelled'));

-- +goose Down
UPDATE tasks SET status = 'next' WHERE status = 'doing';
ALTER TABLE tasks DROP CONSTRAINT tasks_status_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_status_check
    CHECK (status IN ('inbox', 'next', 'waiting', 'someday', 'done', 'cancelled'));
