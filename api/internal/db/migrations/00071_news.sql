-- +goose Up
-- General news apart from the tech watch (EP-155): a source can be news,
-- whose items stay out of the tech watch and make the news digest.
ALTER TABLE watch_sources ADD COLUMN news boolean NOT NULL DEFAULT false;
ALTER TABLE reminders_sent DROP CONSTRAINT reminders_sent_kind_check;
ALTER TABLE reminders_sent ADD CONSTRAINT reminders_sent_kind_check CHECK (kind IN ('morning', 'evening', 'weekly', 'monthly', 'news'));

-- +goose Down
DELETE FROM reminders_sent WHERE kind = 'news';
ALTER TABLE reminders_sent DROP CONSTRAINT reminders_sent_kind_check;
ALTER TABLE reminders_sent ADD CONSTRAINT reminders_sent_kind_check CHECK (kind IN ('morning', 'evening', 'weekly', 'monthly'));
ALTER TABLE watch_sources DROP COLUMN news;
