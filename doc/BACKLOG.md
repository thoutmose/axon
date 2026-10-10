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
| EP-04 | Security, authentication & privacy | Passkeys/TOTP, sessions, ownership checks, audit, export/erasure, consent. | MUST | 55 pts | EP-00, EP-01 | Planned |
| EP-05 | Core data model & universal links | Entity conventions, typed links, event log, retention and erasure hooks, search. | MUST | 34 pts | EP-04 | Planned |
| EP-06 | API platform | Spec-first REST/OpenAPI 3.1 with errors, pagination, idempotency, and tokens. | MUST | 34 pts | EP-05 | Planned |
| EP-07 | Design system & frontend shell | Calm, accessible SvelteKit shell, components, and auth screens. | MUST | 34 pts | EP-04, EP-06 | Planned |
| EP-08 | Dashboard & calendar | One trustworthy daily view of priorities, commitments, and progress. | SHOULD | 34 pts | EP-09, EP-11, EP-13 | Planned |
| EP-09 | Tasks & recurring work | Capture, plan, and complete daily/weekly/monthly work with GTD contexts and recurrence. | MUST | 34 pts | EP-05, EP-06, EP-07 | Planned |
| EP-10 | Projects, Kanban, docs & delivery agents | Projects, epics/issues/sub-tasks, Kanban, docs, repo analysis, and code review. | MUST | 89 pts | EP-09, EP-11, EP-19 | Planned |
| EP-11 | Notes, Zettelkasten, PARA & Markdown sync | Knowledge capture with wikilinks and safe sync with `~/SecondBrain`. | MUST | 55 pts | EP-05, EP-06, EP-07 | Planned |
| EP-12 | Habits & Habits agent | Evidence-based habit design with gentle recovery. | SHOULD | 55 pts | EP-07, EP-19 | Planned |
| EP-13 | Goals & milestones | Outcomes linked to measurable milestones, tasks, habits, and notes. | MUST | 21 pts | EP-05, EP-06, EP-07, EP-09 | Planned |
| EP-14 | Roadmaps, Tutor agent & skill graph | Import or generate learning roadmaps and visualize progress. | SHOULD | 89 pts | EP-11, EP-13, EP-19 | Planned |
| EP-15 | Gamification engine | Transparent, opt-in XP, streaks, badges, and earned freezes. | SHOULD | 55 pts | EP-09, EP-12, EP-13 | Planned |
| EP-16 | Learning engine, Teacher agent & Anki | Retrieval practice with FSRS scheduling, reviewed cards, and Anki sync. | SHOULD | 89 pts | EP-11, EP-14, EP-19 | Planned |
| EP-17 | Media trackers & Media agent | Films, series, and books with attributed metadata and explained recommendations. | COULD | 34 pts | EP-06, EP-07, EP-19 | Planned |
| EP-18 | Health, nutrition, sport & coach agents | Food and training logs with consent-gated, educational coaching. | COULD | 89 pts | EP-04, EP-06, EP-07, EP-19 | Planned |
| EP-19 | AI agent platform, skills & safety | Approval-gated agents with tool boundaries, budgets, evals, and audit; ships Task Manager and Goals agents. | MUST | 89 pts | EP-04, EP-05, EP-06, EP-07, EP-09, EP-13 | Planned |
| EP-20 | Analytics & insights | Explainable personal metrics and weekly reviews from the event log. | COULD | 55 pts | EP-03, EP-05, EP-09 | Planned |
| EP-21 | Documentation, release 1.0 & hardening | Supportable release: docs, upgrade path, performance, recovery, security review. | MUST | 55 pts | All MUST Epics | Planned |

## Delivery order

