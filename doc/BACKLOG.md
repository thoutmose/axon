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
| EP-49 | Audit: fixes and legacy removal | A whole-project audit: bugs fixed, unused code and services removed, documentation brought back to what runs. Owner request. | MUST | 8 pts | EP-48 | Done |
| EP-50 | Diagrams GitHub can display | Every README diagram in Mermaid, up to date with what runs. Owner request. | MUST | 2 pts | EP-49 | Done |
| EP-51 | Pictures in notes | Paste, drop, or pick pictures and PDFs in the editor; Markdown and Obsidian embeds render; files sync with the vault both ways and are exported. Owner request. | MUST | 8 pts | EP-11, EP-36 | Done |
| EP-52 | Learning and memory engine | Spaced, interleaved, retrieval-first reviews with a daily plan; free recall of notes; memory palaces; cards with the why, an example, a vivid image, and AI drawings; cards drafted from notes; deliberate practice. Owner request. | MUST | 21 pts | EP-16, EP-46, EP-51 | Done |
| EP-53 | Agents on every page | Each page offers the agents whose skills fit it, with ready-made requests, the page's context, and approvals in place. Owner request. | SHOULD | 5 pts | EP-19, EP-52 | Done |
| EP-54 | Navigation groups | Eighteen menu links become Today and five groups (Do, Think, Grow, Life, Reflect); one Menu sheet on phones. Owner decision on the EP-49 proposals. | SHOULD | 2 pts | EP-53 | Done |
| EP-55 | Installable app, offline capture | A web app manifest, icons, and a service worker: Axon installs on phones and desktops, opens offline, and captures made offline are sent once, later. Owner decision on the EP-49 proposals. | SHOULD | 3 pts | EP-54 | Done |
| EP-56 | Reminders | A morning digest and an evening nudge by Web Push (encrypted end to end) and optional email, at local times, once a day, only with something to say. Owner decision on the EP-49 proposals. | SHOULD | 5 pts | EP-55 | Done |
| EP-57 | Quick add for anything | One line in Capture becomes a task with a context and a day, a note with tags, a book, film, or series, a habit, a goal with a date, a project, or a flashcard; English and French. Owner decision on the EP-49 proposals. | SHOULD | 3 pts | EP-55 | Done |
| EP-58 | Calendars in | Follow Google, CalDAV/Nextcloud, Outlook, or any ICS calendar, read only: events (repeats expanded, time zones kept) beside Axon's own in the month and agenda views. Owner decision on the EP-49 proposals. | SHOULD | 5 pts | EP-13 | Done |
| EP-59 | Guided weekly review | A ten-step weekly review after Getting Things Done, on Axon's own data, ending in the weekly check-in; the weekend digest suggests it until it is done. Owner decision on the EP-49 proposals. | SHOULD | 3 pts | EP-20, EP-56 | Done |
| EP-60 | Imports | Todoist and TickTick tasks, Goodreads books, Letterboxd films, Firefly III and bank OFX/QFX money, Notion pages; preview first, nothing twice. Owner decision on the EP-49 proposals. | SHOULD | 5 pts | EP-11, EP-17, EP-38 | Done |
| EP-61 | Household sharing | People on the same Axon join a household by a one-time code and share lists (shopping, chores, anything), with items given to members; nothing personal is shared. Owner decision on the EP-49 proposals. | SHOULD | 5 pts | EP-02, EP-56 | Done |
| EP-62 | Search by meaning | Embeddings from the person's own service (local, OpenAI, or Voyage AI) behind their own consent, in pgvector: notes found by meaning in the command center and "Close in meaning" on each note. Closes IS-37.04. Owner decision on the EP-49 proposals. | COULD | 5 pts | EP-37 | Done |
| EP-63 | Customisable Today | Choose and order Today's cards, with new ones for learning, habits, today's calendar, and the household. Owner decision on the EP-49 proposals. | COULD | 2 pts | EP-30, EP-52, EP-58, EP-61 | Done |
| EP-64 | Bulk actions, drag onto the calendar | Select tasks and change them all at once (status, context, due date, delete); drag tasks between days and undated next actions onto a day, with a date field for the keyboard. Owner decision on the EP-49 proposals. | COULD | 3 pts | EP-07, EP-13 | Done |
| EP-65 | Audit 2: fixes | A review of EP-50 to EP-64: vault templates honour the mirror folder, learning days follow the person's time zone, unpacked imports are capped, form controls are styled everywhere, unused exports removed. Owner request. | MUST | 3 pts | EP-64 | Done |
| EP-66 | Trash and undo | Every deletion goes to a 30-day trash with what was deleted along with it; Undo after each deletion, and Account → Trash to restore or forget. Owner request: essential functionality. | MUST | 5 pts | EP-03 | Done |
| EP-67 | Share to Axon | The installed app in the phone's share sheet, a "Save to Axon" bookmarklet, and a clipper that keeps a page's readable text as a literature note. Owner request: essential functionality. | SHOULD | 3 pts | EP-55 | Done |
| EP-68 | Task reminders at a time | A task can carry a reminder time: pushed (and emailed when set up) then, once, from the task editor or quick add ("at 15:00", "à 15h"). Owner request: essential functionality. | SHOULD | 2 pts | EP-56 | Done |
| EP-69 | Clean code and the person's day | Duplicated helpers become one tested module each (time zone and "today", dates, text that reaches the database, editor text changes); the bugs the copies had are fixed; tests written first, including a browser suite run in a far time zone. Owner request: fix, refactor, clean code, TDD, documentation. | MUST | 5 pts | EP-68 | Done |
| EP-70 | Every foreign key indexed | Twenty foreign keys had no index, so deletes and lookups by them scanned whole tables (every calendar refresh, the recall queue); one migration, and a test that keeps it so. | SHOULD | 1 pt | EP-69 | Done |
| EP-71 | Scripts checked | ShellCheck (warnings and errors) runs in `make check` and CI on every shell script; the deploy and backup scripts pass. | SHOULD | 1 pt | EP-70 | Done |
| EP-72 | Lighter offline install | The service worker no longer keeps the API reference (Swagger UI, 1.8 MB) for offline use: each device downloads about half as much when Axon is installed or updated. | SHOULD | 1 pt | EP-55 | Done |
| EP-73 | No query per item | Lists cost the same number of queries for 2 items as for 12; the Goals list (two queries per goal) and Today (one per task) no longer slow down as data grows. | SHOULD | 2 pts | EP-72 | Done |
| EP-74 | Compressed responses | Caddy compresses pages, scripts, and API answers (zstd or gzip); nothing was compressed before (the OpenAPI document: 324 kB → 48 kB). | SHOULD | 1 pt | EP-73 | Done |
| EP-75 | Today and the page title in every browser | Owner report: Today stayed on "Loading your day…" and the page title showed as text at the top of the page (Firefox). Both fixed, with tests that fail without the fix. | MUST | 2 pts | EP-74 | Done |
| EP-76 | Covers for every film, series, and book | Owner request: pictures of the books, series, and films, and a better media card. "Find missing covers" looks up items added by hand, imported, or from the demo; the shelf shows poster cards, with a cover made from the title when there is none. | SHOULD | 2 pts | EP-17 | Done |
| EP-77 | Your memory, measured | Each review of a card in long-term memory keeps what the scheduler predicted (earlier ones by replaying the history); Learn shows predicted against remembered, by prediction, against the chosen retention, and says what to change. | SHOULD | 2 pts | EP-52 | Done |
| EP-78 | Topics at scale | Grouping notes into topics compared every pair of notes (4.4 s for 3,000); an inverted index compares only notes that share a term (151 ms). | SHOULD | 1 pt | EP-46 | Done |
| EP-79 | Key phrases kept | The key phrases of each note (TextRank, a third of the Notes page graph and analysis time) are kept by the hash of its text: at 1,500 notes, the analysis went from 1.84 s to 0.70 s and a repeat visit to the Zettelkasten graph from 2.85 s to 1.68 s. | SHOULD | 1 pt | EP-78 | Done |
| EP-80 | Other sites kept out | Measured over HTTPS: session cookie, CSP, HSTS, and origin checks were right; two isolation headers were missing and are added. axe found no serious issue on 30 pages in Firefox or Chromium. | SHOULD | 1 pt | EP-79 | Done |
| EP-81 | Containers without privileges | Every service but PostgreSQL runs with no Linux capability (Caddy keeps only the one its binary carries) and cannot gain privileges; images were already non-root and pinned by digest. | SHOULD | 1 pt | EP-80 | Done |
| EP-82 | Skills radar | Owner request ("graphe en toile d'araignée"): Roadmaps shows the hard and the soft skills acquired as two radar charts, one spoke per roadmap (or per section under three), done and being learnt, with a table view. | SHOULD | 2 pts | EP-14 | Done |
| EP-83 | Focus music | Owner request ("listen to the best music for productivity related to what I do during the Pomodoro session"): music composed in the browser, without words, matched to what the session is about (lo-fi beats, ambient, calm piano), chosen before or during a session. | SHOULD | 3 pts | EP-35 | Done |
| EP-84 | Teaser on the home page | Owner request: with the demo on, the home page shows "Axon in under a minute", a tour of the demo with Axon's own music, captions in English and French, served by Axon itself; recorded by a script anyone can run again. | SHOULD | 2 pts | EP-82, EP-83 | Done |
| EP-85 | Focus and Roadmaps in French | IS-31.06 says a page is translated when next touched: Focus, Roadmaps, the skills radar, and the focus sounds in Preferences now follow the language setting. | SHOULD | 1 pt | EP-31, EP-82, EP-83 | Done |
| EP-86 | French everywhere in the web app | The rest of IS-31.06: all 86 remaining components in French (about 1,800 strings), dates in the interface's language, and two tests that keep it so. | SHOULD | 8 pts | EP-85 | Done |
| EP-87 | The API speaks French | Everything the API writes for people follows `Accept-Language`: 480 French messages (errors, explanations, insights, badges, consents, media reasons) and the agents' introductions, with a test that parses the code and fails on any message without French. | SHOULD | 5 pts | EP-86 | Done |
| EP-88 | Daily needs | Owner request: the macronutrients needed to be healthy, from activity, age, sex, and size, with a warning to check with a nutritionist. Reference intakes for keeping one's weight, worked out and explained from published references; none before 18 or in pregnancy; no weight-loss targets (ADR 0028). | SHOULD | 3 pts | EP-18 | Done |
| EP-89 | A photo of a meal | Owner request: take a picture of the food and get its macronutrients. The person's AI provider estimates each food and portion, which they correct before logging, behind a new `food_photos` consent; the photo is stripped of metadata and never kept. A photo of a barcode is read on the device. | SHOULD | 5 pts | EP-18, EP-44 | Done |
| EP-90 | Chef & Nutritionist | Owner request: meals by budget, dietary restrictions, constraints, and time, by an AI agent that is a nutritionist and a cook. A planning form on Health, an agent that reads the daily needs and proposes a plan with recipes, a shopping list, and a cost, saved as a note once approved. | SHOULD | 5 pts | EP-44, EP-88 | Done |
| EP-91 | Daily journal | Mood, energy, and stress on five faces, sleep, three things to be grateful for, and a note, a day at a time, under the health data consent; a Journal page, a Today card, and the last 30 days. | SHOULD | 3 pts | EP-88 | Done |
| EP-92 | What helps you | Which habits go with better days: sleep, training, focus, habits kept, tasks finished, and gratitude against mood, energy, and stress over 90 days, with rank correlation, a permutation test, and a correction for many tests; only clear links, in plain words, never as causes. Computed on the server, no AI. | SHOULD | 3 pts | EP-91 | Done |
| EP-93 | Journal days in the vault | Each journal day becomes a daily note in the vault, with its scales in the front matter, written one way when health notes are on, and removed with the day or the consent. | SHOULD | 1 pt | EP-48, EP-91 | Done |
| EP-94 | Only what you use | A new account is asked what it wants Axon for; the menu and Today show just those pages, and Preferences hides or brings back any page. Fixes Preferences resetting Today's cards. | SHOULD | 3 pts | EP-54, EP-63 | Done |
| EP-95 | The Top-5% roadmap | Owner request: the owner's roadmap from their vault is the only roadmap, used by default and in the demo, with its tasks, subtasks, projects, goal, milestones, side projects, habits, and resources. | MUST | 8 pts | EP-14, EP-27, EP-41 | Done |
| EP-96 | A sky that follows the sun | Owner request: a colour theme that follows the sunlight and the moonlight in the person's time zone, and the home picture following the sun over a landscape of their choice (forest, ocean, earth view, monuments of each continent, desert, mountains). | SHOULD | 5 pts | EP-07, EP-38 | Done |
| EP-97 | The roadmap on the calendar | Owner request: the roadmap's schedule of which domain to work on each day, following the phases, on the calendar, and its projects created like the vault's. | SHOULD | 3 pts | EP-08, EP-95 | Done |
| EP-99 | A year you can read | Owner report: the 12-month activity chart did not show the current month and had no legend; the demo showed one active day. | SHOULD | 2 pts | EP-30 | Done |
| EP-98 | Every page its picture | Owner request: illustrations for each page, type of note, and section, with a short description (paintings, films, games), and a background for each Reflect page. | SHOULD | 3 pts | EP-38 | Done |
| EP-100 | Your vault's templates | Owner request: the SecondBrain vault's templates for each type of note, implemented in Axon. | SHOULD | 3 pts | EP-11, EP-48 | Done |
| EP-101 | Water | Owner request: a tracker of the water drunk, in health and food. | SHOULD | 3 pts | EP-18, EP-88 | Done |
| EP-102 | Patched third-party images | Owner request: check the project for active CVEs and fix them. Caddy and PostgreSQL images rebuilt with security fixes, scanned and shipped like Axon's own. | MUST | 2 pts | EP-02, EP-81 | Done |
| EP-103 | Practice log | Problems, wargame levels, and labs with honest attempts and spaced re-solving, measuring the roadmap's DSA and security proofs. | SHOULD | 5 pts | EP-52, EP-95 | Done |
| EP-104 | Certifications | The roadmap's nine exams and any other: status, exam date, cost, result, credential, and expiry, on the calendar. | SHOULD | 5 pts | EP-08, EP-95 | Done |
| EP-105 | Study hours against the plan | Focus time on a roadmap, by week, day, skill, and project, against the plan's hours and blocks. | SHOULD | 3 pts | EP-35, EP-97 | Done |
| EP-106 | Where the vault lives | Notes live in the database by default; a person can name a server and folder for the Markdown vault, pair it with a one-time code and one command, and an empty vault starts with its structure and templates. Owner request. | SHOULD | 5 pts | EP-36, EP-48 | Done |
| EP-107 | Digests everywhere | Daily, weekly, and monthly digests by email, Slack, Discord, Teams, Telegram, ntfy, or a webhook: what to do, what comes next, what to learn, how the period went, illustrated, with a quote for the day. Owner request. | SHOULD | 8 pts | EP-56 | Done |
| EP-108 | Answering the digest | Answer in Slack, Discord, or Telegram: "done 2", "I finished the passport", "add … tomorrow at 3pm", "snooze", "note", "undo"; Axon follows at once. Owner request. | SHOULD | 8 pts | EP-107 | Done |
| EP-109 | Sign in with other accounts | Google, Microsoft, Apple, Discord, Slack, or any OpenID Connect provider, linked by their owner; never matched by email. Owner request. | SHOULD | 8 pts | EP-04 | Done |
| EP-110 | Tech watch | Follow feeds, sites, Medium, Substack, YouTube, GitHub (by hand, CSV, OPML, FreshRSS) on a cron schedule, midnight by default; each story stored once; NLP labels; feed notes; health. Owner request. | SHOULD | 13 pts | EP-67 | Done |
| EP-111 | Tech watch analysis & knowledge graph | Concepts, NPMI links, communities, trends, and a knowledge graph of the news with hover summaries, apart from the Zettelkasten graph and bridged to it. Owner request. | SHOULD | 8 pts | EP-110 | Done |
| EP-112 | Tech watch exports & sharing | CSV, ODS, Markdown, feed notes, JSON, PDF by kind, source, topic, label, and period; read-only share links; "from Axon Project" on each. Owner request. | SHOULD | 5 pts | EP-110 | Done |
| EP-113 | Podcast, agent, digest template, scenery | A podcast of the news (off by default), the Tech watch agent, the watch in the digests with a template the person can change, and illustrations. Owner request. | SHOULD | 8 pts | EP-110, EP-107 | Done |
| EP-114 | Decision journal | Decisions with the outcome expected and its chance, reviewed on their day and scored (Brier score, reliability), so a person learns how far to trust their own forecasts. | SHOULD | 5 pts | EP-91 | Done |
| EP-115 | Search in the menu bar | A search field in the menu bar opens the command center, which finds content and every page of the menu. | SHOULD | 2 pts | EP-62 | Done |
| EP-116 | Checked items stay | A task or habit checked off stays where it is, crossed out, until the next day; unchecking puts it back. | SHOULD | 3 pts | EP-07, EP-22 | Done |
| EP-117 | Coffee and tea | Caffeine against EFSA's daily limit and what is left at bedtime, with the last coffee that keeps sleep intact (Gardiner et al., 2023). | SHOULD | 5 pts | EP-101 | Done |
| EP-118 | Huberman's protocols | Andrew Huberman's health habits with what to do, why, how strong the evidence is, their studies, cautions, and a habit in one click. | COULD | 3 pts | EP-22, EP-117 | Done |
| EP-119 | The collection | Cards to collect, raised from bronze to gold: a trophy per roadmap skill, a card per habit, and milestones for problems, focus, reviews, the streak, and decisions. | COULD | 5 pts | EP-15, EP-95 | Done |
| EP-120 | Invitations | The owner invites one person at a time with a link used once, even while registration is closed, optionally into the household. | SHOULD | 3 pts | EP-03, EP-37 | Done |
| EP-121 | Share as a picture | A picture of a level, a card of the collection, a goal, or a habit's run, drawn on the device, to share, download, copy, or post on X, LinkedIn, or Bluesky. | COULD | 3 pts | EP-119 | Done |
| EP-122 | A practical knowledge graph | Both graphs (the news and the notes) pan, zoom, fit, find a node, go full screen, and place labels without overlap. | SHOULD | 3 pts | EP-47, EP-111 | Done |
| EP-123 | YouTube in the tech watch | Videos' words from their captions, else the operator's Whisper server, in their analysis and graph; Spanish too; links without tracking. | SHOULD | 8 pts | EP-110, EP-111 | Done |
| EP-124 | Podcasts from Audiobookshelf | The person's Audiobookshelf, connected in Preferences, brings each podcast into the tech watch, each episode with its notes and, with Whisper, its words from their own copy. | SHOULD | 5 pts | EP-110, EP-123 | Done |
| EP-125 | A trailer and a gallery | An Apple-style trailer drawn frame by frame with a composed soundtrack, replacing the recorded tour, and a README gallery of today's Axon. | COULD | 3 pts | EP-84 | Done |
| EP-126 | The original and its source | Every news item links to its original (watch, listen, read) and to its source (the channel, the blog, the show), with the full clean link to copy. | SHOULD | 1 pt | EP-110, EP-123 | Done |
| EP-127 | Every new feature in sight | The menu names YouTube videos, Podcasts, the Collection, Coffee and tea, the Huberman protocols, and invitations; the tech watch opens on tabs by kind, with their counts; Today lists what is new. | MUST | 1 pt | EP-116 to EP-126 | Done |
| EP-128 | A shorter board | Each column of the GTD board and the kanban shows its first five cards, then "Show N more". | SHOULD | 1 pt | EP-07 | Done |
| EP-129 | One set of icons | One set of line icons in SVG replaces the emoji: the kinds of items (goal, skill, note, project…), the notes' side links, and the journal's scales. | SHOULD | 2 pts | EP-38 | Done |
| EP-130 | A journal without the past moods | The journal stops showing past moods and energy: no chart, no faces in the last 30 days, which stay listed to reopen. | SHOULD | 1 pt | EP-91 | Done |
| EP-131 | The life web | A radar ("spider web") of the day across food, water, sleep, activity, caffeine, and mood, scored from 0 to 100, with a level, its best day to beat, and last week behind it. | SHOULD | 2 pts | EP-18, EP-88, EP-117 | Done |
| EP-132 | A podcast with real voices | The podcast is spoken by a self-hosted neural voice (Kokoro, `AXON_TTS_URL`), alone or as two hosts in conversation, falling back to the device's voices. | SHOULD | 3 pts | EP-110 | Done |
| EP-133 | Send the digest from its page | The Digest page shows where it goes (email and each chat app) and sets them up there, with a test. | SHOULD | 1 pt | EP-107 | Done |
| EP-134 | Every cover | Each film, series, and book gets its cover: a closer match when the catalogue names it differently (Dune for Dune: Part One, the year agreeing), and the search for missing covers runs by itself. | MUST | 1 pt | EP-17 | Done |
| EP-135 | Media cards worth opening | Larger cards with the cover, the rating, the progress, and the next step at a glance, and a page per item. | SHOULD | 2 pts | EP-17 | Done |
| EP-136 | Notes for films, series, and books | Vault note templates for a film, a series, and a book, alongside literature, permanent, and resource notes, opened from the item. | SHOULD | 1 pt | EP-104, EP-17 | Done |
| EP-137 | A reading habit turns the pages | A habit linked to a book ("Read 20 pages") moves the book's page forward when checked, and the pages left go down. | SHOULD | 1 pt | EP-12, EP-17 | Done |
| EP-138 | Budget charts | A pie of spending by category for a period (a month by default), the same for income, and a year of income against spending by month. | SHOULD | 2 pts | EP-39 | Done |
| EP-139 | A clean start | A new install has no notes, only the templates and courses; a guide says how to set everything up with the agents in a few hours. | MUST | 1 pt | EP-28 | Done |
| EP-140 | The life web on Today | The e2e tests walk the life web; Today links to it. | SHOULD | 1 pt | EP-131 | Done |
| EP-141 | The Top-5% roadmap, version 2 | 35 domains in 18 phases from Monday 19 October 2026, a certification for every domain (43), the skills assessment with a domain each and retaken at every phase's end. | MUST | 5 pts | EP-95, EP-104 | Done |
| EP-142 | The plan and its milestones in sight | The roadmap's page shows the plan phase by phase; the library shows it before it is started; a roadmap started from an earlier version says so and starts the new one. | MUST | 3 pts | EP-95, EP-141 | Done |
| EP-143 | Notes drawn, in Axon or in the vault | Say where notes live (Axon only, or a vault kept in step) at the top of Notes; draw in a note, saved as an SVG the vault shows too. | MUST | 3 pts | EP-48, EP-51 | Done |
| EP-144 | YouTube and Podcasts their own pages | Each tab has its own scenery, title, and follow panel with suggestions for the roadmap; line icons on the tabs. | SHOULD | 2 pts | EP-122, EP-129 | Done |
| EP-145 | Your roadmap today | Today shows the plan's phase, today's study block, its project, and the next certification, with the next steps; sections new in a release show until turned off. | MUST | 2 pts | EP-63, EP-142 | Done |
| EP-146 | Database architecture and GitOps | Two more domains in the roadmap, its certifications, projects, and assessment. | MUST | 2 pts | EP-141 | Done |
| EP-147 | Budget charts where they are expected | The month's expense and income pies and the year's bars on the Budget page itself. | MUST | 1 pt | EP-138 | Planned |
| EP-148 | YouTube in a few words | Each video: 5 to 10 keywords and a 5-to-10-line summary of what it says, instead of its subtitles. | MUST | 2 pts | EP-122 | Planned |
| EP-149 | Video games | A media kind for video games, with covers, a page, and a note template. | SHOULD | 2 pts | EP-134, EP-136 | Planned |
| EP-150 | Maths in notes | LaTeX ($…$, $$…$$) rendered in notes, with a helper to write expressions. | SHOULD | 2 pts | EP-25 | Planned |
| EP-151 | A model per agent | Preferences choose the model for all agents or for each one, Jarvis included. | MUST | 2 pts | EP-22 | Planned |
| EP-152 | AI tokens, measured and compacted | Tokens used overall and per agent; tool results compacted before they reach the model, raw against sent. | SHOULD | 3 pts | EP-19, EP-151 | Planned |
| EP-153 | Jarvis | An opt-in floating button opens a side panel where Jarvis, the master agent, works only through Axon's agents, on the chosen provider or Ollama. | MUST | 8 pts | EP-19, EP-151 | Planned |
| EP-154 | The roadmap's resources as notes | The best resources of the active roadmap as resource notes linked to its projects, milestones, goal, and certifications, opening in a new tab. | MUST | 3 pts | EP-141 | Planned |
| EP-155 | A news digest | A digest of the most interesting general news from the FreshRSS server, apart from the tech watch. | SHOULD | 3 pts | EP-107 | Planned |
| EP-156 | The vault in step, verified | Notes in step with the vault set in Preferences, checked against ~/SecondBrain on srv-secondbrain. | MUST | 2 pts | EP-106, EP-143 | Planned |
| EP-157 | Monitoring of every feature | Per-feature CPU, memory, storage, and request metrics, with dashboards and alerts on srv-monitoring (fleet repository). | SHOULD | 5 pts | EP-21 | Planned |
| EP-158 | A banner and the hardware needed | A README banner, and srv-prod's specification as the example requirements. | SHOULD | 1 pt | — | Planned |
| EP-159 | The trailer, again | The trailer recorded from the current demo. | COULD | 1 pt | EP-158 | Planned |
| EP-160 | Clean, documented, and safe | Refactoring, technical and functional documentation, no critical active CVE, resource use checked. | MUST | 5 pts | — | Planned |
| EP-161 | Everything checked | Every feature walked through in the demo and fixed where it fails. | MUST | 3 pts | EP-146..EP-160 | Planned |
| EP-162 | Version 1.0 | The release prepared for the owner's merge and tag, then deployed to srv-prod. | MUST | 1 pt | EP-161 | Planned |
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
| 1.1 — Owner requests | EP-47 → EP-48 → EP-49 → EP-50 → EP-51 → EP-52 → EP-53 → EP-54 → … → EP-62 → EP-63 → … → EP-68 → EP-69 → EP-70 → … → EP-105 → EP-106 → EP-107 → EP-108 → EP-109 → EP-110 → EP-111 → EP-112 → EP-113 → EP-114 → EP-115 → EP-116 → EP-117 → EP-118 → EP-119 → EP-120 → EP-121 → EP-122 → EP-123 → EP-124 → EP-125 → EP-126 → EP-127 → EP-128 → EP-129 → EP-130 → EP-131 → EP-134 → EP-135 → EP-136 → EP-137 → EP-138 → EP-139 → EP-132 → EP-133 → EP-140 → EP-141 → EP-142 → EP-143 → EP-144 → EP-145 → EP-146 → … → EP-162 | Zettelkasten graph, everything in the vault, audit, diagrams, pictures in notes, the learning and memory engine, agents on every page, the twelve proposed features |

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
- [x] **EP-31/IS-31.06 — French for every page** · 8 pts · SHOULD
  - [x] Shell, landing, sign-in, demo banner, Today and the command dashboard, account tabs, and Preferences are in French.
  - [x] Every other page, dialog, and component (EP-85, EP-86), checked by tests: no text outside `t()` in any template, and a French string for every key the code and Axon's own tables use.
  - [x] Text the API writes (problem details, validation messages, dashboard and insight explanations, goal progress, privacy purposes, badges, media suggestions, agent introductions) follows the language too: EP-87. The demo's sample data and the agents' tool descriptions, written for the model, stay English.

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
- [x] **EP-37/IS-37.04 — Embeddings** · 8 pts · COULD — delivered by EP-62: each person may choose an embedding service behind the `ai_embeddings` consent; vectors in pgvector. Without it, similarity stays local TF-IDF cosine.
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

