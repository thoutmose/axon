-- +goose Up
-- Budget (EP-39, first slice; owner request EP-42): categories with a
-- monthly budget, and income and expense transactions. Amounts are integer
-- cents, never floats. Finance data is not in the entity graph, so search
-- and agents never see it; it is exported with the account and erased with
-- the user.
CREATE TABLE budget_profiles (
    user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    currency   text NOT NULL DEFAULT 'EUR' CHECK (currency ~ '^[A-Z]{3}$'),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE budget_categories (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    kind         text NOT NULL CHECK (kind IN ('income', 'expense')),
    monthly_cents bigint CHECK (monthly_cents >= 0 AND monthly_cents <= 100000000000),
    archived     boolean NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, kind, name)
);

CREATE TABLE budget_transactions (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    occurred_on  date NOT NULL,
    kind         text NOT NULL CHECK (kind IN ('income', 'expense')),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0 AND amount_cents <= 100000000000),
    category_id  uuid REFERENCES budget_categories(id) ON DELETE SET NULL,
    payee        text NOT NULL DEFAULT '' CHECK (length(payee) <= 200),
    note         text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX budget_transactions_month ON budget_transactions (user_id, occurred_on DESC, id);

-- +goose Down
DROP TABLE budget_transactions;
DROP TABLE budget_categories;
DROP TABLE budget_profiles;
