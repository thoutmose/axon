# 0021 — Single host, measured

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-21

## Context

[ADR 0003](0003-single-host-first.md) chose one host per environment and asked for a new decision only when a measurement showed a need. EP-21 asks to revisit it with measurements before 1.0.

## Measurements (2026-10-11)

Host: 4 vCPU, 7 GB of memory (the maintainer's development machine, similar to `srv-prod`). Stack: the release images with Caddy, API, web, PostgreSQL 16, Valkey, and the Anki bridge. Data: the demo account plus 5,000 tasks. Tool: `api/cmd/axon-load`, a mix of today, tasks, notes, search, dashboard overview, insights, habits, and task creation (10 % writes), with a personal access token.

| Clients | Requests/s | p50 | p95 | p99 | Errors |
|---|---|---|---|---|---|
| 10 | 481 | 10 ms | 79 ms | 219 ms | 0 |
| 50 | 713 | 35 ms | 197 ms | 219 ms | 0 |

The slowest endpoint is the dashboard overview (p95 237 ms at 10 clients), which runs many aggregate queries; every other endpoint stays under 100 ms at p95 at 10 clients. Right after the run, the containers used about 290 MB of memory together (API 24 MB, PostgreSQL 40 MB, Caddy 64 MB).

Other measurements:

- A year of activity (7,300 tasks) keeps the insights series, weekly review, and 104-week CSV under 3 s together (integration test; 0.7 s locally).
- Recovery: an instance archive of the demo data (47 tables) restored onto a fresh installation in 42 s; a database dump restored in 4 s (drill of 2026-10-10).
- Upgrade from v0.1.0 to this release: 83 s including image builds, with every row and the API token intact.

## Decision

Keep one host per environment for 1.0. A household (a handful of people) produces a small fraction of 10 concurrent clients, so the single host has more than an order of magnitude of headroom. The recovery targets (RPO ≤ 24 h, RTO ≤ 4 h) are met by nightly encrypted backups or instance archives and a restore measured in seconds.

Revisit when one of these is observed in production: p95 above 500 ms on everyday endpoints, PostgreSQL memory or IOPS pressure, or a recovery target the backups cannot meet.

## Alternatives

- Move PostgreSQL to `srv-db` now: more isolation, but no measured need and more to secure and patch.
- Cache the dashboard overview: not needed at this load; noted as the first optimisation if the threshold above is crossed.

## Consequences

The architecture stays as built. `axon-load` is kept in the repository so operators can measure their own hardware.
