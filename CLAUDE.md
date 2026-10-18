# Axon — agent operating rules

Axon is a self-hosted life OS. Product and architecture: [README.md](README.md). Work items and delivery rules: [doc/BACKLOG.md](doc/BACKLOG.md). Decisions: [doc/adr/](doc/adr/).

## Current state

Every Epic in the backlog through EP-81 is delivered, and EP-106 to EP-113 (the vault server, digests on every channel and answering them, sign-in with other accounts, and the tech watch with its graph, exports, sharing, podcast, and agent) and EP-114 to EP-125 (decisions, search in the menu bar, checked items that stay, coffee and tea, Huberman's protocols, the collection, invitations, sharing as a picture, a practical graph, YouTube and Audiobookshelf in the tech watch, the trailer), EP-126 to EP-127 (links to sources, every feature in sight), and EP-128 to EP-140 (a shorter board, line icons, the life web, media cards, pages, notes and covers, reading habits, budget charts, a clean start, natural podcast voices, the digest's channels), EP-141 to EP-145 (the Top-5% roadmap v2, its plan in sight, notes drawn, YouTube and Podcasts pages, the roadmap on Today), and EP-146 to EP-158 (database architecture and GitOps, budget charts on the Budget page, videos in a few words, video games, maths in notes, a model per agent, AI tokens and compaction, Jarvis, the roadmap's resources as notes, a news digest, the vault verified, monitoring by feature, the README banner), and EP-159 to EP-162 (the trailer again, a clean-up with no critical flaw in reach, every page walked in both languages, and 1.0 prepared), and EP-163 to EP-169 (the radar by domain, the collection in reach, YouTube subscriptions imported, the plan's goals, habits, and areas, habits on the calendar, axon.thoutmose.me, and Claude Code on Axon through MCP) are delivered; 1.0.0 is tagged on `main` (2026-10-18), and the external security review and an observed install follow it. The twelve features proposed in the EP-49 audit are delivered (EP-54 to EP-64, pictures in notes in EP-51), then audit fixes, trash and undo, share to Axon, task reminders, and the clean-code pass (EP-65 to EP-69), then performance and hardening (EP-70 to EP-81) and owner requests: the skills radar, focus music, the home page teaser, Axon in French from end to end (EP-82 to EP-87), and health: daily needs, a photo of a meal, and a Chef & Nutritionist agent (EP-88 to EP-90), then a daily journal, what helps you, journal days in the vault, onboarding by use, the owner's Top-5% roadmap as the one roadmap, a sky that follows the sun, the roadmap on the calendar, every page its picture, a readable year of activity, the vault's note templates, water, patched Caddy and PostgreSQL images, a practice log, certifications, and study hours (EP-91 to EP-105). Everything is on `main`, the only branch; `main` is deployed to staging and production after green CI. Open items and the delivery order are in [doc/BACKLOG.md](doc/BACKLOG.md#delivery-order). Servers: staging `srv-staging`, production `srv-prod`; runbook in [doc/operations.md](doc/operations.md).

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
