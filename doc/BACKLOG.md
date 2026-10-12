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
| EP-15 | Gamification engine | Transparent, opt-in XP, streaks, badges, and earned freezes. | SHOULD | 55 pts | EP-09, EP-12, EP-13 | Done |
| EP-16 | Learning engine, Teacher agent & Anki | Retrieval practice with FSRS scheduling, illustrated cards, practice tests, a Teacher agent, and Anki packages. | SHOULD | 76 pts | EP-11, EP-14, EP-19 | Done |
| EP-43 | Daily-use fixes | Full window width, GTD and Kanban task boards by default, a month calendar by default, skill graphs made of skills (resources under each), Areas, Resources, and Archive pages, and templates for every kind of note, project, area, resource, goal, milestone, and habit. Merged from the parallel branch `claude/axon-ui-features-9ryyt4`. | MUST | 21 pts | EP-36 | Done |
| EP-42 | Anki sync to srv-services | Approved cards reach the existing `anki-sync-server` on `srv-services` automatically, by hand or after a Teacher proposal. Split from EP-16. | SHOULD | 13 pts | EP-16 | Done |
| EP-17 | Media trackers & Media agent | Films, series, and books with attributed metadata and explained recommendations. | COULD | 34 pts | EP-06, EP-07, EP-19 | Done |
| EP-18 | Health logging | Food and training logs behind an explicit health-data consent, with Open Food Facts and USDA lookups. Coaching split to EP-44. | COULD | 44 pts | EP-04, EP-06, EP-07 | Done |
| EP-44 | Coach agents | Personal Trainer and Nutritionist agents with a separate consent to share health data with the AI provider, and a safety eval suite. Split from EP-18. | COULD | 42 pts | EP-18, EP-19 | Done |
| EP-45 | Daily-use fixes 2 | Pomodoro with 4 work sessions by default, a cleaner Today header, the dashboard network made of hard and soft skills only, and savings pots (tirelire) in the budget. Owner request. | MUST | 13 pts | EP-30, EP-35, EP-39 | Done |
| EP-46 | Advanced note NLP | Professional NLP for literature and permanent notes: lemmatisation, key phrases, topics, a richer English/French thesaurus, and concept graphs. Owner request. | SHOULD | 21 pts | EP-37 | Done |
| EP-47 | Zettelkasten knowledge graph | A knowledge graph of literature and permanent notes on the Notes page, from combined NLP evidence, with communities, labels, centrality, and bridges. Owner request. | SHOULD | 13 pts | EP-46 | Done |
| EP-48 | Everything in the vault, visible templates | Tasks, goals, milestones, projects, areas, archives, habits, health, and media as Markdown notes kept in step both ways; templates for every note type shown with their sections. Owner request. | MUST | 21 pts | EP-36, EP-43 | Done |
| EP-19 | AI agent platform, skills & safety | Approval-gated agents with tool boundaries, budgets, evals, and audit; ships Task Manager and Goals agents. | MUST | 89 pts | EP-04, EP-05, EP-06, EP-07, EP-09, EP-13 | Done |
| EP-20 | Analytics & insights | Explainable personal metrics and weekly reviews from the event log. | COULD | 55 pts | EP-03, EP-05, EP-09 | Done |
| EP-21 | Documentation, release 1.0 & hardening | Supportable release: docs, upgrade path, performance, recovery, security review. | MUST | 55 pts | All MUST Epics | Done (external review and observed install pending) |
| EP-22 | AI providers | Agents on Anthropic, OpenAI, Mistral, Gemini, DeepSeek, or a local model, chosen per person. | MUST | 21 pts | EP-19 | Done |
| EP-23 | Visual design refresh | A cinematic, calm interface: imagery, depth, translucency, generous type. | MUST | 21 pts | EP-07 | Done |
| EP-24 | Connect everything | Link any element to any other from wherever it is shown, and see every connection. | MUST | 13 pts | EP-05, EP-12 | Done |
| EP-25 | Notes workspace & Second Brain | A note-taking workspace on par with dedicated apps: tree, live editor, Zettelkasten, PARA, daily notes, templates, quick switcher. | MUST | 34 pts | EP-11 | Done |
| EP-26 | Delivery agents | Side-Projects agent (read-only repository analysis drafting issues) and Code Review agent. | SHOULD | 55 pts | EP-10, EP-19 | Done |
| EP-27 | Tutor & Resource agents | Interview-based, cited draft roadmaps, a curated library of the best resources per domain, and resource finding. | SHOULD | 34 pts | EP-14, EP-19 | Done |
| EP-28 | Demo account | A resettable demo account with realistic sample data, to try every feature without touching real data. | MUST | 13 pts | EP-04, EP-27 | Done |
| EP-29 | Docs in two languages & API reference | README in English and French with a language switch, an interactive OpenAPI (Swagger) reference, and consistent Axon naming. | MUST | 8 pts | EP-06 | Done |
| EP-30 | Command dashboard | The home page as a detailed dashboard: KPIs, charts, tables, and a network graph of hard and soft skills. | MUST | 34 pts | EP-08, EP-14 | Done |
| EP-41 | Skills assessment | Before the first roadmap, a short assessment of your situation, job, goals, and current level, which places you on roadmaps and feeds the Tutor. | SHOULD | 21 pts | EP-27, EP-30 | Done |
| EP-31 | Settings, languages & YAML configuration | One place for language (English, French), theme, fonts, icon theme, templates, AI keys and limits; editable in the UI or as a YAML file. | MUST | 34 pts | EP-22, EP-25 | Done (French for the remaining pages: IS-31.06) |
| EP-32 | Keyboard shortcuts & Vim mode | Shortcuts for every page with a help overlay, and optional Vim keybindings in the note editor. | SHOULD | 13 pts | EP-25, EP-31 | Done |
| EP-33 | Profile & avatars | Profile pictures with a generated default avatar. | SHOULD | 5 pts | EP-04 | Done |
| EP-34 | Atomic Habits | James Clear's four laws, habit stacking, implementation intentions, identity, and breaking bad habits. | SHOULD | 21 pts | EP-12 | Done |
| EP-35 | Focus & Pomodoro | Configurable Pomodoro sessions linked to tasks, projects, skills, and notes, with focus sounds. | SHOULD | 21 pts | EP-09, EP-14 | Done |
| EP-36 | Live vault sync | The Markdown vault, the notes database, and templates stay in step continuously, not only when a sync runs. | SHOULD | 21 pts | EP-11, EP-25 | Done |
| EP-37 | Knowledge explorer & NLP graph | Explore notes with PARA and Second Brain filters, charts, and a knowledge graph built from NLP (keywords, entities, synonyms, similarity). | SHOULD | 34 pts | EP-25, EP-36 | Done (embeddings open) |
| EP-38 | Themed spaces | Each area gets its own scenery and effects: a Da Vinci notebook for notes, an Oxford library for learning, a workshop for projects, and more. | SHOULD | 21 pts | EP-23, EP-31 | Done |
| EP-39 | Personal finance | Accounts, income, expenses, transfers, categories, budgets, recurring transactions, and rules, in the spirit of Firefly III. | SHOULD | 55 pts | EP-04, EP-06, EP-07 | Done |
| EP-40 | Instance export & import | Move a whole Axon (database, files, settings) to another server or configuration. | MUST | 13 pts | EP-01, EP-31 | Done |

## Delivery order

