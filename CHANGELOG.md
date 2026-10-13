# Changelog

All notable changes to Axon. Versions follow [Semantic Versioning](https://semver.org/); the support and deprecation policy is in [doc/support.md](doc/support.md).

## Unreleased (1.1)

Owner requests delivered after the 1.0 release candidate: EP-47 to EP-68 are merged into `main` (12 October 2026); EP-69 is in review in pull request #81.

### Added

- **Notes and learning**: a Zettelkasten knowledge graph, every item (tasks, goals, habits, media, health) as a note in the vault both ways with visible templates, diagrams GitHub can display, pictures in notes, and a learning and memory engine (free recall, memory palaces, illustrated flashcards, an automatic day plan) (EP-47, EP-48, EP-50, EP-51, EP-52).
- **Everyday use**: agents on every page, navigation groups, an installable app with offline capture, reminders by push and email, quick add for anything, task reminders at a time of day, bulk actions and dragging tasks onto the calendar, and a customisable Today (EP-53 to EP-57, EP-63, EP-64, EP-68).
- **Connections**: follow other calendars (ICS, CalDAV), imports from Todoist, TickTick, Goodreads, Letterboxd, Firefly III, OFX, and Notion, household lists shared with others, search by meaning with a chosen embedding service, and share to Axon from any app or page (EP-58, EP-60, EP-61, EP-62, EP-67).
- **Safety nets**: a guided weekly review, and a trash: every deletion can be undone for 30 days (EP-59, EP-66).
- **Skills radar**: Roadmaps shows the hard and the soft skills acquired as two radar charts (EP-82).
- **Focus music**: music composed in the browser as you work, matched to what you focus on: lo-fi beats, ambient, or calm piano; nothing streamed (EP-83).
- **Teaser**: with the demo on, the home page shows a one-minute tour of Axon, recorded by `make teaser` (EP-84).

### Fixed

- Aligned columns in Markdown tables align again: the alignment was a style attribute, which the Content Security Policy blocks.
- New accounts start in the browser's time zone instead of UTC, so "today" is right from the first night (EP-69).
- Sign-in no longer fails on a user agent that is not valid UTF-8; imports, clipped pages, and calendars in other encodings keep their accents (EP-69).
- Dates no longer show a day early west of UTC; workouts dated after the account's today stay listed (EP-69).
- Two audits' fixes: vault templates follow the mirror folder, learning days follow the time zone, imports are size-capped, and legacy code is removed (EP-49, EP-65).

### Changed

- Shared, tested helpers for the person's day, dates, and stored text replace their copies; see "Shared helpers" in [CONTRIBUTING.md](CONTRIBUTING.md) (EP-69).

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
