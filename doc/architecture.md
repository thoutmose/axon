# Architecture

How Axon is built, what it runs on, and how it is configured. Back to the [README](../README.md) · [Français](architecture.fr.md).

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
| Frontend | SvelteKit + TypeScript, design tokens + scoped CSS ([ADR 0007](adr/0007-styling-without-tailwind.md)) | Small, fast UI with accessible progressive enhancement. |
| Data | PostgreSQL 16 + pgvector | Durable relational core, full text, vectors, constraints, extensions. |
| Background work | In-process workers reading PostgreSQL (event log cursor, queue tables) | No extra service; the database stays the single source of truth ([ADR 0022](adr/0022-no-valkey-yet.md)). |
| Graph view | D3 or Sigma.js | Scalable interactive skill and relationship visualization. |
| Auth | Built-in WebAuthn + TOTP | Passkeys, MFA, self-hosted control. |
| Edge | Caddy or Traefik | Automatic TLS and clear routing. |
| Ops | Prometheus, Grafana, Loki, OpenTelemetry | Metrics, dashboards, logs, traces. |
| Delivery | GitOps or Compose + Ansible/OpenTofu | Reproducible promotion suited to small clusters. |
| Security | SOPS/age, Trivy, SBOM, Renovate, SLSA practices | Reduce supply-chain and secret risk. |

Where the table offers a choice, [ADR 0002](adr/0002-initial-technology-stack.md) proposes one: built-in WebAuthn + TOTP, Caddy, and Compose + Ansible.

**Why Go and not Rust?** Rust would give stronger compile-time guarantees and somewhat lower memory use. Axon's work, though, is I/O-bound (PostgreSQL, HTTP, AI providers), where Go is as fast in practice. Go also compiles in seconds, has first-party SDKs for the services Axon uses, and keeps the codebase easy for one maintainer and AI agents to read. Rewriting the existing API would stop feature work for months without a user-visible gain. The choice is recorded in ADR 0002, and the [change log](BACKLOG.md#change-log) notes that a rewrite was considered and not scheduled.

## Hardware requirements

One host runs everything: Caddy, the API with its background workers, the web app, PostgreSQL with pgvector, the Anki bridge, and the backups ([ADR 0003](adr/0003-single-host-first.md), [ADR 0004](adr/0004-hosts-and-tailnet-only-access.md)). `srv-prod` is the example:

| | Minimum | `srv-prod` (the example) | Axon uses there (measured 2026-10-17) |
|---|---:|---:|---|
| CPU | 2 vCPU | 8 vCPU | Under 1 % at rest; a tech watch fetch or an import uses one core for a few seconds. |
| Memory | 4 GB | 16 GB | About 120 MB for all six containers at rest; 300 MB under load ([ADR 0021](adr/0021-single-host-measured.md)). |
| Disk | 40 GB SSD | 120 GB SSD | 1.8 GB of images, a 19 MB database, 1 MB of encrypted backups; notes' pictures and covers add to the database. |

Measured on 2026-10-11, one host serves 700 requests a second. A separate database host is only worth it if a measurement shows the need. The optional speech server (Kokoro, for natural podcast voices) needs about 8 GB of disk and 2 GB of memory more; Whisper, for videos without captions, more still: run them on another machine through `AXON_TTS_URL` and `AXON_WHISPER_URL` if the host is small.

> [!NOTE]
> Small hosts work because workers are bounded, connections are pooled, lists are paginated, and models run elsewhere: AI calls go to the provider each person chooses, a local Ollama server included, rate-limited and within a monthly budget.

## Configuration

