-- +goose Up
-- Daily needs follow a bounded goal (EP-191, ADR 0039): keep one's weight,
-- lose it gently, or gain. Health data, with the rest of the profile.
ALTER TABLE health_profiles ADD COLUMN goal text NOT NULL DEFAULT 'maintain' CHECK (goal IN ('maintain', 'lose', 'gain'));

-- +goose Down
ALTER TABLE health_profiles DROP COLUMN goal;
