-- +goose Up
-- Goals with an optional measurable outcome, and their milestones. Progress
-- is computed on read from the measure or from linked work, never stored.
CREATE TABLE goals (
    id              uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    outcome         text NOT NULL DEFAULT '' CHECK (length(outcome) <= 20000),
    status          text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'achieved', 'archived')),
    target_date     date,
    -- A measure is all four columns or none: "run 0 → 10 km, now 4".
    measure_unit    text CHECK (length(measure_unit) <= 40),
    measure_start   double precision,
    measure_target  double precision,
    measure_current double precision,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CHECK ((measure_target IS NULL) = (measure_start IS NULL)
       AND (measure_target IS NULL) = (measure_current IS NULL)
       AND (measure_target IS NULL) = (measure_unit IS NULL)),
    CHECK (measure_target IS NULL OR measure_target <> measure_start)
);
CREATE INDEX goals_status ON goals (status);

CREATE TABLE milestones (
    id         uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    goal_id    uuid NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    due_on     date,
    done_at    timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX milestones_goal ON milestones (goal_id, due_on);

-- +goose Down
DROP TABLE milestones;
DROP TABLE goals;
