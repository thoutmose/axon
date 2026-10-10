-- +goose Up
-- The user's IANA time zone decides what "today" means for tasks and check-ins.
ALTER TABLE users ADD COLUMN timezone text NOT NULL DEFAULT 'UTC';

CREATE TABLE tasks (
    id            uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    status        text NOT NULL DEFAULT 'inbox'
                  CHECK (status IN ('inbox', 'next', 'waiting', 'someday', 'done', 'cancelled')),
    notes         text NOT NULL DEFAULT '' CHECK (length(notes) <= 20000),
    context       text CHECK (context ~ '^@[a-z0-9_-]{1,40}$'),
    scheduled_on  date,
    due_on        date,
    recurrence    text CHECK (length(recurrence) <= 200),  -- RFC 5545 RRULE, date-based
    series_id     uuid,          -- first task of a recurring series; NULL for one-off tasks
    occurrence_on date,          -- the series date this task stands for
    completed_at  timestamptz,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    -- Completing an occurrence creates the next one exactly once, even on retry.
    UNIQUE (series_id, occurrence_on)
);
CREATE INDEX tasks_open_dates ON tasks (scheduled_on, due_on) WHERE status NOT IN ('done', 'cancelled');
CREATE INDEX tasks_status ON tasks (status);

CREATE TABLE checkins (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('daily', 'weekly', 'monthly')),
    day        date NOT NULL,   -- local date the check-in covers (week/month start for those kinds)
    note       text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, kind, day)
);

-- +goose Down
DROP TABLE checkins;
DROP TABLE tasks;
ALTER TABLE users DROP COLUMN timezone;
