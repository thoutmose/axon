-- +goose Up
-- PARA areas and resources as their own entities (owner request, EP-42).
-- An area is a responsibility kept to a standard (health, finances); a
-- resource is a topic or reference of ongoing interest. Archiving is a
-- status, so the Archive page can bring either back.
CREATE TABLE areas (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    description text NOT NULL DEFAULT '' CHECK (length(description) <= 20000),
    url         text CHECK (url ~ '^https?://' AND length(url) <= 1000),
    status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE resources (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    description text NOT NULL DEFAULT '' CHECK (length(description) <= 20000),
    url         text CHECK (url ~ '^https?://' AND length(url) <= 1000),
    status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE resources;
DROP TABLE areas;
