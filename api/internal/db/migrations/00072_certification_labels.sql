-- +goose Up
-- Labels on certifications (owner request): the field an exam belongs to
-- (data engineering, cloud, security…) and its vendor, to filter by.
ALTER TABLE certifications ADD COLUMN labels text[] NOT NULL DEFAULT '{}' CHECK (cardinality(labels) <= 20);

-- +goose Down
ALTER TABLE certifications DROP COLUMN labels;
