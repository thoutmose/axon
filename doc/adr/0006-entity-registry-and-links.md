# 0006 — Entity registry and universal links

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-05

## Context

Universal links must connect any two compatible things a user owns: a note to a goal, a task to a project, a habit to a milestone. The usual way to do this is a polymorphic link table (`source_type`, `source_id`) with no foreign keys, plus application code and a periodic job that hunts for dangling links. That pushes integrity into code that every later Epic has to get right.

## Decision

Every linkable thing is first a row in one **`entities`** table (`id`, `user_id`, `type`, `title`, `search_text`, timestamps). Each domain table (`tasks`, `notes`, …) uses the entity's ID as its own primary key and foreign key: `id uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE` ("class table inheritance").

- **Links** reference `entities` with real foreign keys, so the database itself guarantees no dangling links: deleting an entity deletes its links in the same transaction.
- **Owner boundary in the schema:** links reference `entities(user_id, id)`, so a link between two users' entities cannot be inserted.
- **Allowed relations** are declared in Go next to each entity type (`relates_to` is always allowed between distinct entities; specific relations such as `part_of` list their source and target types).
- **Deleting** an entity means deleting its `entities` row; the domain row follows by cascade.
- **Search** uses a generated `tsvector` over `title` and `search_text` with the language-neutral `simple` configuration, so French and English notes are found alike.
- **Events** (`created`, `updated`, `deleted`, `linked`, `unlinked`) record which fields changed, never their values, and survive entity deletion so timelines and analytics keep their shape without keeping content.

## Alternatives

- Polymorphic link table without foreign keys: no extra join, but integrity becomes a job instead of a guarantee.
- One link table per pair of types: exact foreign keys, but the number of tables grows with the square of the types.

## Consequences

Creating a domain row takes two inserts in one transaction (entity, then domain row), wrapped by `graph.CreateEntity`. Cross-type features (search, backlinks, timeline, export) work on one table. A test fails if a domain table references `entities` without registering its type.
