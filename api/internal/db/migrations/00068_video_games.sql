-- +goose Up
-- Video games beside films, series, and books (EP-149), looked up in
-- Wikipedia, which needs no key; their covers are copied like the others.
ALTER TABLE media_items DROP CONSTRAINT media_items_kind_check,
    ADD CONSTRAINT media_items_kind_check CHECK (kind IN ('film', 'series', 'book', 'game')),
    DROP CONSTRAINT media_items_source_check,
    ADD CONSTRAINT media_items_source_check CHECK (source IN ('manual', 'tmdb', 'openlibrary', 'wikipedia'));
ALTER TABLE media_covers DROP CONSTRAINT media_covers_source_check,
    ADD CONSTRAINT media_covers_source_check CHECK (source IN ('tmdb', 'openlibrary', 'wikipedia'));
ALTER TABLE media_lookups DROP CONSTRAINT media_lookups_source_check,
    ADD CONSTRAINT media_lookups_source_check CHECK (source IN ('tmdb', 'openlibrary', 'wikipedia'));

-- +goose Down
DELETE FROM media_items WHERE kind = 'game' OR source = 'wikipedia';
DELETE FROM media_covers WHERE source = 'wikipedia';
DELETE FROM media_lookups WHERE source = 'wikipedia';
ALTER TABLE media_items DROP CONSTRAINT media_items_kind_check,
    ADD CONSTRAINT media_items_kind_check CHECK (kind IN ('film', 'series', 'book')),
    DROP CONSTRAINT media_items_source_check,
    ADD CONSTRAINT media_items_source_check CHECK (source IN ('manual', 'tmdb', 'openlibrary'));
ALTER TABLE media_covers DROP CONSTRAINT media_covers_source_check,
    ADD CONSTRAINT media_covers_source_check CHECK (source IN ('tmdb', 'openlibrary'));
ALTER TABLE media_lookups DROP CONSTRAINT media_lookups_source_check,
    ADD CONSTRAINT media_lookups_source_check CHECK (source IN ('tmdb', 'openlibrary'));
