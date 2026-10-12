-- +goose Up
-- Search by meaning (EP-62): each person chooses an embedding service (a
-- local model server, OpenAI, or Voyage AI) and gives a separate consent;
-- notes are embedded there and the vectors kept here (pgvector).
CREATE TABLE embedding_settings (
    user_id        uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    provider       text NOT NULL CHECK (provider IN ('local', 'openai', 'voyage')),
    base_url       text,
    model          text NOT NULL CHECK (length(model) BETWEEN 1 AND 100),
    api_key_sealed bytea,
    error          text NOT NULL DEFAULT '',
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE note_embeddings (
    note_id    uuid PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model      text NOT NULL,
    hash       text NOT NULL, -- the note's content hash when it was embedded
    embedding  vector NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX note_embeddings_user ON note_embeddings (user_id, model);

-- +goose Down
DROP TABLE note_embeddings;
DROP TABLE embedding_settings;
