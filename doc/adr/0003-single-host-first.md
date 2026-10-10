# 0003 — Single host first

- **Status:** Accepted (2026-10-10, maintainer delegated the decision); staging placement superseded by [0004](0004-hosts-and-tailnet-only-access.md)
- **Date:** 2026-10-10
- **Epic:** EP-01

## Context

The README describes six servers: `srv-dev`, `srv-staging`, `srv-prod`, `srv-db`, plus the existing `srv-services` (Anki sync) and `srv-second-brain` (Markdown notes). Axon serves one user or a household. Before any feature exists, provisioning and securing four new hosts would cost more than the product it hosts.

## Decision

Through release R1, run Axon on **one new host (`srv-prod`)** with Docker Compose:

- Caddy, API, web, worker, PostgreSQL, and Valkey run as containers on `srv-prod`.
- Development happens locally with the same Compose file.
- Staging is a second Compose project on `srv-prod` with its own database and hostname, used to rehearse migrations before promotion.
- `srv-services` and `srv-second-brain` stay as they are and are reached over WireGuard/Tailscale.
- Backups are encrypted and stored off-host; restore is rehearsed.

Move PostgreSQL to `srv-db`, or staging to its own host, only when a measurement shows a need: memory or IOPS pressure, noisy-neighbour effects, or a recovery target the single host cannot meet. Record that move in a new ADR.

## Alternatives

- Six hosts from day one (the README target): strongest isolation, but four times the hardening, patching, and monitoring work.
- Managed cloud database: less operating work, but conflicts with self-hosting and data-ownership goals.

## Consequences

EP-01 shrinks from four hosts to one. A host failure takes down production and staging together, so off-host backups and a tested restore become MUST items. Staging shares hardware with production, so heavy load tests run locally, not on staging.
