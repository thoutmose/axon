# Changelog

All notable changes to Axon. Versions follow [Semantic Versioning](https://semver.org/); the support and deprecation policy is in [doc/support.md](doc/support.md).

## Unreleased (1.1)

Owner requests delivered after the 1.0 release candidate: EP-106 to EP-113 (the vault server, digests on every channel and answering them, sign-in with other accounts, and the tech watch) are on the `dev` branch, on top of the stack; EP-47 to EP-68 are merged into `main` (12 October 2026); EP-69 is in review in pull request #81, EP-70 to EP-85 in pull request #82, EP-86 and EP-87 in #83, EP-88 to EP-90 in #84, EP-91 to EP-105 in #85.

### Added

- **The Top-5% roadmap, version 2** (EP-141): 35 domains in 18 phases from Monday 19 October 2026 to January 2031, a certification for every domain (43 in all) with the project work that proves it, the skills assessment with a domain each (37) and retaken at every phase's end.
- **The plan and its milestones in sight** (EP-142): the roadmap's page shows the plan phase by phase, and the library shows it before a start (`GET /v1/roadmaps/sources/{source}/plan`); a roadmap started from an earlier version of the plan says so and starts the new one, and the calendar shows only the latest.
- **Drawing in notes, and where notes live** (EP-143): a sketch sheet in the editor saves an SVG the vault shows too, and opens again for more drawing; the top of Notes says whether notes live in Axon only or also in a vault kept in step.
- **YouTube and Podcasts their own pages** (EP-144): each tab has its scenery, title, and a follow panel with suggestions for the roadmap.
- **Your roadmap today** (EP-145): Today shows the plan's phase, today's study block, its project, and the next certification (`GET /v1/roadmaps/today`); sections new in a release show on a saved Today until turned off (`today.hidden`).
- **Today's what is new** (EP-140): the life web, natural podcast voices, films, series, and books, and the budget charts, each with a line icon.
- **The digest's channels on its page** (EP-133): email, notifications, and the chat apps (Slack, Discord, Microsoft Teams, Telegram, ntfy, a web address) set up on the Digest page.
- **A podcast with natural voices** (EP-132): the podcast is spoken by your own speech server (the bundled Kokoro with `docker compose --profile tts up -d` and `AXON_TTS_URL`, or any server with the OpenAI speech API), by one host or two in conversation; without it, the device's voices as before. Voices of real people are not offered.
- **A clean start** (EP-139): a new account has nothing of anyone else's, only the templates and the library; [Your first afternoon](doc/getting-started.md) walks through setting Axon up with the agents.
- **Budget charts** (EP-138): Reports opens on a month with donuts of where the money went and where it came from, and a year of income against spending; `GET /v1/budget/report?month=` picks the month.
- **A reading habit turns the pages** (EP-137): a habit linked to a book or series with a number in its name ("Read 20 pages") moves its progress when done, and back when undone.
- **Notes for films, series, and books** (EP-136): their own templates, as literature notes, created and linked from the item's page.
- **Media cards worth opening** (EP-135): larger cards with stars, genres, progress, and what is left, and a page for each film, series, and book.
- **Every cover** (EP-134): a closer match finds covers the catalogues name differently (Dune for Dune: Part One), looked for by themselves.
- **The life web** (EP-131): a radar of the day across food, water, sleep, activity, caffeine, and mood, with a level, the week's best to beat, and what to raise next.
- **One set of icons** (EP-129): line icons replace the emoji for kinds of items, the notes' links, the agents' button, the journal's scales, and the digest page.
- **A journal without past moods** (EP-130) and **a shorter board** (EP-128): the journal's last 30 days no longer show moods; board columns show five cards, then Show more.
- **Every new feature in sight** (EP-127): the menu names YouTube videos, Podcasts, the Collection, Coffee and tea, the Huberman protocols, and invitations; the tech watch opens on tabs by kind with their counts; Today lists what is new.
- **The original and its source** (EP-126): every news item, video, and episode links to its original and to its source (the channel, the blog, the show), and the reader shows the full link, without trackers, to copy.
- **A new trailer and a gallery** (EP-125): a one-minute trailer drawn frame by frame, with a soundtrack composed for it, on the home page and at the top of the README, and a gallery of twelve pages of today's Axon.
- **Podcasts from Audiobookshelf** (EP-124): connect your Audiobookshelf in Preferences and every podcast it downloads follows in the tech watch, each episode with its notes and, with a Whisper server, its words read from your own copy.
- **YouTube in the tech watch** (EP-123): videos get their words from YouTube's captions, or from your own Whisper server when they have none, and those words join their analysis, topics, and the news graph; Spanish is understood beside English and French; links lose their tracking (`si`, `fbclid`, `utm_*`…), and every form of a YouTube link is one video.
- **A practical knowledge graph** (EP-122): both graphs zoom and pan with the wheel, a drag, or a pinch, fit, go full screen, find a node by name, and place their labels without overlap.
- **Share as a picture** (EP-121): share your level, a card of your collection, a goal, or a habit's run as a picture made on your device, through the share sheet, a download, a copy, or a post on X, LinkedIn, or Bluesky.
- **Invitations** (EP-120): the owner invites one person at a time with a link used once, even while registration is closed, and can bring them into the household (Account → Invitations).
- **The collection** (EP-119): on Progress, cards to collect and raise from bronze to silver to gold: a trophy for each roadmap skill, a card for each habit by its best run, and milestones for problems by difficulty, focus hours, reviews, the daily streak, and decisions; each shows what the next tier takes.
- **Huberman's protocols** (EP-118): on Habits, Andrew Huberman's health habits (morning sunlight, a regular wake time, the caffeine cut-off, dim evenings, cyclic sighing, rest, zone 2, cold) with what to do, why, how strong the evidence is, the studies, cautions, and a habit in one click.
- **Coffee and tea** (EP-117): log each coffee or tea in one tap; see the day against EFSA's limit (400 mg, 200 mg when pregnant or breastfeeding), the caffeine left at bedtime, and the last time a coffee keeps your sleep intact (Gardiner et al., 2023), set to your bedtime and metabolism.
- **Checked items stay** (EP-116): a task or habit checked off stays in its list, crossed out, until the next day; uncheck it to put it back. Habits can be checked off on Today.
- **Search in the menu bar** (EP-115): a Search field next to Account opens the command center, which now opens every page as well as finding your items and notes.
- **A decision journal** (EP-114): write a decision down with what you expect and how likely it is; on its review day, say what happened and what you learned; Axon scores your forecasts (Brier score, a reliability chart, over- or under-confidence), and a forecast cannot be changed once made.
- **The Tech watch agent and new scenery** (EP-113): an agent that briefs you on your news, spots what is rising that no topic covers, and proposes sources and topics for you to approve; illustrated backgrounds for the tech watch, its graph, the podcast, and the digest.
- **A podcast of the news and a digest template** (EP-113): turn on a spoken episode of the day's or week's best news, as a briefing, a digest, or a deep dive by topic, in English or French, read by your device's voice; choose the sections of each digest, their order, and whether the quote and picture come along; the digests now carry the tech watch.
- **Tech watch exports and share links** (EP-112): download the news a view shows, by kind, source, topic, label, and period, as PDF, ODS, CSV, Markdown, feed notes, or JSON, each marked as coming from Axon Project; share a read-only link that expires, shows only what the sources published, and can be revoked at once.
- **News graph** (EP-111): the concepts of your news, how strongly they go together, their communities, and what is rising, drawn apart from your notes' graph; hover a concept for a summary taken from its own news and the notes that share it.
- **Tech watch** (EP-110): sources on a schedule (midnight by default) from feeds, sites, Medium, Substack, YouTube, GitHub, CSV, OPML, or FreshRSS; each story stored once; labels by local NLP; feed notes; a reader and a health page; Prometheus metrics.
- **Sign in with other accounts** (EP-109): Google, Microsoft, Apple, Discord, Slack, or any OpenID Connect provider the operator configures, linked by their owner and never matched by email.
- **Answering the digest** (EP-108): in Slack, Discord, or Telegram, "done 2", a sentence such as "I finished the passport", "add … tomorrow at 3pm", "snooze", "note", and "undo" act in Axon at once; the bots connect out, so a private instance needs no public address.
- **Digests everywhere** (EP-107): daily, weekly, and monthly digests by email (HTML with a picture drawn for the day), Slack, Discord, Microsoft Teams, Telegram, ntfy, or a signed webhook, with a quote chosen for the day; a Digest page to preview them.
- **Where the vault lives** (EP-106): notes stay in the database by default; Preferences → Markdown vault names a server and folder, gives a one-line install that pairs with a one-time code, and starts an empty vault with its folders and templates. Axon shows whether the vault is in step.
- **Notes and learning**: a Zettelkasten knowledge graph, every item (tasks, goals, habits, media, health) as a note in the vault both ways with visible templates, diagrams GitHub can display, pictures in notes, and a learning and memory engine (free recall, memory palaces, illustrated flashcards, an automatic day plan) (EP-47, EP-48, EP-50, EP-51, EP-52).
- **Everyday use**: agents on every page, navigation groups, an installable app with offline capture, reminders by push and email, quick add for anything, task reminders at a time of day, bulk actions and dragging tasks onto the calendar, and a customisable Today (EP-53 to EP-57, EP-63, EP-64, EP-68).
- **Connections**: follow other calendars (ICS, CalDAV), imports from Todoist, TickTick, Goodreads, Letterboxd, Firefly III, OFX, and Notion, household lists shared with others, search by meaning with a chosen embedding service, and share to Axon from any app or page (EP-58, EP-60, EP-61, EP-62, EP-67).
- **Safety nets**: a guided weekly review, and a trash: every deletion can be undone for 30 days (EP-59, EP-66).
- **Skills radar**: Roadmaps shows the hard and the soft skills acquired as two radar charts (EP-82).
- **Focus music**: music composed in the browser as you work, matched to what you focus on: lo-fi beats, ambient, or calm piano; nothing streamed (EP-83).
- **Teaser**: with the demo on, the home page shows a one-minute tour of Axon, recorded by `make teaser` (EP-84).
- **Study hours**: focus time on a roadmap against its plan, day by day with the plan's block and over 12 weeks, by skill and project, with a shortcut to focus on today's block (EP-105).
- **Certifications**: exams planned, booked, passed, or failed, with countdowns, costs, credentials, and expiries, on the calendar; the roadmap plans its nine (EP-104).
- **Practice log**: problems, wargame levels, and labs from a pasted link, each attempt solved unaided, with help, or not, re-solved at growing intervals, with stats by difficulty, topic, and week (EP-103).
- **Water**: log a glass or a bottle in one tap on Health or Today, against the drinks' share of your daily water need (EFSA), with the week, and in What helps you (EP-101).
- **Your vault's templates**: the SecondBrain vault's nineteen templates ship as Axon's own, byte for byte (Zettelkasten, PARA, daily, weekly, and monthly notes, roadmaps, goals, milestones, habits, media, and six retrieval tests), filed where the vault keeps them, with their placeholders filled (EP-100).
- **Every page its picture**: Journal, Insights, the weekly review, and Agents get their own scenery (Vermeer's evening light, an observatory, a lighthouse, clockwork automata); every themed page says what its picture shows and the painting, film, or game it nods to; each type of note has a small drawing and a sentence on what it is for (EP-98).
- **A year you can read**: the 12-month activity chart is laid out by Monday-first weeks with month and weekday labels, opens on the current month with today outlined, and has a legend from no activity to more; the demo has a lived-in year (EP-99).
- **The roadmap on the calendar**: the Top-5% roadmap's weekly rhythm shows as study blocks with each phase's tracks, in the calendar and its iCal feeds, and each phase project has its note in the vault's project template (EP-97).
- **A sky that follows the sun**: the home page and Today show the sky of your time zone now, with the sun, the moon and its phase, and the stars, over a landscape you choose (nature, the Earth from orbit, or a monument on each continent); a theme follows it, light from dawn to dusk (EP-96).
- **The Top-5% roadmap**: the library is one roadmap, Top-5% in 26 engineering domains, and starting it creates the whole plan: goal, 14 milestones, a project per phase with its exit check as tasks and subtasks, the 42 cursus, four habits, the first week's tasks, areas, resources, and its note. The demo follows it, a week in. OSSU and the 27 library domains are gone, with the regrouping of old imports (EP-95).
- **Only what you use**: a new account says what it wants Axon for, and the menu and Today show just those pages; Preferences hides or brings back any page. Saving Preferences no longer resets Today's cards (EP-94).
- **Journal days in the vault**: with health notes on, each day is a daily note with its scales in the front matter (EP-93).
- **What helps you**: which of your habits go with your better days, from your journal and logs over 90 days, with statistics that leave chance out and say a link is not a cause; computed on your server, no AI (EP-92).
- **Daily journal**: mood, energy, stress, sleep, gratitude, and a note, a day at a time, with the last 30 days and a Today card (EP-91).
- **Chef & Nutritionist**: meal plans within your budget, restrictions and allergies, constraints, and cooking time, close to your daily needs, with recipes, a shopping list, and a cost estimate, saved as a note once approved (EP-90).
- **A photo of a meal**: your AI provider estimates its foods, portions, and macronutrients for you to check before logging, behind a consent of its own; the photo is not kept. A photo of a barcode finds the product (EP-89).
- **Daily needs**: energy, protein, carbohydrates, fat, fibre, and water for keeping your weight, from your age, sex, size, and activity, each explained with its reference, with a warning to check with a dietitian (EP-88).
- **French everywhere**: every page of the web app in French, with dates in the interface's language, and what the API writes (errors, explanations, insights, badges) too, with tests that keep it so (EP-85 to EP-87).

### Fixed

- Edits of a note or an item large enough to be compressed failed with "changed somewhere else" on their first save: the proxy suffixes the entity tag with its encoding, and the API now reads it through (since EP-74, found in EP-100).
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

- Caddy is rebuilt with the patched Go (four denial-of-service fixes) and PostgreSQL gets Debian's security updates and a rebuilt `gosu` (54 HIGH or CRITICAL fixed); both images are now scanned in CI and shipped by `deploy.sh` like Axon's own (EP-102).
- Recovery sign-in locks for 15 minutes after five wrong authenticator codes for an account, and the failures are audited.
- `Permissions-Policy` header; reachable-vulnerability scanning (govulncheck) in CI; the Go toolchain is pinned to 1.26.9.

### Before tagging

- External security review with no open high-severity issue (see [doc/security/review-2026-10.md](doc/security/review-2026-10.md)).
- A new operator installs from [doc/install.md](doc/install.md) while observed.

## 0.1.0

Accounts with passkeys, recovery codes, and TOTP; privacy controls and export; the universal graph API; tasks with GTD views and recurrence; a daily dashboard; CI and deployment to staging and production.
