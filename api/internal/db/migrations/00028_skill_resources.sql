-- +goose Up
-- A skill is a competency, not a course: the books, courses, and sites that
-- teach it are listed on the skill (owner request, EP-42). Imported skills
-- carry their resources here; older imports keep one resource per skill.
ALTER TABLE skills ADD COLUMN resources jsonb NOT NULL DEFAULT '[]'
    CHECK (jsonb_typeof(resources) = 'array' AND jsonb_array_length(resources) <= 50);

-- +goose Down
ALTER TABLE skills DROP COLUMN resources;
