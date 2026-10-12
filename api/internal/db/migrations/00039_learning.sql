-- +goose Up
-- The learning and memory engine (EP-52), after Understanding How We Learn
-- (spaced and retrieval practice, interleaving, elaboration, concrete
-- examples, dual coding) and Moonwalking with Einstein (vivid images, the
-- memory palace, deliberate practice past the "OK plateau").

-- A memory palace: a place the person knows well, as an ordered walk of loci.
CREATE TABLE memory_palaces (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    loci        text[] NOT NULL DEFAULT '{}' CHECK (cardinality(loci) <= 100),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX memory_palaces_user ON memory_palaces (user_id);

-- Cards gain what makes memories stick: why it is true (elaboration), a
-- concrete example, a vivid image or mnemonic, and a place in a palace.
-- An illustration can be a picture (PNG) or a drawing (sanitised SVG).
ALTER TABLE cards
    ADD COLUMN example     text NOT NULL DEFAULT '' CHECK (length(example) <= 2000),
    ADD COLUMN elaboration text NOT NULL DEFAULT '' CHECK (length(elaboration) <= 2000),
    ADD COLUMN mnemonic    text NOT NULL DEFAULT '' CHECK (length(mnemonic) <= 2000),
    ADD COLUMN image_type  text NOT NULL DEFAULT 'image/png' CHECK (image_type IN ('image/png', 'image/svg+xml')),
    ADD COLUMN palace_id   uuid REFERENCES memory_palaces(id) ON DELETE SET NULL,
    ADD COLUMN locus       smallint CHECK (locus BETWEEN 0 AND 99);

-- Free recall of whole notes: write what you remember, then compare. Each
-- attempt is kept; the next one is scheduled by expanding intervals.
CREATE TABLE note_recalls (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    note_id     uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    score       real NOT NULL CHECK (score BETWEEN 0 AND 1),
    recalled_at timestamptz NOT NULL DEFAULT now(),
    due_on      date NOT NULL
);
CREATE INDEX note_recalls_user ON note_recalls (user_id, note_id, recalled_at DESC);

-- Practice tests taken (low-stakes, mixed): the weekly plan offers the next one.
CREATE TABLE practice_tests (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck        text NOT NULL DEFAULT '' CHECK (length(deck) <= 200),
    questions   smallint NOT NULL CHECK (questions BETWEEN 1 AND 100),
    correct     smallint NOT NULL CHECK (correct BETWEEN 0 AND questions),
    taken_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX practice_tests_user ON practice_tests (user_id, taken_at DESC);

-- +goose Down
DROP TABLE practice_tests;
DROP TABLE note_recalls;
ALTER TABLE cards DROP COLUMN locus, DROP COLUMN palace_id, DROP COLUMN image_type,
    DROP COLUMN mnemonic, DROP COLUMN elaboration, DROP COLUMN example;
DROP TABLE memory_palaces;
