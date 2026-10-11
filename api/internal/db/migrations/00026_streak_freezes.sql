-- +goose Up
-- Streak freezes (EP-15), bought only with earned points, never with money.
-- Experience points themselves are computed from the source data, not stored.
CREATE TABLE streak_freezes (
    id        uuid PRIMARY KEY,
    user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bought_at timestamptz NOT NULL DEFAULT now(),
    bought_on date NOT NULL,  -- local date, so it can cover days from then on
    used_on   date            -- the missed local day it covered
);
CREATE INDEX streak_freezes_user ON streak_freezes (user_id);

-- +goose Down
DROP TABLE streak_freezes;