Dependencies above are satisfied by this order. Change it only through a human decision recorded in the [change log](#change-log).

| Release | Epics, in order | Exit signal |
|---|---|---|
| R0 — Foundations | EP-00 → EP-01 → EP-02 → EP-03 → EP-04 | Secure deploy, CI, observability, auth, privacy baseline |
| R1 — Personal graph | EP-05 → EP-06 → EP-07 → EP-09 → EP-11 → EP-13 → EP-08 | Notes, links, tasks, goals, and dashboard in one daily loop — no AI required |
| R2 — Guided growth | EP-19 → EP-12 → EP-22 → EP-23 → EP-24 → EP-25 → EP-10 → EP-26 → EP-14 → EP-27 | Human-approved agents, habits, projects, roadmaps |
| R2.5 — Daily driver | EP-28 → EP-29 → EP-30 → EP-41 → EP-31 → EP-32 → EP-33 → EP-34 → EP-35 → EP-16 → EP-42 → EP-15 → EP-36 → EP-43 → EP-39 → EP-38 → EP-37 → EP-40 | Demo, dashboard, settings, focus, retrieval and Anki, gamification across every area, live vault, knowledge graph, finance, portability |
| R3 — Whole-life beta | EP-17 → EP-18 → EP-45 → EP-46 → EP-44 → EP-20 | Media, health, owner fixes, note NLP, coaching, analytics with consent and graceful degradation |
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

**Issues** (design in [ADR 0015](adr/0015-demo-account.md))

- [x] **EP-28/IS-28.01 — Seed data** · 5 pts · MUST
  - [x] `axon demo reset` builds, in one transaction:
    - 8 tasks across GTD lists, with a weekly recurrence;
    - a project board with 7 cards and a WIP limit;
    - 8 notes (PARA, Zettelkasten with wikilinks, tags, callouts, a table, a daily note);
    - 2 goals with milestones;
    - 3 habits with a month of check-ins;
    - 2 library roadmaps with progress;
    - links across all of them, and 2 pending agent proposals.
- [x] **EP-28/IS-28.02 — Demo sign-in and limits** · 5 pts · MUST
  - [x] `POST /v1/auth/demo` (only with `AXON_DEMO=1`): a 12-hour session, "Try the demo" on the landing page, and a banner in the app.
  - [x] Refused with 403: tokens, passkeys, recovery codes, TOTP, AI providers, repositories, and erasure. Agents are off (reason `demo`).
  - [x] Isolation is tested both ways, and the demo does not close first-account registration.
- [x] **EP-28/IS-28.03 — Reset job and docs** · 3 pts · MUST
  - [x] The server seeds the demo once migrations have run and resets it nightly after 03:00 UTC; a reset signs visitors out (tested).
  - [x] Non-secret per-environment settings (`deploy/env/*.settings.env`); the demo is on for staging and production; runbook, privacy register, and ADR 0015.

**Exit evidence.** Integration tests pass: the demo is off unless enabled; the sample data, approvals, isolation, refused routes, agents off, and the reset are all checked. Playwright: 20 scenarios with axe, including the new demo journey. Cross-cutting matrix:
- identity: demo session, 12 h;
- authorization: NotDemo plus ownership checks;
- privacy: register row, no personal data;
- accessibility: banner role note, axe;
- reliability: one transaction per reset, retried each minute;
- performance: bounded seed;
- observability: reset logged;
- documentation: runbook, ADR;
- AI safety: agents off;
- supply chain: no new dependency.

### EP-29 — Docs in two languages & API reference

**Goal.** README in English and French with a switch at the top of each, an interactive API reference, and one name everywhere: Axon.

**Issues**

- [x] **EP-29/IS-29.01 — French README and parity check** · 3 pts · MUST
  - [x] `README.md` brought up to date (✅ for what works today, "Planned" for the rest, agents with a status column, demo, API reference, why Go and not Rust); `README.fr.md` translates it; each links to the other.
  - [x] `make check-docs` (part of `make check` and CI) fails when the French README's headings, code blocks, or tables drift from the English one.
- [x] **EP-29/IS-29.02 — Swagger UI** · 3 pts · MUST
  - [x] `/api-docs/index.html` renders the live `/api/v1/openapi.yaml` with Swagger UI 5 (Apache-2.0), copied from `node_modules` at build time and served as static files by Axon: no CDN. Linked from Access tokens. (`/api/*` belongs to the API, so the page lives at `/api-docs`.)
- [x] **EP-29/IS-29.03 — Naming sweep** · 2 pts · MUST
  - [x] Repository, Go module, images, Compose project, UI, and docs all say Axon; the glossary no longer calls it a working name; the OpenAPI summary covers today's API.

**Exit evidence.** `make check` (including check-docs) and the Playwright suite pass, with a new scenario that opens the API reference and expands an operation. Cross-cutting matrix: identity, authorization, privacy, reliability, performance, and AI safety are n/a (static documentation; the page sends requests only as the signed-in browser when the reader uses "Try it out"); accessibility is checked by the page landmark and labelled header, but Swagger UI's own markup is third-party and excluded from axe; observability is n/a; documentation is both READMEs; supply chain adds `swagger-ui-dist` 5.33.1, pinned, Apache-2.0, a dev dependency used only at build time.

### EP-30 — Command dashboard

**Goal.** The home page answers "how am I doing?" at a glance: KPIs, trends, tables, and a skills network.

**Issues**

- [x] **EP-30/IS-30.01 — KPI service** · 8 pts · MUST
  - [x] `GET /v1/dashboard/overview?days=7…365`: seven KPIs, each with a written definition and a link to its items:
    - tasks done, habit consistency (scheduled days met, counted from each habit's start), goal progress, skills learned, learning now, notes written, active projects;
    - each compares with the previous period where that makes sense.
- [x] **EP-30/IS-30.02 — Charts** · 8 pts · MUST
  - [x] Daily activity line chart (tasks, habit check-ins, notes) and a 12-month activity heatmap. Hand-drawn SVG, with no chart dependency, and each has a table alternative.
- [x] **EP-30/IS-30.03 — Tables** · 5 pts · MUST
  - [x] Due in the next 7 days, learning now, and active projects without activity for 14 days.
- [x] **EP-30/IS-30.04 — Skills network** · 13 pts · MUST
  - [x] Skills gain a hard/soft `kind`: library soft skills import as soft, existing ones are backfilled, and the kind can be changed with PATCH.
  - [x] The network shows up to 250 skills plus the goals, projects, and notes linked to them, with prerequisites dashed.
  - [x] Its force layout is deterministic and bounded (300 nodes in under 2 s in unit tests). You can filter All, Hard, or Soft, every node is a link, and there is a list alternative.

**Exit evidence.** Integration test on the demo data (KPIs, consistency bounds, series lengths, a 365-day heatmap, tables, a network with hard and soft skills, goals, and projects, `days` validation, skill kind validation). Vitest covers trend, heatmap levels, and the layout (deterministic, framed, linked nodes closer, no ring, speed). Playwright: the demo journey checks the KPIs, both charts, the network's Soft filter, a due-soon row, and the 90-day period, with axe. Screenshot `doc/screenshots/dashboard.png`.

Cross-cutting matrix:
- identity and authorization: owner-scoped queries;
- privacy: no new data beyond the skill kind;
- accessibility: table and list alternatives, a focusable scroll region, labelled links, axe;
- performance: bounded nodes and indexed queries;
- reliability and observability: read-only endpoint;
- documentation: README and BACKLOG;
- AI safety: n/a;
- supply chain: no dependency.

### EP-41 — Skills assessment

**Goal.** Before starting a roadmap, a short assessment establishes where you are: your situation, job, goals, available time, and current level in each domain. Roadmaps then start at the right place.

**In scope.**
- A guided questionnaire: role, experience, goals, hours per week, and preferred formats.
- Short self-ratings and a few objective check questions per domain, drawn from the library's sections.
- Results stored as a profile you can edit, which:
  - marks skills you already know as done or skipped on import;
  - suggests library roadmaps;
  - is read by the Tutor agent (with consent) instead of asking again;
  - can be retaken at any time, with results compared over time on the dashboard.

**Issues**

- [x] **EP-41/IS-41.01 — Questionnaire and profile** · 8 pts · SHOULD
  - [x] Four steps: situation (role, experience, goals, hours per week, formats), domains to learn, quick checks, results.
  - [x] Stored as an `assessment` entity: exported, erased, deletable, and retakable, with a levels-over-time table.
- [x] **EP-41/IS-41.02 — Domain checks and placement** · 8 pts · SHOULD
  - [x] 72 check questions, three per library domain. Answers are never sent to the browser, and their positions are shuffled.
  - [x] Level = round((self-rating + 5 × correct/total) / 2), with the reason written out.
  - [x] Import with `place: true`: level 3 marks the first section done and level 4 or 5 the first two, always leaving the last section; the roadmap description explains why.
- [x] **EP-41/IS-41.03 — Tutor and Learn integration** · 5 pts · SHOULD
  - [x] `get_assessment` tool for the Tutor (1.1.0) and Resources (1.1.0) agents; the Tutor asks only what is missing.
  - [x] Learn page invitation, results with "Start here" per domain and suggested first domains; the demo includes an assessment.

**Exit evidence.** The integration test covers:
- no answers leaked;
- validation (no domains, unknown domain, missing or out-of-range rating);
- scoring and reasons, and suggestions;
- placement on Python (two sections done, explained) and Rust (none), and a 422 for an unassessed domain;
- the Tutor receiving the profile and levels;
- history and deletion.

Playwright: 22 scenarios with axe, including the full assessment flow and placed import.

Cross-cutting matrix:
- identity and authorization: owner-scoped;
- privacy: register row, exported and erased with the entity registry, agents read it only with consent;
- accessibility: fieldsets and legends, labelled radios and level, axe;
- reliability: one transaction;
- performance: bounded;
- observability: n/a;
- documentation: README and BACKLOG;
- AI safety: read-only tool, no write;
- supply chain: no dependency.

### EP-31 — Settings, languages & YAML configuration

**Goal.** One place to configure Axon, in the browser or as a YAML file.

**In scope.** Language (English and French UI, translated strings with a fallback), theme, font family and size, icon theme, default templates for notes, projects, and resources, AI provider keys and monthly limits; `GET/PUT /v1/me/settings` and YAML export and import of the same document, validated against a schema.

**Issues** (design in [ADR 0016](adr/0016-settings-yaml-and-languages.md))

- [x] **EP-31/IS-31.01 — Settings model and API** · 8 pts · MUST
  - [x] `GET/PUT /v1/me/settings` returns language, appearance, templates, and AI (provider without its key, plus the budget). Routes are session-only; validation is per field.
- [x] **EP-31/IS-31.02 — YAML import and export** · 5 pts · MUST
  - [x] `GET/PUT /v1/me/settings.yaml`: commented, round-trips without loss (tested), unknown keys rejected, errors name the line ("line 4: appearance.theme …"), applied in one transaction.
  - [x] `ai.provider.api_key` is write-only: never exported, kept on re-import, set on upload.
- [x] **EP-31/IS-31.03 — i18n framework** · 5 pts · MUST
  - [x] `t()` with an English-keyed French dictionary, placeholders, and fallback (unit-tested); `lang` follows the choice.
- [x] **EP-31/IS-31.04 — Appearance settings** · 5 pts · MUST
  - [x] Theme, font (system, serif, rounded, mono), text size, icon theme (emoji, symbols, none), and motion, applied through `<html>` attributes and CSS tokens; cached for the first paint.
- [x] **EP-31/IS-31.05 — Template settings** · 3 pts · MUST
  - [x] "My template" for new notes, project defaults (description, WIP limit, docs folder with `{slug}`), and resource defaults (section, hard or soft kind, now editable on skills).
- [ ] **EP-31/IS-31.06 — French for every page** · 8 pts · SHOULD
  - [x] Shell, landing, sign-in, demo banner, Today and the command dashboard, account tabs, and Preferences are in French.
  - [ ] Tasks, notes, goals, projects, roadmaps, habits, calendar, agents, and the remaining account pages still show English. They are translated as each page is next touched, and fully before 1.0 (EP-21).

**Exit evidence.** The integration test covers:
- defaults, a JSON update, and per-field rejection;
- YAML without the stored key, a lossless round trip that keeps the key, and an edited file switching language, budget, provider, and key;
- six line-level errors (theme, language, version, unknown key, broken YAML, provider);
- tokens refused.

Vitest covers translation, placeholders, and icons. Playwright covers switching to French, dark, and large text, the YAML export, persistence across pages, and switching back, with axe.

Cross-cutting matrix:
- identity: session only;
- authorization: own settings, demo cannot set providers;
- privacy: register row, exported, keys write-only;
- accessibility: labelled controls, `lang`, axe;
- reliability: one transaction;
- performance: one row;
- observability: errors logged without content;
- documentation: ADR 0016, README;
- AI safety: n/a;
- supply chain: `go.yaml.in/yaml/v4` promoted from indirect to direct (same version).

### EP-32 — Keyboard shortcuts & Vim mode

**Goal.** Fast, discoverable keyboard use everywhere, and Vim keybindings for people who write notes in Vim.

**In scope.** A shortcut registry with a `?` help overlay; go-to chords (`g t` Today, `g n` Notes, …); per-page actions; optional Vim mode in the Markdown editor (normal, insert, visual; motions, operators, `:w`), turned on in settings.

**Issues**

- [x] **EP-32/IS-32.01 — Shortcut registry and overlay** · 5 pts · SHOULD
  - [x] `?` opens the overlay. `g` plus a letter goes to a page (h Today, t Tasks, n Notes, o Goals, p Projects, l Learn, b Habits, c Calendar, a Agents, s Preferences). `/` searches, `n` captures, and Ctrl+K opens the command center.
  - [x] Shortcuts never fire while typing, and can be turned off in Preferences → Keyboard (`keyboard.shortcuts`, also in YAML) for WCAG 2.1.4.
- [x] **EP-32/IS-32.02 — Vim mode** · 8 pts · SHOULD
  - [x] A pure engine (`web/src/lib/vim.ts`) supports:
    - normal, insert, visual, and visual-line modes;
    - counts; h j k l w b e 0 ^ $ gg G;
    - i a I A o O; x X D C J p P u Ctrl-R;
    - d, c, and y with motions, plus dd cc yy;
    - `:w` to save.
  - [x] 7 unit tests cover motions and operators. A mode line shows NORMAL, INSERT, or VISUAL; it is turned on in Preferences (`keyboard.vim`).

**Exit evidence.** Vitest covers the Vim engine (motions, operators, insert, undo and redo, visual, `:w`, passthrough) and the chords. The integration test covers keyboard defaults and saving. Playwright opens the overlay with axe, navigates with `g t`, then turns on Vim, edits with `G o … Esc k dd :w`, saves, and checks the note after a reload.

Two test fixes along the way:
- the axe helper now waits out transitions that were already running, which fixes the colour-contrast flake seen earlier;
- the Preferences page now sends the keyboard section.

Cross-cutting matrix:
- accessibility: WCAG 2.1.4 opt-out, focusable scroll region, a live mode line;
- documentation: overlay, README, and BACKLOG;
- the remaining rows are n/a: client-side only, apart from one settings field.

### EP-33 — Profile & avatars

**Goal.** A face for every account: an uploaded picture or a generated default.

**In scope.** Upload (PNG, JPEG, WebP; resized and stripped of metadata on the server), a deterministic generated avatar by default, shown in the header and on shared items.

**Issues**

- [x] **EP-33/IS-33.01 — Upload and processing** · 3 pts · SHOULD
  - [x] `PUT /v1/me/avatar` takes PNG or JPEG up to 5 MB and 40 megapixels (decompression bombs are refused before decoding). The picture is cropped to a centred square, box-filtered to 256×256, and re-encoded as PNG, so metadata is dropped (unit-tested with a GPS text chunk).
  - [x] Stored in `users.avatar`: exported as `account/avatar.png` and erased with the account. `DELETE` returns to the default, and the demo cannot change its picture.
- [x] **EP-33/IS-33.02 — Default avatar and display** · 2 pts · SHOULD
  - [x] The default is initials on a gradient whose colours come from the account id: stable and distinct, HTML-escaped, and served inert under a `default-src 'none'` CSP.
  - [x] Shown in the header next to Account, and on the Account page with upload and "Use the default picture"; `avatar_updated_at` refreshes caches.

**Exit evidence.** Unit tests cover metadata stripping, cropping and resizing, JPEG input, rejection of non-images and tiny images, and a stable, escaped default. The integration test covers the default SVG, a 422 for garbage, a 415 for GIF, a 256×256 PNG after upload, the version on `/v1/me`, deletion, and the demo's 403. Playwright uploads a picture, sees it in the header, and returns to the default, with axe.

Cross-cutting matrix:
- privacy: register row, export and erasure;
- security: re-encoding, size limits, inert SVG;
- accessibility: alt text is the person's name, and the header image is decorative;
- supply chain: standard library only;
- the remaining rows are covered by the existing account rules.

### EP-34 — Atomic Habits

**Goal.** Habits designed with James Clear's method: make good habits obvious, attractive, easy, and satisfying; make bad habits invisible, unattractive, difficult, and unsatisfying.

**In scope.** Habit stacking ("After I [current habit], I will [new habit]") and implementation intentions ("I will [behaviour] at [time] in [location]") as structured fields; identity statements ("I am a person who…") with votes cast by check-ins; the habits scorecard; the two-minute rule and gradual levelling up; bad habits with inversion strategies; a habit contract with an accountability partner (optional, opt-in).

**Issues**

- [x] **EP-34/IS-34.01 — Model and API** · 8 pts · SHOULD
  - [x] Habits gain:
    - `kind`: build, or break with the four laws inverted;
    - `identity`, and `votes` (kept check-ins);
    - an implementation intention (`intention_time` as HH:MM, `intention_place`);
    - an optional habit contract (`contract_partner`, `contract_pledge`; nothing is sent to anyone).
  - [x] `GET/PUT /v1/habits/scorecard` (up to 100 behaviours rated +, =, or −) is exported and erased.
  - [x] The Habits agent (1.1.0) uses the same model: the laws, stacking and intention formulas, identity, and breaking.
- [x] **EP-34/IS-34.02 — Design flow** · 8 pts · SHOULD
  - [x] "Design a habit" chooses build or break, then identity, then one step per law, whose wording flips for a habit to break. An optional contract follows. A live "Your plan" preview uses Clear's formulas: "After I…, I will…" and "I will… at… in…".
- [x] **EP-34/IS-34.03 — Scorecard, identity, and stacks** · 5 pts · SHOULD
  - [x] Habit cards show the plan, a "Breaking" badge, and "N votes for “I am…”".
  - [x] "Your habit stacks" shows chains of habits stacked after other habits; cycles are handled (unit-tested).
  - [x] The scorecard is edited in place, and the edit page covers every new field.
  - [ ] Stacks on Today: not done; EP-38 left Today with its own scenery, so this needs its own issue.

**Exit evidence.** The integration test covers the new fields and defaults, votes counted from check-ins, a break habit, validation (kind, time format), and the scorecard round trip with its validation; the demo gains identities, intentions, a habit to break, and a scorecard. Vitest covers plan sentences, chains, and cycles. Playwright designs a habit with identity, time, and place, sees the plan preview and the stored plan, checks in and sees the vote, and rates a scorecard behaviour, with axe.

Cross-cutting matrix:
- privacy: register row (the scorecard can reveal habits, kept private and exportable);
- accessibility: labelled rating buttons with `aria-pressed`, live preview;
- AI safety: the agent proposes, as before;
- remaining rows unchanged.

### EP-35 — Focus & Pomodoro

**Goal.** Focused work sessions that count everywhere.

**In scope.** A session is 25 min work, 5 min break, 25 work, 5 break, 25 work, 15 min long break by default. Durations, the number of rounds, and the number of sessions are editable (one session by default). Each session links to a task, project, skill, note, or review; minutes feed KPIs, streaks, and XP. Focus sounds generated in the browser (white, pink, brown noise, rain), plus music links the person chooses for each kind of work.

**Issues**

- [x] **EP-35/IS-35.01 — Timer and settings** · 8 pts · SHOULD
  - [x] A timestamp-driven timer (`web/src/lib/pomodoro.ts`) that pauses, resumes, and skips, and catches up after the tab sleeps. It is saved and restored across reloads (8 fake-clock unit tests and an e2e reload).
  - [x] Defaults: 25 work + 5 break, twice, then 25 work + 15 long break; one session. Durations, rounds, and number of sessions are set in Preferences or YAML (`focus.*`), with validation.
- [x] **EP-35/IS-35.02 — Links and KPIs** · 5 pts · SHOULD
  - [x] Focus sessions are entities linked to their task, project, skill, note, or anything else through `focused_on`, so they show on that item's connections. Only worked minutes count, never breaks, and they cannot exceed the time the session ran.
  - [x] Dashboard: a "Focus time" KPI with the previous period and a focus-minutes series. The Focus page shows today, 7 days, the daily streak, and recent sessions. Nav "Focus", shortcut `g f`.
- [x] **EP-35/IS-35.03 — Focus sounds** · 8 pts · SHOULD
  - [x] White, pink, and brown noise and rain, generated with Web Audio (no download), playing during work phases only, at the chosen volume. A gentle chime marks phase ends, and phase changes are announced for screen readers.
  - [x] Music links per activity (for example coding or reading) open from the Focus page; nothing third-party plays inside Axon.

**Exit evidence.** The integration test covers:
- linking to a task (visible as an incoming link);
- worked-time and status validation, and double-finish refusal;
- today's minutes and the streak, and the dashboard KPI;
- deletion, settings defaults, and a YAML line-level error.

Vitest covers the timer (8 tests) and the noise generator (2). Playwright picks a task, starts a session, pauses, skips to the break, reloads (still running), and stops and saves, with axe on the timer.

Cross-cutting matrix:
- privacy: register row, exported;
- accessibility: labelled ring, live announcements, step list with `aria-current`;
- reliability: the timer is independent of tick timing;
- performance: indexed by `started_at`;
- AI safety: n/a;
- supply chain: no dependency, no third-party audio.

### EP-36 — Live vault sync

**Goal.** Adding, editing, renaming, or deleting a Markdown file in the vault updates the note in Axon within seconds, and the other way round.

**In scope.** `axon notes-sync -watch` (file events plus a periodic scan) as a service on `srv-second-brain`; Axon changes pushed through an event stream; templates synced as files; conflict handling unchanged (both versions kept).

**Issues**

- [x] **EP-36/IS-36.01 — Watch mode** · 8 pts · SHOULD
  - [x] `axon notes-sync -watch [-interval 2s]` runs the conflict-safe two-way sync at start and whenever the vault changes. Changes are detected by a cheap scan of names, sizes, and times. Errors back off up to a minute.
- [x] **EP-36/IS-36.02 — Change stream** · 8 pts · SHOULD
  - [x] `GET /v1/notes/sync/version` is a fingerprint that changes on every create, edit, rename, or delete in Axon, so the watcher syncs only when needed.
- [x] **EP-36/IS-36.03 — Templates as files** · 5 pts · SHOULD
  - [x] Notes in the vault's `Templates/` folder are offered in Axon's New note dialog, with `{{title}}` and `{{date}}` filled in. `GET /v1/notes?folder=` lists a folder.

**Exit evidence.** The integration test covers the vault to Axon direction (create, edit, rename, delete) and the Axon to vault direction (create, edit, rename, delete), each arriving within 10 s; the run takes about a second. Conflict handling is unchanged and still tested by the EP-11 tests. Playwright creates a note from a vault template. The runbook gives the `srv-secondbrain` command.

Cross-cutting matrix:
- security: a write token per vault, and every file operation through `os.Root`;
- reliability: back-off, and state saved after each run;
- remaining rows unchanged.

### EP-43 — Daily-use fixes

**Goal.** Fix what the owner hit in daily use, keeping the work already done on the parallel branch `claude/axon-ui-features-9ryyt4` (built from EP-34).

**Issues**

- [x] **EP-43/IS-43.01 — Full width** · 2 pts · MUST — pages use the whole width of the window.
- [x] **EP-43/IS-43.02 — Task boards** · 5 pts · MUST — Tasks open on a GTD board (inbox, next, waiting, someday), with a Kanban board (to do, doing, done) one click away.
- [x] **EP-43/IS-43.03 — Calendar grid** · 3 pts · MUST — the calendar opens on a month grid; the agenda is the second view.
- [x] **EP-43/IS-43.04 — Skills, not resources** · 5 pts · MUST — skill-graph nodes are competencies, with the resources that teach them listed under each ([ADR 0020](adr/0020-skills-are-competencies.md)).
- [x] **EP-43/IS-43.05 — Areas, Resources, Archive** · 3 pts · MUST — PARA pages in the nav, started from templates, with archive and restore.
- [x] **EP-43/IS-43.06 — Templates everywhere** · 3 pts · MUST — fleeting, literature, permanent, project, area, resource, archive, and daily note templates kept as files in the vault's `Templates/` folder (with Obsidian and Templater placeholders), plus templates for goals, milestones, habits, and projects.

**Integration notes.**
- The branch's three migrations were renumbered after the existing 00026 (00027–00029), and its ADR 0017 became 0020.
- Its duplicate live-sync and flashcard commits were not taken: EP-36 and EP-16 already deliver them.
- Its budget, themed-space, and explorer commits follow as EP-39, EP-38, and EP-37.

**Exit evidence.** `make check`, the integration suite, and 33 Playwright scenarios with axe pass. The new scenarios cover the GTD and Kanban boards, the month grid, PARA pages from templates with archive and restore, goal, habit, and project templates, and vault note templates.

### EP-37 — Knowledge explorer & NLP graph

**Goal.** See what you know: filter, chart, and navigate notes as a knowledge graph.

**In scope.** Filters by PARA area, Zettelkasten type, tags, dates, and links; charts (notes over time, tag clouds, orphan notes); a knowledge graph combining wikilinks with NLP edges: keywords (TF-IDF), named entities, synonyms from an open thesaurus (WordNet), and semantic similarity (pgvector embeddings when an embedding provider is configured; local methods otherwise).

**Acceptance criteria**

- Every NLP edge says why it exists (shared keyword, entity, synonym, or similarity score).
- Works fully without any AI provider.

**Issues**

- [x] **EP-37/IS-37.01 — Filters and charts** · 8 pts · SHOULD — `GET /v1/notes/explore` filters by PARA, Zettelkasten type, tag, text, creation date, or unconnected notes, and returns counts for notes per month, PARA, type, tags, and shared keywords.
- [x] **EP-37/IS-37.02 — Keyword and entity extraction** · 8 pts · SHOULD — Markdown cleaning, English and French stopwords and stemming, TF-IDF keywords, and entity spotting, unit-tested in `internal/knowledge`.
- [x] **EP-37/IS-37.03 — Thesaurus** · 5 pts · SHOULD — a bundled English/French synonym list (`thesaurus.txt`), small and hand-made rather than WordNet, to keep the image light and the licence clear.
- [ ] **EP-37/IS-37.04 — Embeddings** · 8 pts · COULD — not done: Axon has no embedding provider or pgvector column yet, so similarity uses local TF-IDF cosine. It needs its own consent and provider settings.
- [x] **EP-37/IS-37.05 — Graph view** · 5 pts · SHOULD — `GET /v1/notes/graph` keeps wikilinks and connections, then the strongest local reason per pair (entity, keywords, synonym, or similarity), at most six NLP edges per note and 600 notes; `/notes/explore` shows the graph coloured by PARA, toggles per kind, the reasons for the focused note, and a table version.

**Notes**

- Taken from the parallel branch `claude/axon-ui-features-9ryyt4` (`3e5f098`). No AI provider is involved; queries are scoped to the signed-in person and bounded.

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

**Issues**

- [x] **EP-38/IS-38.01 — Theme system per area** · 8 pts · SHOULD — the page path picks a space (`web/src/lib/spaces.ts`); each space tunes the colour tokens so text keeps AA contrast, and a scrim calms the heading band.
- [x] **EP-38/IS-38.02 — Artwork** · 8 pts · SHOULD — nine sceneries in light and dark (notebook, library, blueprint, garden, summit, almanac, ledger, atlas, desk), drawn from geometry by `web/scripts/spaces.py`: original, shipped under the project's licence, nothing fetched from another origin.
- [x] **EP-38/IS-38.03 — Effects and fallbacks** · 5 pts · SHOULD — slow light effects that stop with reduced motion; reduced transparency removes the artwork; Preferences has a Themed spaces switch (`appearance.spaces`).

**Notes**

- Taken from the parallel branch `claude/axon-ui-features-9ryyt4`; flashcards, focus, and progress (built on this line) were added to the map.
- axe checks every space in light and dark in the e2e suite.
- Stacks on Today (EP-34) stay open: Today keeps its own scenery and was not reorganised here.

### EP-39 — Personal finance

**Goal.** Know where money comes from and goes, in the spirit of Firefly III, without sending data anywhere.

**In scope.** Asset, liability, revenue, and expense accounts; double-entry transactions (income, expense, transfer) with categories, tags, and notes; budgets per period; recurring transactions; rules that categorise automatically; CSV import; reports; full API; finance data classified sensitive, encrypted at rest where supported, excluded from agents unless a separate consent is granted.

**Acceptance criteria**

- Balances always reconcile (property tests on double entry).
- Recurring transactions and rules are idempotent (re-running creates nothing twice).
- Finance endpoints pass the authorization matrix; amounts use exact decimals, never floats.

**Issues**

- [x] **EP-39/IS-39.01 — Accounts and transactions** · 13 pts · SHOULD — asset and liability accounts with opening balances; income, expense, and transfer transactions; balances are opening + income − expenses ± transfers, and transfers never count as spending; an account with history can only be archived.
- [x] **EP-39/IS-39.02 — Categories, tags, budgets** · 8 pts · SHOULD — expense and income categories with monthly budgets, tags on transactions, and filters by tag and account.
- [x] **EP-39/IS-39.03 — Recurring and rules** · 13 pts · SHOULD — recurring transactions on an RRULE, caught up when the budget opens, with a unique (series, date) index; rules on payee or note set a category and add a tag, never overriding a chosen category.
- [x] **EP-39/IS-39.04 — CSV import** · 8 pts · SHOULD — separator, date format, column mapping, decimal commas, and signed amounts; a per-row hash makes re-importing an overlapping export create nothing twice.
- [x] **EP-39/IS-39.05 — Reports and UI** · 13 pts · SHOULD — Budget in the main nav with Overview, Accounts, Recurring, Rules, Import, and Reports (monthly flows, spending by category, savings rate, net worth).

**Notes**

- Taken from the parallel branch `claude/axon-ui-features-9ryyt4` (migrations renumbered 00030 and 00031).
- Amounts are integer cents end to end. Finance data is excluded from search and agents (no separate consent flow yet) and included in the account export. Encryption at rest relies on the host disk; column-level encryption was not added.
- Property test: total of balances = openings + income − expenses over random ledgers; integration tests cover reconciliation, idempotent recurrence and import, and the cross-user 404 matrix.

### EP-40 — Instance export & import

**Goal.** Move an entire Axon to another server or configuration and keep everything.

**In scope.** `axon instance export` (database dump, settings, uploaded files, a manifest with versions and checksums) and `axon instance import` into an empty installation, with migrations applied; documented in operations.

**Acceptance criteria**

- An export imported on a fresh stack passes the restore drill (integration test).
- Secrets sealed with `AXON_ENCRYPTION_KEY` are re-sealed or require the same key, as documented.

**Issues**

- [x] **EP-40/IS-40.01 — Export** · 5 pts · SHOULD — `axon instance export [FILE]`: every table in one read-only snapshot, in COPY text format, with a manifest (format, Axon and schema versions, rows and SHA-256 per table, key fingerprint). Uploaded files are in the database, so they are included.
- [x] **EP-40/IS-40.02 — Import** · 5 pts · SHOULD — `axon instance import [FILE]` into a fresh installation only: schema moved to the archive's version, one transaction with counts and checksums checked, then newer migrations. Refuses a newer archive, a damaged one, or a different `AXON_ENCRYPTION_KEY` (sealed secrets need the same key; re-sealing was not added).
- [x] **EP-40/IS-40.03 — Drill and docs** · 3 pts · SHOULD — the integration test runs the restore drill on demo data (every table identical after import) plus an older-schema archive, a wrong key, and a damaged table; a CLI drill on the local stack is recorded in [operations](operations.md#move-an-instance-ep-40).

### EP-16 — Learning engine, Teacher agent & Anki

**Goal.** Retrieval-first learning: spaced scheduling, reviewed practice questions, dual-coded cards, and Anki sync.

**In scope.** FSRS scheduling (SM-2 fallback); review sessions and tests on the website; illustrated flashcards (image, cloze, front/back) that the person can view, add, edit, and delete; Teacher agent generating questions and cards from literature and permanent notes only; approval before cards enter the deck; Anki `.apkg` export and automatic sync of approved cards to the `anki-sync-server` on `srv-services`, by hand or after a Teacher proposal is approved.

**Acceptance criteria**

- FSRS results match the reference implementation on shared fixtures.
- The Teacher agent cannot read fleeting notes (tested).
- Exported decks import into Anki desktop without errors.

**Issues** (split at start: Anki sync moved to EP-42; design in [ADR 0017](adr/0017-flashcards-fsrs-and-anki.md))

- [x] **EP-16/IS-16.01 — Card and review model with images** · 8 pts · SHOULD
  - [x] Cards are entities with a deck, question, answer, optional picture (PNG, fitted to 1200 px, no metadata), and FSRS state. Reviews are logged; cards link to their source (`derived_from`). Everything is exported.
- [x] **EP-16/IS-16.02 — FSRS scheduler** · 8 pts · SHOULD
  - [x] An FSRS-5 port matches the reference implementation (py-fsrs 4.1.2) on 480 shared fixture reviews. `/v1/flashcards/due` previews the next due time for each rating.
- [x] **EP-16/IS-16.03 — Review session and test UI** · 13 pts · SHOULD
  - [x] Review: Space shows the answer, 1–4 rate it, the interval appears on each button, and cards rated Again come back within the session.
  - [x] Practice test: up to ten random cards with typed answers and self-marking. It never changes the schedule.
- [x] **EP-16/IS-16.04 — Flashcard editor** · 8 pts · SHOULD
  - [x] Decks with due and new counts. Add cards with a picture; edit, suspend, resume, or delete them; replace or remove pictures. Learn tabs: Roadmaps · Flashcards · Skills assessment. Shortcut `g r`.
- [x] **EP-16/IS-16.05 — Teacher agent** · 21 pts · SHOULD
  - [x] Tools: `list_study_notes`, `read_study_note`, `list_decks`, `get_assessment`, and `propose_cards`. Only literature and permanent notes are readable (tested: a fleeting note's text never reaches the model). Cards follow retrieval, elaboration, and interleaving principles.
- [x] **EP-16/IS-16.06 — Card approval queue** · 8 pts · SHOULD
  - [x] Teacher cards are proposals in the existing approval queue. On approval they land in a deck named after the note (covered by the every-write-tool test).
- [x] **EP-16/IS-16.07 — Anki export** · 8 pts · SHOULD
  - [x] `.apkg` for all decks or one, pictures included. A CI job imports it with the official `anki` library.
- Moved: **Anki sync to `srv-services`** (13) became EP-42.

**Exit evidence.**
- Go unit tests cover the FSRS reference match, retrievability, image fitting, and the package writer.
- The integration test covers create, link, decks, due previews, review scheduling (Good leads to the 10-minute step), and validation; a 200 KB picture upload; and the `.apkg` opened as SQLite (notes, cards, schema 11, picture). It also covers the Teacher boundary and approval, and an avatar upload above 64 KB.
- The CI `anki` job imports a package into real Anki.
- Vitest covers interval labels and sampling.
- Playwright adds an illustrated card, reviews it with the keyboard (Space, then Easy), takes a test, and checks the export link, with axe.

Cross-cutting matrix:
- privacy: register rows, Teacher scope;
- security: re-encoded images, upload limits;
- accessibility: keyboard review, labelled rating group, image alt text;
- reliability: transactional reviews;
- performance: indexed due query;
- AI safety: read boundary and approval;
- supply chain: `modernc.org/sqlite` promoted (already in the graph), `anki` used only in CI.

### EP-42 — Anki sync to srv-services

**Goal.** Cards in Axon reach the person's Anki through the existing `anki-sync-server` on `srv-services`, without downloading files.

**In scope.** A small sidecar using the official `anki` library: it holds a collection per person, imports new and changed cards (pictures included), and syncs to the server with credentials kept in `deploy/env/*.env.age`. It runs by hand from the Flashcards page, or after a Teacher proposal is approved. Operations docs cover the credentials and restore.

**Issues** (design in [ADR 0018](adr/0018-anki-bridge.md))

- [x] **EP-42/IS-42.01 — Sidecar and collection** · 5 pts · SHOULD
  - [x] `deploy/anki`: the official `anki` library (26.9.3) behind `POST /sync`, with one collection per person.
  - [x] Imports update notes in place by GUID and never delete. Full syncs are resolved safely: download into a new collection, upload only to an empty server.
- [x] **EP-42/IS-42.02 — Sync trigger and status** · 5 pts · SHOULD
  - [x] `GET/PUT/DELETE /v1/me/anki` stores a per-person account with the password sealed and write-only. `POST /v1/me/anki/sync` syncs now.
  - [x] Automatic sync after the Teacher's cards are approved; the last result is shown on Learn → Flashcards.
- [x] **EP-42/IS-42.03 — Deploy and docs** · 3 pts · SHOULD
  - [x] Compose service and server overlay; deploy builds and ships `axon-anki`; CI scans it. `ANKI_SYNC_URL` suggests the `srv-services` server in both settings files. Runbook, privacy register, and ADR 0018.

**Exit evidence.**
- `scripts/check-anki-sync.sh` (also in CI) runs against a disposable `anki-sync-server`: 2 cards arrive in a fresh "Anki desktop" collection; a second sync adds none (updated in place); a note added in Anki survives the next bridge sync.
- The integration test, with a fake bridge, covers validation, a write-only password kept on update, what the bridge receives (a real `.apkg`), the ok and error status with a 502, auto-sync after approving Teacher cards, and disconnect.
- Playwright connects an account, sees the failure shown for an unreachable server, and disconnects, with axe.

Cross-cutting matrix:
- identity: session only, NotDemo;
- security: sealed password, internal-only bridge, no body logging;
- privacy: register rows, the processor is the person's own server;
- reliability: per-user lock and safe full-sync rules;
- observability: status recorded and warnings logged without content;
- supply chain: `anki` pinned, Python base pinned by digest, image scanned by Trivy.

### EP-15 — Gamification engine

**Goal.** Recognize meaningful progress with transparent, opt-in XP, levels, badges, streaks, and freezes bought only with earned points.

**In scope.** XP rules, levels, badges, the streak state machine (see README), earned-point freezes. Every area feeds the same engine: tasks, projects, habits, goals, notes (Zettelkasten, PARA, daily notes), roadmaps and reviews, Pomodoro sessions, and media and book trackers, so one streak and one progress view reflect the whole life. Shared experiences (opt-in sharing of a badge or a streak with another person on the same Axon) are in scope only if they need no public exposure.

**Acceptance criteria**

- The user can see exactly how every point was earned.
- Gamification can be turned off entirely, hiding every surface.
- Freezes cannot be bought with money.
- The streak state machine is covered by property tests, including time zones.
- No notification uses loss-aversion or shaming language.

**Issues** (design in [ADR 0019](adr/0019-progress-from-source-data.md))

- [x] **EP-15/IS-15.01 — XP and rule engine** · 13 pts · SHOULD
  - [x] A ledger computed from every area's source data: tasks (on-time bonus), habits, focus, flashcards, notes (permanent bonus), skills, milestones, goals, and projects. 12 published rules. It sums exactly to the total (tested) and follows undo.
- [x] **EP-15/IS-15.02 — Streak state machine** · 8 pts · SHOULD
  - [x] Inactive, active, at risk, frozen, broken, computed on local dates. Property tests over 3,000 random histories and six time zones across a DST change.
- [x] **EP-15/IS-15.03 — Freezes** · 5 pts · SHOULD
  - [x] 100 earned points each, at most 2 held, no money path. A freeze bridges a gap only when there is a run behind it, is used automatically, and its use is saved.
- [x] **EP-15/IS-15.04 — Badges** · 8 pts · SHOULD
  - [x] Eleven badges, from count thresholds or run lengths, each with the date it was earned.
- [x] **EP-15/IS-15.05 — Levels and progress screens** · 13 pts · SHOULD
  - [x] The Progress page shows the level bar, the run with its state and message, freezes, badges, a 30-day ledger in plain language, and every rule. A chip on Today appears only when rewards are on.
- [x] **EP-15/IS-15.06 — Settings and opt-out** · 3 pts · SHOULD
  - [x] `gamification.enabled` (off by default) is in Preferences and YAML. Off hides every surface and the API refuses the ledger and freezes. There are no notifications; messages avoid loss framing. The demo has it on, with one freeze.
- Deferred: **shared experiences** (sharing a badge or a run with another person on the same Axon), until a person-to-person model exists; production has one account. **Media and books** join the ledger with EP-17.

**Exit evidence.**
- Go property and example tests for the streak.
- The integration test covers: off by default (no data, a 409 on ledger and freezes), turning it on through YAML, points from five areas (26), the ledger matching the total, a badge earned, the freeze refused without points, and the freeze bought once points are earned.
- Playwright turns rewards on, checks the ledger entries and the "First step" badge with axe, and sees the chip on Today.

Cross-cutting matrix:
- privacy: register row, nothing stored but freezes;
- accessibility: progress bar with values, badge contrast without opacity;
- reliability: an advisory lock serialises freeze purchases;
- AI safety: n/a;
- supply chain: no dependency.

### EP-17 — Media trackers & Media agent

**Goal.** Track films, series, and books with metadata from TMDB and Open Library, and get recommendations whose reasons are shown.

**In scope.** Media items with want / in progress / done states and ratings; TMDB and Open Library adapters with caching, quotas, and attribution; Media agent recommendations.

**Acceptance criteria**

- TMDB and Open Library attribution and terms are met (checked against current terms at Epic start).
- Tracking works with both providers unavailable.
- Each recommendation states why it was made.

**Terms checked at Epic start (2026-10-11).** TMDB: show the TMDB logo and the notice "This product uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB." in an About or Credits place; cache at most six months; the base licence is non-commercial, and its content must not train AI. Open Library: identify with a User-Agent and contact (three requests a second, one without); covers by cover ID are not rate-limited; a link back is appreciated.

**Issues**

- [x] **EP-17/IS-17.01 — Media model and API** · 5 pts · COULD — films, series, and books as graph entities with want, doing, done, and dropped, ratings out of 10, progress and total (pages, episodes, or minutes), dates filled in on status changes, and reviews; `/v1/media` CRUD, private per person, in the account export.
- [x] **EP-17/IS-17.02 — TMDB adapter** · 5 pts · COULD — on when the operator sets `TMDB_API_KEY` (their own key; v3 key or v4 token); search and details (genres, director or creators, episodes); text and covers older than six months are pruned daily; the logo and notice are on the Media page and in the README credits.
- [x] **EP-17/IS-17.03 — Open Library adapter** · 5 pts · COULD — search and work details through the search index, identified by User-Agent (`AXON_CONTACT`), paced to three requests a second, covers by ID; HTTPS-only redirects.
- [x] **EP-17/IS-17.04 — Media screens** · 8 pts · COULD — `/media` with kind tabs, search or add by hand, shelves by status with covers, an edit dialog, "Up next" with reasons, and attribution; `g m` shortcut; the library space.
- [x] **EP-17/IS-17.05 — Media agent** · 8 pts · COULD — `list_media` (no reviews or catalogue text), `media_recommendations`, and `suggest_media` proposals; recommendations without AI come from Axon's own explained ranking.

**Notes**

- Lookups need the `external_metadata` consent (defined in EP-04), since the search text goes to the provider; the Media page asks for it, with what is sent, and Privacy turns it off.
- Searches are cached for a week and capped at 30 lookups a minute per person; a catalogue that cannot be reached answers 503 with "add it by hand".
- Covers are copied to Axon (at most 2 MB, type checked from the bytes) so the CSP stays same-origin and readers' IP addresses never reach the providers.
- Catalogue text is never sent to the AI provider, which keeps TMDB content out of models.

### EP-18 — Health logging

**Goal.** Log food and training privately, only with explicit consent.

**In scope.** The health-data consent, food logging (Open Food Facts and USDA FoodData Central lookups, or numbers typed in), and workout logging. The coach agents moved to EP-44 when the Epic was split at start (2026-10-11).

**Acceptance criteria**

- Health data requires explicit consent (GDPR Art. 9) before any is stored.
- Health data is never sent to a model provider (no agent tool reads it in this Epic).
- Logging works with all external providers unavailable.

**Terms checked at Epic start (2026-10-11).** Open Food Facts: data under the ODbL (contents under the Database Contents License, images CC BY-SA); identify with `AppName/Version (contact)`; 10 searches and 15 product reads a minute per IP, with bans beyond. USDA FoodData Central: public domain (CC0), credit appreciated; a data.gov key, 1,000 requests an hour.

**Issues**

- [x] **EP-18/IS-18.01 — Consent flow for health data** · 5 pts · COULD — every health endpoint answers 403 without `health_data`; withdrawing it deletes every food entry and workout in the same transaction (a new privacy hook); a separate `food_lookups` consent covers sending food names or barcodes to the catalogues; "Delete all my health data" on the page.
- [x] **EP-18/IS-18.02 — Food model and adapters** · 13 pts · COULD — food entries keep their nutrients for the amount eaten; Open Food Facts search (Search-a-licious) and barcode, USDA search with `USDA_API_KEY`; foods cached for 30 days, the server paced under Open Food Facts' limits (a search waits at most 3 s, then "busy"), 10 lookups a minute per person, no redirects followed.
- [x] **EP-18/IS-18.03 — Food logging UI** · 13 pts · COULD — `/health` with a day view by meal, totals, a seven-day energy chart without targets, search and choose with grams (one serving by default), or numbers typed in.
- [x] **EP-18/IS-18.04 — Workout model and UI** · 13 pts · COULD — workouts with kind, minutes, effort (RPE), notes, and sets (reps and weight, or distance and time); the last 30 days and this week's sessions, minutes, and active days.

**Notes**

- Health data lives outside the graph: it is never in search, links, the dashboard, or agent tools. It is in the account export and the instance archive.
- No calorie targets or restriction framing; the page says Axon gives no medical or dietary advice.
- The demo account has no health data: consent is personal, and the demo is shared.

### EP-44 — Coach agents

**Goal.** Conservative, educational coaching on food and training that the person explicitly opts into.

**In scope.** A separate consent to send health data to the AI provider; Personal Trainer and Nutritionist skills with read tools over the health logs and proposal-only writes (a planned workout).

**Acceptance criteria**

- Health data reaches a model provider only with its own consent, on top of `ai_agents` and `health_data`.
- Coach agents never diagnose, never propose unsafe restriction, and direct red-flag topics to professionals (scripted eval suite in CI, live-model evals when a key is configured).

**Issues**

- [x] **EP-44/IS-44.01 — Health-to-AI consent** · 3 pts · COULD — a `health_ai` purpose; skills declare `consents:` in their front matter, checked when a run starts (403 naming the consent) and again by the health tools on every call; the Agents page lists what each agent needs; Privacy asks before withdrawing `health_data`, which deletes the logs.
- [x] **EP-44/IS-44.02 — Personal Trainer agent** · 13 pts · COULD — `training_log` (up to 90 days) and proposal-only `create_task` sessions; gradual progression, rest, no medical advice.
- [x] **EP-44/IS-44.03 — Nutritionist agent** · 13 pts · COULD — `food_log` (up to 30 days) and `training_log`; additions and swaps, never targets, fasting, or food-group cuts.
- [x] **EP-44/IS-44.04 — Safety eval suite** · 13 pts · COULD — `safety: health` turns on three guards that do not rely on the model: red-flag requests (English and French: chest pain, fainting, eating disorders, purging, laxatives, pregnancy, insulin, medication, self-harm…) get a fixed referral with no model call; a reply with a daily target under 1,200 kcal or a fast of two days or more is replaced; a write tool with such input is refused. Scripted evals in CI cover both lists of cases and the end-to-end runs; live-model evals still wait for a configured key (as in EP-19).

### EP-45 — Daily-use fixes 2

**Goal.** The owner's corrections after using staging (2026-10-11).

**In scope.**
- Pomodoro defaults to 4 work sessions (25/5 × 4, with the long break after the fourth).
- Today drops the "Today · Axon" line under the header.
- The dashboard network shows hard and soft skills only (no resources).
- Savings pots ("tirelire"): money set aside for a later purchase, with a target, a target date, contributions, and progress.

**Issues**

- [x] **EP-45/IS-45.01 — Pomodoro default** · 1 pt · MUST — 4 rounds by default (100 minutes of focus, the long break after the fourth); saved preferences keep their own value.
- [x] **EP-45/IS-45.02 — Today header** · 1 pt · MUST — the full-height scenery under the header is gone; the greeting, date, and check-in share one line, and the KPIs start right under the navigation.
- [x] **EP-45/IS-45.03 — Skills-only network** · 3 pts · MUST — the dashboard network has hard and soft skills and their prerequisites only (no goals, projects, notes, or resources); skills from roadmaps imported before ADR 0020 (one skill per book or course) are shown as the competency they teach, merged into one node, without changing stored data.
- [x] **EP-45/IS-45.04 — Savings pots** · 8 pts · MUST — `/v1/budget/pots`: a target, an optional date, money put in or taken out (never below zero), what to save each month (rounded up so the date is met), archive, history, in the export; a Savings pots tab in Budget; a demo pot.

### EP-46 — Advanced note NLP

**Goal.** A professional analysis of literature and permanent notes.

**In scope.** Lemmatisation for English and French, key-phrase extraction, topic clusters, a larger open thesaurus (for example Open English WordNet and WOLF for French, if their licences fit), concept co-occurrence graphs, and per-note "related ideas" with reasons; all local, no AI provider.

**Issues**

- [x] **EP-46/IS-46.01 — Stemming and key phrases** · 5 pts · SHOULD — per-note language detection (English or French function words) and the Snowball stemmers (`github.com/kljensen/snowball`, MIT); key phrases from one- to three-word n-grams ranked by frequency and TextRank centrality, a longer phrase replacing the words it covers; readability (Flesch reading ease in English, Kandel–Moles in French), words, sentences, reading time.
- [x] **EP-46/IS-46.02 — Thesaurus import** · 5 pts · SHOULD — 12,456 English synonym groups from Open English WordNet 2024 (CC BY 4.0), each word's most common sense only so rare senses do not link unrelated notes, generated by `scripts/oewn-thesaurus.py`; Axon's English/French list stays for French. WOLF (French WordNet, CeCILL-C) was not added.
- [x] **EP-46/IS-46.03 — Topics and concept graph** · 8 pts · SHOULD — topics by label propagation over TF-IDF cosine similarity (stable order), labelled by their most characteristic words; concepts are key phrases in at least two notes, linked when they appear together more than chance (NPMI ≥ 0.1, in two notes or more) or are synonyms, each link with its reason.
- [x] **EP-46/IS-46.04 — Explorer views** · 3 pts · SHOULD — Notes → Analysis (both kinds, literature, or permanent): summary, topics, the concept graph with a table version; an Analysis panel on every note (key phrases, readability, topic, related ideas with reasons; fleeting notes are compared without joining the corpus).

**Notes**

- `GET /v1/notes/analysis` and `GET /v1/notes/{id}/analysis`; analysis is computed on request, bounded by the explorer's 1,500-note limit, and never leaves the server.


### EP-47 — Zettelkasten knowledge graph

**Goal.** The owner asked for the knowledge graph from the advanced NLP of literature and permanent notes on the Notes page itself (2026-10-12).

**Issues**

- [x] **EP-47/IS-47.01 — Edge evidence** · 3 pts · SHOULD — per pair of notes: 0.45 × TF-IDF cosine + 0.30 × IDF-weighted cosine of key phrases over the concept vocabulary (phrases in two notes or more) + 0.15 × WordNet synonyms + 0.10 × shared named entities; the person's links add 0.25 and are always kept; every edge lists its reasons; key phrases that recur across notes are ranked first.
- [x] **EP-47/IS-47.02 — Structure** · 5 pts · SHOULD — k-nearest-neighbour sparsification (six per note); communities by multi-level Louvain over several fixed node orders (stable, best modularity kept); labels by class-based TF-IDF; weighted PageRank; bridges by the participation coefficient; graph statistics (modularity, density, unconnected notes). `GET /v1/notes/zettelkasten-graph?kind=&min=`.
- [x] **EP-47/IS-47.03 — Notes page view** · 5 pts · SHOULD — at the top of Notes: kind and strength filters, notes coloured by community and sized by centrality, bridges as diamonds, a panel with the selected note's connections and reasons or the communities with their core notes, keyboard selection, and a table version; the demo has a small Zettelkasten in three themes.

### EP-48 — Everything in the vault, visible templates

**Goal.** The owner asked that tasks, goals, projects, milestones, areas, archives, habits, health, and media be notes in the Markdown vault, and why the templates for each type of note and their sections were not visible (2026-10-12).

**Issues**

- [x] **EP-48/IS-48.01 — Vault mirror** · 13 pts · MUST — a worker keeps each item and a note under `Axon/` in step: front matter for fields, sections for text, a free Notes section kept across rewrites; Axon → vault on every change (renames move the file; archived items live in `Archive/`); vault → Axon for the title, editable front matter, and sections; a new note with a known `type` creates the item; deleting the file deletes it; moving into or out of `Archive/` archives or restores. Mirrored notes are left out of search so items are not listed twice. Preferences → Markdown vault (on by default, folder, health).
- [x] **EP-48/IS-48.02 — Health in the vault** · 3 pts · MUST — food days and workouts as one-way notes under `Axon/Health/`, only when turned on with the health consent given, removed when either is withdrawn.
- [x] **EP-48/IS-48.03 — Visible templates** · 5 pts · MUST — templates grouped (Zettelkasten, PARA notes, Axon items, other) with their sections and front matter previewed in New note; an Archive note template; item templates for tasks, goals, milestones, projects, areas, resources, habits, films, series, books; Notes → Templates with every template, its sections, and its Markdown, your vault's templates first; a "Start from a template" strip on the Notes page; `?new=<template>` links.

**Why they were not visible before.** The Notes page's own field created a blank note; the templates were only behind "＋ New note" in the sidebar, listed by name; the vault's templates appear once its `Templates/` folder is synced.
### EP-20 — Analytics & insights

**Goal.** Explainable personal metrics and weekly reviews computed locally from the event log.

**In scope.** Metric definitions; weekly review report; trends for tasks, habits, goals; CSV export.

**Acceptance criteria**

- Every metric documents its formula and links to its source events.
- All computation stays on the deployment.
- Reports stay bounded in time and memory with a year of events.

**Issues**

- [x] **EP-20/IS-20.01 — Metric definitions** · 13 pts · COULD — nine metrics in `internal/insights`, each with a title, unit, plain formula, source columns, a weekly query, and an items query; ISO weeks in the person's time zone; `GET /v1/insights?weeks=4..104`.
- [x] **EP-20/IS-20.02 — Weekly review** · 13 pts · COULD — `GET /v1/insights/review?week=YYYY-Www`: each metric against the week before, highlights for changes of 25 % or more (no praise or blame), the items behind each number (25 per section), open tasks due by the end of the week; previous and next weeks.
- [x] **EP-20/IS-20.03 — Trend views** · 13 pts · COULD — the last four weeks against the four before (up, down, steady, new) in a sentence; `/insights` with 8, 12, 26, or 52 weeks, sparklines, "How it is counted", and drill-down via `GET /v1/insights/metrics/{id}/items`.
- [x] **EP-20/IS-20.04 — CSV export** · 5 pts · COULD — `GET /v1/insights/export.csv?weeks=` (52 by default): a row per week, a column per metric id.
- [x] **EP-20/IS-20.05 — Performance work** · 8 pts · COULD — indexes on completion, milestone, review, and entity creation dates (`00035`); the integration test loads a year of activity (7,300 tasks) and requires the weekly series, review, and 104-week CSV within three seconds together.

**Notes**

- Health and finance data are not insights metrics: health needs its own consent, and finance has its own reports (EP-39).

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

**Issues**

- [x] **EP-21/IS-21.01 — User docs** · 8 pts · MUST — [user-guide.md](user-guide.md): a tour of every area, privacy, and agents.
- [x] **EP-21/IS-21.02 — Operator docs** · 8 pts · MUST — [install.md](install.md): requirements, configuration with generated secrets, start, HTTPS (Tailscale or Caddy), first account, nightly instance archives, restore, upgrade, troubleshooting. A run from a fresh clone following it literally reached a ready instance in 75 s after the documented build-network fix, and its backup and restore steps were run too. **Still open:** an install by someone new, observed (acceptance criterion).
- [x] **EP-21/IS-21.03 — Upgrade testing** · 8 pts · MUST — `TestUpgradeFromThePreviousReleaseKeepsData` (schema 5 → 35 with every row, integrity, and the old token reading old tasks through the new API) in CI, plus a drill with the real v0.1.0 images ([upgrading.md](upgrading.md)).
- [x] **EP-21/IS-21.04 — Load tests** · 8 pts · MUST — `api/cmd/axon-load`; measurements and the single-host decision revisited in [ADR 0021](adr/0021-single-host-measured.md).
- [x] **EP-21/IS-21.05 — Recovery drill** · 5 pts · MUST — instance archive restore 42 s and database dump restore 4 s, within RPO ≤ 24 h and RTO ≤ 4 h; repeated in the install-guide run.
- [x] **EP-21/IS-21.06 — Security review fixes** · 13 pts · MUST — internal review ([security/review-2026-10.md](security/review-2026-10.md)): recovery sign-in locks after five wrong authenticator codes per account (medium, fixed), `Permissions-Policy`, govulncheck in CI, Go toolchain pinned to 1.26.9. **Still open:** the external review (acceptance criterion).
- [x] **EP-21/IS-21.07 — Support policy** · 5 pts · MUST — [support.md](support.md): SemVer, supported releases and backports, `/v1` stability and deprecation (six months, `Deprecation` header), private security reports with response times; [CHANGELOG.md](../CHANGELOG.md).

**Notes**

- Tagging `v1.0.0` waits for the two open acceptance criteria and for the owner to merge the stacked pull requests into `main`.

## Glossary

| Term | Meaning |
|---|---|
| Axon | The name of this self-hosted life OS, used everywhere: repository, module, images, UI, and docs. |
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
| 2026-10-11 | Owner request: EP-41 (skills assessment before the first roadmap) added after EP-30. |
| 2026-10-12 | Owner request: EP-48 puts every item in the vault as a note, both ways, and shows every template with its sections. Delivered. |
| 2026-10-12 | Owner request: EP-47 adds the Zettelkasten knowledge graph to the Notes page, on top of EP-46. Delivered. |
| 2026-10-11 | EP-21 delivered as a 1.0 release candidate: install, user, upgrade, and support docs, upgrade test and drill, load tests (ADR 0021), recovery drill, internal security review with fixes. External review and observed install remain. |
| 2026-10-11 | EP-20 delivered: explainable weekly metrics, trends, weekly review, CSV export. R3 complete; next EP-21. |
| 2026-10-11 | EP-44 delivered: Personal Trainer and Nutritionist with a health_ai consent and deterministic safety guards. |
| 2026-10-11 | EP-46 delivered: local NLP analysis of literature and permanent notes (Snowball, TextRank, Open English WordNet, topics, NPMI concept graph). |
| 2026-10-11 | EP-45 delivered: Pomodoro 4 rounds, compact Today header, skills-only dashboard network, savings pots. |
| 2026-10-11 | EP-18 split at start: health logging (EP-18, delivered) and coach agents (EP-44). Owner requests added as EP-45 (Pomodoro 4 sessions, Today header, skills-only network, savings pots) and EP-46 (advanced note NLP); R3 order is now EP-17 → EP-18 → EP-45 → EP-46 → EP-44 → EP-20. |
| 2026-10-11 | EP-17 delivered: media trackers with Open Library and TMDB (terms checked), explained recommendations, and a Media agent. |
| 2026-10-11 | EP-40 delivered: instance export and import with a checksummed manifest, closing R2.5. Next: R3 (EP-17). |
| 2026-10-11 | EP-37 delivered from the parallel branch: notes explorer, charts, and a local NLP knowledge graph. Embeddings (IS-37.04) stay open. |
| 2026-10-11 | EP-38 delivered from the parallel branch: themed spaces with original artwork for every area, AA contrast in light and dark, and a switch in Preferences. |
| 2026-10-11 | EP-39 delivered from the parallel branch: accounts, transfers, budgets, tags, recurring transactions, rules, CSV import, and reports. Budget is in the main nav. |
| 2026-10-11 | Owner request: keep the work of the parallel branch `claude/axon-ui-features-9ryyt4`. Added EP-43 (daily-use fixes) after EP-36. EP-39 (finance), EP-38 (themed spaces), and EP-37 (explorer) are taken from that branch, in that order. |
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
