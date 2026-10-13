-- +goose Up
-- Daily needs (EP-88): what the reference intakes are worked out from.
-- Special-category data, kept only with the health_data consent and deleted
-- with the rest of the health data when it is withdrawn.
CREATE TABLE health_profiles (
    user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    birth_year smallint NOT NULL CHECK (birth_year BETWEEN 1900 AND 2100),
    sex        text NOT NULL CHECK (sex IN ('female', 'male', 'unspecified')),
    height_cm  numeric(5, 1) NOT NULL CHECK (height_cm BETWEEN 100 AND 250),
    weight_kg  numeric(5, 1) NOT NULL CHECK (weight_kg BETWEEN 25 AND 350),
    activity   text NOT NULL CHECK (activity IN ('sedentary', 'light', 'moderate', 'active', 'very_active')),
    pregnant   boolean NOT NULL DEFAULT false,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE health_profiles;
