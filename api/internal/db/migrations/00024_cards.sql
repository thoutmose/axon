-- +goose Up
-- Flashcards and reviews (EP-16), scheduled with FSRS-5. Cards are entities,
-- so they link to the notes, skills, and roadmaps they come from.
CREATE TABLE cards (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    deck        text NOT NULL DEFAULT 'Default' CHECK (length(deck) BETWEEN 1 AND 200),
    front       text NOT NULL CHECK (length(front) BETWEEN 1 AND 10000),
    back        text NOT NULL CHECK (length(back) <= 10000),
    image       bytea CHECK (length(image) <= 3145728),
    state       smallint NOT NULL DEFAULT 1 CHECK (state IN (1, 2, 3)),
    step        smallint,
    stability   double precision,
    difficulty  double precision,
    due         timestamptz NOT NULL DEFAULT now(),
    last_review timestamptz,
    reps        integer NOT NULL DEFAULT 0,
    lapses      integer NOT NULL DEFAULT 0,
    suspended   boolean NOT NULL DEFAULT false,
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX cards_due ON cards (due) WHERE NOT suspended;

CREATE TABLE card_reviews (
    id          uuid PRIMARY KEY,
    card_id     uuid NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    rating      smallint NOT NULL CHECK (rating BETWEEN 1 AND 4),
    reviewed_at timestamptz NOT NULL,
    state       smallint NOT NULL,
    duration_ms integer CHECK (duration_ms BETWEEN 0 AND 3600000)
);
CREATE INDEX card_reviews_card ON card_reviews (card_id, reviewed_at);

-- +goose Down
DROP TABLE card_reviews;
DROP TABLE cards;
