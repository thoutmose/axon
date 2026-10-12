-- +goose Up
-- Projects with a Kanban board built on tasks (EP-10). A card puts a task on
-- one project's board, in a column, as an epic, an issue, or a sub-task.
CREATE TABLE projects (
    id          uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    description text NOT NULL DEFAULT '' CHECK (length(description) <= 20000),
    status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'on_hold', 'done', 'archived')),
    wip_limit   integer CHECK (wip_limit BETWEEN 1 AND 50),   -- cards allowed in Doing; NULL = no limit
    docs_folder text NOT NULL CHECK (length(docs_folder) BETWEEN 1 AND 300), -- notes under this folder are the project's docs
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE project_cards (
    task_id    uuid PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    col        text NOT NULL DEFAULT 'todo' CHECK (col IN ('backlog', 'todo', 'doing', 'done')),
    position   double precision NOT NULL DEFAULT 0,
    kind       text NOT NULL DEFAULT 'issue' CHECK (kind IN ('epic', 'issue', 'subtask')),
    parent_id  uuid REFERENCES project_cards(task_id) ON DELETE SET NULL,
    CHECK (parent_id IS NULL OR parent_id <> task_id)
);
CREATE INDEX project_cards_board ON project_cards (project_id, col, position);
CREATE INDEX project_cards_parent ON project_cards (parent_id);

-- +goose Down
DROP TABLE project_cards;
DROP TABLE projects;
