-- +goose Up
-- Water drunk (EP-101): health data, kept only with the health_data consent.
CREATE TABLE water_entries (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    drunk_on   date NOT NULL,
    ml         integer NOT NULL CHECK (ml BETWEEN 1 AND 3000),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX water_entries_day ON water_entries (user_id, drunk_on);

-- +goose Down
DROP TABLE water_entries;