Dependencies above are satisfied by this order. Change it only through a human decision recorded in the [change log](#change-log).

| Release | Epics, in order | Exit signal |
|---|---|---|
| R0 — Foundations | EP-00 → EP-01 → EP-02 → EP-03 → EP-04 | Secure deploy, CI, observability, auth, privacy baseline |
| R1 — Personal graph | EP-05 → EP-06 → EP-07 → EP-09 → EP-11 → EP-13 → EP-08 | Notes, links, tasks, goals, and dashboard in one daily loop — no AI required |
| R2 — Guided growth | EP-19 → EP-12 → EP-10 → EP-14 → EP-16 → EP-15 | Human-approved agents, habits, projects, roadmaps, retrieval, Anki |
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

- [ ] **EP-04/IS-04.01 — Accounts and passkeys** · 8 pts · MUST
  - [ ] Register and sign in with WebAuthn; an account can hold several passkeys.
  - [ ] One-time recovery codes are issued at registration and stored hashed.
- [ ] **EP-04/IS-04.02 — TOTP second factor** · 3 pts · SHOULD
  - [ ] TOTP enrolment and verification, with secrets encrypted at rest.
- [ ] **EP-04/IS-04.03 — Sessions** · 5 pts · MUST
  - [ ] Server-side sessions in `Secure`, `HttpOnly`, `SameSite` cookies, rotated at sign-in.
  - [ ] Idle and absolute timeouts; “sign out everywhere” revokes all sessions.
  - [ ] CSRF protection on every state-changing request.
- [ ] **EP-04/IS-04.04 — Ownership checks** · 5 pts · MUST
  - [ ] Every user-owned row has an owner; queries filter by it.
  - [ ] A shared test harness proves, for every endpoint, that user A cannot read or change user B’s data.
- [ ] **EP-04/IS-04.05 — Abuse protection** · 3 pts · MUST
  - [ ] Authentication endpoints are rate-limited with backoff.
- [ ] **EP-04/IS-04.06 — Audit log** · 5 pts · MUST
  - [ ] Sign-ins, factor changes, exports, erasures, and consent changes are recorded append-only and visible to the user.
- [ ] **EP-04/IS-04.07 — Data classification and privacy register** · 5 pts · MUST
  - [ ] Each data category has a classification; health data is marked special-category (GDPR Art. 9).
  - [ ] A processor inventory lists every external service and what it receives.
  - [ ] A retention table states how long each category is kept.
- [ ] **EP-04/IS-04.08 — Export and erasure framework** · 8 pts · MUST
  - [ ] The user can download all their data as JSON plus Markdown.
  - [ ] Erasure removes the user’s rows; a test verifies nothing remains outside documented backup windows.
- [ ] **EP-04/IS-04.09 — Consent records** · 3 pts · MUST
  - [ ] Consent is recorded per purpose (AI agents, each external API), off by default, revocable, and timestamped.
- [ ] **EP-04/IS-04.10 — Threat model and review** · 5 pts · MUST
  - [ ] A threat model covers auth, sessions, and data access.
  - [ ] OWASP ASVS Level 2 checks for authentication and sessions are recorded.
  - [ ] Someone other than the author reviews the authentication code.

**Exit evidence.** ASVS checklist, cross-user access test report, export and erasure demo, threat model.

### EP-05 — Core data model & universal links

**Goal.** Shared conventions for every entity, typed links between any compatible entities, and an event log that records meaningful changes.

**Why.** Links are what make Axon one graph instead of several apps. Getting them right once avoids a migration in every later Epic.

**Out of scope.** Domain entities themselves (tasks, notes, …), which arrive in their own Epics.

**Issues**

- [ ] **EP-05/IS-05.01 — Entity conventions** · 3 pts · MUST
  - [ ] An ADR fixes ID format, timestamps, owner column, and soft-delete rules.
  - [ ] Migrations run up and down in CI.
- [ ] **EP-05/IS-05.02 — Universal links** · 8 pts · MUST
  - [ ] A link stores owner, source type/ID, target type/ID, and relation.
  - [ ] Only allowed type pairs and relations are accepted; a link cannot cross owners.
  - [ ] Deleting an entity removes its links in the same transaction; an integrity check finds no dangling links.
  - [ ] Forward and backward lookups are indexed and paginated.
- [ ] **EP-05/IS-05.03 — Event log** · 8 pts · MUST
  - [ ] Each change writes an append-only event (actor, entity, type, time) in the same transaction.
  - [ ] Event payloads exclude fields classified as sensitive.
- [ ] **EP-05/IS-05.04 — Export and erasure hooks** · 5 pts · MUST
  - [ ] Each entity type registers export and erase handlers with the EP-04 framework.
  - [ ] A test fails when a new entity type has no handlers.
- [ ] **EP-05/IS-05.05 — Search primitive** · 5 pts · SHOULD
  - [ ] PostgreSQL full-text search over entity titles and bodies, scoped to the owner and paginated.

**Exit evidence.** Schema diagram, integrity-check output, handler coverage test.

### EP-06 — API platform

**Goal.** A stable, documented REST API that the web app, scripts, sync, and agents all use the same way.

**Why.** One contract keeps the web app, Markdown sync, and agents consistent, and makes mistakes visible in CI.

**Out of scope.** GraphQL ([ADR 0002](adr/0002-initial-technology-stack.md)).

**Issues**

- [ ] **EP-06/IS-06.01 — Spec-first OpenAPI 3.1** · 5 pts · MUST
  - [ ] The spec lives in the repository, is linted in CI, and generates server types.
  - [ ] Reference docs are served from the API.
- [ ] **EP-06/IS-06.02 — Conventions** · 5 pts · MUST
  - [ ] All routes are under `/v1`.
  - [ ] Errors use RFC 9457 `application/problem+json`.
  - [ ] Lists use cursor pagination with a maximum page size.
- [ ] **EP-06/IS-06.03 — Safe mutations** · 5 pts · MUST
  - [ ] `POST` accepts an `Idempotency-Key`; a replay returns the original result.
  - [ ] `PATCH` requires `If-Match`; a stale version returns `412`.
- [ ] **EP-06/IS-06.04 — Contract tests** · 5 pts · MUST
  - [ ] CI validates real responses against the spec.
- [ ] **EP-06/IS-06.05 — TypeScript client** · 3 pts · MUST
  - [ ] A generated client is used by the web app; CI fails if it is stale.
- [ ] **EP-06/IS-06.06 — Links and timeline endpoints** · 5 pts · MUST
  - [ ] Create, list, and delete links; list backlinks; read an entity’s event timeline.
- [ ] **EP-06/IS-06.07 — Personal access tokens** · 5 pts · SHOULD
  - [ ] Read-only or read-write tokens with expiry, shown once, stored hashed, and revocable; used by scripts and Markdown sync.

**Exit evidence.** Published API docs, contract test report, idempotency and concurrency tests.

### EP-07 — Design system & frontend shell

**Goal.** A calm, responsive, accessible SvelteKit shell that every feature plugs into.

**In scope.** Design tokens (type, spacing, colour) with light and dark themes; core components (button, input, dialog, toast, list, empty/loading/error states); app navigation; sign-in, passkey, and account screens; reduced-motion support.

**Out of scope.** Feature screens (owned by each domain Epic).

**Acceptance criteria**

- Automated accessibility checks (axe) report no serious violations on every shell page.
- Every flow works by keyboard alone, with visible focus.
- Layouts work at 360 px width without horizontal scrolling.
- Animations are disabled when the user prefers reduced motion.
- Slow-network and API-down states show a clear message, not a blank screen.
- Playwright smoke tests cover sign-in, navigation, and sign-out.

**Candidate issues.** Tokens and themes (5) · Core components (8) · App shell and navigation (5) · Auth and account screens (8) · Accessibility and Playwright CI (5) · Error and offline states (3).

### EP-09 — Tasks & recurring work

**Goal.** Capture, plan, and complete daily, weekly, and monthly work with GTD contexts and recurrence.

**In scope.** Task CRUD; inbox capture; due and scheduled dates; today/week/month views; GTD contexts and next actions; recurring tasks (RFC 5545 RRULE); check-ins; links to other entities.

**Out of scope.** Task Manager agent (EP-19), Kanban and projects (EP-10), calendar (EP-08).

**Acceptance criteria**

- A task can be captured from anywhere in the app with one keyboard shortcut and one submit.
- Recurrence is correct across time zones and daylight-saving changes (property tests).
- Completing a recurring task creates the next occurrence exactly once, even on retry.
- Views stay fast and paginated with 10,000 tasks.
- Tasks appear in export and are removed by erasure.

**Candidate issues.** Task model and API (8) · Capture and inbox (5) · Today/week/month views (8) · Contexts and next actions (3) · Recurrence engine (8) · Check-ins (2).

### EP-11 — Notes, Zettelkasten, PARA & Markdown sync

**Goal.** Capture and connect knowledge in Markdown, and keep it in sync with `~/SecondBrain` on `srv-second-brain` without ever losing an edit.

**In scope.** Fleeting, literature, and permanent notes; Markdown editor; `[[wikilinks]]` that create universal links; backlinks; PARA organization; Markdown export and import; Git-based sync.

**Out of scope.** Teacher agent and flashcards (EP-16). A note graph view is COULD here; if it is built, record the graph-library choice in a new ADR ([ADR 0002](adr/0002-initial-technology-stack.md) currently defers it to EP-14).

**Acceptance criteria**

- Export then import reproduces identical Markdown (golden tests).
- Renaming a note updates or preserves its links; no link is broken silently.
- When the same note changes on both sides, sync keeps both versions and shows the conflict; nothing is overwritten.
- Sync runs over the private network only.
- Ship one-way import first, then two-way sync behind a setting.

**Candidate issues.** Note model and API (5) · Editor (8) · Wikilinks and backlinks (8) · PARA organization (3) · Markdown export/import (5) · One-way import from SecondBrain (8) · Two-way Git sync with conflicts (13) · Note graph view (5, COULD).

### EP-13 — Goals & milestones

**Goal.** Turn outcomes into measurable milestones connected to the tasks, habits, and notes that move them.

**In scope.** Goals with a measurable outcome and optional target date; milestones; links to tasks, notes, and (later) habits; progress computed from linked work, with an explanation of how.

**Out of scope.** Goals agent (EP-19).

**Acceptance criteria**

- Every progress figure can be expanded to show exactly what it was computed from.
- No date or commitment is created without the user entering or confirming it.
- Archiving a goal keeps its history and links.

**Candidate issues.** Goal and milestone model and API (5) · Progress calculation and explanation (5) · Goal screens (8) · Weekly goal review prompt (3).

### EP-08 — Dashboard & calendar

**Goal.** One trustworthy daily view of priorities, commitments, and progress.

**In scope.** Today view (due and scheduled tasks, active goals, recent notes); calendar of tasks, goals, and milestones with filters (other types join as their Epics land); read-only iCal feed protected by a revocable token.

**Out of scope.** Writing to external calendars.

**Acceptance criteria**

- Every number on the dashboard links to the items behind it.
- The dashboard responds in under 300 ms at p95 with 10,000 tasks.
- Empty states guide a new user to their first task, note, and goal.
- The iCal feed can be revoked and contains only what the user selected.

**Candidate issues.** Today view (8) · Calendar with filters (8) · iCal feed (5) · KPI definitions and explanations (5) · Empty and onboarding states (3).

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

**Candidate issues.** Provider adapter and budgets (8) · Tool registry and permissions (8) · Proposals and approval queue (13) · Consent and redaction (5) · Run audit and cost view (5) · Memory (8) · Skills format and loader (5) · Eval harness with injection fixtures (13) · Task Manager agent (13) · Goals agent (8). Split into two Epics at refinement if it exceeds one release slot.

### EP-12 — Habits & Habits agent

**Goal.** Habits designed with cues, stacking, and two-minute starts, tracked gently, with recovery instead of punishment.

**In scope.** Habit model (cue, craving, response, reward, stack, two-minute version, schedule); check-ins; recovery after a miss (“never miss twice”); links to goals; Habits agent suggestions as drafts.

**Acceptance criteria**

- A missed day produces a recovery prompt, never shaming copy (copy reviewed against the product quality questions).
- Schedules respect the user’s time zone, including travel and daylight-saving changes.
- Habits work fully with the agent disabled.

**Candidate issues.** Habit model and API (8) · Check-in flow (5) · Schedules and time zones (8) · Recovery flow (5) · Habit screens (13) · Habits agent (13).

### EP-10 — Projects, Kanban, docs & delivery agents

**Goal.** Run personal and side projects in Axon: projects, epics, issues, sub-tasks, Kanban, documentation, repository analysis, and code review.

**In scope.** Project hierarchy built on tasks; Kanban with WIP limits; project docs as notes; Side-Projects agent (read-only repository analysis that drafts issues); Code Review agent (comments on diffs).

**Acceptance criteria**

- Repository access is read-only by default; drafted issues require approval.
- The Code Review agent can comment but cannot approve or merge.
- Kanban moves are keyboard-accessible.

**Candidate issues.** Project and hierarchy model (8) · Kanban (13) · Project docs (8) · Repository connector, read-only (13) · Side-Projects agent (21) · Code Review agent (21). **Split before start** — 89 points is too large for one Epic; separate the agents into their own Epic.

### EP-14 — Roadmaps, Tutor agent & skill graph

**Goal.** Build learning and career roadmaps by hand, by import, or with the Tutor agent, and see progress as a skill graph.

**In scope.** Manual roadmaps linked to goals, notes, and learning items; import from roadmap.sh and OSSU; Tutor agent interview and cited draft roadmaps; skill graph view.

**Acceptance criteria**

- Licenses of roadmap.sh and OSSU content are checked before import is built; attribution is shown.
- Every Tutor suggestion cites its sources and stays a draft until approved.
- The skill graph stays responsive with 500 nodes and has an accessible list alternative.

**Candidate issues.** Roadmap model and API (8) · Manual editor (13) · OSSU import (8) · roadmap.sh import, license permitting (8) · Tutor agent (21) · Skill graph view (13) · Resource agent (13). **Split before start.**

### EP-16 — Learning engine, Teacher agent & Anki

**Goal.** Retrieval-first learning: spaced scheduling, reviewed practice questions, dual-coded cards, and Anki sync.

**In scope.** FSRS scheduling (SM-2 fallback); review sessions; Teacher agent generating questions and cards from literature and permanent notes only; approval before cards enter the deck; Anki `.apkg` export and sync with the existing `anki-sync-server`.

**Acceptance criteria**

- FSRS results match the reference implementation on shared fixtures.
- The Teacher agent cannot read fleeting notes (tested).
- Exported decks import into Anki desktop without errors.

**Candidate issues.** Card and review model (8) · FSRS scheduler (8) · Review session UI (13) · Teacher agent (21) · Card approval queue (8) · Anki export (8) · Anki sync (13). **Split before start.**

### EP-15 — Gamification engine

**Goal.** Recognize meaningful progress with transparent, opt-in XP, levels, badges, streaks, and freezes bought only with earned points.

**In scope.** XP rules, levels, badges, the streak state machine (see README), earned-point freezes.

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
