-- +goose Up
-- Podcasts from the person's Audiobookshelf (EP-124): the server and its
-- API token, sealed when AXON_ENCRYPTION_KEY is set (first byte 0: plain,
-- 1: sealed) and never shown again. Each show becomes a tech watch source
-- of origin audiobookshelf; a podcast episode waits for Whisper like a video.
CREATE TABLE audiobookshelf_servers (
    user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    url        text NOT NULL CHECK (length(url) BETWEEN 1 AND 500),
    token      bytea NOT NULL,
    synced_at  timestamptz,
    error      text NOT NULL DEFAULT '' CHECK (length(error) <= 500),
    created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE watch_sources DROP CONSTRAINT watch_sources_origin_check,
    ADD CONSTRAINT watch_sources_origin_check CHECK (origin IN ('manual', 'csv', 'opml', 'freshrss', 'agent', 'audiobookshelf'));
CREATE INDEX watch_items_unheard ON watch_items (published_at DESC) WHERE transcript_tried_at IS NULL AND audio <> '';

-- +goose Down
DROP INDEX watch_items_unheard;
DELETE FROM watch_sources WHERE origin = 'audiobookshelf';
ALTER TABLE watch_sources DROP CONSTRAINT watch_sources_origin_check,
    ADD CONSTRAINT watch_sources_origin_check CHECK (origin IN ('manual', 'csv', 'opml', 'freshrss', 'agent'));
DROP TABLE audiobookshelf_servers;