### EP-49 — Audit: fixes and legacy removal

**Goal.** The owner asked to analyse everything, fix issues, and remove useless code and legacy (2026-10-12). Tools: `deadcode`, golangci-lint with extra linters, `knip`, `go mod tidy`, and reading the code paths around the newest features.

**Fixed**

- [x] Item notes from the vault mirror counted as notes written: Today's totals and recent notes, dashboard KPIs and charts, Insights, progress points (3 points per task, as a "note"), and the knowledge explorer. They are now left out everywhere they skewed numbers; the Notes list hides them too (`?mirrored=false`), while the `Axon` folder stays in the sidebar.
- [x] Vault mirror: a `## ` heading inside a task's notes, a description, or a review split the section when read back, losing text; only Axon's own section headings start sections now.
- [x] `TMDB_API_KEY`, `USDA_API_KEY`, and `AXON_CONTACT` were never passed to the API container, so setting them in `.env` did nothing; they are now.
- [x] No favicon: browsers showed a blank tab icon and requested `/favicon.ico` on every visit.

**Removed**

- [x] Valkey: the container ran on every host but no code ever used it; `REDIS_URL` with it ([ADR 0022](adr/0022-no-valkey-yet.md)).
- [x] Settings documented but never read: `AXON_ENV`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `S3_BACKUP_BUCKET`.
- [x] `agents.Runner.Wait` and its wait group (never called; interrupted runs are already cleaned up on the next start), two needless conversions, a needless wrapper.
- [x] Legacy documentation: deployment diagrams and the hardware table showing `srv-dev` and `srv-db` hosts that ADR 0003/0004 replaced; queues and caches that did not exist.

**Also**

- [x] Dependabot's `fxamacker/cbor` 2.9.4 → 2.9.6, and `go mod tidy` (snowball is a direct dependency).

**Checked and kept.** Functions only tests reach are deliberate references (the balance property test, FSRS vectors, NLP helpers); no unused web files; `swagger-ui-dist` is used by the build script.

### EP-50 — Diagrams GitHub can display

**Goal.** The owner could not see the UML diagrams: GitHub draws Mermaid but shows PlantUML as raw text (2026-10-12).

- [x] **EP-50/IS-50.01** · 2 pts · MUST — the seven PlantUML diagrams and the two older Mermaid ones become eight Mermaid diagrams in both READMEs: system context, containers, deployment, data model and universal links, learning and memory loop, assessment and Tutor roadmaps, vault mirror both ways, daily streak and freezes; each rendered with the Mermaid CLI before committing.

### EP-51 — Pictures in notes

**Goal.** "Ensure I can put pictures within notes" (owner, 2026-10-12), and see them in Axon and in the vault alike.

- [x] **EP-51/IS-51.01** · 3 pts · MUST — files notes embed: `note_files` keyed by vault path (PNG, JPEG, GIF, WebP, SVG, PDF; 10 MB each, 500 MB per person) with tombstones. `POST /v1/notes/files` stores an editor upload under `Attachments/<name>-<hash>.<ext>`; PNG and JPEG are re-encoded (no EXIF or location) and scaled to 2400 px. `GET /v1/notes/file` serves with `nosniff` and a sandboxing CSP, PDFs as downloads. Bytes are checked against the extension.
- [x] **EP-51/IS-51.02** · 2 pts · MUST — the editor takes pictures by paste, drop, or the 🖼 button and embeds them as `![name](Attachments/…)`; the preview and reading view render `![alt](path)`, `![alt|300](path)`, and Obsidian `![[file.png|300]]`, resolved like Obsidian (exact path, then relative to the note, then by name). External pictures become links (the page loads only its own); missing files say so.
- [x] **EP-51/IS-51.03** · 3 pts · MUST — `axon notes-sync` carries attachments both ways (`GET /v1/notes/files`, `PUT`/`DELETE /v1/notes/file` with `base_hash`), byte for byte, with the same conflict rule as notes (`<name>.axon-conflict.<ext>`); the watch wakes on them; the ZIP import and the account export include them.

**Tests.** `TestNoteFiles` (upload, re-encoding, sandbox headers, ownership, path escapes, type checks, vault sync both ways, conflicts, deletions, export); render tests for embeds and alt-text escaping; e2e "paste a picture into a note".

### EP-52 — Learning and memory engine

**Goal.** "Ensure the learning/memorize/note-taking processes are perfect … automated … and follow the principles of Understanding How We Learn and Moonwalking with Einstein", with "illustrated flash cards generated by an AI agent" (owner, 2026-10-12). Decisions in [ADR 0023](adr/0023-learning-engine.md).

- [x] **EP-52/IS-52.01** · 5 pts · MUST — spaced, paced, interleaved reviews: known cards due, then new cards up to `learning.new_per_day` (20); decks take turns in a review of everything (`learning.interleave`); FSRS aims at `learning.retention` (90%); a deck's due count and Today's "cards to review" say what a review would offer. Settings → Learning.
- [x] **EP-52/IS-52.02** · 3 pts · MUST — retrieval first: write the answer before it is shown (`learning.type_answer`, Ctrl+Enter reveals); after it, the why, a concrete example, and a vivid memory image; leeches (6+ lapses) are named with what to do; "Practise weak cards" drills leeches and the cards closest to being forgotten until recalled, without touching the schedule.
- [x] **EP-52/IS-52.03** · 3 pts · MUST — dual coding by AI: the Teacher proposes cards with elaboration, example, mnemonic, and an SVG drawing, and illustrates existing cards (`list_cards`, `illustrate_cards`), always after approval; SVG is rebuilt from an allowlist and served sandboxed; drawings go to Anki as `.svg` media.
- [x] **EP-52/IS-52.04** · 3 pts · MUST — free recall of literature and permanent notes on a blank page, compared locally with the note's key ideas (stems and synonyms), scheduled by expanding intervals, with history; Learn → Today lists notes due and suggests new ones a day after writing.
- [x] **EP-52/IS-52.05** · 3 pts · SHOULD — memory palaces: ordered places, cards placed at them with a vivid image, and a walk that asks what is at each place before showing it.
- [x] **EP-52/IS-52.06** · 2 pts · MUST — note-taking pipeline: Make cards drafts from `question :: answer`, `Q:`/`A:`, `==cloze==`, and `**Term**: definition`; fleeting notes promoted to literature or permanent in one click; Notes filter by type (`?kind=`); Learn → Today counts fleeting notes older than two days.
- [x] **EP-52/IS-52.07** · 2 pts · MUST — Learn → Today: cards due with minutes from the person's pace, retention against the aim, leeches, notes to recall, weekly mixed test (results recorded), fleeting notes, palaces, streak, and the principles explained where they are used. The demo shows an illustrated card, a palace, and a note due for recall.

**Tests.** `TestLearningEngineCards` (pacing, interleaving, enriched cards, palaces with ownership and shrinking, leeches, practice), `TestLearningEngineNotes` (recall scoring and intervals, drafts, plan, tests, export), `illustrate_cards` in the every-write-tool approval test, SVG sanitiser unit test; e2e "learn: cards from a note, free recall, a memory palace walk, and the day plan".

### EP-53 — Agents on every page

**Goal.** "Why not dispatch the agents around each page following their skills?" (owner, 2026-10-12).

