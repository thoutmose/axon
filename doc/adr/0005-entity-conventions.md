# 0005 — Entity conventions

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-04 (brought forward from EP-05/IS-05.01, because accounts are the first tables)

## Context

Accounts, sessions, and audit events are the first persistent data. Every later table should follow the same rules for identifiers, ownership, time, and deletion, so that authorization checks, export, and erasure work the same way everywhere.

## Decision

- **IDs** are UUIDv7, generated in Go. They are time-ordered, so B-tree indexes stay compact, and they reveal nothing about row counts.
- **Ownership:** every user-owned table has `user_id uuid not null references users(id) on delete cascade`, and every query on it filters by `user_id`.
- **Time:** `timestamptz`, stored in UTC; `created_at` on every table.
- **Deletion is real.** No soft-delete flags: a deleted row is gone, so erasure cannot be undone by forgetting a `where deleted_at is null`. A recoverable trash, if wanted later, is its own table with its own retention.
- **Secrets** (session tokens, recovery codes) are stored as SHA-256 hashes; reversible secrets (TOTP seeds) are encrypted with AES-256-GCM under `AXON_ENCRYPTION_KEY`.
- **Migrations** are plain SQL with `up` and `down` sections (goose).

## Consequences

Erasure is a single `delete from users`, verified by a test that finds no remaining rows referencing the user. Losing `AXON_ENCRYPTION_KEY` disables TOTP for every account (passkeys and recovery codes keep working).
