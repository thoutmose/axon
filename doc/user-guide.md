# Axon user guide

Axon keeps your notes, tasks, goals, habits, learning, focus, money, media, health logs, and reflection in one place, on a server you or someone you trust runs. This guide is a tour; every page also explains itself. Press `?` anywhere for keyboard shortcuts.

## Getting in

- **Create your account** with a passkey (your device's fingerprint, face, or PIN, or a security key). Save the recovery codes: each works once if you lose your devices. The account starts in your device's time zone, which decides what "today" means; change it under **Account**. Add more passkeys and an authenticator app under **Account → Security**.
- **Try the demo** (when offered) opens a shared account with sample data that resets every night.

## Today

Your day at a glance: overdue and today's tasks, your inbox, active goals, recent notes, habits to check in, and focus time. Below, the dashboard compares this period with the previous one, charts daily activity and the year, lists what needs attention, and draws the network of your hard and soft skills. Every number opens the items behind it. Use the **daily check-in** to note how the day went.

## Capture and do: Tasks

- Press `n` anywhere to capture a task. It lands in the **inbox**.
- Tasks open on a **GTD board**: inbox, next, waiting, someday. Clarify each inbox item, give it a context (`@phone`) and a date, or move it to someday. **Kanban** (to do, doing, done) is one click away.
- Tasks can repeat (every Saturday, the first of the month) and belong to projects and goals.
- The **calendar** shows tasks and milestones on a month grid; subscribe to its iCal feed from your phone.

## Think: Notes (Zettelkasten, PARA, Second Brain)

- Notes are Markdown with `[[wikilinks]]`, backlinks, and `#tags`, in PARA folders (Projects, Areas, Resources, Archive) plus Inbox and Zettel.
- **New note** offers templates: fleeting, literature, permanent, project, area, resource, archive, and daily. They live as files in your vault's `Templates/` folder, so edit them there.
- Your **tasks, goals, milestones, projects, areas, resources, habits, and media are notes too**, under `Axon/` in your vault: edit a note's front matter or sections and the item follows, write a new note with `type: task` (or goal, habit, book…) to create one, delete the file to delete it, move it into `Axon/Archive/` to archive it. Turn it off, change the folder, or add your food and training logs in **Preferences → Markdown vault**.
- **Notes → Templates** shows every template with its sections; the strip on the Notes page starts one in a click.
- With **vault sync**, notes stay in step with a Markdown folder (for example Obsidian on your SecondBrain server): add, edit, rename, or delete on either side and it reaches the other.
- **Explore** filters notes by PARA, type, tag, date, or unconnected notes, charts them, and draws a knowledge graph where every link says why.
- The **knowledge graph** at the top of Notes connects your literature and permanent notes by what they say, groups them into communities with names, and marks central notes and bridges; select a note to see why it connects.
- **Analysis** looks at your literature and permanent notes: key phrases, readability, topics, and a concept graph. Each note has an **Analysis** panel with related ideas and why.
- **Vim mode** for the editor is in **Preferences → Keyboard**.

## Aim: Goals, Projects, Areas, Resources

- **Goals** have an outcome, a date, milestones, and linked tasks and habits. Progress shows exactly how it was worked out.
- **Projects** have Kanban boards with WIP limits, epics, issues, and project docs.
- **Areas** (standards to keep: health, home, money) and **Resources** (topics you collect) start from templates; finished things go to the **Archive**, where they can be restored.

## Grow: Habits

Designed with James Clear's *Atomic Habits*:

- Write who you want to become (identity), then make the habit **obvious** (an implementation intention, "I will … at … in …", or a stack, "After I …, I will …"), **attractive**, **easy** (a two-minute version for hard days), and **satisfying** (a reward).
- To break a habit, invert the laws: make it invisible, unattractive, difficult, unsatisfying.
- Check in with the full or the two-minute version. Missing a day is normal; never miss twice is a kindness, not a rule. The **scorecard** lists your current habits as helpful, neutral, or harmful.

## Learn: Roadmaps and flashcards

- Start with the **assessment**: your situation, job, goals, and current level. It places you on roadmaps.
- **Roadmaps** are skill graphs (competencies, with the best books, courses, papers, and videos under each), from a curated library across computing, data, maths, and more, or the OSSU curriculum. Create, edit, and remove your own by hand or with the Tutor agent.
- **Flashcards** (with pictures) are reviewed with FSRS, the Anki scheduler; take practice tests. Export Anki packages or sync to your own `anki-sync-server`. The Teacher agent can draft cards from your notes for your approval.

## Focus

Pomodoro sessions (25 minutes of focus and 5-minute breaks, four rounds, then a 15-minute break, all adjustable) for a task, project, skill, or note. Focus sounds play in the browser; add your own music links per activity.

## Money: Budget

Firefly III–style personal finance: accounts and balances, income, expenses, transfers, categories with monthly budgets, tags, recurring transactions, rules, CSV import from your bank, reports, and **savings pots** (*tirelire*) for something to buy later. Amounts are exact; finance data stays out of search and agents.

## Media and health

- **Media**: films, series, and books you want, are on, have finished, or dropped, with ratings, progress, and reviews. **Up next** explains each suggestion. Searching sends only the typed title to Open Library or TMDB, after you allow lookups.
- **Health**: food and training logs, only after you consent to keeping health data. Turning that consent off deletes them. Axon gives no medical or dietary advice and never sets calorie targets.

## Insights

Nine weekly metrics with their formulas, trends in plain words, and the items behind every number; a **weekly review** against the week before, with what changed and open tasks to carry over; CSV export.

## Progress (optional)

Turn on progress rewards in Preferences: points from every area, each traceable, levels, badges, and one daily streak with freezes bought with earned points. No shame, no notifications.

## AI agents (optional)

Agents draft changes; **nothing changes until you approve**. Each person picks a provider (Anthropic, OpenAI, Mistral, Gemini, DeepSeek, or a local model) and a monthly budget under **Agents**, and turns on **AI agents** in Privacy. Agents: Task Manager, Goals, Habits, Tutor, Resource, Teacher, Media, Side projects, Code review, and, with their own consent, Personal Trainer and Nutritionist.

## Your data

- **Account → Privacy**: consents (all off until you turn them on), the full export (JSON plus your notes as Markdown), and erasure.
- **Preferences**: language (English or French), theme, font, icons, motion, templates, Pomodoro, keyboard, and progress, in the app or as YAML.
- Personal access tokens (**Account → Tokens**) let your own scripts use the API (`/api-docs`).
