-- +goose Up
-- Reminders (EP-56): a morning digest and an evening nudge, as Web Push
-- notifications to the devices the person turned them on for, and by email
-- when the instance can send it.

-- Where to push: what each browser's PushManager returned.
CREATE TABLE push_subscriptions (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint     text NOT NULL UNIQUE CHECK (endpoint LIKE 'https://%' AND length(endpoint) <= 2000),
    p256dh       text NOT NULL CHECK (length(p256dh) BETWEEN 80 AND 100),
    auth         text NOT NULL CHECK (length(auth) BETWEEN 16 AND 30),
    device       text NOT NULL DEFAULT '' CHECK (length(device) <= 100),
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_sent_at timestamptz
);
CREATE INDEX push_subscriptions_user ON push_subscriptions (user_id);

-- Keys the instance makes for itself, such as the VAPID key push services
-- check (sealed with AXON_ENCRYPTION_KEY when it is set).
CREATE TABLE instance_keys (
    name       text PRIMARY KEY,
    value      bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- What was already sent, so each reminder goes out once a day.
CREATE TABLE reminders_sent (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind    text NOT NULL CHECK (kind IN ('morning', 'evening')),
    day     date NOT NULL,
    sent_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, kind, day)
);

-- +goose Down
DROP TABLE reminders_sent;
DROP TABLE instance_keys;
DROP TABLE push_subscriptions;
