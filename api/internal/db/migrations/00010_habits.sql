-- +goose Up
-- Habits (EP-12), designed with the four laws of behaviour change: a cue, a
-- craving, the response (the entity title), and a reward, plus a habit stack
-- and a two-minute version for hard days.
CREATE TABLE habits (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    cue         text NOT NULL DEFAULT '' CHECK (length(cue) <= 500),
    craving     text NOT NULL DEFAULT '' CHECK (length(craving) <= 500),
    reward      text NOT NULL DEFAULT '' CHECK (length(reward) <= 500),
    stack_after text NOT NULL DEFAULT '' CHECK (length(stack_after) <= 500), -- "After I pour my coffee"
    two_minute  text NOT NULL DEFAULT '' CHECK (length(two_minute) <= 500),  -- "Put on running shoes"
    -- ISO weekdays it is scheduled on (1 = Monday … 7 = Sunday), in the user's time zone.
    days        smallint[] NOT NULL DEFAULT '{1,2,3,4,5,6,7}'
                CHECK (cardinality(days) BETWEEN 1 AND 7 AND days <@ '{1,2,3,4,5,6,7}'::smallint[]),
    status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'archived')),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- One check-in per habit and local calendar day: travelling across time
-- zones or a daylight-saving change can never count a day twice.
CREATE TABLE habit_checkins (
    habit_id   uuid NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    day        date NOT NULL,
    status     text NOT NULL CHECK (status IN ('done', 'two_minute', 'skipped')),
    note       text NOT NULL DEFAULT '' CHECK (length(note) <= 1000),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (habit_id, day)
);

-- +goose Down
DROP TABLE habit_checkins;
DROP TABLE habits;
