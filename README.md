<p align="center">
  <img src="https://img.shields.io/badge/status-active%20development-blue" alt="Active development">
  <img src="https://img.shields.io/badge/license-AGPL--3.0-purple" alt="AGPL-3.0">
  <img src="https://img.shields.io/badge/API-OpenAPI%203.1-6BA539" alt="OpenAPI 3.1">
</p>

<h1 align="center">Axon</h1>

<p align="center"><strong>Your self-hosted life, connected.</strong><br>Notes, action, learning, health, projects, and reflection in one humane graph.</p>

<p align="center"><strong>English</strong> · <a href="README.fr.md">Français</a></p>

> [!NOTE]
> Axon is in active development. Features marked ✅ work today; the others are planned, in the order given in the [roadmap](#roadmap). Axon is not a medical device, a therapist, or a promise of production readiness.

## Table of contents

- [Why Axon?](#why-axon)
- [Features](#features)
- [Architecture](#architecture)
- [Tech stack](#tech-stack)
- [Hardware](#hardware-requirements)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Security and GDPR](#security-and-gdpr)
- [AI agents](#ai-agents)
- [Gamification](#gamification)
- [Development](#development-workflow)
- [Testing and operations](#testing-and-operations)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License and acknowledgements](#license-and-acknowledgements)

## Why Axon?

Axon brings together the things that are usually scattered across note apps, task lists, trackers, courses, and spreadsheets. Its center is not a feed or a streak counter: it is a user-owned graph of commitments, knowledge, and evidence. Self-hosting, exportability, explicit consent, and reversible automation are first-class requirements.

> [!WARNING]
> Do not enter medical emergencies, secrets, or another person’s personal data. Coach outputs are informational and require qualified professional judgment.

### Operational principles

1. **You own your data.** Everything exports in open formats; erasure is verified, not just queued.
2. **Private by default.** Consent is explicit, per purpose, and revocable.
3. **Agents propose, people decide.** No AI-generated change takes effect without approval.
4. **Works without AI.** Every feature is useful with agents and external APIs turned off.
5. **Reversible over clever.** Prefer explicit, reversible actions and boring technology.
6. **Recovery, not punishment.** A missed day leads back in, never to shame.
7. **Operable by others.** Anyone with the docs can run, observe, and restore it.

## Features

### 🧠 Second Brain and Zettelkasten
- ✅ A notes workspace: folder tree, live Markdown editor with preview, `[[wikilinks]]` with suggestions, backlinks and mentions, tags, callouts, tables, daily notes, templates, and a command center (Ctrl+K).
- ✅ PARA folders and Zettelkasten notes, Markdown import and export.
- ✅ Two-way sync with the `~/SecondBrain` vault (Git), keeping both versions on conflict.
- ✅ Vim keybindings in the note editor (optional), and keyboard shortcuts everywhere: press `?` for the list, `g` then a letter to jump to a page.
- ✅ Live sync with your Markdown vault: add, edit, rename, or delete a file and the note follows within seconds, and the other way round; vault templates appear in Axon.
- ✅ A knowledge explorer (Notes → Explore): filter by PARA, Zettelkasten type, tag, text, date, or unconnected notes; charts of notes over time, by PARA and type, tags, and shared keywords; and a knowledge graph that adds local NLP edges to your links (shared entities, distinctive keywords, English and French synonyms, similar wording), each saying why it exists. No AI provider involved.
- ✅ Analysis of literature and permanent notes (Notes → Analysis): English and French Snowball stemming, key phrases by TextRank, readability, topics found from similar wording, a concept graph whose links say why (together more than chance by NPMI, or synonyms from Open English WordNet), and on every note its key phrases, topic, and related ideas with reasons. All local.
- ✅ Everything is a note in your vault: tasks, goals, milestones, projects, areas, resources, archives, habits, and films, series, and books are Markdown notes under `Axon/` (fields in the front matter, text in sections, a free Notes section), kept in step both ways: edit the note or the item and the other follows; a new note with a type creates the item; deleting the file deletes it; moving it into `Archive/` archives it. Food and training logs too, one way, when you turn them on.
- ✅ Pictures and PDFs in notes: paste, drop, or pick a file in the editor and it is embedded (`![](Attachments/…)`); Obsidian embeds like `![[photo.png|300]]` show too. Photos lose their metadata and are scaled to 2400 px; files sync with the vault both ways and are part of the export.
- ✅ Templates for every kind of note, with their sections shown before you create: fleeting, literature, permanent; project, area, resource, and archive notes; tasks, goals, milestones, projects, areas, resources, habits, films, series, books; daily and meeting notes; plus your vault's own `Templates/` (Notes → Templates).
- ✅ A Zettelkasten knowledge graph on the Notes page: literature and permanent notes linked by combined evidence (IDF-weighted key phrases, WordNet synonyms, shared names, similar wording, your own links), communities found by Louvain modularity and named by class-based TF-IDF, centrality by PageRank, and bridge notes between communities; select a note to see why it connects.

### ✅ Action and execution
- ✅ Tasks with Today, week, and GTD views (inbox, next, waiting, someday), contexts, recurrence, one-key capture, and daily check-ins.
- ✅ Projects with Kanban boards (keyboard-accessible, WIP limits), epics, issues, sub-tasks, and project docs; tasks open on a GTD board, with a Kanban board one click away; the calendar opens on a month grid.
- ✅ Areas, Resources, and Archive pages (PARA), and templates for every kind of note, project, area, resource, goal, milestone, and habit, kept as files in your vault.
- ✅ Today dashboard with defined KPIs, a calendar, and iCal feeds.
- ✅ A command dashboard on Today: KPIs compared with the previous period, daily activity and 12-month charts, tables of what needs attention, and a network of your hard and soft skills with their prerequisites (skills only; books and courses stay under each skill).
- ✅ Pomodoro focus sessions (25/5 × 4 with a 15-minute long break by default, all adjustable, any number of sessions) linked to the task, project, skill, or note you work on, with focus time on the dashboard, a daily streak, focus sounds generated in the browser (noise, rain), and your own music links per activity.
- ✅ Personal finance in the spirit of Firefly III: asset and liability accounts with balances, income, expenses, and transfers (never counted as spending), categories with monthly budgets, tags, recurring transactions, rules that categorise, CSV import from bank exports without duplicates, reports (monthly flows, spending by category, savings rate, net worth), and savings pots (a *tirelire*) for something to buy later, with a target, an optional date, and how much to put in each month. Amounts are exact integer cents; finance data stays out of search and agents, and is in your export.

### 🌱 Habits, goals, and roadmaps
- ✅ Habits designed with James Clear's Atomic Habits: build good habits (obvious, attractive, easy, satisfying) or break bad ones (invisible, unattractive, difficult, unsatisfying), habit stacking ("After I…, I will…"), implementation intentions ("I will… at… in…"), an identity that every check-in votes for, habit contracts, the habits scorecard, two-minute versions, and gentle recovery prompts.
- ✅ Goals and milestones connected to tasks, notes, habits, projects, and roadmaps, with explained progress and a weekly review.
- ✅ Roadmaps you build by hand, import from OSSU, or start from Axon's curated library: 24 domains of the most recommended books, papers, university lectures, courses, and channels, from computer science and maths to Kubernetes, MLOps, and security. Each one is shown as a list or a layered skill graph.
- ✅ A skills assessment before you start: your situation, goals, and quick checks per domain give an explained level, and roadmaps start at your level.
- ✅ A Tutor agent that interviews you and drafts a roadmap where every skill cites its source, and a Resources agent that finds material for what you are learning. Both wait for your approval.
- ✅ Connect any element to any other, from wherever it is shown.

### 📚 Learning and media
- ✅ Illustrated flashcards you can view, add, edit, and remove, reviewed with FSRS-5 (the Anki scheduler, matched to its reference implementation), practice tests, and a Teacher agent that writes retrieval questions from your literature and permanent notes for your approval.
- ✅ A learning and memory engine after *Understanding How We Learn* and *Moonwalking with Einstein*: Learn → Today plans the day (cards due with the minutes they take, notes due for recall, a weekly mixed test, fleeting notes to process, leeches to practise); a few new cards a day and decks mixed (interleaving); write your answer before you see it; every card can carry the why, a concrete example, a vivid memory image, and a picture or an AI drawing (dual coding); free recall of whole notes on a blank page, compared with their key ideas and spaced further each time; memory palaces you walk place by place; cards drafted from what a note marks (`question :: answer`, `==term==`, `**Term**: definition`); and the Teacher writes illustrated cards for your approval.
- ✅ Reminders: a morning digest (tasks, cards to review with the minutes they take, notes to recall, habits, the weekly test) and an evening nudge (habits still open, with their two-minute version), as notifications on your devices, encrypted end to end, and by email if your server can send it; once a day, only with something to say, off by default.
- ✅ Installable on phones and desktops; offline, Capture keeps tasks and notes on the device and sends them, once, when you are back online.
- ✅ A menu in five groups (Do, Think, Grow, Life, Reflect) plus Today, and one Menu sheet on phones.
- ✅ Agents where you work: every page offers the agents whose skills fit it (Task Manager on Tasks and Today, Goals, Habits, the Teacher on Notes, a note, Learn, and Flashcards, the Tutor and Resources on Roadmaps, Media, the Personal Trainer and Nutritionist on Health, Side-Projects and Code Review on Projects), with ready-made requests and what the page shows sent along; proposals are approved right there.
- ✅ Anki packages (.apkg) per deck, pictures included, and automatic sync to your own `anki-sync-server` (for example on `srv-services`), by hand or right after you approve the Teacher's cards; Anki-only cards are never deleted.
- ✅ Films, series, and books: want, in progress, finished, or dropped, with ratings out of 10, progress in pages, episodes, or minutes, and reviews. Once you allow lookups (they send the title you type), books come from Open Library, and films and series from TMDB when the server has a key; anything can be added by hand. Covers are copied to Axon, so pages never load images from elsewhere. "Up next" ranks your list with the reasons (same creator or genre as what you rated 8 or more, or something you started), and a Media agent proposes new titles for your approval.

### 🥗 Health and sport
- ✅ Food and training logs, only after you consent to keeping health data (turning it off deletes them): meals with energy and macronutrients, looked up in Open Food Facts (by name or barcode) or USDA FoodData Central once you allow lookups, or typed in; workouts with sets, minutes, and effort. Never in search or AI agents, no targets or restriction.
- ✅ Personal Trainer and Nutritionist agents that read your logs only with a separate consent to share them with your AI provider, propose sessions or simple meal tasks for approval, hand red-flag topics (chest pain, fainting, eating disorders, pregnancy, medication…) to a professional without calling the model, and can never suggest under 1,200 kcal a day or multi-day fasts.

### ✨ Humane progress
- ✅ Opt-in progress rewards: points from every area (tasks, projects, habits, focus, flashcards, notes, roadmaps, goals), each traceable in a ledger, levels, badges, one daily streak with freezes bought only with earned points, and no shame or notifications.
- ✅ Insights: nine weekly metrics (tasks completed and captured, habit check-ins, milestones, focus minutes, notes, flashcards reviewed, recall rate, skills learned), each with its formula, its source, a trend in plain words, and the items behind every number; a weekly review against the week before, with what changed and open tasks to carry over; CSV export. All computed on your server.
- ✅ Ethical design: no dark patterns, no shame, no hidden consent, GDPR-aware controls.
- ✅ A profile picture, with a generated default (your initials on your own colours); uploads are re-encoded so no location data survives.
- ✅ Themed spaces: each area has its own original scenery, in light and dark: a Da Vinci notebook for notes, an Oxford library for learning, resources, and flashcards, a workshop blueprint for projects, a garden for habits, a contour map for goals and progress, a star chart for the calendar, a ledger for the budget, an old atlas for areas and the archive, and dotted paper for tasks and focus. Effects are slow, stop with reduced motion, and can be turned off in Preferences.
- ✅ Move your whole Axon to another server: `axon instance export` and `axon instance import` carry every table and picture with a checksummed manifest, into a fresh installation, older versions included.
- ✅ Preferences in one place: English or French, theme, font, text size, icons, motion, templates for notes, projects, and resources, and your AI provider and budget. Edit them in the app or as a YAML file (keys are write-only).

## Screenshots

| Home | Today (light) | Today (dark, phone) |
|---|---|---|
| ![Home page with a night mountain scene and feature sections](doc/screenshots/home.png) | ![Today view with a scenic hero, task counts, and cards](doc/screenshots/today.png) | ![Today view in dark mode on a phone](doc/screenshots/today-mobile-dark.png) |

| Command dashboard | Notes workspace | Command center |
|---|---|---|
| ![Command dashboard with KPIs, an activity chart, a heatmap, a skills network, and tables](doc/screenshots/dashboard.png) | ![Notes workspace with a folder tree, Markdown editor, and live preview](doc/screenshots/notes-edit.png) | ![Command center searching notes](doc/screenshots/command-center.png) |

Captured from a test account with sample data.

<details><summary>Design direction</summary>

Large typography, generous whitespace, dark mode, translucent glass over bundled scenery (ADR 0010). Motion should clarify state like a well-made instrument—not compete for attention. Respect reduced-motion preferences and keep every action usable without animation.

</details>

## Architecture

Axon uses a Go modular monolith first, PostgreSQL 16 + pgvector as the system of record (background work runs in-process, coordinated through PostgreSQL), and SvelteKit + TypeScript for the web application. Agents call bounded tools through an approval-aware orchestration layer.

### Mermaid — system context

```mermaid
flowchart LR
  user(["Person: owns the data, approves every AI change"])
  axon["Axon: self-hosted life OS"]
  ai["AI provider of each person's choice: Anthropic, OpenAI, Mistral, Gemini, DeepSeek, or a local model"]
  cat["Catalogues: Open Library, TMDB, Open Food Facts, USDA"]
  anki["anki-sync-server"]
  vault["Markdown vault, e.g. Obsidian"]
  user -->|"plans, captures, learns, reviews"| axon
  axon -->|"approved prompts and tools, with consent"| ai
  axon -->|"lookups, with consent"| cat
  axon -->|"flashcards"| anki
  axon <-->|"notes and items, both ways"| vault
```

### Mermaid — containers

```mermaid
flowchart TB
  user(["Browser or phone"])
  subgraph host["One host per environment, Docker Compose"]
    caddy["Caddy: HTTPS front, headers"]
    web["Web: SvelteKit + TypeScript"]
    api["API: Go + chi, REST and OpenAPI 3.1"]
    workers["In-process workers: agents, vault mirror, demo reset, media prune"]
    bridge["Anki bridge: official Anki library"]
    db[("PostgreSQL 16 + pgvector: system of record")]
    backup["Backup: encrypted dumps"]
  end
  user --> caddy
  caddy --> web
  caddy -->|"/api"| api
  web -->|"server-side calls"| api
  api --- workers
  api --> db
  workers --> db
  api --> bridge
  backup --> db
```

### Mermaid — deployment

```mermaid
flowchart LR
  dev["Your machine: make up, make check"] -->|"deploy.sh staging, green CI only"| staging["srv-staging: Axon + PostgreSQL"]
  staging -->|"deploy.sh prod, only what staging serves"| prod["srv-prod: Axon + PostgreSQL"]
  prod -->|"Anki sync, encrypted backups, tailnet"| services["srv-services: anki-sync-server, backups"]
  prod <-->|"notes-sync -watch, tailnet"| brain["srv-secondbrain: Markdown vault"]
```

### Mermaid — data model and universal links

```mermaid
erDiagram
  USER ||--o{ ENTITY : owns
  ENTITY ||--o{ LINK : "source or target"
  ENTITY ||--o{ EVENT : "history"
  ENTITY ||--o| NOTE : "is a"
  ENTITY ||--o| TASK : "is a"
  ENTITY ||--o| GOAL : "is a"
  GOAL ||--o{ MILESTONE : has
  ENTITY ||--o| PROJECT : "is a"
  ENTITY ||--o| HABIT : "is a"
  ENTITY ||--o| ROADMAP : "is a"
  ROADMAP ||--o{ SKILL : has
  ENTITY ||--o| CARD : "is a"
  CARD ||--o{ CARD_REVIEW : "reviewed in"
  ENTITY ||--o| MEDIA_ITEM : "is a"
  ENTITY ||--o| VAULT_NOTE : "mirrored as"
  ENTITY {
    uuid id PK
    uuid user_id FK
    string type
    string title
  }
  LINK {
    uuid id PK
    uuid source_id FK
    string relation
    uuid target_id FK
  }
  EVENT {
    uuid id PK
    string kind
    string fields
  }
```

### Mermaid — learning and memory loop

```mermaid
sequenceDiagram
  actor P as Person
  participant N as Notes
  participant T as Teacher agent
  participant R as Review engine, FSRS
  participant L as Learn: today's plan
  participant A as anki-sync-server
  P->>N: captures fleeting notes, rewrites them as literature and permanent notes
  P->>N: marks what to remember: q :: a, ==term==, **Term**: definition
  N-->>R: Make cards: drafts from the marks, added by the person
  P->>T: illustrated cards from these notes
  T->>N: reads only literature and permanent notes
  T-->>P: proposes cards with the why, an example, a vivid image, an SVG drawing
  P->>R: approves, then places cards in a memory palace
  loop Every day
    L-->>P: cards due, a few new ones, decks mixed, and notes due for recall
    P->>R: writes the answer, then sees it with the why, the example, and the image, and rates it
    R->>R: schedules the next review at the chosen retention
    P->>N: recalls a whole note on a blank page
    N-->>L: ideas recalled and missed, next recall further away when recalled well
  end
  L-->>P: weekly mixed test, and leeches to practise and rewrite
  R->>A: syncs approved cards
```

### Mermaid — assessment and Tutor roadmaps

```mermaid
sequenceDiagram
  actor P as Person
  participant As as Assessment
  participant Tu as Tutor agent
  participant L as Curated library
  participant Ro as Roadmaps
  P->>As: situation, job, goals, current level
  As-->>Ro: places you on roadmaps
  P->>Tu: what to learn, time, format
  Tu->>L: searches the best resources
  L-->>Tu: books, courses, papers, videos
  Tu-->>P: draft roadmap, every skill cites its source
  P->>Ro: approves, edits, or removes
```

### Mermaid — vault mirror, both ways

```mermaid
sequenceDiagram
  participant I as Item, e.g. a task
  participant M as Vault mirror worker
  participant N as Note under Axon/
  participant V as Markdown vault
  I->>M: created, changed, or deleted
  M->>N: writes front matter and sections
  N->>V: notes-sync carries the file
  V->>N: you edit, add, move, or delete the file
  N->>M: change in the event log
  M->>I: updates, creates, archives, or deletes the item
```

### Mermaid — daily streak and freezes

```mermaid
stateDiagram-v2
  [*] --> Idle
  Idle --> Active: first qualifying action of the day
  Active --> Active: an action on each following day
  Active --> AtRisk: a day passes without an action
  AtRisk --> Active: act today
  AtRisk --> Frozen: a freeze bought with earned points covers the day
  Frozen --> Active: act on the next day
  AtRisk --> Restart: two days without an action
  Restart --> Active: start again, without shame
```

## Tech stack

| Layer | Choice | Why |
|---|---|---|
| Backend | Go + chi | Low memory, explicit concurrency, simple deployment, strong tooling. |
| API | REST + OpenAPI 3.1; optional GraphQL later | Cacheable, observable, generated clients; avoid premature flexibility. |
| Frontend | SvelteKit + TypeScript, design tokens + scoped CSS ([ADR 0007](doc/adr/0007-styling-without-tailwind.md)) | Small, fast UI with accessible progressive enhancement. |
| Data | PostgreSQL 16 + pgvector | Durable relational core, full text, vectors, constraints, extensions. |
| Background work | In-process workers reading PostgreSQL (event log cursor, queue tables) | No extra service; the database stays the single source of truth ([ADR 0022](doc/adr/0022-no-valkey-yet.md)). |
| Graph view | D3 or Sigma.js | Scalable interactive skill and relationship visualization. |
| Auth | Built-in WebAuthn + TOTP | Passkeys, MFA, self-hosted control. |
| Edge | Caddy or Traefik | Automatic TLS and clear routing. |
| Ops | Prometheus, Grafana, Loki, OpenTelemetry | Metrics, dashboards, logs, traces. |
| Delivery | GitOps or Compose + Ansible/OpenTofu | Reproducible promotion suited to small clusters. |
| Security | SOPS/age, Trivy, SBOM, Renovate, SLSA practices | Reduce supply-chain and secret risk. |

Where the table offers a choice, [ADR 0002](doc/adr/0002-initial-technology-stack.md) proposes one: built-in WebAuthn + TOTP, Caddy, and Compose + Ansible.

**Why Go and not Rust?** Rust would give stronger compile-time guarantees and somewhat lower memory use. Axon's work, though, is I/O-bound (PostgreSQL, HTTP, AI providers), where Go is as fast in practice. Go also compiles in seconds, has first-party SDKs for the services Axon uses, and keeps the codebase easy for one maintainer and AI agents to read. Rewriting the existing API would stop feature work for months without a user-visible gain. The choice is recorded in ADR 0002, and the [change log](doc/BACKLOG.md#change-log) notes that a rewrite was considered and not scheduled.

## Hardware requirements

| Host | Baseline | Comfortable | Workload rationale |
|---|---:|---:|---|
| `srv-staging` | 2 vCPU / 4 GB / 40 GB SSD | 4 / 8 GB / 80 GB | The release before production: same stack, migrations rehearsed. |
| `srv-prod` | 2 vCPU / 4 GB / 60 GB SSD | 4 / 8 GB / 160 GB | Caddy, API with its workers, web, PostgreSQL + pgvector, Anki bridge, backups. |
| `srv-services` | 1 vCPU / 1 GB / 20 GB | 2 / 2 GB / 50 GB | Existing Anki sync service; receives encrypted backups. |
| `srv-secondbrain` | 1 vCPU / 1 GB / 50 GB | 2 / 2 GB / 100 GB | The Markdown vault, kept in step by `notes-sync -watch`. |

Each environment is one host ([ADR 0003](doc/adr/0003-single-host-first.md), [ADR 0004](doc/adr/0004-hosts-and-tailnet-only-access.md)); measured on 2026-10-11, the whole stack uses about 300 MB of memory and serves 700 requests a second ([ADR 0021](doc/adr/0021-single-host-measured.md)). A separate database host is only worth it if a measurement shows the need.

> [!NOTE]
> Low-memory targets assume bounded workers, connection pooling, pagination, and no local model inference. External-model calls are rate-limited and budgeted.

## Quick start

> [!IMPORTANT]
> Axon 1.0 is a release candidate: every area in this README works today. To run it for real, follow the **[install guide](doc/install.md)**; then read the **[user guide](doc/user-guide.md)**, how to **[upgrade](doc/upgrading.md)**, and the **[support policy](doc/support.md)**. Release notes: [CHANGELOG.md](CHANGELOG.md).

### Prerequisites

- Docker Engine and Compose v2; Git; GNU Make for the checks.
- Only if you use agents: a key for Anthropic, OpenAI, Mistral, Google Gemini, or DeepSeek, or a local model server (Ollama, LM Studio, vLLM). Each person can choose their own in the app.
- Optional: TMDB, Open Library, USDA, and Open Food Facts credentials or public endpoints according to their terms.

```bash
git clone git@github.com:thoutmose/axon.git   # private repository
cd axon
cp .env.example .env
docker compose up -d --build --wait
docker compose exec api axon migrate up
```

Then open <http://localhost:3000>. The API is served under `/api` (for example `/api/healthz` and `/api/readyz`). The stack listens on `127.0.0.1` only. To try it with sample data, set `AXON_DEMO=1` in `.env`, restart, run `docker compose exec api axon demo reset`, and press **Try the demo**.

**API reference.** The OpenAPI 3.1 document is at `/api/v1/openapi.yaml` ([source](api/openapi/openapi.yaml)), and an interactive Swagger UI is at `/api-docs/index.html`, served by Axon itself.

### Checks

| Command | What it runs |
|---|---|
| `make check` | `gofmt`, golangci-lint, Go unit tests; `svelte-check`, Prettier, Vitest (Node 24 in a container); README language parity |
| `make test-integration` | API, OpenAPI contract, agent eval, and migration tests against PostgreSQL 16 + pgvector |
| `make up` / `make down` | Build and start, or stop, the local stack |

## Configuration

| Variable | Required | Example | Purpose |
|---|---|---|---|
| `AXON_BASE_URL` | Yes | `https://axon.example` | Canonical URL and callback origin. |
| `DATABASE_URL` | Yes | `postgres://...` | PostgreSQL connection; store as a secret. |
| `AXON_ENCRYPTION_KEY` | Prod | `openssl rand -hex 32` | Encrypts TOTP secrets at rest; never commit. |
| `AXON_REGISTRATION` | No | `first-user` | `first-user` lets only the first account register; `open` allows anyone who can reach Axon. |
| `AXON_DEMO` | No | `1` | Offer a shared demo account with sample data, reset every night. |
| `WEBAUTHN_RP_ID` | Prod | `axon.example` | Passkey relying-party ID. |
| `ANTHROPIC_API_KEY` | Optional | secret | Server default Anthropic key for agents (people can also set their own provider). |
| `AXON_ANTHROPIC_OAUTH` | Optional | config | `1` to use an `ant auth login` profile instead of a key. |
| `TMDB_API_KEY` | Optional | secret | Film and series lookups (your own non-commercial TMDB key, v3 key or v4 read token). |
| `USDA_API_KEY` | Optional | secret | USDA FoodData Central food search (a free data.gov key). |
| `AXON_SMTP_URL` | Optional | secret | Reminder emails through your mail server: `smtp://user:pass@host:587` or `smtps://user:pass@host:465`. |
| `AXON_SMTP_FROM` | Optional | value | The sender address of reminder emails. |
| `AXON_CONTACT` | Optional | `you@example.org` | How Open Library can reach the operator (sent in the User-Agent); defaults to `AXON_BASE_URL`. |
| `ANKI_SYNC_URL` | Optional | private URL | Suggested anki-sync-server address for the Anki bridge (each person enters their own account). |

Never put secrets in Markdown, issue text, prompts, screenshots, or logs. Use SOPS/age and a secret manager appropriate to the deployment.

## Security and GDPR

Axon follows OWASP ASVS as a verification baseline and CIS guidance for hosts. Network traffic between servers uses WireGuard or Tailscale; public ingress is limited to the reverse proxy. WebAuthn/passkeys and TOTP protect accounts; sessions rotate and revoke. RBAC and per-user ownership checks apply at every API boundary.

Privacy controls include consent records, purpose limitation, retention policies, export, erasure requests, audit trails, processor inventories, and redaction of prompts and logs. Backups are encrypted and have documented deletion windows. Health and learning data are sensitive; default visibility is private.

> [!NOTE]
> GDPR compliance is a continuing operational program, not a badge. Obtain appropriate legal advice for your deployment and jurisdiction.

## AI agents

| Agent | Responsibility | Guardrail | Status |
|---|---|---|---|
| Task Manager | Triage and schedule work. | Preview before changing due dates. | ✅ |
| Goals/Milestones | Break outcomes into measurable steps. | No silent commitments or dates. | ✅ |
| Habits | Suggest cues, stacks, recovery, and reviews. | No shame or manipulative pressure. | ✅ |
| Side-Projects | Analyze repos into epics, issues, and tasks. | Read-only; issues are proposals. | ✅ |
| Code Review | Review diffs, tests, security, and maintainability. | Can comment after approval; can never approve or merge. | ✅ |
| Tutor | Interview, research, and draft roadmaps. | Every skill cites a source from the library or an allowlisted site; user approves. | ✅ |
| Resource | Find and classify cited resources. | Allowlisted sites only, robots.txt honoured, 15 pages per run, links not copies. | ✅ |
| Teacher | Generate reviewed retrieval practice and flashcards. | Reads only literature/permanent notes. | ✅ |
| Media | Suggest what to watch or read next, and propose new titles. | Sees only your list (no reviews, no catalogue text); every addition needs approval. | ✅ |
| Personal Trainer | Suggest next sessions from your training log, as tasks to approve. | Own consent (`health_ai`); red flags referred without a model call; non-medical. | ✅ |
| Nutritionist | Gentle suggestions from your food log, as tasks to approve. | Own consent; never calorie targets, fasting, or diagnosis (enforced, not only prompted). | ✅ |

All agents share a constitution: simple, explicit, small diffs, read before write, think in data, test everything, do not be clever. Skills are versioned `SKILL.md` files. Each task ends with reflection, lessons learned, evaluation evidence, and—when needed—human-in-the-loop approval.

## Gamification

Progress systems use XP, levels, streaks, badges, variable rewards, commitment devices, and earned-point streak freezes. The design is opt-in and transparent: users can disable competitive surfaces, see how points are calculated, and recover from missed days without humiliation. Inspiration includes *Hooked*, Octalysis, and Self-Determination Theory; the ethical constraint is stronger than engagement metrics.

## Development workflow

1. Pick the next ready issue in [BACKLOG.md](doc/BACKLOG.md); do not work across Epics.
2. Create a scoped branch and write tests or acceptance examples first.
3. Implement a small vertical slice; keep migrations reversible and APIs documented.
4. Run formatting, linting, unit/integration/contract/accessibility/security checks.
5. Open a Conventional Commit PR/MR with evidence, risk, rollback, and docs.
6. Request independent review, human approval, merge, deploy to staging, and observe before promotion.

## Testing and operations

- **Testing:** Go unit tests, PostgreSQL integration tests, OpenAPI contract tests, Playwright browser tests, accessibility checks, agent evals, migration rehearsal, and backup restore drills.
- **Observability:** Prometheus metrics, Grafana dashboards, Loki logs, OpenTelemetry traces, correlation IDs, queue depth, provider latency, token cost, and redacted agent decisions.
- **Reliability:** Define SLOs, alerts, RPO/RTO, rate limits, idempotency keys, circuit breakers, and runbooks before calling a feature done.
- **Supply chain:** Renovate, SBOMs, Trivy scans, pinned images, signed artifacts where practical, and SLSA-informed provenance.

## Roadmap

| Phase | Outcome | Backlog, in order |
|---|---|---|
| Foundation | Secure deployment, CI, observability, auth, privacy. | EP-00 → 01 → 02 → 03 → 04 |
| Personal graph | Links, tasks, notes, goals, dashboard — no AI required. | EP-05 → 06 → 07 → 09 → 11 → 13 → 08 |
| Guided growth | Agents, habits, projects, roadmaps, notes workspace, delivery and tutor agents. | EP-19 → 12 → 22 → 23 → 24 → 25 → 10 → 26 → 14 → 27 |
| Daily driver | Demo, bilingual docs, dashboard, settings, shortcuts, avatars, Atomic Habits, Pomodoro, learning and Anki, gamification, live vault, knowledge graph, themed spaces, finance, portability. | EP-28 → 29 → 30 → 41 → 31 → 32 → 33 → 34 → 35 → 16 → 42 → 15 → 36 → 43 → 39 → 38 → 37 → 40 |
| Whole-life beta | Media, health, nutrition, analytics. | EP-17 → 18 → 45 → 46 → 44 → 20 |
| 1.0 | Hardening, recovery, documentation, upgrade path. | EP-21 |

Current status: **EP-00–EP-16, EP-19, EP-22–EP-35, EP-36, EP-41, EP-42, EP-43, EP-39, EP-38, EP-37, EP-40, EP-17, EP-18, EP-45, EP-46, EP-44, EP-20, EP-21, EP-47, EP-48, and EP-49 done; 1.0 awaits the external security review and an observed install.** To look around without an account, use **Try the demo** on the landing page: a shared account with sample data, reset every night. Operations: [doc/operations.md](doc/operations.md). Details in [doc/BACKLOG.md](doc/BACKLOG.md#delivery-order).

## Contributing

Contributions are welcome when they improve user agency, accessibility, privacy, or maintainability. Please read [CONTRIBUTING.md](CONTRIBUTING.md) and the [Code of Conduct](CODE_OF_CONDUCT.md), open an issue with context and acceptance criteria, and keep changes small. For security vulnerabilities, follow [SECURITY.md](SECURITY.md) rather than opening a public issue.

## License and acknowledgements

Axon is licensed under **AGPL-3.0-or-later** so hosted modifications remain available to the community. See [LICENSE](LICENSE) for the authoritative text.

Acknowledgements and reading: [roadmap.sh](https://roadmap.sh/), [OSSU](https://github.com/ossu/computer-science), [Anki sync server](https://github.com/ankicommunity/anki-sync-server), [TMDB API](https://developer.themoviedb.org/docs/getting-started), [Open Library API](https://openlibrary.org/developers/api), [Open Food Facts](https://world.openfoodfacts.org/data), [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/), [Twelve-Factor App](https://12factor.net/), [Google SRE book](https://sre.google/sre-book/table-of-contents/), [FSRS](https://github.com/open-spaced-repetition/free-spaced-repetition-scheduler), and [Anthropic documentation](https://docs.anthropic.com/en/docs).

Catalogue data: book data and covers come from [Open Library](https://openlibrary.org/); English synonyms from [Open English WordNet](https://github.com/globalwordnet/english-wordnet) (CC BY 4.0); food data from [Open Food Facts](https://world.openfoodfacts.org/) (ODbL) and [USDA FoodData Central](https://fdc.nal.usda.gov/) (public domain). This product uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.

Books that shape the product: *Understanding How We Learn* (Weinstein, Sumeracki, Caviglioli), *Moonwalking with Einstein* (Foer), *Make It Stick*, *Atomic Habits* (Clear), *Hooked* (Eyal), *The Octalysis Framework* (Chou), *Building a Second Brain* (Forte), *How to Take Smart Notes* (Ahrens), and *Designing Data-Intensive Applications* (Kleppmann).

---

<p align="center"><sub>Axon — build a life you can understand, one small link at a time.</sub></p>
