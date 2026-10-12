-- +goose Up
-- Atomic Habits (EP-34): habits to build or break, the identity they vote
-- for, an implementation intention (time and place), an optional habit
-- contract, and the habits scorecard.
ALTER TABLE habits
    ADD COLUMN kind text NOT NULL DEFAULT 'build' CHECK (kind IN ('build', 'break')),
    ADD COLUMN identity text NOT NULL DEFAULT '' CHECK (length(identity) <= 500),
    ADD COLUMN intention_time text NOT NULL DEFAULT '' CHECK (intention_time ~ '^(([01][0-9]|2[0-3]):[0-5][0-9])?$'),
    ADD COLUMN intention_place text NOT NULL DEFAULT '' CHECK (length(intention_place) <= 500),
    ADD COLUMN contract_partner text NOT NULL DEFAULT '' CHECK (length(contract_partner) <= 200),
    ADD COLUMN contract_pledge text NOT NULL DEFAULT '' CHECK (length(contract_pledge) <= 2000);

CREATE TABLE habit_scorecards (
    user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    items      jsonb NOT NULL DEFAULT '[]',
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE habit_scorecards;
ALTER TABLE habits DROP COLUMN kind, DROP COLUMN identity, DROP COLUMN intention_time,
    DROP COLUMN intention_place, DROP COLUMN contract_partner, DROP COLUMN contract_pledge;
