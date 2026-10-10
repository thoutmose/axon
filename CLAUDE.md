# Axon — agent operating rules

Axon is a self-hosted life OS. Product and architecture: [README.md](README.md). Work items and delivery rules: [doc/BACKLOG.md](doc/BACKLOG.md). Decisions: [doc/adr/](doc/adr/).

## Current state

EP-00 and EP-01 delivered. Next Epic: EP-02 (see [doc/BACKLOG.md](doc/BACKLOG.md#delivery-order)). Servers: staging `srv-staging`, production `srv-prod`; runbook in [doc/operations.md](doc/operations.md).

- `api/` — Go API (`chi`, `pgx`, `goose` migrations embedded in `internal/db/migrations`).
- `web/` — SvelteKit 2 + TypeScript, Node 24.
- `compose.yaml`, `deploy/Caddyfile` — local stack on http://localhost:3000.

## Commands

- `make check` — all formatting, lint, and unit tests (run before every commit).
- `make test-integration` — migrations against real PostgreSQL.
- `make up` / `make down` — local stack; then `docker compose exec api axon migrate up`.
- `deploy/deploy.sh staging|prod [--rollback]` — deploy the committed state (staging first).

Server host configuration belongs to the fleet repository `thoutmose/terraform-proxmox`, not here (ADR 0004).

Docker bridge networks on the maintainer's dev host have no internet access, so image builds and the web check container use host networking.

## Rules

- Work inside the active Epic only. Stop at the Epic boundary and wait for human review.
- Read the target files, tests, and docs before editing. State assumptions; ask when one could affect user data or security.
- Make the smallest coherent diff. Simple, explicit code; no speculative abstractions.
- Run focused checks first, then the relevant suite. Report failures as they are.
- Inspect your diff for scope creep, secrets, generated noise, and unsafe defaults.
- Never merge your own work, mutate production data, rotate credentials, or expand scope silently.
- AI-generated plans and mutations are drafts until a human approves them.
- If a requirement conflicts with privacy, safety, or consent, stop and ask.

Conventions (branches, commits, PRs, ADRs) are in [CONTRIBUTING.md](CONTRIBUTING.md).
