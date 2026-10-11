-- +goose Up
-- Savings pots (EP-45), the "tirelire": money set aside for something to
-- buy later, with a target and an optional date. The pot's balance is the
-- sum of what was put in and taken out.
CREATE TABLE budget_pots (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    target_cents bigint NOT NULL CHECK (target_cents > 0 AND target_cents <= 100000000000),
    target_date  date,
    note         text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
    archived     boolean NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX budget_pots_user ON budget_pots (user_id);

CREATE TABLE budget_pot_moves (
    id           uuid PRIMARY KEY,
    pot_id       uuid NOT NULL REFERENCES budget_pots(id) ON DELETE CASCADE,
    occurred_on  date NOT NULL,
    amount_cents bigint NOT NULL CHECK (amount_cents <> 0 AND abs(amount_cents) <= 100000000000), -- in, or out when negative
    note         text NOT NULL DEFAULT '' CHECK (length(note) <= 500),
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX budget_pot_moves_pot ON budget_pot_moves (pot_id, occurred_on);

-- +goose Down
DROP TABLE budget_pot_moves;
DROP TABLE budget_pots;
