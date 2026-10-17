-- +goose Up
-- The version of the library plan a roadmap was started from (EP-146), so a
-- roadmap started from an earlier one is recognised whatever its titles.
-- Version 1 started on its own date, 5 October 2026, and its first phase has
-- not ended, so every roadmap started from it has that date; version 2
-- starts on 19 October 2026 or a later Monday.
ALTER TABLE roadmaps ADD COLUMN plan_version integer;
UPDATE roadmaps SET plan_version = CASE WHEN plan_start < '2026-10-19' THEN 1 ELSE 2 END
    WHERE plan_id IS NOT NULL;

-- +goose Down
ALTER TABLE roadmaps DROP COLUMN plan_version;
