-- +goose Up
-- Learning roadmaps and their skills (EP-14). Prerequisites are graph links
-- with the "requires" relation, so skills also link to goals and notes.
CREATE TABLE roadmaps (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    description text NOT NULL DEFAULT '' CHECK (length(description) <= 20000),
    source      text NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'ossu')),
    attribution text NOT NULL DEFAULT '' CHECK (length(attribution) <= 2000), -- shown with imported content
    license_url text CHECK (length(license_url) <= 500),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE skills (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    roadmap_id  uuid NOT NULL REFERENCES roadmaps(id) ON DELETE CASCADE,
    description text NOT NULL DEFAULT '' CHECK (length(description) <= 5000),
    url         text CHECK (url ~ '^https?://' AND length(url) <= 1000),
    section     text NOT NULL DEFAULT '' CHECK (length(section) <= 200),
    status      text NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'learning', 'done', 'skipped')),
    position    integer NOT NULL DEFAULT 0,
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX skills_roadmap ON skills (roadmap_id, position);

-- +goose Down
DROP TABLE skills;
DROP TABLE roadmaps;
