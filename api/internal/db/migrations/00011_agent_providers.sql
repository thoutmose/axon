-- +goose Up
-- Which AI provider runs a person's agents (EP-22). Without a row, the
-- server's default provider (if any) is used. Keys are sealed with
-- AXON_ENCRYPTION_KEY and never returned by the API.
CREATE TABLE agent_providers (
    user_id        uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    provider       text NOT NULL CHECK (provider IN ('anthropic', 'openai', 'mistral', 'gemini', 'deepseek', 'local')),
    model          text NOT NULL CHECK (length(model) BETWEEN 1 AND 100),
    base_url       text CHECK (length(base_url) <= 300),   -- local servers only
    api_key_sealed bytea,
    -- US dollars per million tokens, for the budget; NULL means Axon's default.
    input_price    double precision CHECK (input_price >= 0 AND input_price <= 1000),
    output_price   double precision CHECK (output_price >= 0 AND output_price <= 1000),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE agent_providers;
