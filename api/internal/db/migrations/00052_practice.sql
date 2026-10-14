-- +goose Up
-- The practice log (EP-103): problems and their attempts.
CREATE TABLE practice_problems (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      text NOT NULL CHECK (length(title) BETWEEN 1 AND 300),
    url        text CHECK (url ~ '^https?://' AND length(url) <= 2000),
    platform   text NOT NULL,
    difficulty text NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    topics     text[] NOT NULL DEFAULT '{}' CHECK (cardinality(topics) <= 10),
    language   text NOT NULL DEFAULT '' CHECK (length(language) <= 20),
    skill_id   uuid REFERENCES skills(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX practice_problems_url ON practice_problems (user_id, url) WHERE url IS NOT NULL;
CREATE INDEX practice_problems_skill ON practice_problems (skill_id);
CREATE TABLE practice_attempts (
    id         uuid PRIMARY KEY,
    problem_id uuid NOT NULL REFERENCES practice_problems(id) ON DELETE CASCADE,
    tried_on   date NOT NULL,
    outcome    text NOT NULL CHECK (outcome IN ('solved', 'with_help', 'not_solved')),
    minutes    integer CHECK (minutes BETWEEN 1 AND 1440),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX practice_attempts_problem ON practice_attempts (problem_id, tried_on);

-- +goose Down
DROP TABLE practice_attempts;
DROP TABLE practice_problems;
