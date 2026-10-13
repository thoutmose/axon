-- +goose Up
-- The daily journal (EP-91): how a day felt, how long the night was, what
-- the person is grateful for. Mood and sleep are health data: kept only with
-- the health_data consent and deleted when it is withdrawn.
CREATE TABLE journal_entries (
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day         date NOT NULL,
    mood        smallint CHECK (mood BETWEEN 1 AND 5),
    energy      smallint CHECK (energy BETWEEN 1 AND 5),
    stress      smallint CHECK (stress BETWEEN 1 AND 5),
    sleep_hours numeric(3, 1) CHECK (sleep_hours BETWEEN 0 AND 24),
    gratitude   text[] NOT NULL DEFAULT '{}' CHECK (cardinality(gratitude) <= 3),
    note        text NOT NULL DEFAULT '' CHECK (length(note) <= 10000),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, day)
);

-- +goose Down
DROP TABLE journal_entries;
