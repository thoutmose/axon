-- +goose Up
-- Media trackers (EP-17): films, series, and books a person wants, is on, or
-- has finished. Catalogue data (TMDB, Open Library) is cached, never relied
-- on: every item can be entered by hand.

-- Covers are fetched once on the server and served from Axon, so pages never
-- load images from another origin. Shared by everyone on the instance.
CREATE TABLE media_covers (
    id           uuid PRIMARY KEY,
    source       text NOT NULL CHECK (source IN ('tmdb', 'openlibrary')),
    source_key   text NOT NULL CHECK (length(source_key) <= 200),
    content_type text NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),
    data         bytea NOT NULL CHECK (length(data) <= 2097152),
    fetched_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, source_key)
);

CREATE TABLE media_items (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    kind        text NOT NULL CHECK (kind IN ('film', 'series', 'book')),
    status      text NOT NULL DEFAULT 'want' CHECK (status IN ('want', 'doing', 'done', 'dropped')),
    year        integer CHECK (year BETWEEN 1000 AND 3000),
    creators    text NOT NULL DEFAULT '' CHECK (length(creators) <= 500),
    genres      text[] NOT NULL DEFAULT '{}' CHECK (cardinality(genres) <= 12),
    rating      smallint CHECK (rating BETWEEN 1 AND 10),
    progress    integer NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100000),
    total       integer CHECK (total BETWEEN 1 AND 100000),
    started_on  date,
    finished_on date,
    review      text NOT NULL DEFAULT '' CHECK (length(review) <= 10000),
    source      text NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'tmdb', 'openlibrary')),
    source_id   text NOT NULL DEFAULT '' CHECK (length(source_id) <= 100),
    overview    text NOT NULL DEFAULT '' CHECK (length(overview) <= 4000),
    cover_id    uuid REFERENCES media_covers(id) ON DELETE SET NULL,
    fetched_at  timestamptz, -- when catalogue data was cached; TMDB allows at most six months
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX media_items_kind_status ON media_items (kind, status);
CREATE INDEX media_items_cover ON media_items (cover_id);

-- Catalogue searches, cached for a week to spare the providers.
CREATE TABLE media_lookups (
    source     text NOT NULL CHECK (source IN ('tmdb', 'openlibrary')),
    query      text NOT NULL CHECK (length(query) <= 300),
    result     jsonb NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source, query)
);

-- +goose Down
DROP TABLE media_lookups;
DROP TABLE media_items;
DROP TABLE media_covers;
