-- +goose Up
-- A domain is proven, not ticked (EP-181, after École 42's defenses): each
-- attempt at a skill's evaluation sheet keeps its items, the explanation
-- given, and the mark.
CREATE TABLE skill_validations (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    skill_id uuid NOT NULL REFERENCES skills (id) ON DELETE CASCADE,
    attempt integer NOT NULL,
    items jsonb NOT NULL,
    explanation text NOT NULL DEFAULT '',
    mark integer NOT NULL,
    flag text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (skill_id, attempt)
);
CREATE INDEX skill_validations_user ON skill_validations (user_id);

-- +goose Down
DROP TABLE skill_validations;
