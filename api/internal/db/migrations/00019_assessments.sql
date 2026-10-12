-- +goose Up
-- Skills assessments (EP-41): the person's situation and their level per
-- domain, taken before (and again during) their roadmaps.
CREATE TABLE assessments (
    id         uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    profile    jsonb NOT NULL,
    results    jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE assessments;
