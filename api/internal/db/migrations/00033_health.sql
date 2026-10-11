-- +goose Up
-- Health logging (EP-18): food and training. Special-category data under GDPR
-- Art. 9, stored only with the health_data consent and deleted when it is
-- withdrawn. Kept out of the graph, so it never appears in search, links,
-- or agent tools.

-- Foods found in Open Food Facts or USDA FoodData Central, per 100 g.
-- Shared by everyone on the instance; nothing in it is personal.
CREATE TABLE foods (
    id          uuid PRIMARY KEY,
    source      text NOT NULL CHECK (source IN ('off', 'usda')),
    source_id   text NOT NULL CHECK (length(source_id) <= 40),
    name        text NOT NULL CHECK (length(name) BETWEEN 1 AND 300),
    brand       text NOT NULL DEFAULT '' CHECK (length(brand) <= 200),
    kcal        numeric(7, 2) CHECK (kcal BETWEEN 0 AND 1000),
    protein_g   numeric(6, 2) CHECK (protein_g BETWEEN 0 AND 100),
    carbs_g     numeric(6, 2) CHECK (carbs_g BETWEEN 0 AND 100),
    fat_g       numeric(6, 2) CHECK (fat_g BETWEEN 0 AND 100),
    fiber_g     numeric(6, 2) CHECK (fiber_g BETWEEN 0 AND 100),
    serving_g   numeric(7, 2) CHECK (serving_g > 0 AND serving_g <= 5000),
    fetched_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, source_id)
);

CREATE TABLE food_lookups (
    source     text NOT NULL CHECK (source IN ('off', 'usda')),
    query      text NOT NULL CHECK (length(query) <= 200),
    food_ids   uuid[] NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source, query)
);

-- What a person ate. Nutrients are stored for the amount eaten, so the log
-- stays right if the catalogue changes or disappears.
CREATE TABLE food_entries (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    eaten_on   date NOT NULL,
    meal       text NOT NULL CHECK (meal IN ('breakfast', 'lunch', 'dinner', 'snack')),
    name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 300),
    grams      numeric(7, 1) CHECK (grams > 0 AND grams <= 10000),
    kcal       numeric(7, 1) CHECK (kcal BETWEEN 0 AND 20000),
    protein_g  numeric(6, 1) CHECK (protein_g BETWEEN 0 AND 2000),
    carbs_g    numeric(6, 1) CHECK (carbs_g BETWEEN 0 AND 2000),
    fat_g      numeric(6, 1) CHECK (fat_g BETWEEN 0 AND 2000),
    food_id    uuid REFERENCES foods(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX food_entries_day ON food_entries (user_id, eaten_on);

CREATE TABLE workouts (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    done_on    date NOT NULL,
    kind       text NOT NULL CHECK (kind IN ('strength', 'cardio', 'mobility', 'sport', 'other')),
    title      text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    minutes    integer CHECK (minutes BETWEEN 1 AND 1440),
    effort     smallint CHECK (effort BETWEEN 1 AND 10), -- how hard it felt (RPE)
    notes      text NOT NULL DEFAULT '' CHECK (length(notes) <= 4000),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX workouts_day ON workouts (user_id, done_on);

CREATE TABLE workout_sets (
    workout_id  uuid NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    position    smallint NOT NULL CHECK (position BETWEEN 0 AND 199),
    exercise    text NOT NULL CHECK (length(exercise) BETWEEN 1 AND 120),
    reps        smallint CHECK (reps BETWEEN 1 AND 1000),
    weight_kg   numeric(6, 2) CHECK (weight_kg BETWEEN 0 AND 1000),
    distance_km numeric(7, 3) CHECK (distance_km > 0 AND distance_km <= 1000),
    seconds     integer CHECK (seconds BETWEEN 1 AND 86400),
    PRIMARY KEY (workout_id, position)
);

-- +goose Down
DROP TABLE workout_sets;
DROP TABLE workouts;
DROP TABLE food_entries;
DROP TABLE food_lookups;
DROP TABLE foods;
