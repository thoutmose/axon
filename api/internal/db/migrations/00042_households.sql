-- +goose Up
-- Household sharing (EP-61). People on the same Axon join a household by
-- a one-time invite code and share lists: shopping, chores, anything. Only
-- what is put on a household list is shared; nothing personal is.
CREATE TABLE households (
    id         uuid PRIMARY KEY,
    name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE household_members (
    household_id uuid NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE, -- one household per person
    owner        boolean NOT NULL DEFAULT false,
    joined_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (household_id, user_id)
);

-- Invite codes are stored as hashes, used once, and expire after a week.
CREATE TABLE household_invites (
    code_hash    bytea PRIMARY KEY,
    household_id uuid NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    created_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    expires_at   timestamptz NOT NULL,
    used_at      timestamptz
);

CREATE TABLE household_lists (
    id           uuid PRIMARY KEY,
    household_id uuid NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    title        text NOT NULL CHECK (length(title) BETWEEN 1 AND 100),
    kind         text NOT NULL DEFAULT 'other' CHECK (kind IN ('shopping', 'chores', 'other')),
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX household_lists_household ON household_lists (household_id);

CREATE TABLE household_items (
    id         uuid PRIMARY KEY,
    list_id    uuid NOT NULL REFERENCES household_lists(id) ON DELETE CASCADE,
    title      text NOT NULL CHECK (length(title) BETWEEN 1 AND 300),
    due_on     date,
    assignee   uuid REFERENCES users(id) ON DELETE SET NULL,
    created_by uuid REFERENCES users(id) ON DELETE SET NULL,
    done_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    done_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX household_items_list ON household_items (list_id, done_at, created_at);

-- +goose Down
DROP TABLE household_items;
DROP TABLE household_lists;
DROP TABLE household_invites;
DROP TABLE household_members;
DROP TABLE households;
