-- +goose Up
-- Coffee and tea (EP-117): caffeine taken, when, from what. Health data,
-- kept only with the health_data consent, like water.
CREATE TABLE caffeine_entries (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    drunk_at   timestamptz NOT NULL,
    kind       text NOT NULL CHECK (kind IN ('espresso', 'coffee', 'instant', 'tea', 'green_tea', 'matcha', 'cola', 'energy', 'decaf', 'other')),
    mg         integer NOT NULL CHECK (mg BETWEEN 1 AND 1000),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX caffeine_entries_at ON caffeine_entries (user_id, drunk_at);

-- +goose Down
DROP TABLE caffeine_entries;
