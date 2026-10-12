-- +goose Up
-- Focus sessions (EP-35): Pomodoro work linked to what it was for through a
-- graph link, so focus time shows up on tasks, projects, skills, and notes.
CREATE TABLE focus_sessions (
    id              uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    started_at      timestamptz NOT NULL,
    ended_at        timestamptz,
    planned_minutes integer NOT NULL CHECK (planned_minutes BETWEEN 1 AND 1440),
    worked_minutes  integer NOT NULL DEFAULT 0 CHECK (worked_minutes BETWEEN 0 AND 1440),
    rounds          integer NOT NULL DEFAULT 0 CHECK (rounds BETWEEN 0 AND 100),
    status          text NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'stopped')),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX focus_sessions_started ON focus_sessions (started_at);

-- +goose Down
DROP TABLE focus_sessions;
