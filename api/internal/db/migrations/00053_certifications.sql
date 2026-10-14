-- +goose Up
-- Certifications (EP-104): exams planned, booked, passed, or failed.
CREATE TABLE certifications (
    id              uuid PRIMARY KEY,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            text NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
    provider        text NOT NULL DEFAULT '' CHECK (length(provider) <= 100),
    code            text NOT NULL DEFAULT '' CHECK (length(code) <= 40),
    url             text CHECK (url ~ '^https?://' AND length(url) <= 2000),
    status          text NOT NULL CHECK (status IN ('planned', 'booked', 'passed', 'failed')),
    target_on       date,
    exam_on         date,
    passed_on       date,
    expires_on      date,
    validity_months integer CHECK (validity_months BETWEEN 1 AND 240),
    score           text NOT NULL DEFAULT '' CHECK (length(score) <= 40),
    cost_cents      bigint CHECK (cost_cents >= 0),
    currency        text NOT NULL DEFAULT 'USD' CHECK (currency ~ '^[A-Z]{3}$'),
    credential_url  text CHECK (credential_url ~ '^https?://' AND length(credential_url) <= 2000),
    notes           text NOT NULL DEFAULT '' CHECK (length(notes) <= 5000),
    skill_id        uuid REFERENCES skills(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX certifications_user ON certifications (user_id);
CREATE INDEX certifications_skill ON certifications (skill_id);
ALTER TABLE calendar_feeds DROP CONSTRAINT calendar_feeds_types_check;
ALTER TABLE calendar_feeds ADD CONSTRAINT calendar_feeds_types_check
    CHECK (cardinality(types) > 0 AND types <@ ARRAY['task', 'goal', 'milestone', 'study', 'certification']);

-- +goose Down
UPDATE calendar_feeds SET types = array_remove(types, 'certification');
DELETE FROM calendar_feeds WHERE cardinality(types) = 0;
ALTER TABLE calendar_feeds DROP CONSTRAINT calendar_feeds_types_check;
ALTER TABLE calendar_feeds ADD CONSTRAINT calendar_feeds_types_check
    CHECK (cardinality(types) > 0 AND types <@ ARRAY['task', 'goal', 'milestone', 'study']);
DROP TABLE certifications;
