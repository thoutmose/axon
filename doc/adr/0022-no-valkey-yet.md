# 0022 — No Valkey until something needs it

- **Status:** Accepted; supersedes the "Ephemeral" row of [ADR 0002](0002-initial-technology-stack.md)
- **Date:** 2026-10-12
- **Epic:** EP-49

## Context

ADR 0002 planned Valkey for cache and job queues. The stack has run a Valkey container since EP-01, but no code ever connected to it: rate limits live in the single API process, and background work (agent runs, demo resets, media pruning, the vault mirror) runs in-process, coordinated through PostgreSQL (advisory locks, the event log, and small queue tables). The audit of EP-49 found the container, the `REDIS_URL` setting, and the documentation describing queues that did not exist.

## Decision

Remove Valkey and `REDIS_URL`. Background work stays in-process and durable in PostgreSQL. Bring a cache or queue back only when a measurement shows the need (several API replicas, or job volumes PostgreSQL cannot hold), in a new ADR.

## Consequences

One container and one setting fewer on every host; `deploy.sh` removes the old container (`--remove-orphans`). Rate limits stay per process, which is right for a single API process ([ADR 0021](0021-single-host-measured.md)).