- [x] **EP-53/IS-53.01** · 3 pts · SHOULD — an agent panel on Today, Tasks, Goals, Habits, Notes, a note, Learn, Flashcards, Roadmaps, Media, Health, and Projects: the page's agents first (one map, `pageAgents.ts`), ready-made requests from each skill, the page's context (a note's title, id, type, and path) sent with the request, the reply, follow-up questions, and approve or reject in place; when agents cannot run, it says why and links to their setup.
- [x] **EP-53/IS-53.02** · 2 pts · SHOULD — other pages hand a request to the Agents page (`/agents?agent=…&ask=…`), as Flashcards and Make cards do for the Teacher.

**Tests.** A unit test checks that every page names existing agents and suggests only for them; e2e "each page offers its own agents, and says when none can run yet".

### EP-54 — Navigation groups

**Goal.** The first of the twelve features proposed in the EP-49 audit and approved by the owner ("do all the functionalities you have found", 2026-10-12): eighteen links in one row were hard to scan.

- [x] **EP-54/IS-54.01** · 2 pts · SHOULD — Today, then Do (Tasks, Projects, Calendar, Focus), Think (Notes, Areas, Resources, Archive), Grow (Learn, Goals, Habits), Life (Health, Media, Budget), and Reflect (Insights, Agents) as disclosure menus; the current group is marked; menus close on navigation, Escape, or a click elsewhere. Under 760 px, one Menu button opens every group in a sheet. French labels included.

**Tests.** e2e "the menu groups pages: Do, Think, Grow, Life, Reflect" (disclosure state, Escape, the current section, the phone sheet, accessibility).

### EP-55 — Installable app, offline capture

**Goal.** Proposed in the EP-49 audit and approved by the owner: open Axon like an app, and never lose a thought for lack of a connection. Reminders (EP-56) build on its service worker.

- [x] **EP-55/IS-55.01** · 1 pt · SHOULD — a web app manifest (start on Today, standalone, shortcuts to Today, flashcard review, and Notes), PNG and maskable icons from the logo, theme colour, Apple touch icon; Preferences → App installs it where the browser offers to, and says how elsewhere.
- [x] **EP-55/IS-55.02** · 2 pts · SHOULD — a service worker keeps the app's code and a prerendered offline page; nothing personal is cached (the API always goes to the network). Offline, Capture keeps a task or a fleeting note on the device (localStorage) and the offline page lists what waits; back online, every page sends them in order, once: each carries an Idempotency-Key and tabs take turns (Web Locks), so two tabs cannot create it twice.

**Tests.** e2e "installable, and a capture made offline is kept on the device and sent later" (manifest, service worker, offline capture, offline page, sent once on reconnect).

### EP-56 — Reminders

**Goal.** Proposed in the EP-49 audit and approved by the owner: Axon says what today holds without being opened. Decisions in [ADR 0024](adr/0024-web-push-in-house.md).

- [x] **EP-56/IS-56.01** · 3 pts · SHOULD — Web Push with the standard library: RFC 8291 encryption (checked against the RFC's example) and RFC 8292 VAPID signing with an instance key made on first use and sealed when `AXON_ENCRYPTION_KEY` is set; subscriptions only to browser push services; dropped ones forgotten; the service worker shows them and opens the right page.
- [x] **EP-56/IS-56.02** · 2 pts · SHOULD — the morning digest (tasks and overdue ones, cards with minutes, notes to recall, habits, the weekly test) and the evening nudge (habits still open with their two-minute version, cards still due) at the person's local times, once a day (`reminders_sent`), skipped when more than three hours late or when there is nothing to say; Preferences → Reminders turns them on, sets the times, adds this device, removes devices, sends a test, and an email address when the instance has `AXON_SMTP_URL`/`AXON_SMTP_FROM`. The shared demo sends nothing.

**Tests.** RFC 8291 vector and VAPID token tests; `TestReminders` with a fake TLS push service that decrypts as a browser (off until enabled, once a day at local time, the digest and the nudge, a dropped device forgotten, the endpoint allowlist, ownership, the test button, settings validation); mailer composition and header-injection test; e2e "reminders: times saved, and a test says when no device can receive it".

### EP-57 — Quick add for anything

**Goal.** Proposed in the EP-49 audit and approved by the owner: capture anything from anywhere (press n) in one line.

- [x] **EP-57/IS-57.01** · 3 pts · SHOULD — a parser (`quickAdd.ts`): a task by default, with `@context`, a day at the end (today, tomorrow, a weekday, in N days, next week, a date; French too), and `someday`; prefixes `note:` (tags from `#tags`), `book:`, `film:`, `series:`, `habit:`, `goal:` (`by <date>`), `project:`, `card: question :: answer #Deck`, in English and French. Capture shows what it will add as you type and names its button after it; offline, a note waits as a note and anything else as a task. Words that are also names ("Call Sam") are not read as days.

**Tests.** Parser unit tests (dates relative to a fixed day, prefixes, tags, goals, cards, words that are not days); e2e "quick add: one line becomes a task with a context and a day, a book, a habit, or a card".

### EP-58 — Calendars in

**Goal.** Proposed in the EP-49 audit and approved by the owner: see the rest of life's calendar in Axon. (Pictures and PDFs in notes, also proposed, were delivered by EP-51.)

- [x] **EP-58/IS-58.01** · 3 pts · SHOULD — an iCalendar reader: unfolded lines, escaped text, all-day and timed events, UTC, TZID (IANA and Outlook's Windows names) and floating times, DTEND or DURATION, cancelled events, EXDATE, moved occurrences (RECURRENCE-ID), and RRULE expansion (daily, weekly with BYDAY, monthly by day or by "2nd Tuesday"/"last Thursday", yearly; INTERVAL, COUNT, UNTIL), at the same wall-clock time across DST.
- [x] **EP-58/IS-58.02** · 2 pts · SHOULD — Calendar → Calendars you follow: name, colour, and an https:// or webcal:// address, fetched at once (a wrong address is said right away) and every 30 minutes; events kept from 60 days back to a year ahead; the secret address sealed with `AXON_ENCRYPTION_KEY` and shown only as its host; private network addresses refused at connection time (whatever the name resolves to) unless the operator sets `AXON_CALENDAR_PRIVATE=1`; up to 20 calendars, 5 MB each; shown in the month grid and the agenda with their colour, can be hidden, in the export (without addresses).

**Tests.** ICS and RRULE unit tests (a Google-style weekly stand-up with an exception and a moved occurrence across the DST change, all-day ranges, durations, cancellations, "last Thursday", COUNT over short months, INTERVAL); `TestFollowedCalendars` (private and plain addresses refused, not a calendar, missing, events by date and multi-day ranges, the secret address never returned, refresh and kept failures, ownership, unfollow); e2e "follow another calendar by its ICS address and see its events beside your own" (Axon's own feed, through Caddy).

### EP-59 — Guided weekly review

**Goal.** Proposed in the EP-49 audit and approved by the owner: the weekly review of Getting Things Done, guided, with Axon's insights in it.

- [x] **EP-59/IS-59.01** · 3 pts · SHOULD — Reflect → Weekly review, ten steps: get clear (empty your head with quick add; the task inbox to zero with next, waiting, someday, done, delete; fleeting notes to process), get current (the week's highlights from Insights; the next two weeks from Axon and followed calendars; waiting for; projects without a next action; goals, habits, and learning), get creative (someday/maybe), and into next week (what went well, what to change, what matters most, saved as the weekly check-in). From Friday to Sunday the morning digest suggests it until that check-in exists.

**Tests.** `TestReminders` (the Saturday digest suggests the review, and stops once it is done); e2e "the weekly review walks from an empty head to next week".

### EP-60 — Imports

**Goal.** Proposed in the EP-49 audit and approved by the owner: move to Axon without retyping.

- [x] **EP-60/IS-60.01** · 3 pts · SHOULD — parsers, each a pure function: Todoist (CSV; @labels as contexts, ISO dates as due dates, spoken ones kept in the notes), TickTick (CSV backup with its preamble; status, first tag as context), Goodreads (shelves as statuses, stars ×2, read dates), Letterboxd (the export ZIP or one CSV: watched, ratings ×2, diary dates, watchlist), Firefly III (withdrawals and deposits; transfers left out), OFX/QFX bank statements (SGML or XML), Notion (ZIP of Markdown and pictures: ids dropped from names, page links as [[wikilinks]], pictures moved with their pages).
- [x] **EP-60/IS-60.02** · 2 pts · SHOULD — `POST /v1/import/{source}` (50 MB) with `dry_run` and, for money, `account_id`: each item in its own savepoint so a refused one does not undo the rest; nothing twice (tasks by title and due date, media by kind, title, and year, transactions by the budget's import identity, notes and pictures by path; a note with other content is left alone and counted). The budget's CSV import now shares `budget.ImportRows`. Account → Import: how to export from each app, a preview, then the import.

**Tests.** Parser unit tests for every source; `TestImports` (dry run, import, nothing twice for tasks, books, and money, Notion paths and links, ownership, a wrong file, an unknown source); e2e "import from another app: preview, import, and nothing twice".

### EP-61 — Household sharing

**Goal.** Proposed in the EP-49 audit and approved by the owner: share the lists of a home without sharing a life. Decisions in [ADR 0025](adr/0025-household-shares-lists-only.md).

- [x] **EP-61/IS-61.01** · 3 pts · SHOULD — households, members, one-time invite codes (hashed, a week), lists (up to 50) and items (up to 500 open per list) with a due date and an assignee among members; one household per person; leave at any time (the last one out deletes it, an owner hands it on); the owner removes members, whose items are unassigned; done items stay a week so a tick can be undone.
- [x] **EP-61/IS-61.02** · 2 pts · SHOULD — Life → Household: start or join, members, invite code shown once, shared lists with ticks and assignees, refreshed every 15 seconds while open; the morning digest counts household items given to you; the export holds your items, not other members' names.

**Tests.** `TestHousehold` (codes used once and refused when wrong, members, shared lists, assignees limited to members, outsiders refused, ticks, nothing personal shared, owner-only removal, hand-over on leaving, the last one out); e2e "a household: start it, invite with a code, share a list, tick an item, and leave".

### EP-62 — Search by meaning

**Goal.** Proposed in the EP-49 audit (IS-37.04, left open) and approved by the owner. Decisions in [ADR 0026](adr/0026-embeddings-are-opt-in.md).

- [x] **EP-62/IS-62.01** · 3 pts · COULD — per-person embedding service (local OpenAI-style server, OpenAI, Voyage AI; model defaults nomic-embed-text, text-embedding-3-small, voyage-3.5-lite; key sealed and never returned), the `ai_embeddings` consent (withdrawing deletes the vectors), and a background indexer: new and changed notes by content hash, 32 per request, 256 per minute per person, mirror notes left out, failures shown; vectors in pgvector, one space per service and model.
- [x] **EP-62/IS-62.02** · 2 pts · COULD — `GET /v1/notes/semantic?q=` and `GET /v1/notes/{id}/similar` by cosine distance; the command center adds notes close in meaning (not already found by words); notes show "Close in meaning"; Preferences → Search by meaning sets the service, the consent, and shows progress.

**Tests.** `TestSearchByMeaning` with a fake service that embeds by concept (off until set up, keys refused or kept, nothing sent without consent, unchanged notes not re-embedded, "my car" finds the note about an automobile, similar notes, other people's notes never returned, withdrawal deletes the vectors, off forgets the service); e2e "search by meaning is off until a service and the consent are given".

### EP-63 — Customisable Today

**Goal.** Proposed in the EP-49 audit and approved by the owner: a Today that shows what each person looks at first.

- [x] **EP-63/IS-63.01** · 2 pts · COULD — `today.sections` in the settings (known sections, each once): counts at the top and progress at the bottom when shown; the cards between (today's tasks, coming up, goals, recent notes, and new ones: learning today, habits today, calendar today from followed calendars, household items for you) in the chosen order, each loading its data only when shown; Customise Today chooses and orders them.

**Tests.** `TestTodaySections` (saved in order, repeats and unknown sections refused); e2e "Today can be customised: cards chosen and ordered".

### EP-64 — Bulk actions and dragging onto the calendar

**Goal.** The last of the twelve features proposed in the EP-49 audit and approved by the owner.

- [x] **EP-64/IS-64.01** · 2 pts · COULD — `POST /v1/tasks/bulk`: up to 200 tasks changed (status, context, scheduled or due date, recurrence, notes; titles one at a time) or deleted in one transaction, all or none; Tasks → Select, a bar to move them to Next, Waiting, Someday, or Done, set a context or a due date, or delete them.
- [x] **EP-64/IS-64.02** · 1 pt · COULD — Calendar: drag a task to another day (a due date stays a due date, a scheduled one stays scheduled); "Next actions without a date" can be dragged onto a day, or given a date in their field, for the keyboard and for touch screens.

**Tests.** `TestBulkTasks` (status, context, and date at once, all or none with another person's task, titles refused, either delete or a change, dates cleared, delete); e2e "bulk actions change several tasks at once" and "drag a task onto a day of the calendar, or pick its date".

### EP-65 — Audit 2: fixes

**Goal.** "Improve and fix everything" (owner, 2026-10-12): a second audit, over what EP-50 to EP-64 added.

**Fixed**

- [x] A vault template of an item type (task, goal, habit, film…) was created under `Axon/` even when the vault mirror's folder is set to another one, so it never became an item; it now follows Preferences → Markdown vault.
- [x] The learning engine counted days in UTC: near midnight a note's next recall, the notes due today, and the learning streak were off by a day for anyone far from UTC. They now follow the person's time zone (test in UTC+14).
- [x] Letterboxd and Notion imports trusted the sizes a ZIP declares; what is actually unpacked is now capped (100 MB and 200 MB).
- [x] Inputs, selects, text areas, and file pickers outside the shared text field were unstyled browser controls (Import, the weekly review, Calendar, Household): one base style now covers them, with a visible focus ring.

**Removed**

- [x] Unused exports found by knip (`pictureTypes`, `fileURL`, two template helpers that only the module uses).

**Checked and kept.** `deadcode` lists only the test-only helpers EP-49 kept on purpose; the API docs script and swagger-ui are used by the build.

### EP-66 — Trash and undo

**Goal.** "And other functionalities which could be essential" (owner, 2026-10-12): a deletion was final; a life OS must forgive a wrong click. Decisions in [ADR 0027](adr/0027-trash-keeps-rows.md).

- [x] **EP-66/IS-66.01** · 3 pts · MUST — `graph.DeleteEntity` saves the rows a deletion removes (entity, domain row, cascading rows two levels down found from the foreign keys, child entities, links) in `trash`, by transaction batch; `Restore` puts a batch back parents first, skipping links whose other end is gone and refusing (with no change) when a path or name was taken; restored notes leave the sync's tombstones and are newer than every cursor; a `created` event tells the mirror; the mirror's own deletions are not trashed; 30 days, purged daily; in the export.
- [x] **EP-66/IS-66.02** · 2 pts · MUST — `GET /v1/trash`, `POST /v1/trash/undo` (an entity's latest deletion), `POST /v1/trash/{id}/restore`, `DELETE /v1/trash/{id}`, `DELETE /v1/trash`; toasts with **Undo** after deleting a note, task (one or in bulk), goal, milestone, habit, card, film, series or book, project, or memory palace; Account → Trash.

**Tests.** `TestTrash` (undo of a note with its links and the sync seeing it again, a goal with its milestones as one deletion, a habit's check-ins and a card's reviews restored, a taken path refused with nothing half-restored, a bulk deletion as one line, ownership, forget, empty); the full suite with the hook on every deletion; e2e "a deleted note can be undone, and restored from the trash".

### EP-67 — Share to Axon

**Goal.** Capture where reading happens: from any app on a phone, and any page on a computer.

- [x] **EP-67/IS-67.01** · 1 pt · SHOULD — the manifest's `share_target`: the installed app appears in the share sheet and opens `/share` with the title, text, and link (a link inside the text is taken out); a "Save to Axon" bookmarklet sends the page's title, address, and selected text there; sign-in keeps the whole address, so a share survives it. Save as a fleeting note or a task, offline too.
- [x] **EP-67/IS-67.02** · 2 pts · SHOULD — `POST /v1/clip`: the page's readable part (its article, main, or body without scripts, menus, headers, footers, forms) as Markdown with headings, lists, quotes, code, and absolute links; HTML only, 5 MB, through `netsafe`, the private-network-safe client now shared with followed calendars. "Save the page's text" makes a literature note with source, site, and date, opened for annotation.
- [x] Fixed on the way: a capture waiting offline was dropped when another tab was still creating it (the server's "still being processed" answer); it now waits and is sent once.

**Tests.** Clipper unit test (what is kept and left out); `TestClip` (private and non-HTTP addresses refused, a page clipped, a PDF refused); followed calendars still pass on `netsafe`; e2e "share to Axon: a shared link becomes a note, or the page's text a literature note".

### EP-68 — Task reminders at a time

**Goal.** Tasks had days, not times: "remind me at 3 pm" is a to-do list's most basic promise.

- [x] **EP-68/IS-68.01** · 1 pt · SHOULD — `tasks.remind_at` and `reminded_at`: the reminders worker sends "Reminder: <task>" at that time, once, to the person's devices and email, even with the daily reminders off (this one was asked for); a new time means a new reminder; one more than 12 hours late is marked, not sent; a repeating task's next occurrence is reminded at the same time of its day; the notification opens the task.
- [x] **EP-68/IS-68.02** · 1 pt · SHOULD — "Remind me at" in the task editor; quick add reads "at 15:00", "at 3pm", "à 18h30" (a time already gone today, without a day said, means tomorrow); tasks show 🔔 and the time.

**Tests.** `TestReminders` (a task's reminder sent once with the daily reminders off, a new time reminds again, a two-day-old one is not sent); quick-add unit tests for times; e2e "a task can carry a reminder at a time, set by quick add or in the task".

### EP-69 — Clean code and the person's day

**Goal.** The owner asked to fix and refactor everything, with clean code, tests first, and documentation. The audit looked for code written more than once, since each copy can go wrong on its own, and for logic no test reached.

- [x] **EP-69/IS-69.01** · 1 pt · MUST — API `clock` package: the person's time zone and "today" were looked up in twelve places; now one tested function. Progress and focus passed a zone name Go had refused straight to PostgreSQL; every caller now passes the zone Go resolved.
- [x] **EP-69/IS-69.02** · 1 pt · MUST — Text is always UTF-8 before PostgreSQL sees it (`textx.Cut`): a user agent cut inside an accent, or not UTF-8 at all, failed sign-in with a 500. Imports read Windows-1252 exports as such (accents survive), the clipper decodes a page's declared encoding, and a calendar with a bad byte still loads.
- [x] **EP-69/IS-69.03** · 1 pt · MUST — New accounts start in the browser's time zone instead of UTC; before, "today" (tasks, habits, reminders, Today) was yesterday's for part of every night outside UTC.
- [x] **EP-69/IS-69.04** · 1 pt · MUST — Web `lib/dates` replaces six copies of date arithmetic and the `'sv'` locale trick. Fixed on the way: the recall history showed the next date a day early west of UTC; workouts dated after the account's today (a device a day ahead) vanished from the list. Calendar events over several days (`spreadEvents`) and the editor's text changes (`lib/markdownEdit`) moved out of their pages into tested modules; a list numbered "1)" now continues "2)", not "2.".
- [x] **EP-69/IS-69.05** · 1 pt · SHOULD — Tests where none reached: account balances from SQL checked against the property-tested model over random movements; vault edits reaching habits and films, and workouts as notes; a Notion export's pictures. `CONTRIBUTING.md` lists the shared helpers to use.

**Kept as they are.** The 150 `catch` blocks that show an error toast: a helper would save a line each and hide the error path, against "simple, explicit code".

**Tests.** `clock`, `textx`, `dates`, `markdownEdit`, `spreadEvents`, and encoding unit tests, each written before its code; `TestSignInWithAnyUserAgent` (failed with a 500 before the fix), `TestRegistrationTakesTheBrowsersTimeZone`, `TestAccountBalancesMatchTheModel` (fails when the SQL is broken on purpose), `TestVaultEditsReachHabitsMediaAndWorkouts`, a workout dated tomorrow in `TestHealth…`, Notion pictures in `TestImports`. The e2e suite now runs the browser at UTC+14 or UTC−11, whichever has a different date from UTC's at the time; it found the workout and calendar test bugs.

### EP-70 — Every foreign key indexed

**Goal.** Measured, not guessed: a catalogue query found twenty foreign keys without an index starting with their columns. Deleting a parent row (an erasure, a deleted note or category) or looking up by the key then scans the whole child table, for every user. Two are on hot paths: each calendar refresh deletes its events by `source_id` every 30 minutes, and the recall queue reads each note's latest attempt by `note_id`.

- [x] **EP-70/IS-70.01** · 1 pt · SHOULD — Migration `00046` indexes all twenty; `note_recalls (note_id, recalled_at DESC)` serves the recall queue's lateral lookup without a sort. `TestEveryForeignKeyIsIndexed` fails for any foreign key added later without an index.

**Tests.** `TestEveryForeignKeyIsIndexed` (red with the twenty listed, then green); `TestMigrateUpDown`; `EXPLAIN` of the recall queue and the calendar refresh delete now shows index scans.

### EP-71 — Scripts checked

**Goal.** The deploy, backup, and check scripts run as root on servers and touch backups; nothing linted them.

- [x] **EP-71/IS-71.01** · 1 pt · SHOULD — `make check-scripts` (ShellCheck 0.11.0, pinned) and a CI step in the small `spec` job lint every tracked `*.sh` at warning level. All pass; the remaining notes are intentional client-side expansions in `deploy.sh`. The same audit ran govulncheck (no reachable vulnerability) and `npm audit --omit=dev` (none).

**Tests.** The gate reports SC2115 on `rm -rf $DIR/*`; `actionlint` passes on `ci.yml`.

### EP-72 — Lighter offline install

**Goal.** Measured: the app's own code is small (no page over 14 kB gzipped), but the service worker kept every static file for offline use, including Swagger UI for the API reference (1.8 MB of about 3.5 MB). Each update changes the cache name, so every installed phone downloaded it again after each deploy.

- [x] **EP-72/IS-72.01** · 1 pt · SHOULD — `lib/offlineShell` leaves `/api-docs/` out of the offline shell; the API reference still works online.

**Tests.** `offlineShell` unit tests; the e2e "installable…" test reads the browser's caches: the offline page is there, nothing under `/api-docs/`.

### EP-73 — No query per item

**Goal.** Measured: the test pool counts queries, and each list is read with 2 items, then 12. Two lists ran a query per item (an "N+1"), so they slowed as data grew: Goals (6 queries for 2 goals, 26 for 12: milestones and progress per goal) and Today (one per task due). Today also lists active goals, so it paid both.

- [x] **EP-73/IS-73.01** · 1 pt · SHOULD — `goals.List` reads the milestones and the supporting tasks of all listed goals in one query each, counted as `Explain` counts them (a task supporting a goal and its milestone counts once).
- [x] **EP-73/IS-73.02** · 1 pt · SHOULD — `tasks.DueBy` returns Today's tasks in one query; the rule for what is due today lives in the tasks package.

**Tests.** `TestListsDoNotQueryPerItem` over 13 lists (tasks, notes, habits, goals, projects, areas, media, Today, calendar, palaces, budget accounts and transactions, flashcards): red for Goals and Today, then green. `TestGoalProgressIsExplainedByItsInputs` now checks the listed goal against the goal itself, with a task supporting both a goal and its milestone.

### EP-74 — Compressed responses

**Goal.** Measured: no response was compressed, not by Caddy and not by the web server; scripts compress about 3×, JSON and YAML more.

- [x] **EP-74/IS-74.01** · 1 pt · SHOULD — `encode zstd gzip` in the Caddyfile for HTML, CSS, JavaScript, JSON (problem and manifest too), YAML, SVG, and text. The OpenAPI document goes from 324 kB to 48 kB. No response streams; sessions travel in cookies (headers), which are not compressed. Responses under 512 bytes stay as they are.

**Tests.** e2e "pages, scripts, and the API are sent compressed" (red before the change) and "API answers are sent compressed".

### EP-75 — Today and the page title in every browser

**Goal.** Owner report with a screenshot of staging in Firefox: Today never left "Loading your day…", and "Today · Axon" showed as text under it. A probe of staging in Firefox and Chromium found both causes.

- [x] **EP-75/IS-75.01** · 1 pt · MUST — The page title text is SvelteKit's announcer for screen readers, hidden by an inline style that the Content Security Policy (`style-src 'self'`) blocks in Firefox. The stylesheet now hides it (it is still read out); `app.html`'s inline style and the calendar's colour styles moved to the stylesheet and `style:` directives; a unit test fails on any `style="…"` left in the markup.
- [x] **EP-75/IS-75.02** · 1 pt · MUST — Today threw "Cannot read properties of null (reading 'includes')": the staging demo's settings had `today.sections: null`, and the page used it as a list. Settings now read a missing list as the default sections (an empty list stays a choice), and Today keeps its defaults whatever it is sent.

**Tests.** e2e "the page title announced to screen readers is not shown on the page" (removes the inline style, as Firefox does; red before the fix); `TestTodaySectionsAreNeverNull` (red without the fix); `csp.test.ts`; a Firefox and Chromium probe of the same state locally: Today loads, the announcer is 1×1 px.

### EP-76 — Covers for every film, series, and book

**Goal.** Owner request: "pictures of the books, TV series, movies, and a better UI/UX for the card of a media". Covers came only with items added from a catalogue search; items added by hand, imported (Goodreads, Letterboxd), or from the demo had none, and the card was a small thumbnail beside text.

- [x] **EP-76/IS-76.01** · 1 pt · SHOULD — `POST /v1/media/covers/find`: up to 20 items without a cover, oldest first, are looked up in Open Library (books) and TMDB (films and series, when a key is set), and get the cover of the same work: same title (case, accents, punctuation aside), and the same author or year when both sides know one. A wrong cover is worse than none. It needs the `external_metadata` consent, spends the lookup quota, and reuses cached lookups; covers are copied into Axon as before, so pages load no outside image.
- [x] **EP-76/IS-76.02** · 1 pt · SHOULD — The shelf is a grid of poster cards: the cover fills the card, the rating is a badge on it, and the title, kind, year, and creators sit below, with "210 / 611 pages" under the progress bar. An item without a cover gets one made from its title, in a colour of its own (`coverHue`). "Find missing covers" appears when lookups are allowed and some item has none. French included.

**Tests.** `TestBestMatchTakesOnlyTheSameWork` (written first); `TestFindMissingCovers` against the fake catalogue (refused without consent; Dune by Frank Herbert gets its cover, Dune by someone else and a film without TMDB do not; a second run uses the cache); `coverHue` unit test; the media e2e test checks the generated cover, the rating badge, and that the button waits for consent. Checked in Firefox. Run against the real Open Library and TMDB on the demo: 5 of 6 items got their cover (searching by title alone, since title and author together found nothing for *Designing Data-Intensive Applications*); *Dune: Part One* stays without one, since TMDB calls it "Dune" and nothing confirms it is the 2021 film rather than the 1984 one. Two faults found that way are fixed: cached searches left out the cover's key, so a cover whose download failed once (archive.org, which serves Open Library covers, fails now and then) could never be found again — the work's details now supply it; and a run with slow covers outlasted the server's 15-second limit (a 502) — a run now stops at 12 seconds and says to press again.

### EP-77 — Your memory, measured

**Goal.** Learning is the owner's priority, and the schedule's promise is measurable: FSRS predicts the chance of recalling each card at each review. Whether that holds for this person was never checked, so neither they nor Axon could tell if the chosen retention (90% by default) was what they got.

- [x] **EP-77/IS-77.01** · 1 pt · SHOULD — Migration `00047` adds `card_reviews.predicted`: the scheduler's predicted recall at reviews of cards in long-term memory (learning steps have none). `GET /v1/flashcards/calibration`: the last year's reviews, overall and by prediction (0–50, 50–70, 70–80, 80–90, 90–95, 95–100%), predicted against remembered (anything but Again), with the target.
- [x] **EP-77/IS-77.02** · 1 pt · SHOULD — Learn shows "Your memory, measured" and a verdict: too early under 20 reviews; below the target by more than 5 points (reviews come late: raise the retention, or fewer new cards a day); above it (reviews could come later); or on target. A table by prediction backs it.

**Tests.** `TestCalibrateComparesPredictionWithRecall`, `TestCalibrateWithNoReviews`, and the `verdict` unit tests, written first; `TestFlashcardCalibration` (an Easy review moves the card to long-term memory, the next two reviews are predicted, one remembered: 2 reviews, 50%), red before the endpoint existed.

- [x] **EP-77/IS-77.03** · 1 pt · SHOULD — Earlier reviews get their prediction by replaying each card's history through the scheduler (daily, with the other maintenance), so the measure starts from the person's whole past. Only cards whose whole history is in Axon are replayed (not imported cards with earlier reviews), and a history that does not fit the scheduler's states is left alone.

**Replay test.** `TestReplayGivesEarlierReviewsTheirPrediction`: six reviews over 60 days, predictions erased, replayed: each equals what was recorded live (within 10⁻⁶); a card whose earlier reviews are missing stays without.

### EP-78 — Topics at scale

**Goal.** Measured, not guessed: every simple GET answers in under 30 ms on the demo, so the question is how pages scale with years of notes. The notes analysis groups notes into topics by comparing every pair (up to 1,500 notes of each kind): a benchmark on synthetic notes (60 terms each, 5,000-word vocabulary) took 114 ms for 500 notes, 1.0 s for 1,500, and 4.4 s for 3,000.

- [x] **EP-78/IS-78.01** · 1 pt · SHOULD — `similarPairs` builds an inverted index (term → notes) and accumulates dot products only for notes that share a term: 11 ms, 56 ms, and 151 ms (10×, 19×, 29×), the same neighbours in the same order.

**Tests.** `TestSimilarPairsMatchesEveryPair` (the every-pair comparison as oracle on 300 notes: same neighbours, scores within 10⁻¹², ascending order), written first; `BenchmarkTopics`; the knowledge tests unchanged.

**Measured next.** The explorer graph (up to 1,500 notes) and the Zettelkasten graph (up to 600) also compare every pair, with keyword synonyms that a term index does not catch; they need their own benchmark before a change.

### EP-79 — Key phrases kept

**Goal.** Measured at 1,500 permanent notes (a throwaway probe: real notes in PostgreSQL, the real functions timed): the explorer graph 0.69 s, the Zettelkasten graph 2.85 s (it keeps 600 notes), the notes analysis 1.84 s. A CPU profile showed TextRank (`KeyPhrases`) at a third of the time, spread across map lookups, string building, and stemming: no single fault, but the same work redone on every note at every visit.

- [x] **EP-79/IS-79.01** · 1 pt · SHOULD — `cachedKeyPhrases` keeps each text's phrases by a SHA-256 of k and the text (an edited note is a new entry), cleared at 20,000 entries; callers get a copy, since they reweigh it. Analysis 1.84 s → 0.70 s; a repeat Zettelkasten graph 2.85 s → 1.68 s.

**Tests.** `TestKeyPhrasesAreCachedByText` (same result as `KeyPhrases`, one entry per text and k, a caller's change does not reach the cache), written first; the knowledge tests unchanged.

**Measured next.** What remains of the Zettelkasten graph is its pair loop (1.2 s at 600 notes): 10 × 10 keyword synonym checks per pair. Tried and dropped: skipping pairs whose keywords share no thesaurus group gave nothing measurable (1.66 s against 1.68 s), since with real vocabulary nearly every pair shares one.

### EP-80 — Other sites kept out

**Goal.** A security and accessibility pass over HTTPS (the owner's preview behind Tailscale) and in Firefox, the owner's browser. Already right: the session cookie is `HttpOnly; Secure; SameSite=Lax`, the CSP forbids framing and inline code, HSTS is on, and a write without a matching `Origin` is refused (403). axe found no serious or critical issue on 30 pages, in Firefox and in Chromium, and no page error.

- [x] **EP-80/IS-80.01** · 1 pt · SHOULD — Caddy sends `Cross-Origin-Opener-Policy: same-origin` (a page another site opened gets no handle on Axon's window) and `Cross-Origin-Resource-Policy: same-origin` (no other site can embed Axon's responses, pictures and JSON included). Calendar feeds are read by servers, not browsers, so they are not affected.

**Tests.** e2e "responses keep other sites out: no framing, opening, or embedding" (red before); the whole e2e suite (passkeys, pictures, the offline service worker) and covers in Firefox through the preview still pass.

### EP-81 — Containers without privileges

**Goal.** The images were already right: every one runs as a non-root user (distroless `nonroot` for the API) and is pinned by digest. At run time, though, each container kept Docker's default Linux capabilities and could gain more through a setuid binary.

- [x] **EP-81/IS-81.01** · 1 pt · SHOULD — `no-new-privileges` and `cap_drop: [ALL]` for Caddy, the API, the web server, the Anki bridge, and the backup job. Caddy keeps `NET_BIND_SERVICE` alone: its binary carries that file capability, and without it the kernel refuses to start it (found by trying). PostgreSQL keeps its defaults: its entrypoint changes file ownership.

**Tests.** The whole e2e suite on the hardened stack; `docker inspect` shows the dropped capabilities; the backup image's tools (`pg_dump`, `age` encryption, `ssh`, `rsync`) run under the same restrictions as user `backup`.

### EP-82 — Skills radar

**Goal.** Owner request: a spider chart ("graphe en toile d'araignée") of the hard and soft skills acquired. Skills already carried a kind (hard or soft) and a status; only per-roadmap progress bars showed them.

- [x] **EP-82/IS-82.01** · 1 pt · SHOULD — `GET /v1/roadmaps/radar`: for each kind, one spoke per roadmap holding skills of that kind, or one per section when fewer than three roadmaps do (a radar needs three spokes), with skills done, being learnt, and in all; skipped skills are left out, as in a roadmap's progress. Nine spokes at most: the smallest areas add up to "Other". One grouped query.
- [x] **EP-82/IS-82.02** · 1 pt · SHOULD — Roadmaps shows "Skills acquired": a radar for hard skills and one for soft skills, the filled shape for skills done and a dashed one adding those being learnt, figures on hover, and the same figures as a table. Under three areas, the table alone. The demo follows a few more roadmaps (Linux and Bash, databases), so its radar has a shape.

**Tests.** `TestRadar*` unit tests (spokes by roadmap, by section, folding into Other, empty), written first; `TestSkillsRadar` against PostgreSQL (a skipped skill is left out, another person sees nothing); `radar.test.ts` for the geometry and labels. Seen in Firefox and Chromium on the demo.

### EP-83 — Focus music

**Goal.** Owner request, first made with the Pomodoro (EP-35): "the ability to listen to the best music for productivity related to what I do during the Pomodoro session". EP-35 shipped noise and links that open elsewhere; nothing musical played in Axon. Streaming from a music service would tell it when and how long the person works, so Axon composes the music itself.

- [x] **EP-83/IS-83.01** · 2 pts · SHOULD — `lib/music`: three styles composed bar by bar in the browser, never the same twice: lo-fi beats (72 BPM, a ii–V–I–vi loop with ninths, swung hats, a record's hiss), ambient (60 BPM, open pads and bells in a long reverb), and calm piano (66 BPM, broken chords through I–V–vi–iii–IV–I–IV–V). No words, which compete with reading and writing; steady, slow tempos; every note in key. "Matched to what you focus on" picks lo-fi for tasks and projects, ambient for notes, resources, and areas, and piano for skills, roadmaps, and flashcards. Settings accept `music`, `lofi`, `ambient`, and `piano`; the demo uses `music`.
- [x] **EP-83/IS-83.02** · 1 pt · SHOULD — Focus offers music and noise in one list, says what "matched" will play for the chosen item, and keeps a sound picker on the timer to change it mid-session; music stops for breaks as the noise does. One audio context per page, created in the click that starts the session: Safari only lets sound start from a click, and the session is created over the network first.

**Tests.** `music.test.ts` (written first): the style for each type, notes inside their bar, every pitched note in key, drums only in lo-fi, the chord progression bar by bar, a seeded bar replays exactly, calm tempos. `TestFocusSessionsLinkToWhatTheyWereFor` accepts every sound and refuses an unknown one. The e2e Pomodoro test picks matched music for a task. Rendered offline in Chromium and Firefox: no clipping, no silent second, the three styles within 3.5 dB of each other (piano was 12 dB quieter and was raised). A live session runs its audio context in Chromium and WebKit; headless Firefox suspends every audio context, even one made in a click, so it was checked offline only.

**Found on the way.** Aligned columns of Markdown tables never aligned: the alignment was a `style` attribute, which the Content Security Policy blocks; it is a class now.

### EP-84 — Teaser on the home page

**Goal.** Owner request: "insert the movie teaser on the main page in the demo". Visitors to a demo saw a hero and five illustrated cards, never the app itself until they signed in.

- [x] **EP-84/IS-84.01** · 1 pt · SHOULD — `make teaser` (`scripts/teaser/make-teaser.sh`) records a 55-second tour of the demo in Chromium: Today, tasks, the Zettelkasten graph, goals, the skills radar, Learn, a focus session, the media shelf, and a closing card, each with a caption. Frames come from the browser's screencast (sharper text than a recorded video); the soundtrack is the lo-fi focus music rendered offline; the English and French caption files are timed from the tour itself. Encoded as VP9/Opus (WebM) and H.264/AAC (MP4, for Safari), 2.1 MB each, with a poster.
- [x] **EP-84/IS-84.02** · 1 pt · SHOULD — When the demo is on, the home page shows "Axon in under a minute" under the hero: the video with controls, played only on request (`preload="none"`: nothing downloads, nothing moves on its own), WebM first and MP4 as the fallback, both caption tracks. It comes from Axon's own server, like everything else on the page; the service worker leaves it out of the offline install.

**Tests.** `offlineShell` leaves `/teaser/` out (written first). The home page e2e test finds two sources and two tracks and gets partial content (206) for each file, as Safari asks for it. Played on the home page in Chromium, Firefox, and WebKit: each picks the WebM and plays; caption tracks load. ShellCheck passes on the script.

### EP-85 — Focus and Roadmaps in French

**Goal.** IS-31.06 translates a page when it is next touched; EP-82 and EP-83 touched Focus and Roadmaps, which were English only.

- [x] **EP-85/IS-85.01** · 1 pt · SHOULD — Every string on Focus and Roadmaps, the radar's captions, table, and hover text, the sound choices (Focus and Preferences), and the item types shown on Focus go through `t()`, with 104 French strings. Sentences around a link are split where the French reads well split. "0 jour d'affilée": the day count picks its form with `Intl.PluralRules`, since French counts zero as singular. Roadmap titles and library content are the person's own data and stay as they are.

**Tests.** `i18n.test.ts` checks every French string keeps its placeholders; the English e2e suite is unchanged but for the matched-music line ("Lo-fi beats for “Buy timber”."). Both pages seen in French in Firefox.

### EP-86 — French everywhere in the web app

**Goal.** Finish IS-31.06 ("fully before 1.0"). After EP-85, 86 of 102 components still showed English only, about 1,500 strings in templates and 570 in code, with no way to tell what was left except reading every page.

- [x] **EP-86/IS-86.01** · 2 pts · SHOULD — `i18nCoverage.test.ts`, written first: it fails on text between tags, in the attributes people read (placeholder, aria-label, title, label, alt, message, and the empty, hint, and description props), and on string literals in template expressions (both sides of a ternary, template literals and their `${…}` parts) that do not go through `t()`. A second test fails on any key passed to `t()`, in code or in Axon's own tables (templates, PARA kinds, weekdays, Atomic Habits laws, recurrence presets), that has no French; names and codes that read the same are listed.
- [x] **EP-86/IS-86.02** · 5 pts · SHOULD — Every component goes through `t()`, with 1,800 French strings. Plurals and fragments that built English from parts (`passkey{s}`, "vote"/"votes", "card keeps"/"cards keep") became whole sentences per case. Loop variables named `t` were renamed so `t()` is never shadowed; tables of labels follow a language change (`$derived`), and Axon's own tables are translated where they are shown. `t()` shows a missing value as nothing, as `{value}` does in markup.
- [x] **EP-86/IS-86.03** · 1 pt · SHOULD — Dates and numbers follow the interface's language (`appLocale()`: French gives French dates, in the browser's own French when it has one); "Today", "Tomorrow", and "Yesterday" are translated. Before, a French page said "Échéance Fri, Oct 22".

**Tests.** The two coverage tests (each made to fail by adding an untranslated string); `relativeDay` and `formatDate` in French; the 66 English e2e tests unchanged and green; a Firefox walk through 42 pages in French with no page error, whose remaining English is the demo's own data (titles the person wrote) and text the API writes (EP-87).

### EP-87 — The API speaks French

**Goal.** After EP-86 the pages were French but what the API wrote was not: every error and validation message (they show in toasts), the dashboard and insight explanations, goal progress, privacy purposes, badges, and media suggestions. 480 messages in all.

- [x] **EP-87/IS-87.01** · 2 pts · SHOULD — `internal/l10n`: the language from `Accept-Language` (French when it is preferred over English, q-values honoured), put in the request's context and said in `Content-Language`, with `Vary: Accept-Language`. `httpx.Problem` reads `Content-Language` from the response, so the 384 problem and validation messages are translated without touching their call sites. `T`/`Tr` translate (a message with arguments is a format), `Mark` flags strings translated where they are shown, `Num` and `Date` write numbers and dates the French way. The web app sends the stored language, known before the layout applies it, and reloads once when a device first meets an account in another language, so nothing already fetched stays in the old one.
- [x] **EP-87/IS-87.02** · 2 pts · SHOULD — Dynamic text: dashboard KPIs and their definitions, the task counts, insight titles, formulas, trend sentences (one format per direction, so French phrases each naturally) and the weekly review's highlights, goal progress, point rules, badges, streak messages, consent purposes, and media reasons. Each agent's `SKILL.md` gains `title_fr` and `description_fr`; tool descriptions are written for the model and stay English.
- [x] **EP-87/IS-87.03** · 1 pt · SHOULD — `catalog_test.go` parses every Go file and fails on a message given to `httpx.Problem`, `ValidationError`, or `l10n.T`/`Tr`/`Mark` without French, on a catalog entry nothing uses, and on a French format whose verbs differ from the English. The French catalog is excluded from the English spell-checker and from gosec's credential guess (a message about tokens).

**Tests.** `l10n` unit tests (negotiation, middleware, catalog, numbers, dates); `TestProblemSpeaksTheResponseLanguage`; trend sentences in French; `TestTheAPISpeaksFrenchWhenAsked` against PostgreSQL (a problem, insights, and consents in French, and in English for `en-GB`); `TestEverySkillSpeaksFrench`; the full Go suite and the 66 e2e tests. A Firefox walk through 42 pages in French leaves only the demo's sample data and the tool descriptions in English.

### EP-88 — Daily needs

**Goal.** Owner request: "following science and nutrition, define the amount of macronutrients needed to be healthy following activities, age, gender, size", with a warning to check with a nutritionist. Until now Axon set no numbers at all (EP-44's coaches must never set calorie targets).

- [x] **EP-88/IS-88.01** · 2 pts · SHOULD — `health.Calc`: Mifflin–St Jeor basal rate × activity factor (1.2 to 1.9); protein from EFSA's 0.83 g/kg to 1.6 g/kg with hard training (ISSN), at least 1.0 g/kg from 65 (ESPEN); fat 30 % of energy, carbohydrates the rest; fibre 25 g and water 2.0–2.5 L (EFSA). Each figure comes with how it was worked out, in the person's language, and the sources. No numbers before 18 or during pregnancy or breastfeeding; no weight-loss or gain target (ADR 0028). Migration `00048` keeps the profile under the `health_data` consent: exported with the health data and deleted when the consent is withdrawn. `GET /v1/health/needs`, `PUT /v1/health/profile`.
- [x] **EP-88/IS-88.02** · 1 pt · SHOULD — Health shows "Your daily needs": the warning first, always; a short profile form ("Prefer not to say" for sex takes the middle of the two equations); the needs, the day's log against energy, protein, carbohydrates, and fat (over a reference is said in words, not in red), and "How is this worked out?" with the references.

**Tests.** `needs_test.go`, written first: two worked examples computed by hand, no sex given, older adults' protein, no numbers for a minor or in pregnancy, French explanations, profile validation. `TestDailyNeedsFromAProfile` against PostgreSQL (consent required, a minor gets no numbers, the profile is deleted with the consent). The e2e health test fills the profile and finds the warning, 2,760 kcal, and the energy bar.

### EP-89 — A photo of a meal

**Goal.** Owner request: "use a camera and take a picture of the food, analyse it, and determine the macronutrients; or the user gives the product and the macros". Typing the macros and finding a product by name or barcode already worked (EP-18); a photo did not.

- [x] **EP-89/IS-89.01** · 3 pts · SHOULD — `POST /v1/health/food/estimate`: the photo (JPEG or PNG, 10 MB) is re-encoded at 1,568 px on its long side, which drops camera and location metadata, and sent once to the person's AI provider with a fixed prompt: one line per food with the portion's weight and its energy and macronutrients from composition tables, a note on what was uncertain, names in the person's language, no comment on health or weight, and text in the photo treated as data. The answer is untrusted: out-of-range lines are dropped, twelve at most. It needs `ai_agents`, `health_data`, and the new `food_photos` consent, stops before crossing the monthly budget, and is recorded like a run (cost, never content). Never in the shared demo. The OpenAI-compatible adapter sends images as `image_url` parts, so local and other vision models work too.
- [x] **EP-89/IS-89.02** · 2 pts · SHOULD — Health: "Photo of a meal" opens the phone's camera (a file picker on a computer); the photo is made small in the browser first. Before anything is sent, missing consents are shown with what they share, to allow at once. The estimate comes back as editable lines (name, grams, energy, protein, carbohydrates, fat), each checked by default, under a warning that photo estimates can be far off, and is logged to the chosen meal only when the person says so. "Photo of a barcode" (where the browser can read barcodes) finds the code on the device and looks the product up as before.

**Tests.** `parseEstimate` (bounded, untrusted answers) and the adapter's image parts, written first; `TestMealPhotoEstimate` against PostgreSQL with the scripted provider: refused without each consent, the photo reaches the provider without its EXIF location, out-of-range lines are dropped, the call is recorded with its cost, a non-picture is refused. `fitSize` unit test. The e2e health test picks a photo and finds the consent request, with no request sent.

### EP-90 — Chef & Nutritionist

**Goal.** Owner request: "give the user meals following the budget, dietary restrictions, constraints, and the time one wants to spend; perhaps with an AI agent who is a nutritionist and a chef".

- [x] **EP-90/IS-90.01** · 3 pts · SHOULD — The `chef` skill (Chef & Nutritionist, in French too) with `daily_needs` (the EP-88 reference intakes, or why there are none), `food_log`, `draft_meal_plan`, and `remember`, behind `health_data` and `health_ai` with the health safety on (red flags referred without a model call). It plans each day close to the needs for one adult and scales to the number of people, never below them to make someone lose weight; with no numbers (no profile, a minor, pregnancy) it plans balanced plates without figures. Restrictions and allergies are strict, with a reminder to check labels; prices are estimates in the person's currency, within the budget; recipes stay within the cooking time. `draft_meal_plan` proposes Markdown (days, meals, recipes, a shopping list as checkboxes, the cost) that becomes `Meal plans/<date>-<title>.md` once approved, with a note that everything is an estimate; a second plan the same day gets `-2`.
- [x] **EP-90/IS-90.02** · 1 pt · SHOULD — The daily-target guard read any "per day" in a text with any kcal figure under 1,200, so a meal plan's "breakfast, about 450 kcal" every day would have been refused. A figure is now a daily target only when "a day", "daily", or the like sits next to it on its line; "Keep to 900 kcal a day" is still refused, in a reply or a proposal.
- [x] **EP-90/IS-90.03** · 1 pt · SHOULD — Health: "Plan my meals with the chef" asks for days, people, meals, a budget and currency, restrictions and allergies (twelve to tick), the cooking time, and other constraints, then hands the request, written in the interface's language, to the page's agent dock, where the run and its proposal appear for approval.

**Tests.** `TestMealPlansAreNotDailyTargets` (written first, with the existing safety suite unchanged); `TestChefPlansMealsAsANote` with the scripted provider (the agent gets 1,520 kcal from the needs, the plan with kcal per meal is proposed and becomes a note in "Meal plans/", a second one the same day too, a 900 kcal a day plan is refused); `mealRequest` unit tests; the e2e health test sends the form to the Chef & Nutritionist.

### EP-91 — Daily journal

**Goal.** A life OS for well-being had a daily check-in that was only a line of text: no mood, energy, stress, or sleep to look back on, and nothing for EP-92 to learn from.

- [x] **EP-91/IS-91.01** · 2 pts · SHOULD — Migration `00049` adds `journal_entries`, one per person and day: mood, energy, and stress from 1 to 5, hours of sleep, up to three things the person is grateful for, and a note. Mood and sleep are health data (GDPR art. 9), so the journal is kept only with `health_data`, exported, and deleted when it is withdrawn. Consent withdrawal could hold only one listener per purpose, so a second package would have silently replaced Health's deletion; `privacy.OnWithdraw` now keeps a list. `GET /v1/journal` (the last 30 days by default, and today in the person's time zone), `GET`, `PUT` (replaces the day), and `DELETE /v1/journal/{day}`; never a future day, and never an empty one.
- [x] **EP-91/IS-91.02** · 1 pt · SHOULD — Reflect → Journal: three scales of five faces (native radio buttons under the faces, so the keyboard and screen readers work; choosing again clears), sleep, three gratitude lines, a note, and the last 30 days as a mood and energy chart and a list to reopen any day. "How are you today?" can be added to Today; it merges into the day so nothing written is lost.

**Tests.** Field checks (written first); `TestJournal` against PostgreSQL (consent, replace, future day, empty day, other people, export, deletion with the consent); `TestEveryListenerHearsAWithdrawal`; `hasContent`; an e2e test that clicks the faces, saves, and finds the day in the last 30 days, with axe.

### EP-92 — What helps you

**Goal.** A journal is worth keeping when it gives something back: which of a person's habits go with their better days, without reading chance as a pattern or a pattern as a cause (ADR 0029).

- [x] **EP-92/IS-92.01** · 2 pts · SHOULD — `GET /v1/journal/helps` reads the last 90 days in one query (the journal, and per day the minutes trained and focused, habits kept, and tasks finished, in the person's time zone) and tests six habits against mood, energy, and stress: Spearman rank correlation, a permutation test of 2,000 shuffles seeded by the pair, and the Benjamini–Hochberg correction over every pair. A link is kept when q ≤ 0.1 and |ρ| ≥ 0.3, with 14 days and 5 on each side of the split (sleep at 7 h, the least adults need; counts at their median); at most six, strongest first, each as a sentence with the two averages, in the person's language. Before 14 days, it says how many more are needed.
- [x] **EP-92/IS-92.02** · 1 pt · SHOULD — Journal: a "What helps you" card with each link, its strength, the averages on those days and the others as bars, the correlation and q-value, "a link is not a cause", and the method. The demo has 60 days of journal and runs, so it shows links at once.

**Tests.** Spearman, the permutation test, and Benjamini–Hochberg against reference values from Python (written first); a planted link between sleep and mood found and noise left out, the 9-day case, and French; `TestWhatHelps` against PostgreSQL; the demo shows at least two links; the e2e journal test reads how many more days are needed.

### EP-93 — Journal days in the vault

**Goal.** Everything in Axon is a note in the vault (EP-48), but the journal was not: people who keep daily notes in Obsidian could not see or query their days there.

- [x] **EP-93/IS-93.01** · 1 pt · SHOULD — With health notes turned on, each journal day is `Axon/Health/Journal/Journal YYYY-MM-DD.md`: `type: journal`, the date, and mood, energy, stress, and sleep hours in the front matter (only those given), then "Grateful for" and "Note". Written one way like the other health notes, in the same transaction as the change, rewritten when the day changes, and removed when it is deleted or the consent is withdrawn. Turning health notes on writes every existing day.

**Tests.** `TestVaultMirror` (written first): a day becomes a note with its front matter and sections, deleting it removes the note, and withdrawing the consent removes every health note.

### EP-94 — Only what you use

**Goal.** Axon has nineteen pages, and a new account met them all at once: the first minutes decide whether someone stays, and a life OS that shows a budget, a household, and flashcards to someone who came for tasks feels like work before it helps.

- [x] **EP-94/IS-94.01** · 1 pt · SHOULD — Settings gain `menu`: the pages left out of the navigation (any but Today, each once) and whether the person has chosen. A hidden page still opens from a link or a search, and nothing in it changes.
- [x] **EP-94/IS-94.02** · 1 pt · SHOULD — A new account is first asked, on Today, what it wants Axon for (eight uses, from tasks and projects to a shared home). The answer sets the menu and Today's cards in one save; "Show me everything" keeps both as they are; imports are one link away. The menu is remembered on the device, so it does not flash every page before the settings answer.
- [x] **EP-94/IS-94.03** · 1 pt · SHOULD — Preferences → Pages in the menu. Preferences sent the settings without Today's cards, and the API fills a part left out with its default, so saving Preferences reset a customised Today; it now sends them back as they are.

**Tests.** `TestMenuPages` (written first: default, saved, duplicates, Today, unknown pages, YAML); `planFor` unit tests; an e2e test that chooses two uses, finds the menu without Budget and Today with the calendar, brings every page back in Preferences, and finds Today unchanged, with axe.

### EP-95 — The Top-5% roadmap

**Goal.** The owner asked (2026-10-13) for the roadmap in their vault, *Top-5% in 26 engineering domains*, to be the only roadmap, used by default and in the demo, with everything related to it: tasks, subtasks, projects, goal, milestones, side projects, habits, and resources (ADR 0030).

- [x] **EP-95/IS-95.01** · 3 pts · MUST — The library is that roadmap: the 26 domains as skills in seven areas, each with its top-5% proof and the resources the phases use (86), and six soft skills the plan relies on, so both radars have something to show. OSSU and the 27 library domains are removed, with the regrouping of imports made before skills were competencies; roadmaps people already imported stay theirs.
- [x] **EP-95/IS-95.02** · 3 pts · MUST — Starting it creates the plan in one transaction: the goal with its outcome and a measure of 26 domains, 14 milestones, a project per phase (the first active, the others on hold) whose board holds the exit check as an epic, issues, and subtasks, a *42 cursus* side project with each phase's 42 projects, the four habits with their cue, routine, and reward, the first week's tasks (two on pipekit's board) and the monthly roadmap check, the areas *Engineering mastery* and *Maths* (reused if they exist), three resources (42 subjects and testers, certification exams, books platform), and the roadmap's own note, unless the vault already has it. Everything is connected to the goal or the roadmap. Started while phase 1 runs, it keeps its dates; later, it starts on the coming Monday. It starts once.
- [x] **EP-95/IS-95.03** · 1 pt · SHOULD — Placement follows the skills assessment: a domain rated at level 3 of 5 or more starts as being learnt, never done, since each needs its proof.
- [x] **EP-95/IS-95.04** · 1 pt · MUST — The demo follows the plan a week in: phase 1's skills being learnt, the first week's tasks done but one, the habits checked in with a rest day, a 6.042J literature note with its 15 flashcards and their diagrams, and the certification and reward funds in the budget.

**Tests.** Library and phase-date checks and the plan's dates (written first); `TestStartingTheTop5RoadmapStartsThePlan` against PostgreSQL; placement in `TestAssessmentPlacesYouOnRoadmapsAndInformsTheTutor`; the demo and overview tests; e2e tests that start the roadmap at your level from the assessment and walk the plan (skill graph, goal, projects, habits). The e2e tests now run on a stack of their own, so a running local stack is left alone.

### EP-96 — A sky that follows the sun

**Goal.** The owner asked (2026-10-13) for a colour theme that follows the sun and the moon, set by the time zone, and for the home page's picture to follow the sunlight over a landscape they choose.

- [x] **EP-96/IS-96.01** · 2 pts · SHOULD — The sun's and the moon's positions in the browser, from the Astronomical Almanac's low-precision formulas (about 0.01° for the sun), and the moon's phase from a known new moon. The place comes from the account's time zone (its main city, or a longitude from its offset), never from the device's location. The sky's colours are blended from the sun's altitude across night, the three twilights, the golden hour, and day.
- [x] **EP-96/IS-96.02** · 2 pts · SHOULD — The scenery on the home page and in a strip above Today: the sun and the moon (its lit part for its phase) where they stand, stars fading in after dusk, and eleven landscapes drawn in SVG and shipped with Axon (mountains, forest, ocean with the light on the water, desert, the Earth from orbit, the Eiffel Tower, the Pyramids of Giza, the Taj Mahal, the Statue of Liberty, Christ the Redeemer, the Sydney Opera House). It moves every minute.
- [x] **EP-96/IS-96.03** · 1 pt · SHOULD — The theme "Follow the sun and the moon": light from civil dawn to civil dusk, dark otherwise, checked every minute, in the settings and on the header's theme button. Preferences choose the landscape and preview the sky later in the day.

**Tests.** Sun and moon against known positions and phases (Paris solstice noon and sunrise, Sydney winter noon, the equator at the equinox, the October 2026 full and new moons) and the palette and theme rules, written first; `TestTheThemeFollowsTheSunOverAChosenLandscape`; an e2e test that chooses the theme and a landscape and finds both on Today. Every landscape was looked at, at noon, sunset, night, and dawn, in Firefox.

### EP-97 — The roadmap on the calendar

**Goal.** The owner asked (2026-10-13) for the roadmap's schedule (which track to work on each day, following the phases and milestones) to be on the calendar, and for its projects to be created as in their vault.

- [x] **EP-97/IS-97.01** · 2 pts · SHOULD — Migration `00050` records a started roadmap's plan and start date. The calendar's new type `study` works the weekly rhythm out from the plan: each day's block (maths, computer science, the language track, catching up, spaced tests and DSA, the weekend's project days) with the tracks and project of the phase the day falls in, so the blocks change with the phases and nothing is stored per day. They link to the roadmap, show dashed in the month and the agenda, and iCal feeds can include them (each day its own event). Milestones and the goal's date were already on the calendar.
- [x] **EP-97/IS-97.02** · 1 pt · SHOULD — Each phase project has its note in `1-Projects/<project>/<project>.md`, in the vault's project template (outcome, next actions on the board, tracks, log), unless the vault has one already.

**Tests.** The rhythm by weekday and phase, shifted plans, and nothing outside the plan (written first); the integration test reads a week of blocks and the pipekit note; the e2e test finds the blocks on the calendar, linking to the roadmap.

### EP-99 — A year you can read

**Goal.** The owner reported (2026-10-13) that the 12-month activity chart did not show the current month and had no labels for its colours; the demo read "970 changes on 1 of 365 days".

- [x] **EP-99/IS-99.01** · 1 pt · SHOULD — The chart is laid out by Monday-first weeks (worked out in `heatGrid`, tested), labels each month at the week it starts and the weekdays beside it (kept in view while the weeks scroll), opens scrolled to the current month with today outlined, and has a legend: no activity, less to more, today.
- [x] **EP-99/IS-99.02** · 1 pt · SHOULD — The demo was all created on the night it resets, so its year had one active day. It now also holds a few edits per item over the past eleven months, more of them recently, added to the append-only event log (never rewritten) with time-ordered ids, the same every night.

**Tests.** `heatGrid` (written first); the overview test wants activity on at least 60 days of the demo's year; the demo e2e test finds the legend and the chart scrolled to its end.

### EP-98 — Every page its picture

**Goal.** The owner asked (2026-10-13) for illustrations on each page, type of note, and section, each with a short description (a painting, a picture, a film scene, an artwork, a video game), and for a background on each Reflect page: Journal, Insights, the weekly review, and Agents.

- [x] **EP-98/IS-98.01** · 1 pt · SHOULD — Four new themed spaces, original and drawn from geometry by `scripts/spaces.py` in light and dark (the earlier drawings are unchanged, byte for byte): evening light through a leaded window on a diary (Journal, after Vermeer), star trails and Galileo's moon sketches over an observatory (Insights), a lighthouse sweeping the sea (the weekly review, after Hopper), and clockwork gears and cams (Agents, after Jaquet-Droz's Writer).
- [x] **EP-98/IS-98.02** · 1 pt · SHOULD — Every themed page ends with what its picture shows and what it nods to, from Leonardo's notebooks to Celeste's mountain; hidden with themed spaces off.
- [x] **EP-98/IS-98.03** · 1 pt · SHOULD — Each type of note (permanent, literature, fleeting, daily, weekly, monthly, project, area, resource, roadmap, goal, milestone, habit, media, test, map of content, journal, task, workout, food log, meeting, archive) has a small line drawing and a sentence on what it is for, on the note and on the templates page.

**Tests.** The Reflect pages' spaces, French for every caption and note type (passed by name, so the coverage test cannot see them), and the fallback drawing; the e2e suite runs axe on these pages. Every new space was looked at in light and dark.

### EP-100 — Your vault's templates

**Goal.** The owner asked (2026-10-13) for the templates of their SecondBrain vault, one per type of note, to be implemented in Axon.

- [x] **EP-100/IS-100.01** · 2 pts · SHOULD — The vault's nineteen templates ship byte for byte (kept out of the formatter): literature, permanent, project, area, resource, daily, weekly, monthly, roadmap, goal, milestone, habit, media, and the six retrieval tests, in groups on the templates page and in the new-note dialog, beside Axon's fleeting, archive, meeting, and blank notes. Each files its note where the vault does (`Zettel/literature/<id>-<slug>.md`, `2-Areas/<slug>/<slug>.md`, `Tracker/goals/`, `Weekly/2026-W41.md`, `Reviews/<note>/5w.md`…). Placeholders are filled: `{{created}}` as date and time, `{{slug}}`, a test's `{{note}}`, a media note's kind and status; any other is left blank. Today's daily note uses the daily template. A vault template named like a built-in one (Templates/goal.md) replaces it.
- [x] **EP-100/IS-100.02** · 1 pt · MUST — Found on the way: since EP-74 the proxy compressed large responses and suffixed their entity tag with the encoding (`"…-zstd"`); the browser sent it back in If-Match and the API answered 412, so a note or item over the compression threshold showed "changed somewhere else" on its first save. The API now reads the suffix and a weak `W/` through.

**Tests.** Paths, contents with no placeholder left, the daily note, the round trip through the vault's templates folder (written first); `TestIfMatchAcceptsWhatTheProxyMadeOfTheTag`; e2e tests that create a note from the vault's literature template and find the retrieval tests' sections.

### EP-101 — Water

**Goal.** The owner asked (2026-10-14) for a tracker of the water drunk, with the food and health logs. Axon already worked out the daily water need (EP-88) but could not log a drink.

- [x] **EP-101/IS-101.01** · 2 pts · SHOULD — Migration `00051` adds `water_entries` (1 to 3,000 ml a drink, never on a later day), health data kept only with `health_data`, exported, and deleted when it is withdrawn. `GET /v1/health/water` gives a day's drinks, total, and week, and its target: the drinks' share, 80 %, of the daily need (EFSA's adequate intake counts drinks and food), none without a profile, under 18, or during pregnancy (ADR 0028). `POST` logs a drink, `DELETE` removes one.
- [x] **EP-101/IS-101.02** · 1 pt · SHOULD — A Water card on Health (for the day shown) and, if added, on Today: a glass or a bottle in one tap, any amount, the day as glasses, the drinks with their time to remove one, and the last seven days. "Health and well-being" at the first question adds it to Today. What helps you (EP-92) has a factor for the days water was logged: 1.5 L or more. The demo has a month of drinks.

**Tests.** `TestWaterTracker` against PostgreSQL (consent, bounds, a later day, the target from a profile, another day, other people, export, withdrawal; written first), the water factor in What helps you, the glasses and litres helpers, and the e2e health test (two drinks against the target, one removed, with axe).

### EP-102 — Patched third-party images

**Goal.** The owner asked (2026-10-14) for a check of active CVEs and their fixes. Axon's own code and images were clean; the two third-party images it runs were not, and upstream had not rebuilt them.

- [x] **EP-102/IS-102.01** · 1 pt · MUST — `deploy/caddy`: Caddy v2.11.7 with its standard modules, built with Go 1.26.9 and `golang.org/x/net` 0.60.0 on the upstream image (four HIGH denial-of-service flaws fixed). Its file capability is gone, and so is `cap_add: NET_BIND_SERVICE`.
- [x] **EP-102/IS-102.02** · 1 pt · MUST — `deploy/db`: pgvector 0.8.7 on PostgreSQL 16 with Debian's security updates and `gosu` 1.19 rebuilt with Go 1.26.9 (54 HIGH or CRITICAL fixed); same major version and Debian release, so existing data directories start as they are. Both images join the CI build, Trivy gate, and SBOM matrix and are built and copied by `deploy.sh`; servers run the copied images.

**Tests.** Trivy (the CI gate: no fixable HIGH or CRITICAL) on both built images; the preview's database restarted on the new image with the same 1,146 entities and pgvector working; the full Go suite on it; all e2e tests on fresh stacks of the new images.

### EP-103 — Practice log

**Goal.** The roadmap asks for 400 LeetCode problems, OverTheWire levels, every PortSwigger lab, and Kaggle medals (EP-95), and Axon could not log one. Practising problems also forgets like anything else: re-solving a problem from scratch, not recognising its solution, is the skill.

- [x] **EP-103/IS-103.01** · 3 pts · SHOULD — Migration `00052` adds `practice_problems` (title, link unique per person, platform, difficulty, up to ten topics, language, an optional roadmap skill) and `practice_attempts` (day, solved unaided, with help, or not solved, minutes). `POST /v1/practice` logs an attempt, creating the problem the first time (the same link is the same problem); `POST /v1/practice/{id}/attempts`, `GET /v1/practice` (with `due=true`), `GET /v1/practice/stats`, `DELETE`. The re-solve day: tomorrow after a miss, three days after help, a week after a first unaided solve (an easy one is then done), then 2.5 times the last gap, until it would pass 90 days. In the export.
- [x] **EP-103/IS-103.02** · 2 pts · SHOULD — Learn → Practice: paste a link and the platform and title fill in (LeetCode, Codeforces, HackerRank, Advent of Code, Exercism, Project Euler, OverTheWire, PortSwigger, Hack The Box, TryHackMe, Kaggle); re-solve today with one tap per outcome; solved by difficulty, topic, and the last twelve weeks; every problem with its attempts and next re-solve. The demo has twelve weeks of classic problems and the first Bandit levels, linked to the roadmap's skills.

**Tests.** The re-solve schedule (written first, ten cases); the link reader; `TestPracticeLog` against PostgreSQL (create, same link, attempts, validation, due, stats, other people); an e2e test that pastes a link, logs with help and unaided, and reads the schedule, with axe.

### EP-104 — Certifications

**Goal.** Nine of the roadmap's proofs are exams, about $3,000 over three years, and some expire (EP-95). Axon knew their names only as text in a resource.

- [x] **EP-104/IS-104.01** · 3 pts · SHOULD — Migration `00053` adds `certifications` (name, provider, code, link, planned, booked, passed, or failed, target, exam, pass, and expiry dates, validity, score, cost and currency, credential link, notes, a roadmap skill). A booked or failed exam has its date; a pass defaults to the exam day and sets its expiry from the validity. `GET`, `POST`, `PUT`, and `DELETE /v1/certifications`; in the export. The calendar's new type `certification` shows booked exams and the expiry of passed ones (and iCal feeds can carry them). Starting the roadmap plans its nine exams for the end of their phases, with their price and validity, each linked to its skill.
- [x] **EP-104/IS-104.02** · 2 pts · SHOULD — Learn → Certifications: a card per exam with its status, target, countdown, result, credential, and expiry; an update form for booking and results; passed of planned, the next exam, paid of planned per currency, and a warning 90 days before a certification expires.

**Tests.** `TestCertifications` against PostgreSQL (written first: create, validation, a pass setting the expiry, exam and expiry on the calendar, other people); the roadmap test finds its nine exams; costs, next exam, and expiring helpers; an e2e test that books the CKA and reads its countdown, with axe.

### EP-105 — Study hours against the plan

**Goal.** The roadmap's monthly check asks for "hours vs plan" (EP-95), and its weekly reflection for "hours done vs 22 h"; Axon timed focus sessions but could not say which were for the roadmap.

- [x] **EP-105/IS-105.01** · 2 pts · SHOULD — `GET /v1/roadmaps/{id}/hours` adds up the focus sessions for the roadmap, one of its skills, its goal and milestones, the projects, habits, and tasks supporting the goal, and the tasks on those projects' boards (in the person's time zone): the last 12 weeks, this week day by day with the plan's block (EP-97), and minutes by skill and by project. The weekly target is the low end of the plan's hours. A session for nothing in particular is not counted.
- [x] **EP-105/IS-105.02** · 1 pt · SHOULD — A roadmap's page shows the week against its target, the days beside their blocks, twelve weeks of bars under the target line, and the time by skill and project; **Focus on today's block** opens Focus aimed at the roadmap (Focus now takes `?target=`). The demo has its first week of sessions.

**Tests.** `TestStudyHoursAgainstThePlan` against PostgreSQL (written first: sessions on a skill, the roadmap, and a project count, one for nothing does not; a roadmap without a plan has no target); the e2e roadmap test reads 0 h of 22 h and opens Focus on the block, with axe.

### EP-106 — Where the vault lives

**Goal.** "Easily define the server where the vault Markdown will be; set up the synchronisation with a new server automatically; generate the structure and the templates if it is empty; the server is an option: by default everything is in the database" (owner, 2026-10-14). Decisions in [ADR 0031](adr/0031-vault-servers-pair.md).

- [x] **EP-106/IS-106.01** · 2 pts · SHOULD — `vault_sources` (one server and folder per person, none by default); `PUT /v1/vault/source` makes a one-time pairing code (hashed, 30 min, rate limited) and two ready commands; `POST /v1/vault/pair` gives the server a write token valid a year, replacing the previous one; `DELETE` revokes it and keeps the notes; host and folder are checked so they cannot break the shell quoting.
- [x] **EP-106/IS-106.02** · 2 pts · SHOULD — `axon notes-sync -pair CODE` saves the token (0600) and, in an empty folder, makes the folders while Axon adds `Home.md` and the built-in templates, which the first sync writes; a folder with notes is left alone. The sync reports after each run and every 5 minutes (`/v1/vault/heartbeat`): in step, silent, or failing, with the conflicts. `/v1/vault/install.sh` and `/v1/vault/agent` give a one-line install with no root and a user systemd service; the Docker command does the same.
- [x] **EP-106/IS-106.03** · 1 pt · SHOULD — Preferences → Markdown vault: "In Axon only" (default) or "Also as files on a server", the commands with a copy button, and the vault's state.

**Tests.** `TestVaultServer` (default none, unsafe host and folder refused, a code used once and typed loosely, scaffold once, heartbeat and status, another person's token refused, pairing again revokes the old token, removal revokes and keeps notes, public script and program); `TestVaultPairingFillsAnEmptyVault` (pair with the real `notesync` client, token file 0600, folders, first sync writes Home and templates, a full vault untouched); a web test keeps the API's templates equal to the web's; e2e "the vault lives in Axon by default, and a server is set up with one command". Checked by hand: the one-line install against a running Axon paired, made the folders, and wrote 65 files into an empty folder.

### EP-107 — Digests everywhere

**Goal.** "A digest each morning by mail, Discord, Slack, Teams…, with the tasks to validate today, the next important or due tasks, projects, milestones, learning; the same weekly and monthly; aesthetic, illustrated, with a motivating quote related to the tasks, the day, or the profile" (owner, 2026-10-14). Decisions in [ADR 0032](adr/0032-digests-on-chat-channels.md).

- [x] **EP-107/IS-107.01** · 3 pts · SHOULD — `internal/digest`: daily (today's tasks numbered, late ones first; habits with their two-minute version; coming up: tasks due within a week, milestones within two, goals within a month; cards, notes to recall, the weekly test, household, the weekly review), weekly (last week's numbers, still open, the week ahead, the review), monthly (last month's numbers, goals with progress bars, the month ahead); English and French; a quote chosen for the day's theme and the person's domains; a landscape PNG drawn for the day.
- [x] **EP-107/IS-107.02** · 3 pts · SHOULD — `notify_channels`: Slack (webhook or bot), Discord (webhook or bot, with the picture), Teams (Adaptive Card), Telegram (bot, picture then text), ntfy, signed webhooks; addresses checked per service, free addresses through `netsafe`; secrets sealed, never returned or exported, kept out of errors; per channel: which digests, reminders or not, on or off, a test, the last error. Reminders and the evening nudge reach channels too.
- [x] **EP-107/IS-107.03** · 2 pts · SHOULD — the weekly (day, time) and monthly (day 1–28, time) settings; the email digest as text plus HTML with the picture inline; `GET /v1/me/digest` and `/picture`; a Digest page (today, this week, this month) with "Send it to me now"; Preferences → Reminders and digests, Chat apps.

**Tests.** `TestDigestsOnEveryChannel` (the digest numbered with a quote and a picture; each kind's address checked; no secret in any answer; ownership; a test to one channel; an unlinked bot refused; changing the name keeps the secret; the morning digest once to every channel taking it, Discord and Telegram with the picture, ntfy with its token; the weekly digest signed for the webhook; a failing channel recorded without stopping the others; the monthly digest; the evening nudge only where reminders are on); digest unit tests (every string in French, quotes by theme and domain, renderers escape text and keep Discord's limits, the picture is stable per day); `TestRichMail` (multipart parsed as a mail app does); e2e "the digest: numbered items, a picture drawn for the day, and a quote", "chat apps: connect a channel without showing its secret, and disconnect it", and the reminders test with the weekly day.

### EP-108 — Answering the digest

**Goal.** "The user can directly respond to the message and the answer will be analysed: with Slack, I can say during the day when I have achieved an element of my to-do list, synchronised with everything in the website" (owner, 2026-10-14). Decisions in [ADR 0033](adr/0033-answers-through-outbound-listeners.md).

- [x] **EP-108/IS-108.01** · 3 pts · SHOULD — listeners that only connect out: Telegram long polling, Slack Socket Mode, Discord gateway (`internal/wsclient`, RFC 6455), one per bot, reconnecting with back-off, stopped cleanly; a chat is linked with a one-time code; only the linked person, in a private chat, is obeyed; a flood limit.
- [x] **EP-108/IS-108.02** · 3 pts · SHOULD — commands in English and French: done by number (tasks, habits, milestones), by words or by sentence (Snowball stems; one clear match, or a numbered choice), add with a day and a time (a reminder), note, snooze, list, help, undo; longer free text becomes a fleeting note; the latest numbers are kept (`digest_items`).
- [x] **EP-108/IS-108.03** · 2 pts · SHOULD — Preferences shows each bot's state (linked to whom, a new code) and how to create the bot; the Digest page explains the answers.

**Tests.** `TestParse`, `TestResolve` (English and French commands, numbers and ranges, days, weekdays, times, "buy 2 apples" is not 2 o'clock); `TestAgainstAServer` (the WebSocket client against a standard server, large messages, close); `TestAnsweringTheDigest` (nobody obeyed before linking, a wrong or used code refused, a stranger refused, list, done, already done, undo, a habit by number, an unknown number, a sentence, a choice between close matches, no match, a repeating task, add with a day and a time, undo of an add, snooze, a note, French, a disabled channel); `TestChatListeners` (Telegram, Slack, and Discord each answer a direct message through their real protocol against fakes; group and server messages ignored; the Slack acknowledgement, the Discord identify with the DM intent, and a heartbeat); the race detector on both.

### EP-109 — Sign in with other accounts

**Goal.** "Allow other types of connections like with Google, Microsoft, Discord, Slack, Apple" (owner, 2026-10-14). Decisions in [ADR 0034](adr/0034-sign-in-with-other-accounts.md).

- [x] **EP-109/IS-109.01** · 5 pts · SHOULD — providers from the environment (Google, Microsoft with its tenant, Apple with an ES256 client secret, Discord as OAuth 2, Slack, any OpenID Connect issuer by discovery); the code flow with PKCE, a cookie-bound state, and a nonce; ID token claims checked (issuer, audience and azp, expiry, nonce); sign-in, link, or a new account when registration is open; never matched by email; one identity per account; the last way in kept; audit events; in the personal export; starts rate limited.
- [x] **EP-109/IS-109.02** · 3 pts · SHOULD — buttons on the sign-in page with clear errors; Account → Sign-in & security lists linked accounts (last use, unlink) with link buttons; a new account made with a provider is asked to add a passkey.

**Tests.** `TestProvidersFromEnv`, `TestAppleClientSecret` (the JWT verifies with the key), `TestIDTokenChecks` (issuer, audience, azp, expiry, nonce, subject, Microsoft's tenant issuer and a look-alike, safe redirects); `TestSignInWithAnotherAccount` (PKCE, state, and nonce in the authorize address; an unlinked account refused with registration closed; link; sign-in from a new browser to the asked page; another browser's state, a used state, a wrong nonce, a wrong issuer, a cancel, an off-site next; Discord's user endpoint; unlink); `TestSignUpWithAnotherAccount` (a new account, someone else's account refused, the last way in kept).

### EP-110 — Tech watch

**Goal.** "A database dedicated to tech watch: domains, sources, or RSS feeds defined by hand, by importing a normalised CSV, or by an AI agent; a crontab by default each midnight; new elements stored in a relational database with no duplicates or similarity; NLP labels; each news a feed note with its template; access to Medium, Substack, etc.; easy monitoring" (owner, 2026-10-14). Decisions in [ADR 0035](adr/0035-tech-watch-stores-each-story-once.md).

- [x] **EP-110/IS-110.01** · 5 pts · SHOULD — RSS 2.0, RSS 1.0, Atom, JSON Feed; discovery (Medium, Substack, YouTube, GitHub, Reddit, dev.to, Hacker News, `<link rel="alternate">`, common paths); conditional GET; full text for excerpt-only feeds; cron schedules in the person's time zone (midnight by default, 15 minutes at least); back-off; four fetches at a time; 180-day retention of what nobody kept.
- [x] **EP-110/IS-110.02** · 3 pts · SHOULD — each story once: per-source seen entries, canonical links, SimHash and Jaccard near-duplicates over ten days, an advisory lock per person; merged sources listed as "also in".
- [x] **EP-110/IS-110.03** · 2 pts · SHOULD — labels: topics with thesaurus synonyms and distinct evidence, TextRank key phrases, feed categories, the person's own; outliers set aside; relabelling when a topic changes.
- [x] **EP-110/IS-110.04** · 3 pts · SHOULD — imports (normalised CSV, OPML, FreshRSS's Google Reader API) and OPML export; the feed-note template and **Keep as a note** into `Feeds/`; news as entities linkable from anywhere; the reader (filters, labels, j/k, mark all read, connections) and Sources and health (overview, 30 days, per-source health, schedule presets and cron, fetch now); Prometheus metrics.

**Tests.** Parser tests (RSS with content and enclosures, Atom with YouTube media groups and HTML titles, RDF, JSON Feed, garbage refused); canonical links, SimHash and Jaccard, cron (time zones, weekdays, either-day rule, too frequent refused), known feeds, alternates; analysis (topics with synonyms and distinct evidence, a passing mention refused, outliers, French); `TestTechWatch` (discovery from a blog page, follow, a duplicate refused, the same link and the same story from another source stored once, filters by status, label, words, source, topic, and period, ownership, 304 after an ETag, a failing source retried within 20 minutes, full text read from the article page, the worker, the feed note, keep as a note in `Feeds/`, labels and status, mark all read, CSV with a bad line and a bad schedule, OPML folders as topics, FreshRSS with a wrong and a right password, OPML export with attribution, the overview, schedules, topic and source removal, the retention purge); e2e "tech watch: topics, an imported source waiting for its first fetch, and an empty reader".

### EP-111 — Tech watch analysis & knowledge graph

**Goal.** "With the NLP analysis (chunks, lemmatisation, stopwords, outlier layers, some ML), define a knowledge graph; hovering a node gives a short summary really related to it; distinct from the Zettelkasten graph, or integrated with a clear distinction between feed notes and Zettelkasten notes" (owner, 2026-10-14). Decisions in [ADR 0036](adr/0036-a-news-graph-apart-from-the-notes-graph.md).

- [x] **EP-111/IS-111.01** · 5 pts · SHOULD — `GET /v1/watch/graph` (period, topic, source, link strength, fringe): concepts from key phrases by stems and thesaurus synonyms, found in each item's text; NPMI links; Louvain communities; PageRank; trends against the weeks before; extractive hover summaries (TF-IDF centroid); bridges to literature and permanent notes; the fringe layer.
- [x] **EP-111/IS-111.02** · 3 pts · SHOULD — the **News graph** page: a night-sky constellation apart from the notes' graph; hover or focus a star for its summary, news count, sources, trend, related concepts, and notes; select it to keep it with its news and notes; rising concepts; community legend; a table view; French.

**Tests.** `TestWatchGraph` (two threads of news make two communities; "linux kernel" and "memory safety" together, "pgvector" apart; a hover summary that names its concept; a bridge to a permanent note; no trend without history; another person sees nothing; a bad period and a bad strength refused); `trendWord` and `lastDays`; e2e "tech watch" opens the News graph (empty state, accessibility).

### EP-112 — Tech watch exports & sharing

**Goal.** "Export the news by types, sources, labels (applied by the NLP), period; share them with other people; export them (PDF, ODS, CSV, MD, etc.) with an explicit mention that they come from Axon Project" (owner, 2026-10-14). Decisions in [ADR 0037](adr/0037-watch-exports-and-share-links.md).

- [x] **EP-112/IS-112.01** · 3 pts · SHOULD — `GET /v1/watch/export` in CSV (formula-safe), ODS, Markdown, a zip of feed notes, JSON, and PDF, each with the attribution, its date, and the filter in words; a kind filter (article, video, podcast) in the reader and the API; **Export or share** in the reader, with a period.
- [x] **EP-112/IS-112.02** · 2 pts · SHOULD — share links: made and revoked from a session only, audited, expiring, hashed; a public page and its downloads showing only what sources published, not indexed, no referrer, rate-limited; **Shared links** on Sources and health with views; links in the personal export, without tokens.

**Tests.** Export unit tests (CSV header, formula guard, and attribution row; ODS stored media type, XML, dates, escaping; Markdown by day; feed-note zip with duplicate names; JSON; PDF cross-reference offsets, pages, links, WinAnsi text, a footer on every page; wrapping; the filter in words; kinds); the PDF checked with poppler and the ODS opened by LibreOffice; `TestWatchExportsAndShares` (kinds, exports with a filter, another person's empty export, a bad format, share validation, only Saved kept, a token refused, the public page and downloads without status, name, or ids, views, an unknown, expired, and revoked link, ownership); e2e "tech watch: export the view, share it with a link anyone can open, then revoke it" (a download by plain navigation, a stranger's view, accessibility).

### EP-113 — Podcast, agent, digest template, scenery

**Goal.** "A podcast format, disabled by default but can be activated, with the type of voice, the language, the format; an AI agent dedicated to the tech watch; illustrated backgrounds for the tech watch, analysis, knowledge graph, podcast, and digest, which should be templated but customisable" (owner, 2026-10-14). Decisions in [ADR 0038](adr/0038-podcast-and-digest-template.md).

- [x] **EP-113/IS-113.01** · 3 pts · SHOULD — the podcast, off by default: an episode script from the period's best news in three formats and two languages, sized to a length, ending with Axon Project; read by the device's voices with its pace, chapters, skip, and the script as a file; settings `podcast`.
- [x] **EP-113/IS-113.02** · 2 pts · SHOULD — the digest template (settings `digest`): sections and order per digest, the quote and the picture optional, on every channel; the tech watch's best stories in each digest; **Customise** on the Digest page.
- [x] **EP-113/IS-113.03** · 2 pts · SHOULD — the Tech watch agent: reads topics, sources and their health, news, and the news graph's trends; proposes sources (https) and topics for approval; offered on the reader.
- [x] **EP-113/IS-113.04** · 1 pt · SHOULD — scenery for the tech watch (signal), its graph (constellation), the podcast (studio), and the digest (dispatch), light and dark.

**Tests.** `TestSpeakable`; `TestWatchPodcast` (off until turned on, an episode of six stories with no links, a three-minute briefing within its words, a French deep dive announcing each topic once, bad options and settings refused, an empty week); `TestTemplateFollowsTheSettings`, `TestArrangeOrdersDropsAndRenumbers`, `TestRenderersWithoutQuoteOrPicture`; `TestDigestNewsAndTemplate` (news last and unnumbered in the daily and weekly digests, a template's order, numbering, no quote, no picture, bad templates refused); `TestTechWatchAgent` (tools read as data, news filtered by topic, an http address refused, a topic proposed and applied, then labelling the news) and the two new write tools in `TestEveryWriteToolNeedsApproval`; `spaceFor` for the new spaces; e2e "tech watch: the podcast is off until turned on, then makes an episode" (with the reader's agent), "the digest follows its template", and "themed spaces keep AA contrast" over the four new spaces.

### EP-114 — Decision journal

**Goal.** Axon tracks what a person does but not how well they decide. A decision journal (the habit behind architecture decision records and *Superforecasting*) writes down a choice, what is expected, and how likely it is, before the outcome is known; reviewing it on its day, and scoring the chance against what happened, shows whether one's confidence can be trusted.

- [x] **EP-114/IS-114.01** · 3 pts · SHOULD — Migration `00060` adds `decisions` (title, context, choice, the expected outcome and its chance from 1 to 99 percent, the decision and review days, and once reviewed whether it happened, what happened, and the lessons). `POST /v1/decisions`, `GET /v1/decisions` (`status=open|due|reviewed`), `PATCH` (title, context, choice, review day; the forecast is refused, so the score stays honest), `POST /v1/decisions/{id}/review`, `DELETE`, and `GET /v1/decisions/calibration`: the Brier score, the mean chance given and the share that came true, and the reliability buckets by tenth. In the export; French messages.
- [x] **EP-114/IS-114.02** · 2 pts · SHOULD — Reflect → Decisions: record a decision with a chance slider and a review in one, three, six, or twelve months or on a day; review those due with what happened and what you learned; the calibration as a score with a verdict against always saying 50 percent, over- or under-confidence after ten reviews, and a reliability chart with a table; scenery after Frost's *The Road Not Taken*. The demo has a year of decisions, twelve reviewed, one due, two waiting.

**Tests.** `TestScore` (written first: calibrated, overconfident, the buckets, a certainty); `TestDecisionJournal` against PostgreSQL (create, validation, the forecast fixed, due, review, the score, the export, other people); the verdict, confidence, reliability geometry, and months helpers; an e2e test that records a decision, reviews a due one, and reads the score, with axe.

### EP-115 — Search in the menu bar

**Goal.** "A search bar on the menu bar which can search within all the website." The command center (Ctrl+K or /) already searched every item and note, by words and by meaning, but nothing on screen said so, and it opened only eleven pages.

- [x] **EP-115/IS-115.01** · 2 pts · SHOULD — a Search field in the menu bar, for signed-in people (an icon on phones), opens the command center; the command center opens every page of the menu, and Roadmaps, Progress, the account, its preferences, and the trash.

**Tests.** Every page of the menu is offered and exists as a route (vitest); an e2e test that searches from the menu bar, opens a task, then a page with Ctrl+K.

### EP-116 — Checked items stay

**Goal.** A task checked off vanished from its list at once, and habits could not be checked off on Today. The owner chose: a checked item stays where it is, crossed out, with Undo, and leaves the list the next day, so the day's work stays in sight.

- [x] **EP-116/IS-116.01** · 2 pts · SHOULD — Migration `00061` adds `tasks.done_from`, the list a task was in when it was done or cancelled, cleared when it is reopened. Each task view (all, inbox, next, doing, waiting, someday, today, week, month) returns, on its first page, `done_today`: its tasks done or cancelled today in the person's time zone; the dashboard returns today's. Reopening puts a task back in `done_from`.
- [x] **EP-116/IS-116.02** · 1 pt · SHOULD — Tasks and Today show those tasks crossed out and checked, and unchecking reopens them in their list; Today's habits get a checkbox, and a habit checked in today is crossed out on Today and on Habits, with Undo.

**Tests.** `TestTasksDoneTodayStayInTheirList` (each view, the dashboard, a task done days ago left out, Undo back to its list); an e2e test that checks off a task, reloads, sees it on Today, and unchecks it there, with axe.

### EP-117 — Coffee and tea

**Goal.** "Like water, a coffee/tea consumption tracker which follows scientific papers and determines the quantity we can drink, and when we should stop so as not to harm sleep."

The science: caffeine leaves the body by half every half-life, about 5 hours on average (1.5 to 9.5 in adults; longer in pregnancy and with oral contraceptives, shorter for smokers). EFSA's scientific opinion on the safety of caffeine (2015) finds no concern up to 400 mg a day for adults, and sets 200 mg a day in pregnancy. Gardiner et al. (2023, *Sleep Medicine Reviews*, a meta-analysis of 24 studies) found that caffeine shortens sleep by about 45 minutes, and that a coffee (107 mg) should be drunk at least 8.8 hours before bed to avoid it: what that coffee leaves at bedtime, about 32 mg at a 5-hour half-life, is the threshold Axon uses.

- [x] **EP-117/IS-117.01** · 3 pts · SHOULD — Migration `00062` adds `caffeine_entries` (when, which drink, mg), health data under the health consent, deleted with it and in the export. `GET /v1/health/caffeine` gives the day's total against the limit (400 mg, 200 mg when the profile says pregnant or breastfeeding), what is left now and at bedtime from the day and the two before, the cut-off (the last moment a coffee still leaves no more than the threshold at bedtime, or none), the week, and the drinks with their usual caffeine (USDA FoodData Central, EFSA); `POST` (a drink now or in the last 30 days, its own mg if known), `DELETE`. Settings `caffeine`: bedtime (23:00) and metabolism (fast 3 h, average 5 h, slow 7 h).
- [x] **EP-117/IS-117.02** · 2 pts · SHOULD — A Coffee and tea card on Health and as a Today section: one tap per drink, the day against the limit, a sentence on tonight (a coffee is fine until a time, that time has passed, or no more today), the week with days over the limit, the sources, and "for information, not medical advice"; bedtime and metabolism in Preferences. The demo has a week of coffee and tea, and Today shows its health cards.

**Tests.** The model (written first: decay, cut-off at 8.8 hours with nothing drunk, earlier after a morning coffee, none after a late strong one, earlier for a slow metabolism; bedtime after midnight); `TestCaffeineTracker` (consent, validation, the limit, decay, metabolism and pregnancy, other people, the export, withdrawal); the advice and the drinks' French; an e2e step on Health, with axe.

### EP-118 — Huberman's protocols

**Goal.** "For the healthy habits, you can follow the habits of Andrew Huberman." His podcast's protocols are popular, and some rest on more evidence than others: each one comes with its studies and an honest grade, and is educational, not medical advice.

- [x] **EP-118/IS-118.01** · 3 pts · COULD — Habits → Protocols for health: morning sunlight, waking at the same time, waiting 90 minutes before coffee (graded "his advice, not tested"), the last coffee 9 hours before bed (linked to EP-117), dimming the lights in the evening, cyclic sighing, non-sleep deep rest, zone 2 cardio and strength (the WHO's recommendation), and deliberate cold. Each says what to do and why, grades the evidence (strong, some, not tested), links its sources (Khalsa 2003, Phillips 2017, Gardiner 2023, Gooley 2011, Balban 2023, Kjaer 2002, WHO 2020, Søberg 2021, Huberman Lab's Toolkit for Sleep), warns where needed, and fills in the habit form in one click, in the interface's language. The demo has a Morning sunlight habit.

**Tests.** Each protocol has a reason, an https source, and a habit; the untested one says so; French for every text; an e2e test that opens two protocols, follows a source, and makes a habit, with axe.

### EP-119 — The collection

**Goal.** "Rewards, like cards, trophies, or badges, I can collect with the streak and the experience, difficulties, domains, habits." Points, levels, badges, and the streak (EP-15) reward activity; a collection rewards depth in each thing, and shows what the next step takes.

- [x] **EP-119/IS-119.01** · 3 pts · COULD — `GET /v1/progress/collection` (409 while rewards are off), worked out from what is recorded, no new table: a trophy for each roadmap skill (bronze when started, silver after 10 hours of focus on it, gold when learned); a card for each habit by its best run of scheduled days kept, a rest day neither counting nor breaking it (7, 21, 66 days, the median time to automaticity in Lally et al., 2010); problems solved unaided by difficulty (easy 10/30/60, medium 5/25/60, hard 1/10/30, after the roadmap's LeetCode proofs); focus hours (10/50/200); flashcard reviews (100/1,000/5,000); the longest daily streak (7/30/100); decisions reviewed (5/20/50). Each card has its tier, value, and next tier; the total counts bronze 1, silver 2, gold 3.
- [x] **EP-119/IS-119.02** · 2 pts · COULD — Progress → Your collection: tiers collected and medals, then roadmap trophies, habit cards, and milestones as cards in their metal, locked ones dashed, each with a bar to the next tier and what it takes, in English and French.

**Tests.** `TestBestRun` (rest days, two-minute versions, days off the schedule), the tier steps and the longest run (written first); `TestCollection` (off, a habit kept 7 days, a hard problem, locked cards, the total); the names and next steps; an e2e check on Progress, with axe.

### EP-120 — Invitations

**Goal.** "Functionalities to invite." Registration closes after the first account unless the operator opens it, so a partner could not even join the owner's household. An invitation lets the owner bring in one person at a time, without opening registration to everyone.

- [x] **EP-120/IS-120.01** · 2 pts · SHOULD — Migration `00063` adds `invites` (the creator, the SHA-256 of a 256-bit token, a note, whether it brings the person into the household, expiry, when and by whom it was used). `POST /v1/invites` (the owner, the first account, only; not the demo; 1 to 30 days) answers once with the link `/register?invite=…`; `GET /v1/invites` lists them without links; `DELETE` revokes an unused one, a used one staying as a record. `GET /v1/auth/invite` (rate-limited with sign-in) tells the holder who invites them. Registration accepts the token when registration is closed; the token's hash travels in the WebAuthn ceremony, and the invitation is spent with a conditional update in the transaction that creates the account, so one link makes one account, and the person joins the owner's household when it says so.
- [x] **EP-120/IS-120.02** · 1 pt · SHOULD — Account → Invitations, for the owner: a note, how long, whether into the household; the link shown once with Copy and the device's Share; the list with who used each, and Revoke. The registration page says who invites the person.

**Tests.** `TestInvitations` (registration closed to strangers; validation; who invites; signing up with the link joins the household; used once; a member cannot invite; the list; revoked links open nothing; a used one cannot be revoked); `TestEveryForeignKeyIsIndexed`; an e2e test that creates a link and opens it in another browser, with axe.

### EP-121 — Share as a picture

**Goal.** "Easily share an element, take a 'screenshot' and send it on social networks." A picture travels where a link to a private Axon cannot: it is drawn on the person's device, at the size social networks show (1200 × 630), and nothing leaves until they share it.

- [x] **EP-121/IS-121.01** · 3 pts · COULD — A Share button on the level (Progress), on each card earned in the collection, on a goal's progress, and on a habit's run: a dialog with the picture (a title, the big figure, a sentence, "Made with Axon", in the card's metal), the device's share sheet with the picture where it can share files, Download, Copy, and links that open a post with the words on X, LinkedIn, and Bluesky (the picture to attach). The text is wrapped to the picture's width; one dialog per button, with its own ids.

**Tests.** Wrapping to a width with an ellipsis, and the posting links (written first); what an earned card stands for; an e2e step that opens the level's picture and its X link, with axe.

### EP-122 — A practical knowledge graph

**Goal.** "Improve the UI/UX design of the knowledge graph to be more practical." Labels piled on each other, a large graph could not be explored closer, and a concept could not be found but by eye.

- [x] **EP-122/IS-122.01** · 3 pts · SHOULD — The news graph and the notes' graph: the wheel and a pinch zoom about the pointer (0.25 to 8 times), a drag pans (only after a few pixels, so a click on a node still selects it), + and − zoom about the middle, Fit shows everything, Full screen; Find in the graph (with the names to choose from) selects a node and centres on it; labels are placed most central first and only where they overlap no other, the focused node and its neighbours first, and more appear as you zoom in; lines and labels keep their size on screen.

**Tests.** Zooming about a point, fitting, centring, and label placement (written first); an e2e step on the notes' graph that finds a note, zooms, and fits, with axe.

### EP-123 — YouTube in the tech watch

**Goal.** "Like the tech watch, but with YouTube channels, and advanced NLP analysis following the subtitles (you can generate them), with the title and metadata, in more than one natural language, and with links cleaned." The owner chose: YouTube's captions first, then their own Whisper server.

- [x] **EP-123/IS-123.01** · 2 pts · SHOULD — Links: every form of a YouTube link (`youtu.be`, `m.`, `/shorts/`, `/embed/`, `/live/`) is one video, `youtube.com/watch?v=…` (with `t` kept), and the trackers of YouTube's share button, Facebook, LinkedIn, X, Google, and Microsoft ads are dropped, from each item's link and from the links in its text. A YouTube link of any form is a video, never set aside as too short.
- [x] **EP-123/IS-123.02** · 4 pts · SHOULD — Transcripts: after each round the worker reads three untried videos; Axon asks YouTube's player (as its Android app does) for the captions, takes the author's in the spoken language, else any of the author's, else the automatic ones, and reads them (both caption formats); with none, and `AXON_WHISPER_URL` set, the smallest audio stream (at most 60 MB, 3 hours) goes to that OpenAI-compatible server. The words join the item's text under "Transcript", and the item is analysed again: language, terms, key phrases, topics, labels, score. Migration `00064` records the source, language, and attempt, so a video is tried once; `POST /v1/watch/items/{id}/transcript` tries now (not in the demo). Only the text is kept.
- [x] **EP-123/IS-123.03** · 1 pt · SHOULD — More than one language: Spanish is recognised beside English and French, by its function words (without those that are English words too), and stemmed with Snowball; Whisper's language names become codes.
- [x] **EP-123/IS-123.04** · 1 pt · SHOULD — The reader says where a video's transcript came from and its language, or offers **Read the transcript**.

**Tests.** Canonical links and video ids, links in a text (written first); both caption formats; the track choice; a fake YouTube and Whisper (captions, none, the smallest audio, the token, the language); `TestWatchVideoTranscript` (a `youtu.be` link cleaned and kept as a video, captions that bring a topic, YouTube asked once); Spanish detection and stems, English left alone.

### EP-124 — Podcasts from Audiobookshelf

**Goal.** "The same for podcasts. The podcasts are downloaded somewhere else, on a server, so I can set up the connection to it directly from the preferences." The owner's server is Audiobookshelf.

- [x] **EP-124/IS-124.01** · 3 pts · SHOULD — Migration `00065` adds `audiobookshelf_servers` (the address, and the API token sealed with `AXON_ENCRYPTION_KEY`, never shown again), allows tech watch sources of origin `audiobookshelf`, and indexes episodes waiting for a transcript. `PUT /v1/audiobookshelf` checks the address and token with the server (a private address only with the operator's `AXON_CALENDAR_PRIVATE=1`), saves them, and syncs: every show of the podcast libraries becomes a source, every episode an item (its notes, date, cover, the public file as its audio, else its page on the server). `POST /v1/audiobookshelf/sync` follows new shows and reads new episodes; the worker reads followed shows on their schedule through the origin, the tech watch knowing nothing of Audiobookshelf. `DELETE` stops following, the episodes staying. Not in the YAML settings, so no export carries the token.
- [x] **EP-124/IS-124.02** · 1 pt · SHOULD — With `AXON_WHISPER_URL`, podcast episodes get their words like videos (three per round): an Audiobookshelf episode from the person's own copy on their server, another podcast's from its public file (at most 250 MB).
- [x] **EP-124/IS-124.03** · 1 pt · SHOULD — Preferences → Podcasts from Audiobookshelf: the address, the token (write-only), Connect, Sync now, Disconnect, the last sync and its error; a link from Sources and health. Links keep their scheme, host, and port (a LAN server over http), only their trackers going; identity still uses the canonical form.

**Tests.** The client against a fake server (podcast libraries only, shows, episodes as entries, the audio by a checked guid, a wrong token); `TestAudiobookshelf` (a private address refused, a wrong token, connected and synced without showing the token, a second sync, an episode transcribed from the person's copy, disconnected with the episodes kept); `TestClean`; an e2e check of the Preferences section, with axe.

### EP-125 — A trailer and a gallery

**Goal.** "Improve the README files, add a screenshots gallery, and update the trailer with a really better one, like Apple presents a new product. The music should be less robotic, more dynamic, more aesthetic." The EP-84 teaser recorded a live tour (scrolling in real time, a jerky picture) over the focus timer's simple oscillators.

- [x] **EP-125/IS-125.01** · 2 pts · COULD — `make teaser` now runs `scripts/trailer/make-trailer.sh`: `shots.mjs` takes fourteen screenshots of the demo in the light theme (the visitor's banner hidden in that capture only); `trailer.html` draws the film for any instant (`render(t)`), so `render.mjs` captures 30 frames a second with no live scrolling: a title on black, the name, ten features (a headline beside, or above, the page in a floating window that tilts and pushes in), a grid of nine pages, "Private by design", and the end card, in Inter (SIL OFL, used to make the film only). The soundtrack is composed for it in the browser: detuned pads through a filter that opens section by section, a reverb, a piano motif, bass, soft drums from the first feature, a side-chained pulse, a rise into the full part, and a last chord (D major, I–V–vi–IV, 96 BPM, −15.7 LUFS). 63 seconds, captions in English and French, 1280 × 720, about 5 MB in WebM and MP4, and a poster.
- [x] **EP-125/IS-125.02** · 1 pt · COULD — The README (English and French) opens its screenshots with the trailer's poster linked to the film, then a gallery of twelve pages of today's Axon (Today, tasks, notes, Learn, the roadmap, Focus, the tech watch and its graph, health, decisions, the collection, habits), described for screen readers.

**Tests.** Rendered and measured: loudness and true peak (EBU R128), the spectrogram's sections, preview frames of each kind of scene; `make check` (shellcheck on the script).

### EP-126 — The original and its source

**Goal.** "For the tech watch, YouTube, etc., make sure I can easily find the source's link too." The reader had one "Open the original" button, and the source was plain text.

- [x] **EP-126/IS-126.01** · 1 pt · SHOULD — Each item carries `source_url`, where its source lives (the YouTube channel, the blog, the show; its feed when it gives none). In the list, each item links to its original, named by its site (↗ youtube.com), and to its source. In the reader, the button says what opening it does (Watch on youtube.com, Open the episode on changelog.com, Read on go.dev), the source's name links to it, the full link without trackers is shown with **Copy the link**, and a podcast links to its audio file.

**Tests.** `TestWatchVideoTranscript` checks the source's address; the site names and labels, and their French (written first); an e2e test on the demo's YouTube talk (the list's link, the reader's button, the full link, the copy button, the source), with axe.

### EP-127 — Every new feature in sight

**Goal.** The owner, in the demo on srv-dev, could not find the YouTube videos, the collection, or coffee and tea: they were there, but under a filter, inside Progress, and at the bottom of Today.

- [x] **EP-127/IS-127.01** · 1 pt · MUST — The menu names each feature: Think has **YouTube videos** and **Podcasts** (the tech watch on that kind), Grow has **Huberman protocols** and **Collection** (its own page), Life has **Coffee and tea** (its own page) and **Invite someone**; a link to a part of a page waits for that part to render. The tech watch opens on tabs **Everything · Articles · YouTube · Podcasts** with their counts (`items.articles`, `items.videos`, `items.podcasts` in `/v1/watch/stats`), the kind kept in the address, and each video or episode carries a badge. Today shows **New in Axon**, a card linking each new feature, until closed. The search offers the new pages. Tests: the stats count each kind, the demo has at least five videos and two episodes, and an end-to-end walk of the demo through the menu.

### EP-128 — A shorter board

**Goal.** "Reduce the size of the GTD view: hide when a column has more than a given number of elements." A column of twenty-two cards pushed the others off the screen.

- [x] **EP-128/IS-128.01** · 1 pt · SHOULD — Each column of the GTD board and the kanban shows its first five cards (across its contexts on the GTD board), then a **Show N more** button that opens the rest, and **Show less**. The count in the header stays the whole column.

**Tests.** A unit test of the split; an e2e test on the demo's kanban (five shown, Show more, all shown).

### EP-129 — One set of icons

**Goal.** "The icons for goal, skill and note are not professional and aesthetic; same problem on Notes; for the journal, the icons are really not professional."

- [x] **EP-129/IS-129.01** · 2 pts · SHOULD — An `Icon` component draws one set of line icons (24 × 24, 1.75 stroke, `currentColor`, after Lucide, ISC licence): the kinds of items in place of the emoji and symbols, the notes' side links (explore, analysis, templates), the agents' button, and the journal's mood, energy, and stress (a face that smiles, a battery that fills, a gauge that swings). The icons follow the theme's colour; Preferences → Icons is now Shown or None. The Digest page draws its sections and figures with the same icons; emails and chat apps keep their emoji, which they can show.

**Tests.** Every kind has an icon; the scales keep their labels for screen readers; axe on the journal and notes.

### EP-130 — A journal without the past moods

**Goal.** "We don't need to see our previous mood." (Owner's choice: the journal's history.)

- [x] **EP-130/IS-130.01** · 1 pt · SHOULD — The journal's last 30 days lose the mood and energy chart and the faces: each day stays listed by date with its sleep and first line, to reopen. What helps you (EP-92) is unchanged.

**Tests.** The e2e journal test checks no chart and no faces in the list.

### EP-131 — The life web

**Goal.** "For Life, with food, health and activities, gamify much more, a spider-web chart."

- [x] **EP-131/IS-131.01** · 2 pts · SHOULD — Health opens on a radar of today across six axes scored from 0 to 100: food (energy and protein against the daily needs), water (against the target), sleep (hours against 7 to 9), activity (minutes against 30 a day, WHO), caffeine (under the limit), and mood (the journal). Last week's average is drawn behind today's; the day's score is their mean, with a level (Bronze, Silver, Gold, Platinum), the best day to beat, and the next axis to raise. Axes without data say so. Without a profile, the targets are the EU reference intakes (2,000 kcal, 50 g of protein) and EFSA's 1.6 L of drinks. The menu's Life names **Life web**.

**Tests.** Unit tests of each axis's score and the level; an e2e test of the radar on the demo, with axe.

### EP-132 — A podcast with real voices

**Goal.** "Improve drastically the podcast: the voices are really too robotic." (Owner's choice: a self-hosted Kokoro. Voices of real, famous people are not offered: copying a person's voice without consent is impersonation.)

- [x] **EP-132/IS-132.01** · 2 pts · SHOULD — `compose.yaml` gains an optional `tts` service (Kokoro-FastAPI v0.2.4, CPU image, profile `tts`; 7.8 GB on disk); `AXON_TTS_URL` (with `AXON_TTS_MODEL`, `AXON_TTS_TOKEN`) points the API at any server speaking the OpenAI speech API. `GET /v1/watch/podcast/voices` lists its voices with the language Kokoro's names say; `POST /v1/watch/podcast/speech` speaks one part (at most 3,000 characters, MP3, three minutes to answer). Without the server, the device's voices as before. Settings gain `podcast.hosts` (1 or 2) and `podcast.voice2`.
- [x] **EP-132/IS-132.02** · 1 pt · SHOULD — The podcast page offers the natural voices first (named by voice, accent, and sex), **Hosts: a conversation** where the first host opens and closes and the two take turns between, a player that plays parts back to back with the next one spoken ahead, and the pace sent to the server.

**Tests.** The speech handler against a fake server (size limit, error); the conversation's turns; the e2e test with a fake voice.

### EP-133 — Send the digest from its page

**Goal.** "The digest is really cool; add the ability to send it by mail, Discord, Slack, Teams…" It already goes to all of them (EP-107), but the setup sat in Preferences.

- [x] **EP-133/IS-133.01** · 1 pt · SHOULD — The Digest page shows **Where it goes**: the email address (saved there; a note when the operator has no mail server), how many devices get notifications, and the chat apps (Slack, Discord, Microsoft Teams, Telegram, ntfy, a web address), added, linked, and disconnected there. **Send it to me now** tests them all; **Times** leads to Preferences.

**Tests.** The e2e digest test finds the channels on the Digest page.

### EP-134 — Every cover

**Goal.** "For films, series, and books, ensure each element has a cover!" The demo's Dune: Part One had none: TMDB calls it Dune (2021).

- [x] **EP-134/IS-134.01** · 1 pt · MUST — When no candidate has the same title, one whose title starts the item's (or the other way round, at a word) matches if the year is the same on both sides, or the authors overlap. The media page looks for missing covers by itself when some lack one, once per visit; the demo's films carry their year.

**Tests.** `TestBestMatch` with Dune and a near title of another year; the demo shows no drawn cover.

### EP-135 — Media cards worth opening

**Goal.** "The cards are too small and not really useful."

- [x] **EP-135/IS-135.01** · 2 pts · SHOULD — The media page shows larger cards: the cover, the title, the creator and year, the rating as stars, the progress with what is left, and the next step (start, update, finish, rate). Each opens the item's page (`/media/{id}`, where links now lead): the cover large, the overview, genres, progress with quick steps (+10, +20, +50 pages; +1 episode; +30 minutes), status, rating, review, and its connections.

**Tests.** e2e: open an item from its card, change its progress.

### EP-136 — Notes for films, series, and books

**Goal.** "A movie, a series, and a book should have a template note, like the literature, permanent, and resource notes, made for their purpose."

- [x] **EP-136/IS-136.01** · 1 pt · SHOULD — Three vault templates, Film, Series, and Book: their details filled from the item (title, creator, year, genres, cover), and sections for why it, quotes and scenes, ideas, what it changes, the rating and a verdict; a book adds a reading log and chapters, a series its seasons. They are literature notes (`Zettel/literature/`, `source_type: film | series | book`), so they join the knowledge graph and its retrieval tests; **Write a note** on the item's page creates one from its template, linked to the item. The vault's Templates/ gets them too.

**Tests.** The templates fill their placeholders; e2e: a note from a book.

### EP-137 — A reading habit turns the pages

**Goal.** "If I have the habit Read 20 pages today, connected to a book like Dune, the pages read should increase and the pages to read decrease."

- [x] **EP-137/IS-137.01** · 1 pt · SHOULD — A habit linked to a book or series (Connections) with a number in its name or its target ("Read 20 pages", "Watch 1 episode") moves that item's progress forward by that number when a day becomes done, back when it is undone or skipped, between 0 and its total; a book wanted becomes in progress. The two-minute version moves nothing. The demo's Read 20 pages turns Designing Data-Intensive Applications.

**Tests.** API: check in, the progress goes up; uncheck, it goes back; the total caps it.

### EP-138 — Budget charts

**Goal.** "A pie chart of the expenses of a period, by default a month, per category; the same for income; and a chart of expenses and income over a year."

- [x] **EP-138/IS-138.01** · 2 pts · SHOULD — Budget → Reports opens on a period (this month by default, any month or range) with two donuts, spending and income by category, each slice named with its amount and share, and a table for screen readers; below, a year of income and spending by month as paired bars with the balance, and the figures as a table. `GET /v1/budget/report?month=2026-09` ends the report with that month. A category keeps its colour (eight hues validated for colour vision deficiency, light and dark) whatever the period; past seven, Other. The demo has a year of money.

**Tests.** Unit tests of the slices; e2e on the demo's budget, with axe.

### EP-139 — A clean start

**Goal.** "By default, on one's own setup, there is no note, but still templates and courses; with the AI agents one can set up everything for one's needs in a few hours."

- [x] **EP-139/IS-139.01** · 1 pt · MUST — A new account starts with no notes, tasks, habits, goals, projects, roadmaps, media, or sources (checked: registration creates only the account; the demo keeps its sample); the note templates, the library roadmap with its courses, and Huberman's protocols are offered. `doc/getting-started.md` and `doc/getting-started.fr.md`, linked from both READMEs, walk through a first afternoon with the agents: preferences and a provider, the roadmap, goals, projects, habits, tasks, health, notes, the tech watch, the digest, the budget, the weekly review.

**Tests.** A new account has zero notes and the templates; the docs check.

### EP-140 — The life web on Today

- [x] **EP-140/IS-140.01** · 1 pt · SHOULD — An e2e test of the life web on the demo (the score, the radar, the menu's link), with axe. Today's **New in Axon** lists the life web, the podcast's natural voices, films, series, and books, and the budget charts, each with a line icon instead of an emoji, and shows again for this release.

### EP-141 — The Top-5% roadmap, version 2

**Goal.** "Improve drastically more this roadmap. It should be the best roadmap to be one of the best engineers (DSA, CS, OS, Linux, Bash, Python, C/C++, Java, Rust, Docker, Podman, Kubernetes, DevOps, DataOps, DevSecOps, data-intensive, Kafka, Spark, data engineering, data analysis, data science, machine learning, deep learning, maths, statistics, probability, API design, system design, web development, cybersecurity, cloud…), with the certifications (one for each domain), the skills assessment, projects… and it starts Monday 19 October 2026" (owner, 2026-10-17).

- [x] **EP-141/IS-141.01** · 3 pts · MUST — The library roadmap has 35 hard domains in eight sections and its six soft skills, 122 resources, and 18 phases of 12 weeks (the last 16) from Monday 2026-10-19 to 2031-01-05, each with its tracks, a project that builds on the previous one, an exit check, and 42 projects where they fit. Every domain has a certification taken at the end of a phase that works on it, 43 in all: the nine new ones are the Mathematics for Machine Learning Specialization (Imperial College), LFD103 (Linux kernel development), LPIC-1 (Bash), a course from a Rust Foundation accredited trainer (no individual Rust exam exists), Oracle 1Z0-071 (SQL), EDB PostgreSQL Associate and Professional, Cloud Computing Concepts 1 and 2 (UIUC, distributed systems), IEEE CS Professional Software Developer, and MuleSoft API Design Associate. Each domain also has project work that proves it beyond the exam. The goal's outcome, the roadmap's note (domain matrix, certification table, budget), and the certification fund follow.
- [x] **EP-141/IS-141.02** · 2 pts · MUST — The skills assessment has a domain for every hard domain: Bash, database administration, and software engineering are assessed apart (37 domains, five questions each from easy to hard). Every phase's exit check ends with retaking the assessment for its domains. Starting the plan again skips the certifications (by code) and habits (by title) the person already has.

**Tests.** Every hard domain has a certification in a phase that works on it, and its own assessment domain; phases follow day after day from 2026-10-19; starting the plan plans 43 certifications; the assessment serves 185 questions without answers.

### EP-142 — The plan and its milestones in sight

**Goal.** "There is no milestones actually, why???" (owner, 2026-10-17): the milestones were only made when the roadmap was started, and shown on the goal.

- [x] **EP-142/IS-142.01** · 2 pts · MUST — `GET /v1/roadmaps/{id}/plan` shows a started plan phase by phase: dates, past, current, or coming, the milestone (tick it from there), the project and its board's progress, the exit check, the 42 projects, and the certifications with their status. `GET /v1/roadmaps/sources/{source}/plan` lays out the library plan at the dates starting it today would give, so the library card's **See the plan** shows its 18 milestones, projects, and certifications before a start.
- [x] **EP-142/IS-142.02** · 1 pt · MUST — A roadmap started from an earlier version of the plan (none of its milestones is a current phase) says so and offers **Start the new version**; the earlier one then points to the newer, and the calendar's study blocks come from the latest started roadmap of each plan only.

**Tests.** The preview before a start (no milestone, projects without ids, PCPP1 in phase 1, the assessment retake last); a roadmap made outdated in the database is flagged, then names the newer one; the second start keeps 43 certifications and 4 habits; the calendar shows one week of blocks, from the new roadmap.

### EP-143 — Notes drawn, in Axon or in the vault

**Goal.** "Ensure the notes could be drawn on the website or in a Markdown vault; I should define it somewhere" (owner, 2026-10-17).

- [x] **EP-143/IS-143.01** · 1 pt · MUST — **Where your notes live** opens the Notes page: Axon only, or Axon and a vault on a server (paired with the vault server, in step both ways within seconds) with its state, or a folder kept in step by hand with `axon notes-sync -watch`.
- [x] **EP-143/IS-143.02** · 2 pts · MUST — The editor's **Draw** (✎) opens a sheet with a pen, a marker, lines, arrows, boxes, ellipses, text, an eraser, six colours, three thicknesses, undo and redo. It saves an SVG under `Attachments/` and embeds it, so it shows in Axon, in Obsidian, and in the vault's Git history; on a line with a drawing Axon made, Draw opens it again and the saved one replaces it. The shapes ride in the SVG's `<metadata>` and are rebuilt field by field when read back, since a file can come from anywhere.

**Tests.** Unit tests: the SVG round trip with escaped text, a crafted file's shapes dropped, smoothing, the eraser's reach, finding a drawing's embed on a line.

### EP-144 — YouTube and Podcasts their own pages

- [x] **EP-144/IS-144.01** · 2 pts · SHOULD — The tech watch's YouTube and Podcasts tabs each get their own scenery (a cinema screen after the Lumière Cinématographe, a 1930s wireless set; light and dark, from `scripts/spaces.py`), their own title, and a follow panel: the channels or shows followed, a field to follow one more, and suggestions for the roadmap. The kind tabs and each story's badge use the line icons.

**Tests.** The spaces test maps the tabs to their scenery; the e2e demo walk finds the badges.

### EP-145 — Your roadmap today

**Goal.** "Deliver the best website you can to guide a self-learner in their learning journey" (owner, 2026-10-17).

- [x] **EP-145/IS-145.01** · 2 pts · MUST — `GET /v1/roadmaps/today` and Today's **Your roadmap today**: the phase of the latest started plan (or when it starts), its week, the days to its milestone, the milestones reached, today's study block, the phase's project with its board, and the next certification not passed, with **Focus on today's block**, review, practice, and the plan. Without a plan it points to the assessment and the roadmap. Settings keep `today.hidden`, the sections turned off, so a section new in a release (this one) shows on a saved Today until it is turned off; the onboarding's "Learning" adds it.

**Tests.** The endpoint before and after a start; settings keep and validate `today.hidden`, and their defaults include the roadmap.

### EP-146 — Database architecture and GitOps

**Goal.** "Add also to the roadmaps, certifications, projects: database architecture, database administration, GitOps, DevOps, DevSecOps" (owner, 2026-10-17). Database administration, DevOps, and DevSecOps were domains already.

- [x] **EP-146/IS-146.01** · 1 pt · MUST — Database architecture (data modelling, indexing and partitioning strategy, polyglot persistence; phases 5, 6, 18) and GitOps (Argo CD, Flux, progressive delivery, rollback by revert; phases 11, 12) are domains 15 and 34 of 37, each with its resources, its assessment domain (database architecture is new: 38 domains, 190 questions), its exit checks (an architecture decision record per raft-polystore store; every environment rebuilt from Git alone), and its certification: Google Cloud Professional Cloud Database Engineer (phase 6) and the CNCF GitOps Certified Associate (phase 11): 45 in all.
- [x] **EP-146/IS-146.02** · 1 pt · MUST — The plan has a version, and migration `00066` stores the one each roadmap was started from (1 for those started on 5 October 2026, version 1's own date, 2 after). Started, outdated, and the library card's "Open the roadmap" read the plan's id and version, never the title, so the roadmap can be renamed and version 1 (which shares five phase titles with version 2) is still recognised. A certification the plan gained after a start is planned from the plan view with **Plan it**.

**Tests.** Every domain has a certification and an assessment domain (37 and 45); a roadmap set to version 1 is listed as not current and flagged outdated; the assessment serves 190 questions.

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
| 2026-10-12 | The owner merged EP-47 to EP-68 into `main`. |
| 2026-10-12 | Owner request: "fix everything, refactor everything, clean code, TDD, documentation". Added EP-69. |
| 2026-10-12 | Owner request: "improve and fix everything, and other essential functionalities". Added and delivered EP-65 to EP-68: audit fixes, trash and undo, share to Axon, task reminders at a time. |
| 2026-10-14 | Owner request: the tech watch, as EP-110 (sources, schedule, no duplicates, labels, feed notes, monitoring), EP-111 (analysis and its knowledge graph), EP-112 (exports and sharing), EP-113 (podcast, agent, digest template, scenery). |
| 2026-10-14 | Owner requests: EP-106 (where the vault lives), EP-107 (digests on every channel), EP-108 (answering the digest), EP-109 (sign in with other accounts). |
| 2026-10-16 | Owner requests: a cover for every film, series, and book; larger media cards; media note templates; a reading habit that turns a book's pages; budget charts; a clean start for a new install. Planned as EP-134 to EP-139, before EP-132 and EP-133. |
| 2026-10-16 | Owner requests: a shorter board, one set of icons, no past moods in the journal, a life radar, natural podcast voices (self-hosted Kokoro; famous people's voices declined), and the digest's channels on its page. Planned as EP-128 to EP-133. |
| 2026-10-17 | Owner request: a far better Top-5% roadmap (a certification for every domain, the assessment, projects, milestones, a start on Monday 19 October 2026), notes drawn in Axon or in the vault, YouTube and Podcasts their own pages, and a site that guides a self-learner. Planned and delivered as EP-141 to EP-145. |
| 2026-10-17 | Owner requests: database architecture and GitOps in the roadmap, Jarvis, the vault verified, video games, YouTube keywords and summaries, the budget charts, token compaction (rtk itself compresses shell output for coding agents, so Axon compacts its agents' tool results instead), maths in notes, the roadmap's resources as notes, a model per agent, a clean-up, a news digest, a banner, the trailer, monitoring, a full check, then 1.0. Planned as EP-146 to EP-162. |
| 2026-10-15 | Owner requests: a search bar in the menu bar, checked items that stay, a caffeine tracker, Huberman's habits, collectible rewards, invitations and sharing, a better knowledge graph, YouTube with subtitles, podcasts from Audiobookshelf, screenshots and a new trailer. Planned as EP-115 to EP-125 (sharing split into invitations, EP-120, and sharing an item, EP-121); EP-115 to EP-125 delivered. |
| 2026-10-15 | Owner request: "continue to improve, by adding new features". Added and delivered EP-114, the decision journal. |
| 2026-10-15 | The work asked for on 2026-10-14 (vault server, digests, answers, sign-in, tech watch) was built in a cloud session from `main` as EP-69 to EP-76, numbers this stack already used: renumbered EP-106 to EP-113 (migrations 00054 to 00059, ADRs 0031 to 0038) and rebuilt on top of the stack on `dev`, which `srv-dev` serves. |
| 2026-10-12 | Owner requests: EP-50 (diagrams), then EP-51 pictures in notes, EP-52 learning and memory engine, EP-53 agents on every page, and EP-54 to EP-62 for the twelve proposed features. |
| 2026-10-12 | Owner request: EP-49 audit; fixes and legacy removal delivered. |
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
