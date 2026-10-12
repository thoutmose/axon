-- +goose Up
-- Personal settings (EP-31): language, appearance, and templates. AI provider
-- and budget stay in their own tables; keys are never stored here.
CREATE TABLE user_settings (
    user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    doc        jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_settings;
