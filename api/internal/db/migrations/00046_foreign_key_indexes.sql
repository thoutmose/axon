-- +goose Up
-- Every foreign key gets an index starting with its columns (EP-70), so
-- deleting a parent row, or looking up by the key, never scans the whole
-- child table. calendar_events(source_id) is hit by every calendar refresh;
-- note_recalls(note_id, recalled_at) serves the recall queue's "latest
-- attempt of each note".
CREATE INDEX agent_proposals_run ON agent_proposals (run_id);
CREATE INDEX agent_runs_parent ON agent_runs (parent_run_id);
CREATE INDEX budget_recurring_user ON budget_recurring (user_id);
CREATE INDEX budget_recurring_account ON budget_recurring (account_id);
CREATE INDEX budget_recurring_to_account ON budget_recurring (to_account_id);
CREATE INDEX budget_recurring_category ON budget_recurring (category_id);
CREATE INDEX budget_rules_user ON budget_rules (user_id);
CREATE INDEX budget_rules_category ON budget_rules (category_id);
CREATE INDEX budget_transactions_category ON budget_transactions (category_id);
CREATE INDEX calendar_events_source ON calendar_events (source_id);
CREATE INDEX cards_palace ON cards (palace_id);
CREATE INDEX food_entries_food ON food_entries (food_id);
CREATE INDEX household_invites_household ON household_invites (household_id);
CREATE INDEX household_invites_created_by ON household_invites (created_by);
CREATE INDEX household_items_assignee ON household_items (assignee);
CREATE INDEX household_items_created_by ON household_items (created_by);
CREATE INDEX household_items_done_by ON household_items (done_by);
CREATE INDEX note_recalls_note ON note_recalls (note_id, recalled_at DESC);
CREATE INDEX repo_connections_project ON repo_connections (project_id);
CREATE INDEX webauthn_ceremonies_user ON webauthn_ceremonies (user_id);

-- +goose Down
DROP INDEX agent_proposals_run, agent_runs_parent, budget_recurring_user, budget_recurring_account,
  budget_recurring_to_account, budget_recurring_category, budget_rules_user, budget_rules_category,
  budget_transactions_category, calendar_events_source, cards_palace, food_entries_food,
  household_invites_household, household_invites_created_by, household_items_assignee,
  household_items_created_by, household_items_done_by, note_recalls_note, repo_connections_project,
  webauthn_ceremonies_user;
