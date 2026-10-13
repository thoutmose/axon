# Axon — agent operating rules

Axon is a self-hosted life OS. Product and architecture: [README.md](README.md). Work items and delivery rules: [doc/BACKLOG.md](doc/BACKLOG.md). Decisions: [doc/adr/](doc/adr/).

## Current state

Every Epic in the backlog through EP-81 is delivered (1.0 release candidate); 1.0 waits for the external security review and an observed install. The twelve features proposed in the EP-49 audit are delivered (EP-54 to EP-64, pictures in notes in EP-51), then audit fixes, trash and undo, share to Axon, task reminders, and the clean-code pass (EP-65 to EP-69), then performance and hardening (EP-70 to EP-81) and owner requests: the skills radar, focus music, the home page teaser, Axon in French from end to end (EP-82 to EP-87), and health: daily needs, a photo of a meal, and a Chef & Nutritionist agent (EP-88 to EP-90). Everything through EP-68 is merged into `main`; EP-69 is in pull request #81, with the CI-minutes change and a dependency bump; EP-70 to EP-85 are in pull request #82, EP-86 and EP-87 in #83 (based on #82), and EP-88 to EP-90 in #84 (based on #83). Open items and the delivery order are in [doc/BACKLOG.md](doc/BACKLOG.md#delivery-order). Servers: staging `srv-staging`, production `srv-prod`; runbook in [doc/operations.md](doc/operations.md).

- `api/` — Go API (`chi`, `pgx`, `goose` migrations embedded in `internal/db/migrations`).
- `web/` — SvelteKit 2 + TypeScript, Node 24.
- `compose.yaml`, `deploy/Caddyfile` — local stack on http://localhost:3000.

## Commands

- `make check` — all formatting, lint, and unit tests (run before every commit).
- `make test-integration` — migrations against real PostgreSQL.
- `make up` / `make down` — local stack; then `docker compose exec api axon migrate up`.
- `deploy/deploy.sh staging|prod [--rollback]` — deploy the committed state; needs green CI, and prod only takes what staging serves.

Server host configuration belongs to the fleet repository `thoutmose/terraform-proxmox`, not here (ADR 0004).

Docker bridge networks on the maintainer's dev host have no internet access: its `.env` sets `AXON_BUILD_NETWORK=host`, and the web check container uses host networking.

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
