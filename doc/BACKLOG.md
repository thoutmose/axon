# Axon Backlog — *A humane operating system for a life well lived*

> **Status:** Product backlog and delivery contract. Estimates are directional story points, not promises. Epics after EP-06 are outlined; refine each one into issues at Epic start.

## Purpose

Axon is a self-hosted, open-source life OS that connects notes, action, learning, health, media, goals, and projects without turning a person into a scoreboard. This backlog is the canonical sequence for building it responsibly. It favors small, observable increments, local ownership of data, and reversible decisions.

## How agents and humans must use this backlog

- Work on **one Epic at a time**, in the order given by [Delivery order](#delivery-order). Do not start implementation in another Epic while the active Epic is incomplete or explicitly paused by a human maintainer.
- At Epic start, confirm scope, dependencies, Definition of Ready, threat impact, migration plan, and measurable acceptance criteria. Outlined Epics must be split into issues first.
- When an Epic is complete: run tests and security checks; request code review; update documentation and ADRs; open a PR/MR with Conventional Commits; obtain human approval; then select the next Epic.
- Agents may propose, draft, test, and explain changes. They must not silently mutate production data, merge their own work, rotate credentials, or expand scope.
- Read before write. Prefer simple explicit code, small diffs, boring technology, data-informed decisions, and “don’t be clever.”
- Every task records evidence: test output, screenshots where useful, migration notes, performance observations, and known caveats.
- If a requirement conflicts with privacy, safety, or consent, stop and request a human decision.

## Delivery rules

### Definition of Ready

- User or operator outcome is named.
- Acceptance criteria are testable and unambiguous.
- Dependencies and data migrations are identified.
- Security, privacy, accessibility, observability, and rollback impact are noted.
- Design/API contract is reviewed when the change crosses a boundary.
- Issue has an estimate, priority, owner, and labels.

### Definition of Done

- Acceptance criteria pass in CI and the relevant environment.
- Unit, integration, contract, accessibility, and security tests are proportionate to risk.
- Logs, metrics, traces, and runbook entries exist for operational behavior.
- Documentation, OpenAPI, screenshots, and ADRs are updated.
- Migration is reversible or has a documented recovery plan.
- The [cross-cutting acceptance matrix](#cross-cutting-acceptance-matrix) is attached to the Epic’s exit evidence.
- A reviewer who did not author the change approves it.
- Conventional Commit PR/MR is merged only after human approval.

### Labels

- `type:feature`, `type:bug`, `type:docs`, `type:security`, `type:ops`, `type:research`, `type:migration`
- `area:api`, `area:web`, `area:data`, `area:agent`, `area:infra`, `area:privacy`, `area:docs`
- `risk:low`, `risk:medium`, `risk:high`, `needs-human-decision`, `good-first-issue`
- `status:ready`, `status:in-progress`, `status:blocked`, `status:review`, `status:done`

### Estimation and priority

Story points use relative Fibonacci sizing: 1, 2, 3, 5, 8, 13, 21. A 21-point item should be split before implementation. MoSCoW priority means **MUST**, **SHOULD**, **COULD**, or **WON’T this release**. Estimates include testing and documentation, not only coding.

### Branches, commits, PRs, and ADRs

- Branches: `feat/EP-09-task-planner`, `fix/EP-04-session-rotation`, `docs/EP-00-agent-constitution`.
- Use Conventional Commits: `feat(tasks): add recurring task schedule`; include issue IDs in the body.
- PR title: `<type>(<area>): <imperative outcome> [EP-xx/IS-xx.xx]`.
- PR template must include purpose, screenshots/API examples, tests, migrations, security/privacy impact, rollback, and checklist.
- ADR files use `doc/adr/NNNN-short-title.md` ([template](adr/0000-template.md)), with context, decision, alternatives, consequences, and status. Supersede rather than rewrite accepted decisions.

### PR review checklist

- Does the change solve the stated user outcome without scope creep?
- Is the simplest design used, with failure paths explicit?
- Are authorization, tenant/data boundaries, secrets, injection, and auditability covered?
- Are loading, empty, offline, error, keyboard, screen-reader, and reduced-motion states covered?
- Are queries indexed and bounded? Are retries idempotent?
- Can an operator observe, roll back, and restore it?
- Are docs and tests updated?

## Epic index

| ID | Epic | Goal | Priority | Estimate | Dependencies | Status |
|---|---|---|---|---:|---|---|
| EP-00 | Project foundation & governance | Repository, license, decision records, contributor norms, agent rules, and a walking skeleton. | MUST | 34 pts | None | Done |
| EP-01 | Infrastructure & IaC | Axon deployed to `srv-prod` and `srv-staging` on the tailnet, with rollback and tested backups. | MUST | 27 pts | EP-00 | Done |
| EP-02 | CI/CD & promotion | Every change is checked, built reproducibly, and promoted staging → production with rollback. | MUST | 34 pts | EP-00, EP-01 | Done |
| EP-03 | Observability | Actionable logs, metrics, traces, dashboards, alerts, and SLOs. | MUST | 34 pts | EP-01, EP-02 | In progress (fleet apply pending) |
| EP-04 | Security, authentication & privacy | Passkeys/TOTP, sessions, ownership checks, audit, export/erasure, consent. | MUST | 55 pts | EP-00, EP-01 | Done (independent review owed) |
| EP-05 | Core data model & universal links | Entity conventions, typed links, event log, retention and erasure hooks, search. | MUST | 34 pts | EP-04 | Done |
| EP-06 | API platform | Spec-first REST/OpenAPI 3.1 with errors, pagination, idempotency, and tokens. | MUST | 34 pts | EP-05 | Done |
| EP-07 | Design system & frontend shell | Calm, accessible SvelteKit shell, components, and auth screens. | MUST | 34 pts | EP-04, EP-06 | Done |
| EP-08 | Dashboard & calendar | One trustworthy daily view of priorities, commitments, and progress. | SHOULD | 34 pts | EP-09, EP-11, EP-13 | Done |
| EP-09 | Tasks & recurring work | Capture, plan, and complete daily/weekly/monthly work with GTD contexts and recurrence. | MUST | 34 pts | EP-05, EP-06, EP-07 | Done |
| EP-10 | Projects, Kanban & docs | Projects, epics/issues/sub-tasks, Kanban, and project docs. | MUST | 34 pts | EP-09, EP-11 | Done |
| EP-11 | Notes, Zettelkasten, PARA & Markdown sync | Knowledge capture with wikilinks and safe sync with `~/SecondBrain`. | MUST | 55 pts | EP-05, EP-06, EP-07 | Done |
| EP-12 | Habits & Habits agent | Evidence-based habit design with gentle recovery. | SHOULD | 55 pts | EP-07, EP-19 | Done |
| EP-13 | Goals & milestones | Outcomes linked to measurable milestones, tasks, habits, and notes. | MUST | 21 pts | EP-05, EP-06, EP-07, EP-09 | Done |
| EP-14 | Roadmaps & skill graph | Manual roadmaps, OSSU import, and a skill graph view. | SHOULD | 42 pts | EP-11, EP-13 | Done |
| EP-15 | Gamification engine | Transparent, opt-in XP, streaks, badges, and earned freezes. | SHOULD | 55 pts | EP-09, EP-12, EP-13 | Planned |
| EP-16 | Learning engine, Teacher agent & Anki | Retrieval practice with FSRS scheduling, reviewed cards, and Anki sync. | SHOULD | 89 pts | EP-11, EP-14, EP-19 | Planned |
| EP-17 | Media trackers & Media agent | Films, series, and books with attributed metadata and explained recommendations. | COULD | 34 pts | EP-06, EP-07, EP-19 | Planned |
| EP-18 | Health, nutrition, sport & coach agents | Food and training logs with consent-gated, educational coaching. | COULD | 89 pts | EP-04, EP-06, EP-07, EP-19 | Planned |
| EP-19 | AI agent platform, skills & safety | Approval-gated agents with tool boundaries, budgets, evals, and audit; ships Task Manager and Goals agents. | MUST | 89 pts | EP-04, EP-05, EP-06, EP-07, EP-09, EP-13 | Done |
| EP-20 | Analytics & insights | Explainable personal metrics and weekly reviews from the event log. | COULD | 55 pts | EP-03, EP-05, EP-09 | Planned |
| EP-21 | Documentation, release 1.0 & hardening | Supportable release: docs, upgrade path, performance, recovery, security review. | MUST | 55 pts | All MUST Epics | Planned |
| EP-22 | AI providers | Agents on Anthropic, OpenAI, Mistral, Gemini, DeepSeek, or a local model, chosen per person. | MUST | 21 pts | EP-19 | Done |
| EP-23 | Visual design refresh | A cinematic, calm interface: imagery, depth, translucency, generous type. | MUST | 21 pts | EP-07 | Done |
| EP-24 | Connect everything | Link any element to any other from wherever it is shown, and see every connection. | MUST | 13 pts | EP-05, EP-12 | Done |
| EP-25 | Notes workspace & Second Brain | A note-taking workspace on par with dedicated apps: tree, live editor, Zettelkasten, PARA, daily notes, templates, quick switcher. | MUST | 34 pts | EP-11 | Done |
| EP-26 | Delivery agents | Side-Projects agent (read-only repository analysis drafting issues) and Code Review agent. | SHOULD | 55 pts | EP-10, EP-19 | Done |
| EP-27 | Tutor & Resource agents | Interview-based, cited draft roadmaps, a curated library of the best resources per domain, and resource finding. | SHOULD | 34 pts | EP-14, EP-19 | Done |
| EP-28 | Demo account | A resettable demo account with realistic sample data, to try every feature without touching real data. | MUST | 13 pts | EP-04, EP-27 | Planned |
| EP-29 | Docs in two languages & API reference | README in English and French with a language switch, an interactive OpenAPI (Swagger) reference, and consistent Axon naming. | MUST | 8 pts | EP-06 | Planned |
| EP-30 | Command dashboard | The home page as a detailed dashboard: KPIs, charts, tables, and a network graph of hard and soft skills. | MUST | 34 pts | EP-08, EP-14 | Planned |
| EP-31 | Settings, languages & YAML configuration | One place for language (English, French), theme, fonts, icon theme, templates, AI keys and limits; editable in the UI or as a YAML file. | MUST | 34 pts | EP-22, EP-25 | Planned |
| EP-32 | Keyboard shortcuts & Vim mode | Shortcuts for every page with a help overlay, and optional Vim keybindings in the note editor. | SHOULD | 13 pts | EP-25, EP-31 | Planned |
| EP-33 | Profile & avatars | Profile pictures with a generated default avatar. | SHOULD | 5 pts | EP-04 | Planned |
| EP-34 | Atomic Habits | James Clear's four laws, habit stacking, implementation intentions, identity, and breaking bad habits. | SHOULD | 21 pts | EP-12 | Planned |
| EP-35 | Focus & Pomodoro | Configurable Pomodoro sessions linked to tasks, projects, skills, and notes, with focus sounds. | SHOULD | 21 pts | EP-09, EP-14 | Planned |
| EP-36 | Live vault sync | The Markdown vault, the notes database, and templates stay in step continuously, not only when a sync runs. | SHOULD | 21 pts | EP-11, EP-25 | Planned |
| EP-37 | Knowledge explorer & NLP graph | Explore notes with PARA and Second Brain filters, charts, and a knowledge graph built from NLP (keywords, entities, synonyms, similarity). | SHOULD | 34 pts | EP-25, EP-36 | Planned |
| EP-38 | Themed spaces | Each area gets its own scenery and effects: a Da Vinci notebook for notes, an Oxford library for learning, a workshop for projects, and more. | SHOULD | 21 pts | EP-23, EP-31 | Planned |
| EP-39 | Personal finance | Accounts, income, expenses, transfers, categories, budgets, recurring transactions, and rules, in the spirit of Firefly III. | SHOULD | 55 pts | EP-04, EP-06, EP-07 | Planned |
| EP-40 | Instance export & import | Move a whole Axon (database, files, settings) to another server or configuration. | MUST | 13 pts | EP-01, EP-31 | Planned |

## Delivery order

Dependencies above are satisfied by this order. Change it only through a human decision recorded in the [change log](#change-log).

| Release | Epics, in order | Exit signal |
|---|---|---|
| R0 — Foundations | EP-00 → EP-01 → EP-02 → EP-03 → EP-04 | Secure deploy, CI, observability, auth, privacy baseline |
| R1 — Personal graph | EP-05 → EP-06 → EP-07 → EP-09 → EP-11 → EP-13 → EP-08 | Notes, links, tasks, goals, and dashboard in one daily loop — no AI required |
| R2 — Guided growth | EP-19 → EP-12 → EP-22 → EP-23 → EP-24 → EP-25 → EP-10 → EP-26 → EP-14 → EP-27 | Human-approved agents, habits, projects, roadmaps |
| R2.5 — Daily driver | EP-28 → EP-29 → EP-30 → EP-31 → EP-32 → EP-33 → EP-34 → EP-35 → EP-16 → EP-15 → EP-36 → EP-37 → EP-38 → EP-39 → EP-40 | Demo, dashboard, settings, focus, retrieval and Anki, gamification across every area, live vault, knowledge graph, finance, portability |
| R3 — Whole-life beta | EP-17 → EP-18 → EP-20 | Media, health, analytics with consent and graceful degradation |
| 1.0 — Durable release | EP-21 | Recovery drill, security review, docs, upgrade path, support policy |

R1 deliberately ships without agents: every feature must be useful with AI disabled. Agent features for R1 domains (Task Manager, Goals) are delivered by EP-19.

## Global release guardrails

- No telemetry leaves the deployment without explicit consent.
- Health, nutrition, and learning outputs are educational aids, not medical, dietary, or professional advice.
- AI-generated plans are drafts until the user approves them.
- The product works fully with AI and every external API disabled.
- Every external API has attribution, caching, quota handling, and deletion behavior documented.
- All user exports are machine-readable and human-readable; erasure is verified, not merely queued.

## Cross-cutting acceptance matrix

These checks apply to every Epic and are attached to its exit evidence. No specialized feature bypasses the platform baseline.

| Concern | Required evidence | Owner at review | Failure response |
|---|---|---|---|
| Identity | Account ownership and session behavior tested | Security reviewer | Block release |
| Authorization | Positive and negative access cases | API reviewer | Block release |
| Privacy | Classification, consent, retention, export, erasure | Privacy owner | Block release |
| Accessibility | Keyboard, focus, labels, contrast, reduced motion | Frontend reviewer | Fix before merge |
| Reliability | Timeout, retry, idempotency, rollback | Platform reviewer | Add runbook |
| Performance | Bounded query, pagination, representative load | Backend reviewer | Profile and split |
| Observability | Metric, structured log, trace, alert or rationale | SRE reviewer | Add instrumentation |
| Documentation | User and operator instructions | Maintainer | Block Done |
| AI safety | Tool scope, prompt handling, approval gate, eval | AI reviewer | Disable mutation |
| Supply chain | Dependency, image, SBOM, license review | Release owner | Pin, patch, or defer |

Rows that do not apply to an Epic (for example, AI safety for EP-01) are marked “n/a” with a one-line reason.

## Epic detail

Detailed Epics list issues with acceptance criteria. Outlined Epics list scope and candidate issues; split them into issues at Epic start.

### EP-00 — Project foundation & governance

**Goal.** Repository, license, decision records, contributor norms, agent rules, and a walking skeleton that later Epics build on.

**Why.** Every later Epic needs a place to land code, a way to record decisions, and rules agents can read. The walking skeleton gives CI (EP-02) and auth (EP-04) something real to build, test, and deploy.

**Out of scope.** CI pipelines (EP-02), hosting (EP-01), any product feature.

**Issues**

- [x] **EP-00/IS-00.01 — Repository and license** · 2 pts · MUST
  - [x] Git repository initialized on `main`.
  - [x] `LICENSE` contains the canonical AGPL-3.0 text.
  - [x] `.gitignore` excludes secrets, `.env` files, and build output.
  - [x] Initial commit made; private GitHub remote created ([ADR 0002](adr/0002-initial-technology-stack.md)).
- [x] **EP-00/IS-00.02 — Decision records** · 3 pts · MUST
  - [x] ADR template and [ADR 0001](adr/0001-record-architecture-decisions.md) exist.
  - [x] [ADR 0002](adr/0002-initial-technology-stack.md) (technology stack) accepted.
  - [x] [ADR 0003](adr/0003-single-host-first.md) (single host first) accepted.
- [x] **EP-00/IS-00.03 — Contributor norms** · 3 pts · MUST
  - [x] [CONTRIBUTING.md](../CONTRIBUTING.md) states workflow, commit, PR, and review rules.
  - [x] [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md) adopts Contributor Covenant 2.1 with a private reporting route.
  - [x] [SECURITY.md](../SECURITY.md) names a private vulnerability-reporting channel.
  - [x] PR and issue templates in `.github/` match the delivery rules.
- [x] **EP-00/IS-00.04 — Agent operating model** · 3 pts · MUST
  - [x] [CLAUDE.md](../CLAUDE.md) states the rules agents follow in this repository.
  - [x] The maintainer delegated confirmation (2026-10-10).
- [x] **EP-00/IS-00.05 — Backlog and README refinement** · 5 pts · MUST
  - [x] EP-00–EP-06 have specific issues and acceptance criteria; later Epics have scope and candidate issues.
  - [x] Every dependency points to an Epic earlier in the delivery order.
  - [x] README links, paths, and roadmap match this backlog.
  - [x] The maintainer delegated approval (2026-10-10).
- [x] **EP-00/IS-00.06 — Walking skeleton** · 13 pts · MUST · depends on IS-00.02
  - [x] Layout: `api/` (Go module), `web/` (SvelteKit + TypeScript), `compose.yaml`, `deploy/Caddyfile`.
  - [x] `GET /healthz` returns `200` with the build version; `GET /readyz` fails when PostgreSQL is unreachable.
  - [x] The web home page calls the API and shows its status, including an error state when the API is down.
  - [x] `docker compose up` starts API, web, PostgreSQL 16 + pgvector, Valkey, and Caddy locally.
  - [x] One migration (enable pgvector) runs up and down cleanly (`make test-integration`).
  - [x] `make check` runs formatting, linting, and unit tests for Go and TypeScript.
  - [x] `.env.example` lists every variable in the README configuration table, with no real secrets.
  - [x] README quick start works on a clean checkout (evidence in the PR).

**Exit evidence.** Skeleton demo log, accepted ADRs, approved backlog, approved PR.

### EP-01 — Infrastructure & IaC

**Goal.** Axon runs on `srv-prod` (production) and `srv-staging` (staging), reachable only over the tailnet, deployed and rolled back with one command, with tested off-host backups.

**Why.** Self-hosting is the product’s promise; it must be reproducible and recoverable before any user data exists. Hosts are already provisioned and configured by the fleet repository [`terraform-proxmox`](https://github.com/thoutmose/terraform-proxmox); [ADR 0004](adr/0004-hosts-and-tailnet-only-access.md) keeps host configuration there and puts only application deployment here.

**Out of scope.** Host provisioning and hardening (fleet repository), a separate database host, public ingress, Kubernetes.

**Issues**

- [x] **EP-01/IS-01.01 — Server Compose overlay** · 3 pts · MUST
  - [x] Servers run prebuilt images with `restart: unless-stopped`; no source tree is needed on the host.
- [x] **EP-01/IS-01.02 — Encrypted environment files** · 3 pts · MUST
  - [x] `deploy/env/{staging,prod}.env.age` are encrypted to the keys in `deploy/env/recipients.txt` and decrypted only during deployment.
  - [x] A secret scan (gitleaks) of the full history finds nothing.
  - [x] Key holders and the recovery procedure for a lost key are documented ([operations](operations.md#secrets)).
- [x] **EP-01/IS-01.03 — Deploy and roll back** · 8 pts · MUST
  - [x] One command builds, copies, migrates, starts, and smoke-tests a version on staging or production.
  - [x] One command switches back to the previous version; the limits (migrations are not reverted) are documented.
- [x] **EP-01/IS-01.04 — Tailnet HTTPS** · 3 pts · MUST
  - [x] Each host serves Axon on its `*.ts.net` name through Tailscale Serve; nothing listens publicly (port 3000 refused from the LAN).
  - [x] HSTS, a baseline Content-Security-Policy, and `X-Content-Type-Options` are set.
- [x] **EP-01/IS-01.05 — Backups and restore** · 8 pts · MUST
  - [x] Nightly encrypted database dumps are kept on the host and copied off-host.
  - [x] Retention is documented, including how long erased data can survive in backups.
  - [x] A restore to a fresh database is rehearsed and timed; initial targets RPO ≤ 24 h, RTO ≤ 4 h (drill: 4 s on staging).
- [x] **EP-01/IS-01.06 — Operator runbook** · 2 pts · MUST
  - [x] Deploy, rollback, backup, and restore steps are documented in `doc/operations.md`.

**Exit evidence.** Staging and production deploy logs, rollback log, restore drill timings, runbook.

### EP-02 — CI/CD & promotion

**Goal.** Every change is checked automatically, built into versioned images, deployed to staging, and promoted to production after human approval, with a rehearsed rollback.

**Why.** Small changes are only cheap when checking and shipping them is automatic and boring.

**Out of scope.** GitOps controllers and Kubernetes ([ADR 0002](adr/0002-initial-technology-stack.md)).

**Issues**

- [x] **EP-02/IS-02.01 — Pull-request checks** · 8 pts · MUST
  - [x] Every PR runs formatting, linting, unit tests, `svelte-check`, and builds. GitHub cannot enforce required checks on a free private repository, so `deploy.sh` refuses any commit without a green CI run.
  - [x] The pipeline finishes in under 10 minutes on a typical PR (first run: ~4 min).
- [x] **EP-02/IS-02.02 — Integration tests** · 5 pts · MUST
  - [x] Integration tests run against a real PostgreSQL 16 + pgvector service, including migrations up and down.
- [x] **EP-02/IS-02.03 — Supply chain** · 8 pts · MUST
  - [x] Base images are pinned by digest.
  - [x] Trivy fails the build on fixable HIGH/CRITICAL vulnerabilities.
  - [x] An SBOM is produced for each image on every PR and attached to each release.
  - [x] Dependabot (built into GitHub, no app to install) opens grouped weekly dependency PRs.
  - [x] Secret scanning runs on every PR.
- [x] **EP-02/IS-02.04 — Release builds** · 5 pts · MUST
  - [x] A version tag produces versioned images in a registry; `/healthz` reports that version.
- [x] **EP-02/IS-02.05 — Deploy, promote, roll back** · 8 pts · MUST
  - [x] One command deploys a version to staging and runs migrations (`deploy/deploy.sh`; GitHub runners cannot reach the tailnet).
  - [x] Promotion to production is a manual command and only accepts the version staging is serving.
  - [x] Rolling back to the previous version is documented and rehearsed (EP-01 staging log).

**Exit evidence.** A PR that went from commit to production through the pipeline, plus a rollback log.

### EP-03 — Observability

**Goal.** The operator can see whether Axon is healthy, find out why it is not, and is alerted before the user notices.

**Why.** Defects in a personal system are found by the user at the worst time unless they are measured first.

**Out of scope.** Product analytics (EP-20). No telemetry leaves the deployment. Host metrics, log shipping, Prometheus, Grafana, Loki, and ntfy already run fleet-wide from `terraform-proxmox`; this Epic adds Axon’s own signals to them and should be re-estimated at start.

**Issues**

- [x] **EP-03/IS-03.01 — Structured logs** · 3 pts · MUST
  - [x] API and worker log JSON with a request ID; no request bodies, tokens, or personal data (redaction test).
- [x] **EP-03/IS-03.02 — Metrics** · 5 pts · MUST
  - [x] Request rate, errors, and duration per route; database pool; host metrics (fleet node_exporter/cAdvisor). Job queue depth arrives with the first queue.
- [ ] **EP-03/IS-03.03 — Traces** · 5 pts · WON’T this release — the fleet has no trace backend; revisit with EP-19 (agent runs)
  - [ ] OpenTelemetry traces cover web → API → database with configured sampling.
- [ ] **EP-03/IS-03.04 — Observability stack** · 8 pts · MUST
  - [ ] The fleet Prometheus on `srv-monitoring` scrapes Axon’s metrics ([terraform-proxmox#1](https://github.com/thoutmose/terraform-proxmox/pull/1), awaiting apply).
  - [ ] Axon dashboards are provisioned in the fleet Grafana (same PR).
- [ ] **EP-03/IS-03.05 — SLOs and alerts** · 5 pts · MUST
  - [x] API availability and latency SLOs are written down ([operations](operations.md#service-level-objectives)).
  - [x] Alerts are defined for service down, database unreachable, error rate, latency, and stale backups; disk is covered by the fleet’s HostOutOfDiskSpace (same PR).
  - [ ] Alerts reach a channel the operator actually reads (e.g. ntfy or email); a test alert is delivered.
- [x] **EP-03/IS-03.06 — Runbooks** · 3 pts · MUST
  - [x] Every alert links to a runbook with diagnosis and recovery steps ([runbooks](runbooks.md)).

**Exit evidence.** Dashboard screenshots, test-alert delivery, runbook index.

### EP-04 — Security, authentication & privacy

**Goal.** Only the owner can reach their data; they sign in with passkeys, can see what happened to their account, and can export, erase, and control consent for their data.

**Why.** Axon will hold health, learning, and personal reflection data. This baseline must exist before any domain data does.

**Out of scope.** SSO/federation and multi-tenant administration (see [ADR 0002](adr/0002-initial-technology-stack.md)).

**Issues**

- [x] **EP-04/IS-04.01 — Accounts and passkeys** · 8 pts · MUST
  - [x] Register and sign in with WebAuthn; an account can hold several passkeys.
  - [x] One-time recovery codes are issued at registration and stored hashed.
- [x] **EP-04/IS-04.02 — TOTP second factor** · 3 pts · SHOULD
  - [x] TOTP enrolment and verification, with secrets encrypted at rest.
- [x] **EP-04/IS-04.03 — Sessions** · 5 pts · MUST
  - [x] Server-side sessions in `Secure`, `HttpOnly`, `SameSite` cookies, rotated at sign-in.
  - [x] Idle and absolute timeouts; “sign out everywhere” revokes all sessions.
  - [x] CSRF protection on every state-changing request.
- [x] **EP-04/IS-04.04 — Ownership checks** · 5 pts · MUST
  - [x] Every user-owned row has an owner; queries filter by it ([ADR 0005](adr/0005-entity-conventions.md)).
  - [x] A shared test harness proves, for every endpoint, that user A cannot read or change user B’s data (`TestUsersCannotSeeOrChangeEachOthersData`).
- [x] **EP-04/IS-04.05 — Abuse protection** · 3 pts · MUST
  - [x] Authentication endpoints are rate-limited with backoff.
- [x] **EP-04/IS-04.06 — Audit log** · 5 pts · MUST
  - [x] Sign-ins, factor changes, exports, erasures, and consent changes are recorded append-only and visible to the user.
- [x] **EP-04/IS-04.07 — Data classification and privacy register** · 5 pts · MUST
  - [x] Each data category has a classification; health data is marked special-category (GDPR Art. 9) ([privacy register](privacy.md)).
  - [x] A processor inventory lists every external service and what it receives.
  - [x] A retention table states how long each category is kept.
- [x] **EP-04/IS-04.08 — Export and erasure framework** · 8 pts · MUST
  - [x] The user can download all their data as JSON plus Markdown.
  - [x] Erasure removes the user’s rows; a test verifies nothing remains outside documented backup windows.
- [x] **EP-04/IS-04.09 — Consent records** · 3 pts · MUST
  - [x] Consent is recorded per purpose (AI agents, each external API), off by default, revocable, and timestamped.
- [ ] **EP-04/IS-04.10 — Threat model and review** · 5 pts · MUST
  - [x] A threat model covers auth, sessions, and data access ([threat model](security/threat-model.md)).
  - [x] OWASP ASVS Level 2 checks for authentication and sessions are recorded ([checklist](security/asvs-checklist.md)).
  - [ ] Someone other than the author reviews the authentication code. An automated security review ran on the PR; a human or external review is still owed before 1.0 (EP-21).

**Exit evidence.** ASVS checklist, cross-user access test report, export and erasure demo, threat model.

### EP-05 — Core data model & universal links

**Goal.** Shared conventions for every entity, typed links between any compatible entities, and an event log that records meaningful changes.

**Why.** Links are what make Axon one graph instead of several apps. Getting them right once avoids a migration in every later Epic.

**Out of scope.** Domain entities themselves (tasks, notes, …), which arrive in their own Epics.

**Issues**

- [x] **EP-05/IS-05.01 — Entity conventions** · 3 pts · MUST
  - [x] An ADR fixes ID format, timestamps, owner column, and soft-delete rules ([ADR 0005](adr/0005-entity-conventions.md), written in EP-04).
  - [x] Migrations run up and down in CI (`TestMigrateUpDown`).
- [x] **EP-05/IS-05.02 — Universal links** · 8 pts · MUST
  - [x] A link stores owner, source and target entity, and relation; types live on `entities` ([ADR 0006](adr/0006-entity-registry-and-links.md)).
  - [x] Only allowed type pairs and relations are accepted; a link cannot cross owners (enforced by composite foreign keys).
  - [x] Deleting an entity removes its links in the same transaction (foreign keys); `axon integrity` runs on every deployment.
  - [x] Forward and backward lookups are indexed and paginated (keyset on UUIDv7, max 100).
- [x] **EP-05/IS-05.03 — Event log** · 8 pts · MUST
  - [x] Each change writes an append-only event (actor, entity, type, time) in the same transaction.
  - [x] Events store changed field names only, never values.
- [x] **EP-05/IS-05.04 — Export and erasure hooks** · 5 pts · MUST
  - [x] Each entity type registers an export handler; erasure cascades from `users` through `entities`.
  - [x] A test fails when a table references `entities` without a registered type (`TestEveryEntityTableIsRegistered`).
- [x] **EP-05/IS-05.05 — Search primitive** · 5 pts · SHOULD
  - [x] PostgreSQL full-text search over entity titles and bodies, scoped to the owner and paginated (language-neutral `simple` configuration).

**Exit evidence.** Schema diagram ([data-model.md](data-model.md)), integrity-check output (deploy log), handler coverage test.

### EP-06 — API platform

**Goal.** A stable, documented REST API that the web app, scripts, sync, and agents all use the same way.

**Why.** One contract keeps the web app, Markdown sync, and agents consistent, and makes mistakes visible in CI.

**Out of scope.** GraphQL ([ADR 0002](adr/0002-initial-technology-stack.md)).

**Issues**

- [x] **EP-06/IS-06.01 — Spec-first OpenAPI 3.1** · 5 pts · MUST
  - [x] The spec lives in the repository (`api/openapi/openapi.yaml`) and is linted in CI (Redocly). Instead of generated server types, contract tests check every request and response the integration tests make against it, and fail if any operation is untested.
  - [x] The spec is served at `/api/v1/openapi.yaml` (load it in any OpenAPI viewer).
- [x] **EP-06/IS-06.02 — Conventions** · 5 pts · MUST
  - [x] All routes are under `/v1`.
  - [x] Errors use RFC 9457 `application/problem+json`.
  - [x] Lists use cursor pagination with a maximum page size (100).
- [x] **EP-06/IS-06.03 — Safe mutations** · 5 pts · MUST
  - [x] `POST` accepts an `Idempotency-Key`; a replay returns the original result; reuse with a different body gets 422.
  - [x] `PATCH` requires `If-Match` (428 without); a stale version returns `412`.
- [x] **EP-06/IS-06.04 — Contract tests** · 5 pts · MUST
  - [x] CI validates real requests and responses against the spec (`internal/app/contract_test.go`).
- [x] **EP-06/IS-06.05 — TypeScript client** · 3 pts · MUST
  - [x] Generated TypeScript types (`web/src/lib/api-types.ts`) are used by the web app; CI fails if they are stale.
- [x] **EP-06/IS-06.06 — Links and timeline endpoints** · 5 pts · MUST
  - [x] Create, list, and delete links; list backlinks; read an entity’s event timeline; search.
- [x] **EP-06/IS-06.07 — Personal access tokens** · 5 pts · SHOULD
  - [x] Read-only or read-write tokens with expiry, shown once, stored hashed, and revocable; tokens can never manage the account.

**Exit evidence.** Published API docs, contract test report, idempotency and concurrency tests.

### EP-07 — Design system & frontend shell

**Goal.** A calm, responsive, accessible SvelteKit shell that every feature plugs into.

**In scope.** Design tokens (type, spacing, colour) with light and dark themes; core components (button, input, dialog, toast, list, empty/loading/error states); app navigation; sign-in, passkey, and account screens; reduced-motion support.

**Out of scope.** Feature screens (owned by each domain Epic).

**Issues**

- [x] **EP-07/IS-07.01 — Tokens and themes** · 5 pts · MUST
  - [x] Type, space, radius, colour, shadow, and motion tokens for light and dark ([ADR 0007](adr/0007-styling-without-tailwind.md)).
  - [x] Theme follows the system, or the user forces light/dark from the header.
- [x] **EP-07/IS-07.02 — Core components** · 8 pts · MUST
  - [x] Button, TextField, Section, List, Dialog (native `<dialog>`), Toast, and Status (loading/error/empty) in `web/src/lib/ui/`.
- [x] **EP-07/IS-07.03 — App shell and navigation** · 5 pts · MUST
  - [x] Sticky header, skip link, signed-in route group that redirects to sign-in and back (`?next=`, same-site paths only).
- [x] **EP-07/IS-07.04 — Auth and account screens** · 8 pts · MUST
  - [x] Sign-in, registration, and account split into overview, sign-in & security, access tokens, privacy & data, and activity.
- [x] **EP-07/IS-07.05 — Accessibility and Playwright CI** · 5 pts · MUST
  - [x] axe reports no serious or critical violations on every shell page (CI `e2e` job).
  - [x] Every account page is reached by keyboard; focus is visible.
  - [x] Layouts fit 360 px without horizontal scrolling (checked per page).
  - [x] Motion tokens drop to 0 ms under reduced motion.
  - [x] Playwright covers registration, navigation, sign-out, and sign-in with a virtual passkey.
- [x] **EP-07/IS-07.06 — Error and offline states** · 3 pts · MUST
  - [x] Offline and server-down banners; every data panel has loading and retryable error states.

### EP-09 — Tasks & recurring work

**Goal.** Capture, plan, and complete daily, weekly, and monthly work with GTD contexts and recurrence.

**In scope.** Task CRUD; inbox capture; due and scheduled dates; today/week/month views; GTD contexts and next actions; recurring tasks (RFC 5545 RRULE); check-ins; links to other entities.

**Out of scope.** Task Manager agent (EP-19), Kanban and projects (EP-10), calendar (EP-08).

**Issues**

- [x] **EP-09/IS-09.01 — Task model and API** · 8 pts · MUST
  - [x] Tasks are graph entities with GTD status, notes, context, scheduled and due dates; create, read, update (If-Match), delete.
  - [x] Tasks appear in export and are removed by erasure.
- [x] **EP-09/IS-09.02 — Capture and inbox** · 5 pts · MUST
  - [x] Press `n` on any signed-in page, type, Enter: the task lands in the inbox (e2e test).
- [x] **EP-09/IS-09.03 — Today, week, and month views** · 8 pts · MUST
  - [x] “Today” follows the account's time zone; views include overdue and upcoming dated work.
  - [x] Views stay under 300 ms and paginated (keyset cursors) with 10,000 tasks.
- [x] **EP-09/IS-09.04 — Contexts and next actions** · 3 pts · MUST
  - [x] `@contexts` with open counts and a filter; next/waiting/someday lists.
- [x] **EP-09/IS-09.05 — Recurrence engine** · 8 pts · MUST
  - [x] Date-based RFC 5545 RRULEs (daily, weekdays, weekly on days, monthly incl. last day, yearly); DST cannot shift dates (property tests over 15 years).
  - [x] Completing a recurring task creates the next occurrence exactly once, even with retries, double clicks, or reopen-and-complete.
- [x] **EP-09/IS-09.06 — Check-ins** · 2 pts · MUST
  - [x] Daily, weekly, and monthly check-ins with an optional note.

### EP-11 — Notes, Zettelkasten, PARA & Markdown sync

**Goal.** Capture and connect knowledge in Markdown, and keep it in sync with `~/SecondBrain` on `srv-second-brain` without ever losing an edit.

**In scope.** Fleeting, literature, and permanent notes; Markdown editor; `[[wikilinks]]` that create universal links; backlinks; PARA organization; Markdown export and import; Git-based sync.

**Out of scope.** Teacher agent and flashcards (EP-16). A note graph view is COULD here; if it is built, record the graph-library choice in a new ADR ([ADR 0002](adr/0002-initial-technology-stack.md) currently defers it to EP-14).

**Issues**

- [x] **EP-11/IS-11.01 — Note model and API** · 5 pts · MUST
  - [x] Notes are graph entities holding the whole Markdown file, front matter included; create, read, replace (If-Match), delete; documented in OpenAPI and contract-tested.
  - [x] Notes appear in the export as Markdown files under `notes/` and are removed by erasure.
- [x] **EP-11/IS-11.02 — Editor** · 8 pts · MUST
  - [x] Notes pages: list by folder, create, read with a rendered preview, edit Markdown and file path, Ctrl+S, unsaved-change guard, delete with confirmation (e2e and axe covered).
  - [x] The preview escapes everything it does not render, so a note cannot inject markup (unit tests).
- [x] **EP-11/IS-11.03 — Wikilinks and backlinks** · 8 pts · MUST
  - [x] `[[name]]`, `[[name#heading]]`, and `[[name|alias]]` become `references` links, resolved by file name then title; links to notes written later resolve when they appear.
  - [x] Renaming or moving a note rewrites links to it in other notes; no link breaks silently.
  - [x] Backlinks and outgoing links on every note; unresolved links offer “Create note”.
- [x] **EP-11/IS-11.04 — PARA organization** · 3 pts · MUST
  - [x] The top folder (`0-Inbox`, `1-Projects`, `2-Areas`, `3-Resources`, `4-Archive`) sets the PARA category; new notes land in the inbox.
- [x] **EP-11/IS-11.05 — Markdown export and import** · 5 pts · MUST
  - [x] Export then import reproduces identical Markdown (golden test with front matter, tables, CRLF, Unicode).
  - [x] ZIP import never overwrites: differing files are reported as conflicts.
- [x] **EP-11/IS-11.06 — One-way import from SecondBrain** · 8 pts · MUST
  - [x] `axon notes-sync` (default mode) imports a vault folder over HTTPS with a write token; reruns are no-ops; hidden folders and symlinks are skipped.
- [x] **EP-11/IS-11.07 — Two-way Git sync with conflicts** · 13 pts · MUST
  - [x] `-two-way` also writes Axon's changes and deletions to the files; `-git` pulls before and commits + pushes after.
  - [x] A note changed on both sides keeps both versions (`<name>.axon-conflict.md`); nothing is overwritten (tested for edits and deletions on either side).
  - [x] Sync runs over the private network only: Axon is served on the tailnet.
- [ ] **EP-11/IS-11.08 — Note graph view** · 5 pts · COULD — not built; deferred to EP-14 with the graph-library ADR.

### EP-13 — Goals & milestones

**Goal.** Turn outcomes into measurable milestones connected to the tasks, habits, and notes that move them.

**In scope.** Goals with a measurable outcome and optional target date; milestones; links to tasks, notes, and (later) habits; progress computed from linked work, with an explanation of how.

**Out of scope.** Goals agent (EP-19).

**Issues**

- [x] **EP-13/IS-13.01 — Goal and milestone model and API** · 5 pts · MUST
  - [x] Goals (title, outcome, status, optional target date, optional measure) and milestones are graph entities; tasks and notes link to them with `supports`.
  - [x] No date or commitment is created without the user entering it: new goals and milestones have no date unless one is sent (integration test).
  - [x] Archiving a goal is a status change: milestones, links, and timeline stay (integration test).
- [x] **EP-13/IS-13.02 — Progress calculation and explanation** · 5 pts · MUST
  - [x] Progress comes from the measure (start → target, capped 0–100%) or from done milestones and linked tasks (cancelled tasks and notes excluded); unit-tested.
  - [x] Every figure expands to exactly what it was computed from (`GET /v1/goals/{id}/progress`, “How is this worked out?”).
- [x] **EP-13/IS-13.03 — Goal screens** · 8 pts · MUST
  - [x] Goals by status with progress bars; goal page with progress, milestones, linked work, details, measure, and delete confirmation (e2e and axe).
- [x] **EP-13/IS-13.04 — Weekly goal review prompt** · 3 pts · SHOULD
  - [x] Until this week's weekly check-in exists, the goals page invites a short review and saves it as that check-in; “Not this week” dismisses it.

### EP-08 — Dashboard & calendar

**Goal.** One trustworthy daily view of priorities, commitments, and progress.

**In scope.** Today view (due and scheduled tasks, active goals, recent notes); calendar of tasks, goals, and milestones with filters (other types join as their Epics land); read-only iCal feed protected by a revocable token.

**Out of scope.** Writing to external calendars.

**Issues**

- [x] **EP-08/IS-08.01 — Today view** · 8 pts · SHOULD
  - [x] `/today` (where signed-in people land) shows today's tasks overdue first, upcoming milestones, active goals with progress, recent notes, and the daily check-in.
  - [x] The dashboard responds in under 300 ms with 10,000 tasks (slowest of 20 loads; integration test).
- [x] **EP-08/IS-08.02 — Calendar with filters** · 8 pts · SHOULD
  - [x] Month agenda of dated tasks, goal target dates, and milestones, filtered by type; items link to their goal.
- [x] **EP-08/IS-08.03 — iCal feed** · 5 pts · SHOULD
  - [x] Read-only feeds contain only the chosen types, open items only, titles only; each has its own random URL token (stored hashed), can be revoked at once, and creation and revocation are audited.
  - [x] Feeds are managed from browser sessions only; access tokens cannot list or create them.
- [x] **EP-08/IS-08.04 — KPI definitions and explanations** · 5 pts · SHOULD
  - [x] Every number carries its definition and links to exactly the items it counts: Overdue, For today, and Inbox equal the task views of the same name (integration test; new `overdue` view).
- [x] **EP-08/IS-08.05 — Empty and onboarding states** · 3 pts · SHOULD
  - [x] A new account sees “Get started” with its first task, note, and goal, ticked off as each appears; every section has an empty state that says what to do next.

### EP-19 — AI agent platform, skills & safety

**Goal.** Agents that help with real work while the user stays in control of every change, every cost, and every piece of data they see.

**In scope.** Model-provider adapter (Anthropic first); tool registry with per-agent allowlists and read/write separation; mutation proposals shown as previews in an approval queue; per-user budgets with a hard stop; consent gate (off by default); redacted, auditable run records; inspectable and deletable agent memory; versioned `SKILL.md` files; eval harness in CI; reference agents **Task Manager** (triage and schedule proposals) and **Goals** (draft milestones).

**Out of scope.** Agents for later Epics (each ships with its domain).

**Acceptance criteria**

- No agent can change data without an approved proposal (tested for every write tool).
- Disabling agents leaves every feature fully usable.
- A run stops when its budget is exhausted, and the user sees why.
- Content from notes, web pages, or imports is treated as data: injection fixtures in the eval suite cannot trigger unapproved tools or send data out.
- Each run records agent, skill version, tools called, cost, and outcome, without raw sensitive content.

**Issues** (design in [ADR 0008](adr/0008-agents-propose-people-approve.md))

- [x] **EP-19/IS-19.01 — Provider adapter and budgets** · 8 pts · MUST
  - [x] Anthropic Go SDK behind a one-method provider interface; no key means agents report “not set up” and nothing else changes (e2e runs without a key).
  - [x] Monthly budget per user (default $2); a run stops before any call whose worst case could cross it, and the person sees why (`budget_exhausted`).
- [x] **EP-19/IS-19.02 — Tool registry and permissions** · 8 pts · MUST
  - [x] Tools are read or write; each skill lists the only tools it may call; anything else is denied and recorded.
- [x] **EP-19/IS-19.03 — Proposals and approval queue** · 13 pts · MUST
  - [x] Every write tool only creates a proposal, validated by a rolled-back dry run; approval (browser sessions only) applies it as `agent:<name>`; stale proposals fail instead of overwriting (tested for every write tool).
- [x] **EP-19/IS-19.04 — Consent and redaction** · 5 pts · MUST
  - [x] Off by default behind the `ai_agents` consent; withdrawing it stops runs at once. Logs and run records never contain prompts, replies, or tool inputs.
- [x] **EP-19/IS-19.05 — Run audit and cost view** · 5 pts · MUST
  - [x] Each run records agent, skill version, model, tools called with outcomes, tokens, cost, and outcome; shown in run history with this month's spend.
- [x] **EP-19/IS-19.06 — Memory** · 8 pts · SHOULD
  - [x] Agents remember only approved facts; the person can list and delete them; memories reach later runs framed as data.
- [x] **EP-19/IS-19.07 — Skills format and loader** · 5 pts · MUST
  - [x] Versioned, embedded `SKILL.md` files (name, version, model, effort, tools, instructions); an invalid skill stops startup (unit tests).
- [x] **EP-19/IS-19.08 — Eval harness with injection fixtures** · 13 pts · MUST
  - [x] CI evals replay scripted model turns against real data, including injected instructions in tasks and notes: denied tools, exfiltration attempts, out-of-allowlist writes, and malformed calls change nothing and create no proposals; no server-side (network) tool is ever offered.
  - [ ] Live-model quality evals (real model against fixtures) are not in CI: they cost money and need a key. Planned for when a provider key is configured.
- [x] **EP-19/IS-19.09 — Task Manager agent** · 13 pts · MUST
  - [x] Clarifies the inbox into proposed statuses, contexts, and new tasks; never invents dates.
- [x] **EP-19/IS-19.10 — Goals agent** · 8 pts · SHOULD
  - [x] Proposes observable milestones and task links; no dates unless the person gave one.

### EP-12 — Habits & Habits agent

**Goal.** Habits designed with cues, stacking, and two-minute starts, tracked gently, with recovery instead of punishment.

**In scope.** Habit model (cue, craving, response, reward, stack, two-minute version, schedule); check-ins; recovery after a miss (“never miss twice”); links to goals; Habits agent suggestions as drafts.

**Issues**

- [x] **EP-12/IS-12.01 — Habit model and API** · 8 pts · MUST
  - [x] Habits are graph entities with cue, craving, response (title), reward, stack (“After I…”), two-minute version, weekday schedule, and status; they can support goals.
- [x] **EP-12/IS-12.02 — Check-in flow** · 5 pts · MUST
  - [x] Done, two-minute version, or planned rest, for today or up to 7 days back (never the future); undo; 28-day history.
- [x] **EP-12/IS-12.03 — Schedules and time zones** · 8 pts · MUST
  - [x] Check-ins are keyed by local calendar date in the user's time zone; tests cover 300 days across DST in three zones and travel from Paris to Tokyo (no double count, no false miss).
- [x] **EP-12/IS-12.04 — Recovery flow** · 5 pts · MUST
  - [x] After a missed scheduled day the habit shows a recovery prompt (“Never miss twice: even the two-minute version counts today”); copy is tested against a list of shaming words and loss framing; rest days and planned skips are never misses.
- [x] **EP-12/IS-12.05 — Habit screens** · 13 pts · MUST
  - [x] Today's habits with Done / two-minute / rest buttons and undo; habit page with four-week history, goal links, design fields, and delete confirmation (e2e and axe).
- [x] **EP-12/IS-12.06 — Habits agent** · 13 pts · SHOULD
  - [x] Proposes new habits and design changes as drafts the user approves (`create_habit`, `update_habit`; covered by the every-write-tool test); its skill forbids shame, pressure, and medical advice. Habits work fully with agents off (e2e runs without a provider).

### EP-10 — Projects, Kanban & docs

**Goal.** Run personal and side projects in Axon: projects, epics, issues, sub-tasks, Kanban, documentation, repository analysis, and code review.

**In scope.** Project hierarchy built on tasks; Kanban with WIP limits; project docs as notes. The delivery agents moved to EP-26.

**Issues**

- [x] **EP-10/IS-10.01 — Project and hierarchy model** · 8 pts · MUST
  - [x] Projects are graph entities (status, WIP limit, docs folder) that can support goals; epics, issues, and sub-tasks are tasks with a card, with hierarchy rules enforced (integration test).
- [x] **EP-10/IS-10.02 — Kanban** · 13 pts · MUST
  - [x] Backlog, To do, Doing, Done; drag and drop, Shift+arrow keys, and a “Move to” menu; moves are announced and keep focus (e2e).
  - [x] The WIP limit is shown on Doing; going over answers 409 and the board asks for confirmation; nothing is blocked silently (integration and e2e).
  - [x] Done on the board completes the task; a task completed elsewhere shows in Done; moving it out reopens it.
- [x] **EP-10/IS-10.03 — Project docs** · 8 pts · MUST
  - [x] Notes in the project's folder (`1-Projects/<slug>` by default) are its docs; “New doc” creates one there.
- [x] **EP-10/IS-10.04 — Project screens** · 5 pts · MUST
  - [x] Projects by status with progress; project page with board, docs, connections, settings, and delete (tasks and docs stay).

### EP-14 — Roadmaps & skill graph

**Goal.** Build learning and career roadmaps by hand, by import, or with the Tutor agent, and see progress as a skill graph.

**In scope.** Manual roadmaps linked to goals, notes, and learning items; import from openly licensed curricula; skill graph view. The Tutor and Resource agents moved to EP-27.

**Issues** (design in [ADR 0013](adr/0013-roadmaps-licences-and-graph.md))

- [x] **EP-14/IS-14.01 — Licence review** · 2 pts · MUST
  - [x] OSSU: MIT, imported with attribution shown before and after import. roadmap.sh: personal use only, no redistribution, so not imported (link only).
- [x] **EP-14/IS-14.02 — Roadmap model and API** · 8 pts · MUST
  - [x] Roadmaps and skills as graph entities; prerequisites as `requires` links; progress that leaves out skipped skills; a 500-skill roadmap loads in under 300 ms (integration test).
- [x] **EP-14/IS-14.03 — Manual editor** · 13 pts · MUST
  - [x] Create roadmaps, add skills with sections, links, and prerequisites; set status from the list or the skill panel; connect skills and roadmaps to goals and notes.
- [x] **EP-14/IS-14.04 — OSSU import** · 8 pts · SHOULD
  - [x] Bundled snapshot: 63 courses, sections, links, effort, 24 prerequisite edges.
- [x] **EP-14/IS-14.05 — Skill graph view** · 13 pts · SHOULD
  - [x] Layered SVG graph with zoom and status colours; named links for screen readers; the list view is the accessible alternative; the layout is unit-tested at 500 nodes (e2e and axe on both views).

### EP-27 — Tutor & Resource agents

**Goal.** Build roadmaps with the Tutor agent (interview, cited draft roadmaps) and find learning resources with the Resource agent. Split from EP-14.

**Issues** (design in [ADR 0014](adr/0014-tutor-library-and-web-reading.md))

- [x] **EP-27/IS-27.01 — Curated library** · 8 pts · MUST
  - [x] 24 domains, 258 widely recommended books, papers, university courses, videos, channels, and practice sites, chosen for the owner's domains (CS, DSA, maths, OS, Linux/Bash, C/C++, Java, Python, Rust, networking, DevOps/GitOps, Docker/Podman/Kubernetes, web/full stack, API design, system design/data-intensive, databases, data engineering/DataOps/big data, data analysis, data science, ML, MLOps, security/DevSecOps, cloud, soft skills). Titles and links only; every link checked when written.
  - [x] Each domain imports as an ordinary roadmap with prerequisites, and can then be renamed, edited, or deleted (UI and API).
- [x] **EP-27/IS-27.02 — Allowlisted page reading** · 5 pts · MUST
  - [x] `fetch_page`: allowlisted educational hosts (redirects re-checked), https only, private addresses refused, robots.txt honoured, 15 pages per run, text only, framed as data (unit and integration tests).
- [x] **EP-27/IS-27.03 — Conversations** · 5 pts · MUST
  - [x] Answer an agent from the Agents page (`POST /v1/agent-runs/{id}/continue`); the transcript lives in memory for an hour and is never stored; `parent_run_id` links runs; only the owner can continue (tested).
- [x] **EP-27/IS-27.04 — Tutor agent** · 10 pts · SHOULD
  - [x] Interviews first, then proposes `draft_roadmap`: every skill needs a source link and prerequisites name earlier skills; nothing is created before approval (tested, including uncited and refused fetches).
- [x] **EP-27/IS-27.05 — Resource agent** · 6 pts · SHOULD
  - [x] Searches the library and allowlisted pages, and proposes `suggest_resources` with links and attribution; approval adds them as skills.

**Exit evidence.** API unit and integration tests (including the every-write-tool and injection evals), contract coverage, and 19 Playwright scenarios with axe pass. Cross-cutting matrix: identity, authorization (owner-only continue), privacy (new processor row, transcripts memory-only), accessibility (dialogs, labelled thread), reliability (fetch timeouts, bounded pages), performance (bounded tool output), observability (content-free run records), documentation (ADR 0014, README), AI safety (allowlist, citations, approval), supply chain (no new dependency).

### EP-28 — Demo account

**Goal.** Anyone with access to the server can try every feature on realistic data, without a passkey and without seeing or touching real accounts.

**In scope.** `axon demo seed` creates (or resets) one demo user with sample tasks, projects, notes (Zettelkasten and PARA), goals, habits, roadmaps from the library, and agent proposals. `AXON_DEMO=1` shows a "Try the demo" button that signs into that user with a short, demo-only session. A nightly reset. Agents use a scripted provider in the demo, so no AI key or spend is involved.

**Acceptance criteria**

- The demo session cannot reach any other user's data, tokens, settings, or exports (authorization tests).
- The demo cannot register passkeys, create tokens, change providers, or send anything outside Axon.
- Staging and production can each run a demo; real accounts are unaffected (integration test).

**Candidate issues.** Seed data (5) · Demo sign-in and limits (5) · Reset job and docs (3).

### EP-29 — Docs in two languages & API reference

**Goal.** README in English and French with a switch at the top of each, an interactive API reference, and one name everywhere: Axon.

**In scope.** `README.fr.md` kept in step with `README.md` (a CI check compares their section lists); Swagger UI served at `/api/docs` from the bundled OpenAPI document; a repository sweep for leftover names.

**Acceptance criteria**

- Both READMEs describe the current state and link to each other.
- `/api/docs` renders the live OpenAPI document without loading anything from a CDN.

**Candidate issues.** French README and section check (3) · Swagger UI (3) · Naming sweep (2).

### EP-30 — Command dashboard

**Goal.** The home page answers "how am I doing?" at a glance: KPIs, trends, tables, and a skills network.

**In scope.** KPI tiles (tasks done, focus time, habit consistency, goal progress, learning, notes written) with week-over-week change; charts (trends, heatmap, burn-down); tables (due soon, stalled projects, reviews due); a network graph of skills from roadmaps, tagged hard or soft, linked to the goals, projects, and notes that use them. Every number opens the items behind it (as in EP-08).

**Acceptance criteria**

- Every KPI has a written definition, and its number matches the items it opens (tested).
- Charts have table alternatives and pass axe; the page loads in under 500 ms with a year of data.

**Candidate issues.** KPI service (8) · Charts (8) · Tables (5) · Skills network with hard/soft tags (13).

### EP-31 — Settings, languages & YAML configuration

**Goal.** One place to configure Axon, in the browser or as a YAML file.

**In scope.** Language (English and French UI, translated strings with a fallback), theme, font family and size, icon theme, default templates for notes, projects, and resources, AI provider keys and monthly limits; `GET/PUT /v1/me/settings` and YAML export and import of the same document, validated against a schema.

**Acceptance criteria**

- A YAML file round-trips without loss; invalid files are rejected with line-level messages.
- Secrets in YAML are write-only: export never contains keys.
- The whole UI is available in French (no missing-string fallbacks in e2e).

**Candidate issues.** Settings model and API (8) · YAML import and export (5) · i18n framework and French strings (13) · Appearance settings (5) · Template settings (3).

### EP-32 — Keyboard shortcuts & Vim mode

**Goal.** Fast, discoverable keyboard use everywhere, and Vim keybindings for people who write notes in Vim.

**In scope.** A shortcut registry with a `?` help overlay; go-to chords (`g t` Today, `g n` Notes, …); per-page actions; optional Vim mode in the Markdown editor (normal, insert, visual; motions, operators, `:w`), turned on in settings.

**Acceptance criteria**

- Shortcuts never fire while typing in a field (except editor mode keys) and can be turned off (WCAG 2.1.4).
- Vim mode is covered by unit tests for its motions and operators.

**Candidate issues.** Shortcut registry and overlay (5) · Vim mode (8).

### EP-33 — Profile & avatars

**Goal.** A face for every account: an uploaded picture or a generated default.

**In scope.** Upload (PNG, JPEG, WebP; resized and stripped of metadata on the server), a deterministic generated avatar by default, shown in the header and on shared items.

**Acceptance criteria**

- Uploaded images are re-encoded (no EXIF location survives) and limited in size.
- Export includes the picture; erasure removes it.

**Candidate issues.** Upload and processing (3) · Default avatar and display (2).

### EP-34 — Atomic Habits

**Goal.** Habits designed with James Clear's method: make good habits obvious, attractive, easy, and satisfying; make bad habits invisible, unattractive, difficult, and unsatisfying.

**In scope.** Habit stacking ("After I [current habit], I will [new habit]") and implementation intentions ("I will [behaviour] at [time] in [location]") as structured fields; identity statements ("I am a person who…") with votes cast by check-ins; the habits scorecard; the two-minute rule and gradual levelling up; bad habits with inversion strategies; a habit contract with an accountability partner (optional, opt-in).

**Acceptance criteria**

- Each law appears as a guided step when designing a habit, and the Habits agent uses the same model.
- Stacks show as a chain on the habits page and on Today.

**Candidate issues.** Model and API (8) · Design flow (8) · Scorecard and identity (5).

### EP-35 — Focus & Pomodoro

**Goal.** Focused work sessions that count everywhere.

**In scope.** A session is 25 min work, 5 min break, 25 work, 5 break, 25 work, 15 min long break by default. Durations, the number of rounds, and the number of sessions are editable (one session by default). Each session links to a task, project, skill, note, or review; minutes feed KPIs, streaks, and XP. Focus sounds generated in the browser (white, pink, brown noise, rain), plus music links the person chooses for each kind of work.

**Acceptance criteria**

- The timer stays correct across tab sleep and reloads (tested with a fake clock).
- No third-party audio is loaded unless the person added it.

**Candidate issues.** Timer and settings (8) · Links and KPIs (5) · Focus sounds (8).

### EP-36 — Live vault sync

**Goal.** Adding, editing, renaming, or deleting a Markdown file in the vault updates the note in Axon within seconds, and the other way round.

**In scope.** `axon notes-sync -watch` (file events plus a periodic scan) as a service on `srv-second-brain`; Axon changes pushed through an event stream; templates synced as files; conflict handling unchanged (both versions kept).

**Acceptance criteria**

- Create, edit, rename, and delete propagate both ways within 10 s (integration test).
- No change is lost when both sides edit the same note (tested).

**Candidate issues.** Watch mode (8) · Change stream (8) · Templates as files (5).

### EP-37 — Knowledge explorer & NLP graph

**Goal.** See what you know: filter, chart, and navigate notes as a knowledge graph.

**In scope.** Filters by PARA area, Zettelkasten type, tags, dates, and links; charts (notes over time, tag clouds, orphan notes); a knowledge graph combining wikilinks with NLP edges: keywords (TF-IDF), named entities, synonyms from an open thesaurus (WordNet), and semantic similarity (pgvector embeddings when an embedding provider is configured; local methods otherwise).

**Acceptance criteria**

- Every NLP edge says why it exists (shared keyword, entity, synonym, or similarity score).
- Works fully without any AI provider.

**Candidate issues.** Filters and charts (8) · Keyword and entity extraction (8) · Thesaurus (5) · Embeddings (8) · Graph view (5).

### EP-38 — Themed spaces

**Goal.** Each area has its own atmosphere, and the visual language from EP-23 continues everywhere.

**In scope.** Bundled, licence-clear artwork and effects:
- notes: a Da Vinci notebook, with parchment, sepia sketches, and ink;
- learning and resources: an Oxford library, with wood, leather, warm lamps, and serif type;
- projects: a workshop or blueprint;
- the other areas follow the same pattern.

Effects are subtle (parallax, grain, light) and respect reduced motion and reduced transparency. Themes can be turned off in settings.

**Acceptance criteria**

- Text contrast passes WCAG AA on every themed surface (axe in e2e).
- All artwork is original or openly licensed, with attribution in the repository.

**Candidate issues.** Theme system per area (8) · Artwork (8) · Effects and fallbacks (5).

### EP-39 — Personal finance

**Goal.** Know where money comes from and goes, in the spirit of Firefly III, without sending data anywhere.

**In scope.** Asset, liability, revenue, and expense accounts; double-entry transactions (income, expense, transfer) with categories, tags, and notes; budgets per period; recurring transactions; rules that categorise automatically; CSV import; reports; full API; finance data classified sensitive, encrypted at rest where supported, excluded from agents unless a separate consent is granted.

**Acceptance criteria**

- Balances always reconcile (property tests on double entry).
- Recurring transactions and rules are idempotent (re-running creates nothing twice).
- Finance endpoints pass the authorization matrix; amounts use exact decimals, never floats.

**Candidate issues.** Accounts and transactions (13) · Categories, tags, budgets (8) · Recurring and rules (13) · CSV import (8) · Reports and UI (13). **Split before start.**

### EP-40 — Instance export & import

**Goal.** Move an entire Axon to another server or configuration and keep everything.

**In scope.** `axon instance export` (database dump, settings, uploaded files, a manifest with versions and checksums) and `axon instance import` into an empty installation, with migrations applied; documented in operations.

**Acceptance criteria**

- An export imported on a fresh stack passes the restore drill (integration test).
- Secrets sealed with `AXON_ENCRYPTION_KEY` are re-sealed or require the same key, as documented.

**Candidate issues.** Export (5) · Import (5) · Drill and docs (3).

### EP-16 — Learning engine, Teacher agent & Anki

**Goal.** Retrieval-first learning: spaced scheduling, reviewed practice questions, dual-coded cards, and Anki sync.

**In scope.** FSRS scheduling (SM-2 fallback); review sessions and tests on the website; illustrated flashcards (image, cloze, front/back) that the person can view, add, edit, and delete; Teacher agent generating questions and cards from literature and permanent notes only; approval before cards enter the deck; Anki `.apkg` export and automatic sync of approved cards to the `anki-sync-server` on `srv-services`, by hand or after a Teacher proposal is approved.

**Acceptance criteria**

- FSRS results match the reference implementation on shared fixtures.
- The Teacher agent cannot read fleeting notes (tested).
- Exported decks import into Anki desktop without errors.

**Candidate issues.** Card and review model with images (8) · FSRS scheduler (8) · Review session and test UI (13) · Flashcard editor (8) · Teacher agent (21) · Card approval queue (8) · Anki export (8) · Anki sync to `srv-services` (13). **Split before start.**

### EP-15 — Gamification engine

**Goal.** Recognize meaningful progress with transparent, opt-in XP, levels, badges, streaks, and freezes bought only with earned points.

**In scope.** XP rules, levels, badges, the streak state machine (see README), earned-point freezes. Every area feeds the same engine: tasks, projects, habits, goals, notes (Zettelkasten, PARA, daily notes), roadmaps and reviews, Pomodoro sessions, and media and book trackers, so one streak and one progress view reflect the whole life. Shared experiences (opt-in sharing of a badge or a streak with another person on the same Axon) are in scope only if they need no public exposure.

**Acceptance criteria**

- The user can see exactly how every point was earned.
- Gamification can be turned off entirely, hiding every surface.
- Freezes cannot be bought with money.
- The streak state machine is covered by property tests, including time zones.
- No notification uses loss-aversion or shaming language.

**Candidate issues.** XP and rule engine (13) · Streak state machine (8) · Freezes (5) · Badges (8) · Levels and progress screens (13) · Settings and opt-out (3).

### EP-17 — Media trackers & Media agent

**Goal.** Track films, series, and books with metadata from TMDB and Open Library, and get recommendations whose reasons are shown.

**In scope.** Media items with want / in progress / done states and ratings; TMDB and Open Library adapters with caching, quotas, and attribution; Media agent recommendations.

**Acceptance criteria**

- TMDB and Open Library attribution and terms are met (checked against current terms at Epic start).
- Tracking works with both providers unavailable.
- Each recommendation states why it was made.

**Candidate issues.** Media model and API (5) · TMDB adapter (5) · Open Library adapter (5) · Media screens (8) · Media agent (8).

### EP-18 — Health, nutrition, sport & coach agents

**Goal.** Log food and training privately, with conservative, educational coaching that the user explicitly opts into.

**In scope.** Food logging via Open Food Facts and USDA FoodData Central; workout logging; Personal Trainer and Nutritionist agents.

**Acceptance criteria**

- Health data requires explicit consent (GDPR Art. 9) before any is stored.
- Health data is never sent to a model provider without separate consent for that purpose.
- Coach agents never diagnose, never propose unsafe restriction, and direct red-flag topics to professionals (eval suite).
- Logging works with all external providers unavailable.

**Candidate issues.** Consent flow for health data (5) · Food model and adapters (13) · Food logging UI (13) · Workout model and UI (13) · Personal Trainer agent (21) · Nutritionist agent (21). **Split before start.**

### EP-20 — Analytics & insights

**Goal.** Explainable personal metrics and weekly reviews computed locally from the event log.

**In scope.** Metric definitions; weekly review report; trends for tasks, habits, goals; CSV export.

**Acceptance criteria**

- Every metric documents its formula and links to its source events.
- All computation stays on the deployment.
- Reports stay bounded in time and memory with a year of events.

**Candidate issues.** Metric definitions and projections (13) · Weekly review (13) · Trend views (13) · CSV export (5) · Performance work (8).

### EP-22 — AI providers

**Goal.** Let each person run agents on the provider they prefer, including a model on their own hardware, without weakening any EP-19 guarantee.

**In scope.** Provider settings per person (provider, model, key or local endpoint, prices for the budget), keys encrypted at rest; Anthropic through the official SDK with an API key or a server-side OAuth profile (`ant auth login`, no key to manage); OpenAI, Mistral, Gemini, DeepSeek, and local servers (Ollama, LM Studio, llama.cpp) through their OpenAI-compatible chat API; a connection test.

**Out of scope.** Signing in with a consumer chat subscription (Claude.ai Pro/Max, ChatGPT Plus): providers do not permit third-party apps to use those plans.

**Issues** (design in [ADR 0009](adr/0009-ai-providers.md))

- [x] **EP-22/IS-22.01 — Provider settings** · 5 pts · MUST
  - [x] Per-person provider, model, key, local address, and prices; keys sealed with `AXON_ENCRYPTION_KEY`, never returned, logged, or exported; browser sessions only (integration test).
- [x] **EP-22/IS-22.02 — OpenAI-compatible adapter** · 8 pts · MUST
  - [x] OpenAI, Mistral, Gemini, DeepSeek, and local servers share one translated wire format; a full agent run against a fake local server proposes and never writes, with no key sent (integration test); translation unit-tested both ways.
- [x] **EP-22/IS-22.03 — Anthropic without a stored key** · 3 pts · SHOULD
  - [x] `AXON_ANTHROPIC_OAUTH=1` uses an `ant auth login` profile on the server through the SDK's own credential resolution.
- [x] **EP-22/IS-22.04 — Settings screen and connection test** · 5 pts · MUST
  - [x] Agents page: choose provider, model, address, key, prices; test the connection with a request that carries no personal data (e2e).

Consumer chat subscriptions (Claude Pro/Max, ChatGPT Plus) are out of scope: providers do not permit third-party apps to use them.

### EP-23 — Visual design refresh

**Goal.** An interface that feels as considered as a flagship product page: full-bleed imagery, depth, translucent glass, and generous typography, while staying fast, calm, and accessible.

**In scope.** Ambient background artwork bundled with the app (no third-party image hosts), glass surfaces with backdrop blur, a hero on Today and the home page, refined type scale and motion, light and dark themes.

**Issues** (design in [ADR 0010](adr/0010-visual-language.md))

- [x] **EP-23/IS-23.01 — Tokens, glass, and colour field** · 8 pts · MUST
  - [x] Neutral light and dark tokens, translucent surfaces with backdrop blur, pill buttons, display type; reduced transparency gives solid surfaces; reduced motion stops animation (e2e).
- [x] **EP-23/IS-23.02 — Bundled scenery and heroes** · 8 pts · MUST
  - [x] SVG landscape in four time-of-day palettes; full-bleed hero on the home page (with feature sections) and Today (with a two-column layout); no cross-origin requests (e2e).
- [x] **EP-23/IS-23.03 — Navigation polish** · 5 pts · SHOULD
  - [x] Logo mark, current-page highlight, SVG theme icons; menu scrolls inside itself on 360 px screens (e2e).
  - [x] Text contrast at AA on every page checked by axe; entrance motion never lowers contrast.

### EP-24 — Connect everything

**Goal.** Any element can be connected to any other from wherever it appears, and every connection is visible from both sides.

**In scope.** A reusable “Connections” panel on tasks, notes, goals, milestones, and habits: search any entity, choose a relation, see outgoing links and backlinks, remove a link; a task detail page.

**Issues**

- [x] **EP-24/IS-24.01 — Connections panel** · 8 pts · MUST
  - [x] One panel on task, note, goal, and habit pages: search anything by part of its title, choose how they connect (both directions, specific relations first), see every connection with its wording (“Supports” / “Supported by”, “Mentions” / “Mentioned in”), remove links. Automatic wikilink references are shown but not editable.
  - [x] Works by keyboard alone (e2e connects a task to a note without the mouse) and passes axe.
- [x] **EP-24/IS-24.02 — Task page** · 3 pts · MUST
  - [x] `/tasks/{id}` with the task's details, editor, and connections; reachable from the task editor.
- [x] **EP-24/IS-24.03 — Search while typing** · 2 pts · MUST
  - [x] Search also matches part of a title (LIKE wildcards escaped); queries with operators keep exact web-search meaning (integration test).

### EP-25 — Notes workspace & Second Brain

**Goal.** Make Axon the place to think in notes: a workspace on par with dedicated Markdown note apps, built for Zettelkasten, PARA, and a Second Brain.

**In scope.** Sidebar folder tree with PARA groups; split editor with live preview and formatting shortcuts; autosave; quick switcher; full-text search; Zettelkasten IDs and note types (fleeting, literature, permanent); daily notes; templates; tags; unlinked mentions; capture to inbox from anywhere.

**Issues** (design in [ADR 0011](adr/0011-notes-workspace.md))

- [x] **EP-25/IS-25.01 — Workspace and tree** · 5 pts · MUST
  - [x] Sidebar with PARA groups and nested folders, counts, current note, tags; new note and today's note; collapses on small screens.
- [x] **EP-25/IS-25.02 — Editor** · 8 pts · MUST
  - [x] Split editor with live preview, toolbar and shortcuts, list continuation, `[[` suggestions (keyboard), word count; autosave with conflict detection that never overwrites; explicit move/rename (e2e).
- [x] **EP-25/IS-25.03 — Rich Markdown** · 3 pts · MUST
  - [x] Tables with alignment, callouts, highlight, strikethrough, tags, heading anchors, wikilinks shown by title; escape-first and unit-tested.
- [x] **EP-25/IS-25.04 — Zettelkasten, PARA, and daily notes** · 5 pts · MUST
  - [x] Templates for fleeting, literature, permanent (with id), project, area, resource, meeting, and blank notes; daily note linked to yesterday (unit and e2e tests).
- [x] **EP-25/IS-25.05 — Tags and search** · 5 pts · MUST
  - [x] Tags from front matter and `#tags`, filter and counts; full-text note search with highlighted excerpts (integration and e2e tests).
- [x] **EP-25/IS-25.06 — Command center and capture** · 5 pts · MUST
  - [x] Ctrl/Cmd+K searches everything and runs commands; the capture dialog (`n`) can save a fleeting note (e2e).
- [x] **EP-25/IS-25.07 — Unlinked mentions** · 3 pts · SHOULD
  - [x] Notes that name the current note without linking are listed, with a one-click link.

LaTeX, diagrams, and embedded media are not rendered yet (ADR 0011).

### EP-26 — Delivery agents

**Goal.** Side-Projects agent and Code Review agent, split from EP-10.

**Issues** (design in [ADR 0012](adr/0012-delivery-agents.md))

- [x] **EP-26/IS-26.01 — Repository connector, read-only** · 13 pts · MUST
  - [x] Per-person GitHub connections with sealed optional tokens, managed from browser sessions only; the client can only read, plus one plain comment endpoint refused unless comments are allowed (off by default).
- [x] **EP-26/IS-26.02 — Side-Projects agent** · 21 pts · SHOULD
  - [x] Reads the tree, files, and pull requests; drafts issues that become backlog cards only after approval (integration test with a fake GitHub and an injected README).
- [x] **EP-26/IS-26.03 — Code Review agent** · 21 pts · SHOULD
  - [x] Reviews diffs; can propose a comment, posted once after approval and only where allowed; no approve, merge, or review tool exists (tested). External writes are validated with `Check`, never dry-run, so nothing reaches GitHub before approval.

### EP-21 — Documentation, release 1.0 & hardening

**Goal.** A release someone other than the author can install, operate, upgrade, and recover.

**In scope.** User and operator documentation; upgrade path from the previous release; load and performance tests; full recovery drill; external security review; support and deprecation policy; revisit the single-host decision ([ADR 0003](adr/0003-single-host-first.md)) with measurements.

**Acceptance criteria**

- A new operator installs Axon from the docs alone (observed run).
- Upgrading from the previous release keeps all data (tested).
- The recovery drill meets the documented RPO and RTO.
- The external security review finds no open high-severity issues.

**Candidate issues.** User docs (8) · Operator docs (8) · Upgrade testing (8) · Load tests (8) · Recovery drill (5) · Security review fixes (13) · Support policy (5).

## Glossary

| Term | Meaning |
|---|---|
| Axon | Working name for this self-hosted life OS. |
| PARA | Projects, Areas, Resources, Archives organization method. |
| CODE | Capture, Organize, Distill, Express. |
| Zettelkasten | Linked note practice using fleeting, literature, and permanent notes. |
| Universal link | A typed edge between any compatible Axon entities. |
| Event log | Append-only record of meaningful changes used for audit and projections. |
| FSRS / SM-2 | Spaced-repetition scheduling algorithms; FSRS is the recommended default. |
| Agent | A bounded AI workflow with tools, memory, evaluation, and approval rules. |
| Skill | Versioned Markdown instructions and tests an agent may load for a capability. |
| Walking skeleton | The thinnest end-to-end build that deploys and runs, used to grow features on. |
| SLO | Service-level objective that defines a measurable reliability target. |
| ADR | Architecture Decision Record. |
| RPO / RTO | Maximum acceptable data loss / recovery time. |

## Sources and bibliography

- [roadmap.sh](https://roadmap.sh/)
- [OSSU](https://github.com/ossu/computer-science)
- [Anki sync](https://github.com/ankicommunity/anki-sync-server)
- [Open Food Facts](https://world.openfoodfacts.org/data)
- [TMDB](https://developer.themoviedb.org/docs/getting-started)
- [Open Library](https://openlibrary.org/developers/api)
- [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/)
- [Twelve-Factor](https://12factor.net/)
- [SRE](https://sre.google/sre-book/table-of-contents/)
- [DDIA](https://dataintensive.net/)
- [Atomic Habits](https://jamesclear.com/atomic-habits)
- [Second Brain](https://fortelabs.com/blog/basboverview/)
- [Smart Notes](https://takesmartnotes.com/)
- [FSRS](https://github.com/open-spaced-repetition/free-spaced-repetition-scheduler)
- [Anthropic](https://docs.anthropic.com/en/docs)
- [OpenAPI](https://spec.openapis.org/oas/latest.html)
- [RFC 9457 Problem Details](https://www.rfc-editor.org/rfc/rfc9457)
- [PostgreSQL](https://www.postgresql.org/docs/16/)
- [SvelteKit](https://svelte.dev/docs/kit)
- [Go](https://go.dev/doc/)
- [GDPR](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
- [Prometheus](https://prometheus.io/docs/introduction/overview/)
- [CIS](https://www.cisecurity.org/controls)
- [WebAuthn](https://www.w3.org/TR/webauthn-3/)
- [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
- [Trivy](https://aquasecurity.github.io/trivy/)
- [SLSA](https://slsa.dev/spec/v1.0/)

Additional books informing product behavior: *Understanding How We Learn* (Weinstein, Sumeracki, Caviglioli); *Moonwalking with Einstein* (Foer); *Make It Stick* (Brown, Roediger, McDaniel); *Hooked* (Eyal); *The Octalysis Framework* (Chou); *Building a Second Brain* (Forte); *How to Take Smart Notes* (Ahrens); *Atomic Habits* (Clear); *Designing Data-Intensive Applications* (Kleppmann).

## Change log

| Date | Change |
|---|---|
| 2026-10-10 | Initial authored backlog for Axon; all Epics are Planned pending human refinement. |
| 2026-10-10 | Refinement (EP-00/IS-00.05): replaced templated Epic text with specific goals, scope, and acceptance criteria; detailed EP-00–EP-06 into issues. Fixed dependencies so each points to an earlier Epic: EP-08 now follows EP-09/11/13; Task Manager and Goals agents moved into EP-19, so EP-09 and EP-13 no longer depend on it; EP-21 depends on MUST Epics instead of EP-20. EP-01 reduced to one host per ADR 0003 (55 → 34 pts); EP-09 55 → 34 and EP-13 34 → 21 after moving their agents. Added delivery order and a walking skeleton to EP-00. |
| 2026-10-10 | Owner request: added EP-22 (AI providers), EP-23 (visual refresh), EP-24 (connect everything), EP-25 (notes workspace & Second Brain), scheduled before EP-10. Split EP-10 as its text required: delivery agents moved to new EP-26. |
| 2026-10-11 | EP-14 split as its text required: the Tutor and Resource agents moved to new EP-27; roadmap.sh import dropped for licence reasons (ADR 0013). |
| 2026-10-11 | Owner request: EP-27 gained a curated library of the best resources for the owner's domains. Added EP-28 (demo), EP-29 (bilingual docs, API reference), EP-30 (dashboard), EP-31 (settings, i18n, YAML), EP-32 (shortcuts, Vim), EP-33 (avatars), EP-34 (Atomic Habits), EP-35 (Pomodoro), EP-36 (live vault sync), EP-37 (knowledge explorer, NLP), EP-38 (themed spaces), EP-39 (personal finance), EP-40 (instance export/import) as release R2.5, before EP-16 and EP-15, which now include flashcard editing, Anki sync to `srv-services`, and gamification across every area. The backend stays in Go (ADR 0002); a Rust rewrite was considered and not scheduled. |

## Standard issue evidence template

- [ ] User outcome is written in one sentence.
- [ ] Scope boundaries name what is intentionally excluded.
- [ ] Inputs and outputs have examples.
- [ ] Error and degraded behavior are specified.
- [ ] Data owner, visibility, and retention are specified.
- [ ] External calls, quotas, attribution, and cache policy are specified.
- [ ] Migration, backfill, and rollback behavior are specified.
- [ ] Test layers and fixtures are named.
- [ ] Telemetry and redaction are named.
- [ ] Documentation destination is named.
- [ ] Human approval points are named.
- [ ] Reviewer and release evidence are recorded.

## Agent task protocol

1. Load the active Epic context and its approved skills.
2. Read the target files, tests, schemas, and nearby documentation before proposing edits.
3. State assumptions and ask when an assumption could affect user data or security.
4. Make the smallest coherent diff that satisfies the issue.
5. Run focused checks first, then the relevant suite.
6. Inspect the resulting diff for accidental scope, secrets, generated noise, and unsafe defaults.
7. Write a short reflection: what worked, what failed, what should become a skill or test.
8. Present mutations as a preview when approval is required.
9. Update the issue, docs, and evidence before declaring completion.
10. Stop at the Epic boundary and wait for review.

## Product quality questions

- Does this reduce cognitive load rather than add another dashboard to maintain?
- Can a user understand why a recommendation appeared?
- Can a user correct, export, or delete the underlying data?
- Does a missed day lead to recovery rather than punishment?
- Does the feature work with a slow network and a small screen?
- Is the default safe for a new user who does not understand the internals?
- Are external providers optional where practical?
- Can maintainers operate it without relying on the original author?
