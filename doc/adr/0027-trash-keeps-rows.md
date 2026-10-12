# 0027 — The trash keeps the rows a deletion removes

- **Status:** Accepted
- **Date:** 2026-10-12
- **Epic:** EP-66

## Context

Deleting a note, task, goal, habit, card, or any other item was final. For a system that holds a person's notes, plans, and history, a mistaken click should be recoverable. Soft deletion (a `deleted_at` on every table) would touch every query in every package, and forgetting one filter would show deleted data.

## Decision

`graph.DeleteEntity`, the one path every deletion takes, first saves the rows the deletion will remove, as JSON: the entity, its domain row, the rows that cascade from it two levels down (found from PostgreSQL's own foreign keys, so new tables are covered without code), the entity rows of children that are entities themselves, and the links to and from all of them. Derived rows (note embeddings) are left to be rebuilt. Restoring inserts the same rows, parents first, in one transaction; links whose other end is gone are skipped; a path or name taken meanwhile stops the restore with a clear message and changes nothing. Deletions made in one transaction (a goal and its milestones, a bulk delete) share a batch and come back together. Deletions the vault mirror makes on its own (an item's note following its item) are not trashed. Entries are kept 30 days, then purged daily. A restore records a `created` event, so the vault mirror and the sync see the item again.

## Consequences

Every item type is covered, including future ones, with one hook. Deleted data stays 30 days longer in the database: it is part of the personal export, erased with the account, and can be emptied at any time from Account → Trash.