| Variable | Required | Example | Purpose |
|---|---|---|---|
| `AXON_BASE_URL` | Yes | `https://axon.example` | Canonical URL and callback origin. |
| `DATABASE_URL` | Yes | `postgres://...` | PostgreSQL connection; store as a secret. |
| `AXON_ENCRYPTION_KEY` | Prod | `openssl rand -hex 32` | Encrypts TOTP secrets at rest; never commit. |
| `AXON_REGISTRATION` | No | `first-user` | `first-user` lets only the first account register, and others only with the owner's invitation link; `open` allows anyone who can reach Axon. |
| `AXON_DEMO` | No | `1` | Offer a shared demo account with sample data, reset every night, and show the one-minute teaser on the home page. |
| `WEBAUTHN_RP_ID` | Prod | `axon.example` | Passkey relying-party ID. |
| `ANTHROPIC_API_KEY` | Optional | secret | Server default Anthropic key for agents (people can also set their own provider). |
| `AXON_ANTHROPIC_OAUTH` | Optional | config | `1` to use an `ant auth login` profile instead of a key. |
| `TMDB_API_KEY` | Optional | secret | Film and series lookups (your own non-commercial TMDB key, v3 key or v4 read token). |
| `USDA_API_KEY` | Optional | secret | USDA FoodData Central food search (a free data.gov key). |
| `AXON_WHISPER_URL` | Optional | value | A speech-to-text server that speaks the OpenAI transcription API (faster-whisper-server, speaches, LocalAI…), on your network or not: YouTube videos without captions are transcribed there (EP-123). `AXON_WHISPER_MODEL` (default `whisper-1`) and `AXON_WHISPER_TOKEN` (a bearer token, a secret) go with it. |
| `AXON_CALENDAR_PRIVATE` | Optional | value | `1` lets followed calendars live on your own network (a Nextcloud on the LAN or tailnet); off, only public HTTPS addresses are fetched. |
| `AXON_SMTP_URL` | Optional | secret | Reminder emails through your mail server: `smtp://user:pass@host:587` or `smtps://user:pass@host:465`. |
| `AXON_SMTP_FROM` | Optional | value | The sender address of reminder emails. |
| `AXON_GOOGLE_CLIENT_ID`, `AXON_GOOGLE_CLIENT_SECRET` | Optional | secret | Sign in with Google; also `AXON_MICROSOFT_*` (with `AXON_MICROSOFT_TENANT`), `AXON_DISCORD_*`, `AXON_SLACK_*`. Redirect address: `AXON_BASE_URL/api/v1/auth/oauth/<provider>/callback`. |
| `AXON_APPLE_CLIENT_ID`, `AXON_APPLE_TEAM_ID`, `AXON_APPLE_KEY_ID`, `AXON_APPLE_PRIVATE_KEY` | Optional | secret | Sign in with Apple: a Services ID and the `.p8` key (or `AXON_APPLE_PRIVATE_KEY_FILE`). |
| `AXON_OIDC_ISSUER`, `AXON_OIDC_CLIENT_ID`, `AXON_OIDC_CLIENT_SECRET`, `AXON_OIDC_NAME` | Optional | secret | Sign in with any OpenID Connect provider (Authentik, Keycloak, GitLab…). |
| `AXON_CONTACT` | Optional | `you@example.org` | How Open Library can reach the operator (sent in the User-Agent); defaults to `AXON_BASE_URL`. |
| `ANKI_SYNC_URL` | Optional | private URL | Suggested anki-sync-server address for the Anki bridge (each person enters their own account). |
| `AXON_PORT` | No | `3000` | The port Caddy publishes on 127.0.0.1 (compose); a second stack, such as the end-to-end tests, runs beside the first on another. |
| `AXON_VERSION` | No | `dev` | The image tag compose builds and runs; `deploy.sh` sets it to the commit it deploys. |
| `AXON_TRUSTED_PROXIES` | No | `100.95.163.114/32` | Addresses of a proxy in front of Tailscale Serve (one serving Axon's own domain), so Caddy sees each visitor's address rather than the proxy's. |
| `AXON_ADDR`, `AXON_METRICS_ADDR` | No | `:8080`, `:9464` | Where the API listens, and where it serves Prometheus metrics apart from the public routes. |

Never put secrets in Markdown, issue text, prompts, screenshots, or logs. Use SOPS/age and a secret manager appropriate to the deployment.

## Security and GDPR

Axon follows OWASP ASVS as a verification baseline and CIS guidance for hosts. Network traffic between servers uses WireGuard or Tailscale; public ingress is limited to the reverse proxy. WebAuthn/passkeys and TOTP protect accounts; sessions rotate and revoke. RBAC and per-user ownership checks apply at every API boundary.

Privacy controls include consent records, purpose limitation, retention policies, export, erasure requests, audit trails, processor inventories, and redaction of prompts and logs. Backups are encrypted and have documented deletion windows. Health and learning data are sensitive; default visibility is private.

> [!NOTE]
> GDPR compliance is a continuing operational program, not a badge. Obtain appropriate legal advice for your deployment and jurisdiction.

## Testing and operations

- **Testing:** Go unit tests, PostgreSQL integration tests, OpenAPI contract tests, Playwright browser tests, accessibility checks, agent evals, migration rehearsal, and backup restore drills.
- **Observability:** Prometheus metrics, Grafana dashboards, Loki logs, OpenTelemetry traces, correlation IDs, queue depth, provider latency, token cost, and redacted agent decisions.
- **Reliability:** Define SLOs, alerts, RPO/RTO, rate limits, idempotency keys, circuit breakers, and runbooks before calling a feature done.
- **Supply chain:** Renovate, SBOMs, Trivy scans, pinned images, signed artifacts where practical, and SLSA-informed provenance.

## Development workflow

1. Pick the next ready issue in [BACKLOG.md](BACKLOG.md); do not work across Epics.
2. Create a scoped branch and write tests or acceptance examples first.
3. Implement a small vertical slice; keep migrations reversible and APIs documented.
4. Run formatting, linting, unit/integration/contract/accessibility/security checks.
5. Open a Conventional Commit PR/MR with evidence, risk, rollback, and docs.
6. Request independent review, human approval, merge, deploy to staging, and observe before promotion.

## Roadmap

| Phase | Outcome | Backlog, in order |
|---|---|---|
| Foundation | Secure deployment, CI, observability, auth, privacy. | EP-00 → 01 → 02 → 03 → 04 |
| Personal graph | Links, tasks, notes, goals, dashboard — no AI required. | EP-05 → 06 → 07 → 09 → 11 → 13 → 08 |
| Guided growth | Agents, habits, projects, roadmaps, notes workspace, delivery and tutor agents. | EP-19 → 12 → 22 → 23 → 24 → 25 → 10 → 26 → 14 → 27 |
| Daily driver | Demo, bilingual docs, dashboard, settings, shortcuts, avatars, Atomic Habits, Pomodoro, learning and Anki, gamification, live vault, knowledge graph, themed spaces, finance, portability. | EP-28 → 29 → 30 → 41 → 31 → 32 → 33 → 34 → 35 → 16 → 42 → 15 → 36 → 43 → 39 → 38 → 37 → 40 |
| Whole-life beta | Media, health, nutrition, analytics. | EP-17 → 18 → 45 → 46 → 44 → 20 |
| 1.0 | Hardening, recovery, documentation, upgrade path. | EP-21 |
| 1.1 | Owner requests: Zettelkasten graph, everything in the vault, diagrams, pictures in notes, learning and memory engine, agents on every page, installable app, reminders, quick add, calendars in, weekly review, imports, household, search by meaning, custom Today, bulk actions, trash and undo, share to Axon, task reminders, clean code. | EP-47 → 48 → … → 69 |

Current status: **every Epic in [doc/BACKLOG.md](BACKLOG.md#delivery-order) through EP-169 is delivered and released as 1.0.0.** To look around without an account, use **Try the demo** on the landing page: a shared account with sample data, reset every night. Operations: [doc/operations.md](operations.md).
