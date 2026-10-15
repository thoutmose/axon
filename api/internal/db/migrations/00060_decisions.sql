-- +goose Up
-- The decision journal (EP-114): a decision, the outcome expected and the
-- chance given to it, then, on the review day, what happened. The forecast
-- (expected, probability) never changes once made, so the score is honest.
CREATE TABLE decisions (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    context     text NOT NULL DEFAULT '' CHECK (length(context) <= 5000),
    choice      text NOT NULL DEFAULT '' CHECK (length(choice) <= 2000),
    expected    text NOT NULL CHECK (length(expected) BETWEEN 1 AND 500),
    probability double precision NOT NULL CHECK (probability >= 0.01 AND probability <= 0.99),
    decided_on  date NOT NULL,
    review_on   date NOT NULL CHECK (review_on > decided_on),
    happened    boolean,
    outcome     text NOT NULL DEFAULT '' CHECK (length(outcome) <= 5000),
    lessons     text NOT NULL DEFAULT '' CHECK (length(lessons) <= 5000),
    reviewed_on date,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK ((happened IS NULL) = (reviewed_on IS NULL))
);
CREATE INDEX decisions_user ON decisions (user_id, review_on);

-- +goose Down
DROP TABLE decisions;
