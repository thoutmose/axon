-- +goose Up
-- A video's or an episode's words are kept apart from what is shown
-- (EP-148): the item shows five to ten of its sentences and five to ten
-- keywords; the words still feed its analysis. Transcripts that were
-- appended to the content move out, and those items are read again so they
-- get their summary.
ALTER TABLE watch_items ADD COLUMN transcript text NOT NULL DEFAULT '' CHECK (length(transcript) <= 200000);
UPDATE watch_items
   SET content = rtrim(left(content, strpos(content, E'\n\n## Transcript\n') - 1)),
       transcript_source = '', transcript_lang = '', transcript_tried_at = NULL
 WHERE strpos(content, E'\n\n## Transcript\n') > 0;

-- +goose Down
UPDATE watch_items SET content = content || E'\n\n## Transcript\n\n' || transcript || E'\n' WHERE transcript <> '';
ALTER TABLE watch_items DROP COLUMN transcript;
