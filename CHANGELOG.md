# Changelog

All notable changes to Axon. Versions follow [Semantic Versioning](https://semver.org/); the support and deprecation policy is in [doc/support.md](doc/support.md).

## Unreleased

### Added

- **Where the vault lives** (EP-69): notes stay in the database by default; Preferences → Markdown vault names a server and folder, gives a one-line install that pairs with a one-time code, and starts an empty vault with its folders and templates. Axon shows whether the vault is in step.
- **Digests everywhere** (EP-70): daily, weekly, and monthly digests by email (HTML with a picture drawn for the day), Slack, Discord, Microsoft Teams, Telegram, ntfy, or a signed webhook, with a quote chosen for the day; a Digest page to preview them.
- **Answering the digest** (EP-71): in Slack, Discord, or Telegram, "done 2", a sentence such as "I finished the passport", "add … tomorrow at 3pm", "snooze", "note", and "undo" act in Axon at once; the bots connect out, so a private instance needs no public address.
- **Sign in with other accounts** (EP-72): Google, Microsoft, Apple, Discord, Slack, or any OpenID Connect provider the operator configures, linked by their owner and never matched by email.
- **Tech watch** (EP-73): sources on a schedule (midnight by default) from feeds, sites, Medium, Substack, YouTube, GitHub, CSV, OPML, or FreshRSS; each story stored once; labels by local NLP; feed notes; a reader and a health page; Prometheus metrics.
- **News graph** (EP-74): the concepts of your news, how strongly they go together, their communities, and what is rising, drawn apart from your notes' graph; hover a concept for a summary taken from its own news and the notes that share it.
- **Tech watch exports and share links** (EP-75): download the news a view shows, by kind, source, topic, label, and period, as PDF, ODS, CSV, Markdown, feed notes, or JSON, each marked as coming from Axon Project; share a read-only link that expires, shows only what the sources published, and can be revoked at once.
- **A podcast of the news and a digest template** (EP-76): turn on a spoken episode of the day's or week's best news, as a briefing, a digest, or a deep dive by topic, in English or French, read by your device's voice; choose the sections of each digest, their order, and whether the quote and picture come along; the digests now carry the tech watch.
- **The Tech watch agent and new scenery** (EP-76): an agent that briefs you on your news, spots what is rising that no topic covers, and proposes sources and topics for you to approve; illustrated backgrounds for the tech watch, its graph, the podcast, and the digest.

## 1.0.0 (release candidate)

The first release meant for someone other than its author to install, operate, upgrade, and recover. Upgrading from 0.1.0 keeps all data ([doc/upgrading.md](doc/upgrading.md)).

### Added since 0.1.0

- **Second Brain**: Markdown notes with wikilinks, backlinks, tags, PARA folders, templates for every note type kept in the vault, live two-way vault sync, an explorer with a knowledge graph, and local NLP analysis of literature and permanent notes (EP-06, EP-25, EP-36, EP-37, EP-46).
- **Action**: GTD and Kanban task boards, a month calendar with iCal feeds, projects with WIP limits, Areas, Resources, and Archive pages, and templates for goals, milestones, habits, and projects (EP-07, EP-08, EP-13, EP-24, EP-43).
- **Growth**: goals with explainable progress, Atomic Habits design and scorecards, roadmaps from a curated library and OSSU with a skills assessment, FSRS flashcards with Anki export and sync, Pomodoro focus, opt-in progress rewards (EP-07, EP-10, EP-14, EP-15, EP-16, EP-27, EP-34, EP-35, EP-41, EP-42).
- **Life**: Firefly III–style finance with savings pots, film, series, and book trackers with TMDB and Open Library, consent-gated food and training logs with Open Food Facts and USDA, and weekly insights with a weekly review (EP-17, EP-18, EP-20, EP-39, EP-45).
- **Agents**: approval-gated AI agents on the provider each person chooses, with budgets, skills, and evals; coach agents with their own consent and enforced health safety (EP-19, EP-22, EP-26, EP-44).
- **Platform**: a detailed dashboard, settings in the app or YAML, English and French, keyboard shortcuts and Vim mode, profile pictures, themed spaces, a demo account, an OpenAPI reference with Swagger UI, and full instance export and import (EP-28 to EP-33, EP-38, EP-40).
- **Operations**: an install guide, an upgrade guide and test, a load-testing tool with measurements (ADR 0021), and a support policy (EP-21).

### Security

- Recovery sign-in locks for 15 minutes after five wrong authenticator codes for an account, and the failures are audited.
- `Permissions-Policy` header; reachable-vulnerability scanning (govulncheck) in CI; the Go toolchain is pinned to 1.26.9.

### Before tagging

- External security review with no open high-severity issue (see [doc/security/review-2026-10.md](doc/security/review-2026-10.md)).
- A new operator installs from [doc/install.md](doc/install.md) while observed.

## 0.1.0

Accounts with passkeys, recovery codes, and TOTP; privacy controls and export; the universal graph API; tasks with GTD views and recurrence; a daily dashboard; CI and deployment to staging and production.
