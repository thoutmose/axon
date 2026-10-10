-- +goose Up
-- Every linkable thing a user owns (ADR 0006). Domain tables use entities.id as their primary key.
CREATE TABLE entities (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type         text NOT NULL,
    title        text NOT NULL DEFAULT '' CHECK (length(title) <= 500),
    search_text  text NOT NULL DEFAULT '',
    search       tsvector GENERATED ALWAYS AS (to_tsvector('simple', title || ' ' || search_text)) STORED,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, id)
);
CREATE INDEX entities_user_type ON entities (user_id, type, id);
CREATE INDEX entities_search ON entities USING gin (search);

CREATE TABLE links (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL,
    source_id  uuid NOT NULL,
    relation   text NOT NULL,
    target_id  uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Both ends must belong to the link's owner; deleting either end deletes the link.
    FOREIGN KEY (user_id, source_id) REFERENCES entities (user_id, id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, target_id) REFERENCES entities (user_id, id) ON DELETE CASCADE,
    CHECK (source_id <> target_id),
    UNIQUE (source_id, relation, target_id)
);
CREATE INDEX links_outgoing ON links (user_id, source_id, id);
CREATE INDEX links_incoming ON links (user_id, target_id, id);

-- Append-only record of meaningful changes: field names only, never values.
CREATE TABLE events (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor        text NOT NULL,
    entity_type  text NOT NULL,
    entity_id    uuid NOT NULL,
    kind         text NOT NULL CHECK (kind IN ('created', 'updated', 'deleted', 'linked', 'unlinked')),
    fields       text[] NOT NULL DEFAULT '{}',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX events_entity ON events (user_id, entity_id, id DESC);
CREATE INDEX events_user ON events (user_id, id DESC);

-- +goose StatementBegin
CREATE FUNCTION events_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'events is append-only';
END $$;
-- +goose StatementEnd
CREATE TRIGGER events_no_update BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION events_append_only();

-- +goose Down
DROP TRIGGER events_no_update ON events;
DROP FUNCTION events_append_only();
DROP TABLE events;
DROP TABLE links;
DROP TABLE entities;
