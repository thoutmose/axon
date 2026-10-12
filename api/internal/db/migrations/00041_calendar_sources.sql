-- +goose Up
-- Calendars the person follows (EP-58): an ICS address, read only. The
-- address is often a secret link, so it is sealed when AXON_ENCRYPTION_KEY
-- is set and never shown again in full.
CREATE TABLE calendar_sources (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    color       text NOT NULL DEFAULT '#5aa9ff' CHECK (color ~ '^#[0-9a-fA-F]{6}$'),
    url         bytea NOT NULL,          -- first byte 0: plain, 1: sealed
    host        text NOT NULL,           -- shown instead of the address
    fetched_at  timestamptz,
    error       text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX calendar_sources_user ON calendar_sources (user_id);

-- Their events, as occurrences from 60 days ago to a year ahead; replaced
-- at each fetch.
CREATE TABLE calendar_events (
    source_id uuid NOT NULL REFERENCES calendar_sources(id) ON DELETE CASCADE,
    user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    uid       text NOT NULL,
    title     text NOT NULL,
    location  text NOT NULL DEFAULT '',
    starts_at timestamptz NOT NULL,
    ends_at   timestamptz NOT NULL,
    all_day   boolean NOT NULL DEFAULT false
);
CREATE INDEX calendar_events_user_time ON calendar_events (user_id, starts_at);

-- +goose Down
DROP TABLE calendar_events;
DROP TABLE calendar_sources;
