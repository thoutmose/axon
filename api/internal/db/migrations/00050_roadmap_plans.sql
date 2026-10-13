-- +goose Up
-- A roadmap started from a plan (EP-95) remembers which plan and when it
-- starts, so the calendar shows its weekly study blocks (EP-97).
ALTER TABLE roadmaps ADD COLUMN plan_id text, ADD COLUMN plan_start date;
ALTER TABLE roadmaps ADD CONSTRAINT roadmaps_plan CHECK ((plan_id IS NULL) = (plan_start IS NULL));
ALTER TABLE calendar_feeds DROP CONSTRAINT calendar_feeds_types_check;
ALTER TABLE calendar_feeds ADD CONSTRAINT calendar_feeds_types_check
    CHECK (cardinality(types) > 0 AND types <@ ARRAY['task', 'goal', 'milestone', 'study']);

-- +goose Down
UPDATE calendar_feeds SET types = array_remove(types, 'study');
DELETE FROM calendar_feeds WHERE cardinality(types) = 0;
ALTER TABLE calendar_feeds DROP CONSTRAINT calendar_feeds_types_check;
ALTER TABLE calendar_feeds ADD CONSTRAINT calendar_feeds_types_check
    CHECK (cardinality(types) > 0 AND types <@ ARRAY['task', 'goal', 'milestone']);
ALTER TABLE roadmaps DROP CONSTRAINT roadmaps_plan;
ALTER TABLE roadmaps DROP COLUMN plan_id, DROP COLUMN plan_start;
