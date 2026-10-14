-- +goose Up
-- Tech watch (EP-73): the sources a person follows, on a schedule, and the
-- news they bring, each kept once. A news item is an entity (type
-- feed_item), so it can be linked to tasks, projects, and goals, and found
-- by search; it becomes a note in the vault only when the person keeps it.

-- Domains the person watches; their keywords label the news.
CREATE TABLE watch_topics (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
    keywords   text[] NOT NULL DEFAULT '{}' CHECK (cardinality(keywords) <= 50),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE TABLE watch_sources (
    id              uuid PRIMARY KEY,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url             text NOT NULL CHECK (length(url) BETWEEN 1 AND 2000),
    site_url        text NOT NULL DEFAULT '' CHECK (length(site_url) <= 2000),
    title           text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    topic_ids       uuid[] NOT NULL DEFAULT '{}',
    schedule        text NOT NULL DEFAULT '0 0 * * *' CHECK (length(schedule) <= 100),
    enabled         boolean NOT NULL DEFAULT true,
    full_text       boolean NOT NULL DEFAULT false, -- fetch each article's page for its text
    origin          text NOT NULL DEFAULT 'manual' CHECK (origin IN ('manual', 'csv', 'opml', 'freshrss', 'agent')),
    etag            text NOT NULL DEFAULT '' CHECK (length(etag) <= 500),
    last_modified   text NOT NULL DEFAULT '' CHECK (length(last_modified) <= 100),
    next_fetch_at   timestamptz NOT NULL DEFAULT now(),
    last_fetched_at timestamptz,
    last_ok_at      timestamptz,
    last_error      text NOT NULL DEFAULT '' CHECK (length(last_error) <= 500),
    failures        integer NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, url)
);
CREATE INDEX watch_sources_due ON watch_sources (next_fetch_at) WHERE enabled;

CREATE TABLE watch_items (
    id           uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_id    uuid REFERENCES watch_sources(id) ON DELETE SET NULL,
    source_title text NOT NULL DEFAULT '',
    guid         text NOT NULL DEFAULT '' CHECK (length(guid) <= 2000),
    url          text NOT NULL CHECK (length(url) <= 2000),
    url_hash     bytea NOT NULL,
    title        text NOT NULL CHECK (length(title) <= 500),
    author       text NOT NULL DEFAULT '' CHECK (length(author) <= 200),
    published_at timestamptz NOT NULL,
    fetched_at   timestamptz NOT NULL DEFAULT now(),
    summary      text NOT NULL DEFAULT '' CHECK (length(summary) <= 5000),
    content      text NOT NULL DEFAULT '' CHECK (length(content) <= 200000), -- Markdown
    image        text NOT NULL DEFAULT '' CHECK (length(image) <= 2000),
    audio        text NOT NULL DEFAULT '' CHECK (length(audio) <= 2000),
    lang         text NOT NULL DEFAULT '',
    words        integer NOT NULL DEFAULT 0,
    simhash      bigint NOT NULL DEFAULT 0,
    terms        text[] NOT NULL DEFAULT '{}', -- distinct stems, for near-duplicates and analysis
    keywords     text[] NOT NULL DEFAULT '{}',
    labels       text[] NOT NULL DEFAULT '{}', -- applied by Axon's NLP and by the person
    categories   text[] NOT NULL DEFAULT '{}', -- as the feed gives them
    topic_ids    uuid[] NOT NULL DEFAULT '{}',
    score        real NOT NULL DEFAULT 0,      -- how well it fits the person's topics
    outlier      text NOT NULL DEFAULT '',     -- why it stands apart (too short, mostly links…), or empty
    also_in      text[] NOT NULL DEFAULT '{}', -- other sources that carried the same story (it is stored once)
    status       text NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'saved', 'archived')),
    note_id      uuid REFERENCES notes(id) ON DELETE SET NULL,
    UNIQUE (user_id, url_hash)
);
CREATE INDEX watch_items_recent ON watch_items (user_id, published_at DESC);
CREATE INDEX watch_items_source ON watch_items (source_id, guid);
CREATE INDEX watch_items_labels ON watch_items USING gin (labels);

-- What each source already delivered, so a feed's old entries are skipped
-- cheaply, duplicates included.
CREATE TABLE watch_seen (
    source_id uuid NOT NULL REFERENCES watch_sources(id) ON DELETE CASCADE,
    guid_hash bytea NOT NULL,
    seen_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source_id, guid_hash)
);

-- +goose Down
DROP TABLE watch_seen;
DROP TABLE watch_items;
DROP TABLE watch_sources;
DROP TABLE watch_topics;
