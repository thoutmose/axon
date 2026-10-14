-- +goose Up
-- Digests everywhere (EP-70): the daily digest, and new weekly and monthly
-- ones, also go to chat channels a person connects. A channel's address or
-- bot token is a credential: config is JSON sealed with AXON_ENCRYPTION_KEY
-- when it is set (first byte 1), plain otherwise (first byte 0).
CREATE TABLE notify_channels (
    id              uuid PRIMARY KEY,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind            text NOT NULL CHECK (kind IN ('slack', 'discord', 'teams', 'telegram', 'ntfy', 'webhook')),
    name            text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    config          bytea NOT NULL,
    target          text NOT NULL DEFAULT '' CHECK (length(target) <= 200), -- what to show, without the secret part
    digests         text[] NOT NULL DEFAULT '{daily,weekly,monthly}',
    reminders       boolean NOT NULL DEFAULT true, -- task reminders and the evening nudge too
    enabled         boolean NOT NULL DEFAULT true,
    -- A bot answers one person: they link the chat by sending a one-time code (EP-71).
    link_hash       bytea UNIQUE,
    link_expires_at timestamptz,
    peer            text NOT NULL DEFAULT '' CHECK (length(peer) <= 100),
    peer_name       text NOT NULL DEFAULT '' CHECK (length(peer_name) <= 100),
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_sent_at    timestamptz,
    last_error      text NOT NULL DEFAULT '' CHECK (length(last_error) <= 500)
);
CREATE INDEX notify_channels_user ON notify_channels (user_id);

ALTER TABLE reminders_sent DROP CONSTRAINT reminders_sent_kind_check;
ALTER TABLE reminders_sent ADD CONSTRAINT reminders_sent_kind_check CHECK (kind IN ('morning', 'evening', 'weekly', 'monthly'));

-- +goose Down
DELETE FROM reminders_sent WHERE kind IN ('weekly', 'monthly');
ALTER TABLE reminders_sent DROP CONSTRAINT reminders_sent_kind_check;
ALTER TABLE reminders_sent ADD CONSTRAINT reminders_sent_kind_check CHECK (kind IN ('morning', 'evening'));
DROP TABLE notify_channels;
