-- +goose Up
-- Transcripts of YouTube videos (EP-123): where a video's words came from
-- (YouTube's captions, its automatic captions, or the operator's Whisper
-- server), their language, and when Axon last tried, so a video without
-- any is not asked for again and again.
ALTER TABLE watch_items
    ADD COLUMN transcript_source text NOT NULL DEFAULT '' CHECK (transcript_source IN ('', 'captions', 'auto_captions', 'whisper')),
    ADD COLUMN transcript_lang text NOT NULL DEFAULT '' CHECK (length(transcript_lang) <= 10),
    ADD COLUMN transcript_tried_at timestamptz;
CREATE INDEX watch_items_untranscribed ON watch_items (published_at DESC)
    WHERE transcript_tried_at IS NULL AND url LIKE 'https://youtube.com/watch?%';

-- +goose Down
DROP INDEX watch_items_untranscribed;
ALTER TABLE watch_items DROP COLUMN transcript_source, DROP COLUMN transcript_lang, DROP COLUMN transcript_tried_at;
