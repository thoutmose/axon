-- +goose Up
-- Personal finance in the spirit of Firefly III (EP-39): accounts with
-- balances, transfers between them, tags, recurring transactions, rules
-- that categorise automatically, and idempotent CSV import. Still integer
-- cents, still outside the entity graph.
CREATE TABLE budget_accounts (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    type          text NOT NULL CHECK (type IN ('asset', 'liability')),
    role          text NOT NULL DEFAULT 'checking'
                  CHECK (role IN ('checking', 'savings', 'cash', 'credit_card', 'loan', 'investment', 'other')),
    opening_cents bigint NOT NULL DEFAULT 0 CHECK (abs(opening_cents) <= 100000000000),
    archived      boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE TABLE budget_recurring (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    kind         text NOT NULL CHECK (kind IN ('income', 'expense', 'transfer')),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0 AND amount_cents <= 100000000000),
    account_id   uuid REFERENCES budget_accounts(id) ON DELETE RESTRICT,
    to_account_id uuid REFERENCES budget_accounts(id) ON DELETE RESTRICT,
    category_id  uuid REFERENCES budget_categories(id) ON DELETE SET NULL,
    payee        text NOT NULL DEFAULT '' CHECK (length(payee) <= 200),
    note         text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
    tags         text[] NOT NULL DEFAULT '{}',
    rule         text NOT NULL CHECK (length(rule) <= 200), -- RFC 5545 RRULE
    next_on      date,                                      -- null once the series ended
    active       boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (kind <> 'transfer' OR (account_id IS NOT NULL AND to_account_id IS NOT NULL AND account_id <> to_account_id))
);

CREATE TABLE budget_rules (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    field       text NOT NULL CHECK (field IN ('payee', 'note')),
    operator    text NOT NULL CHECK (operator IN ('contains', 'equals', 'starts_with')),
    value       text NOT NULL CHECK (length(value) BETWEEN 1 AND 200),
    category_id uuid REFERENCES budget_categories(id) ON DELETE CASCADE,
    tag         text NOT NULL DEFAULT '' CHECK (tag ~ '^[a-z0-9_-]{0,40}$'),
    position    integer NOT NULL DEFAULT 0,
    active      boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (category_id IS NOT NULL OR tag <> '')
);

ALTER TABLE budget_transactions DROP CONSTRAINT budget_transactions_kind_check;
ALTER TABLE budget_transactions
    ADD CONSTRAINT budget_transactions_kind_check CHECK (kind IN ('income', 'expense', 'transfer')),
    ADD COLUMN account_id    uuid REFERENCES budget_accounts(id) ON DELETE RESTRICT,
    ADD COLUMN to_account_id uuid REFERENCES budget_accounts(id) ON DELETE RESTRICT,
    ADD COLUMN tags          text[] NOT NULL DEFAULT '{}',
    ADD COLUMN recurring_id  uuid REFERENCES budget_recurring(id) ON DELETE SET NULL,
    ADD COLUMN import_hash   text,
    ADD CONSTRAINT budget_transfer_accounts
        CHECK (kind <> 'transfer' OR (account_id IS NOT NULL AND to_account_id IS NOT NULL AND account_id <> to_account_id AND category_id IS NULL)),
    ADD CONSTRAINT budget_to_account_only_transfers CHECK (kind = 'transfer' OR to_account_id IS NULL);
-- Re-running recurring transactions or re-importing a file creates nothing twice.
CREATE UNIQUE INDEX budget_transactions_recurring ON budget_transactions (recurring_id, occurred_on) WHERE recurring_id IS NOT NULL;
CREATE UNIQUE INDEX budget_transactions_import ON budget_transactions (user_id, import_hash) WHERE import_hash IS NOT NULL;
CREATE INDEX budget_transactions_account ON budget_transactions (account_id);
CREATE INDEX budget_transactions_to_account ON budget_transactions (to_account_id);

-- +goose Down
DROP INDEX budget_transactions_to_account;
DROP INDEX budget_transactions_account;
DROP INDEX budget_transactions_import;
DROP INDEX budget_transactions_recurring;
DELETE FROM budget_transactions WHERE kind = 'transfer';
ALTER TABLE budget_transactions
    DROP CONSTRAINT budget_to_account_only_transfers,
    DROP CONSTRAINT budget_transfer_accounts,
    DROP COLUMN import_hash,
    DROP COLUMN recurring_id,
    DROP COLUMN tags,
    DROP COLUMN to_account_id,
    DROP COLUMN account_id,
    DROP CONSTRAINT budget_transactions_kind_check;
ALTER TABLE budget_transactions ADD CONSTRAINT budget_transactions_kind_check CHECK (kind IN ('income', 'expense'));
DROP TABLE budget_rules;
DROP TABLE budget_recurring;
DROP TABLE budget_accounts;
