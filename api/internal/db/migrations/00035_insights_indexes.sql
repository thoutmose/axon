-- +goose Up
-- Insights (EP-20) read a year of completions and reviews by date.
CREATE INDEX tasks_completed ON tasks (completed_at) WHERE completed_at IS NOT NULL;
CREATE INDEX milestones_done ON milestones (done_at) WHERE done_at IS NOT NULL;
CREATE INDEX card_reviews_when ON card_reviews (reviewed_at);
CREATE INDEX entities_user_type_created ON entities (user_id, type, created_at);

-- +goose Down
DROP INDEX entities_user_type_created;
DROP INDEX card_reviews_when;
DROP INDEX milestones_done;
DROP INDEX tasks_completed;
