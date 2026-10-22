# Features

Every feature of Axon, area by area. Back to the [README](../README.md) · [Français](features.fr.md).

## Features

### 🧠 Second Brain and Zettelkasten
- ✅ A notes workspace: folder tree, live Markdown editor with preview, `[[wikilinks]]` with suggestions, backlinks and mentions, tags, callouts, tables, daily notes, templates, and a command center (Ctrl+K).
- ✅ PARA folders and Zettelkasten notes, Markdown import and export.
- ✅ Two-way sync with the `~/SecondBrain` vault (Git), keeping both versions on conflict.
- ✅ Vim keybindings in the note editor (optional), and keyboard shortcuts everywhere: press `?` for the list, `g` then a letter to jump to a page.
- ✅ Live sync with your Markdown vault: add, edit, rename, or delete a file and the note follows within seconds, and the other way round; vault templates appear in Axon.
- ✅ Your notes live in Axon by default; name a server and folder in Preferences and one command pairs it, syncs it both ways, and starts an empty vault with its folders and templates (Axon shows whether it is in step).
- ✅ Drawing in notes: a sheet with a pen, a marker, arrows, boxes, and text saves an SVG the note embeds, shown in Axon and in the vault alike, and opened again for more drawing. The top of Notes says where your notes live: Axon only, or a vault kept in step both ways.
- ✅ Daily, weekly, and monthly digests by email, Slack, Discord, Microsoft Teams, Telegram, ntfy, or a webhook: what to do today, what comes next (tasks, milestones, goals), what to learn and recall, how the week or month went, with a picture drawn for the day and a quote chosen for it.
- ✅ A news digest: sources marked as general news (a FreshRSS import, for one) stay apart from the tech watch, and the day's top stories come ranked by how many outlets carry them.
- ✅ Answer the digest in Slack, Discord, or Telegram: "done 2", "I finished the passport", "add call the bank tomorrow at 3pm", "snooze 3 monday", "note …", "undo" — Axon does it at once, in English or French, with no public address needed.
- ✅ Sign in with Google, Microsoft, Apple, Discord, Slack, or your own OpenID Connect provider, linked by you from Account → Sign-in & security (never matched by email); passkeys stay the safest way in.
- ✅ A tech watch: follow feeds, sites, Medium authors, Substacks, YouTube channels, and GitHub repositories (by hand, CSV, OPML, or from FreshRSS) on a schedule, midnight by default; each story is stored once however many sources carry it, labelled by local NLP, read as a feed note, and kept as a note when it matters; every source shows its health. A news graph shows the concepts of the news, how they go together, and what is rising, apart from the notes' graph, with a summary on hover. Any view exports to PDF, ODS, CSV, Markdown, feed notes, or JSON, or becomes a read-only link to share. Turn on its podcast to hear the best news read by your device, and find them in your digests, whose sections you choose. A Tech watch agent proposes sources and topics for you to approve.
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
- ✅ Pomodoro focus sessions (25/5 × 4 with a 15-minute long break by default, all adjustable, any number of sessions) linked to the task, project, skill, or note you work on, with focus time on the dashboard, a daily streak, focus sounds generated in the browser (noise, rain), music composed as you work and matched to it (lo-fi beats for tasks and projects, ambient for notes, calm piano for skills and flashcards; nothing streamed), and your own music links per activity.
- ✅ Personal finance in the spirit of Firefly III: asset and liability accounts with balances, income, expenses, and transfers (never counted as spending), categories with monthly budgets, tags, recurring transactions, rules that categorise, CSV import from bank exports without duplicates, reports (monthly flows, spending by category, savings rate, net worth), and savings pots (a *tirelire*) for something to buy later, with a target, an optional date, and how much to put in each month; the Budget page counts each account's balance, net worth, and the bills and income still to come this month. Amounts are exact integer cents; finance data stays out of search and agents, and is in your export.

### 🌱 Habits, goals, and roadmaps
- ✅ Habits designed with James Clear's Atomic Habits: build good habits (obvious, attractive, easy, satisfying) or break bad ones (invisible, unattractive, difficult, unsatisfying), habit stacking ("After I…, I will…"), implementation intentions ("I will… at… in…"), an identity that every check-in votes for, habit contracts, the habits scorecard, two-minute versions, and gentle recovery prompts.
- ✅ Goals and milestones connected to tasks, notes, habits, projects, and roadmaps, with explained progress and a weekly review.
- ✅ Roadmaps you build by hand, or the one roadmap Axon ships: Top-5% in 47 engineering domains, 18 phases from Monday 19 October 2026 to January 2031 with their courses, books, certifications for every domain (141 in all, labelled by field and vendor) with the project work that proves it, and the skills assessment retaken at the end of each phase. Starting it creates the whole plan, with its weekly rhythm as study blocks on the calendar: the goal, a milestone and a project per phase with its exit check as tasks and subtasks, the 42 projects, four habits, the first week's tasks, areas, and resources. Each roadmap is shown as a list or a layered skill graph. A domain is proven, not ticked: its evaluation sheet, as at École 42, with the proof's items, their evidence, the explanation in your own words, a mark of 0 to 125, and retries; the plan says how many days behind it runs.
- ✅ Your roadmap today: Today shows the plan's phase and week, the days to its milestone, today's study block, the phase's project, and the next certification, with focus, review, and practice one click away. The library shows the plan's 18 milestones before you start it.
- ✅ A skills radar: the hard and the soft skills you have acquired, and those you are learning, area by area.
- ✅ A skills assessment before you start: your situation, goals, and quick checks per domain give an explained level, and roadmaps start at your level.
- ✅ A Tutor agent that interviews you and drafts a roadmap where every skill cites its source, and a Resources agent that finds material for what you are learning. Both wait for your approval.
- ✅ Connect any element to any other, from wherever it is shown.

### 📚 Learning and media
- ✅ A practice log for problems, wargame levels, and labs (LeetCode, Codeforces, Advent of Code, OverTheWire, PortSwigger, Kaggle…): paste a link, mark each attempt solved unaided, with help, or not, and re-solve at growing intervals; solved by difficulty, topic, and week.
- ✅ A decision journal: write down a choice, what you expect, and how likely it is; review it on its day and see how well calibrated your forecasts are (Brier score, reliability chart).
- ✅ Coffee and tea: caffeine against EFSA's daily limit, what is left at bedtime, and the last coffee that keeps your sleep intact, set to your bedtime and metabolism.
- ✅ Andrew Huberman's health protocols on Habits, each with its studies, an honest grade of the evidence, and a habit in one click.
- ✅ A collection on Progress: a trophy per roadmap skill, a card per habit, and milestones, raised from bronze to silver to gold.
- ✅ Invitations: the owner brings someone in with a link used once, even while registration is closed, and into the household.
- ✅ Share as a picture: your level, a card, a goal, or a habit's run, drawn on your device for the share sheet or a post on X, LinkedIn, or Bluesky.
- ✅ YouTube in the tech watch: a video's words from its captions, or your own Whisper server, in its analysis and the news graph, in English, French, or Spanish, with links cleaned of tracking.
- ✅ Podcasts from your Audiobookshelf in the tech watch, each episode with its notes and, with Whisper, its words.
- ✅ Certifications: planned, booked, passed, or failed, with the exam countdown, cost against the plan, credential, and expiry with a renewal warning; exams and expiries on the calendar. The roadmap plans its 45.
- ✅ Study hours against the plan: focus time on a roadmap, this week day by day beside the plan's block and over 12 weeks against its weekly target, by skill and project.
- ✅ Illustrated flashcards you can view, add, edit, and remove, reviewed with FSRS-5 (the Anki scheduler, matched to its reference implementation), practice tests, and a Teacher agent that writes retrieval questions from your literature and permanent notes for your approval.
- ✅ A learning and memory engine after *Understanding How We Learn* and *Moonwalking with Einstein*: Learn → Today plans the day (cards due with the minutes they take, notes due for recall, a weekly mixed test, fleeting notes to process, leeches to practise); a few new cards a day and decks mixed (interleaving); write your answer before you see it; every card can carry the why, a concrete example, a vivid memory image, and a picture or an AI drawing (dual coding); free recall of whole notes on a blank page, compared with their key ideas and spaced further each time; memory palaces you walk place by place; cards drafted from what a note marks (`question :: answer`, `==term==`, `**Term**: definition`); and the Teacher writes illustrated cards for your approval.
- ✅ Your memory, measured: Learn compares the recall the scheduler predicted with what you remembered, against your target retention, and says what to change.
- ✅ Covers for films, series, books, and games, found for anything added by hand or imported, on a shelf of poster cards.
- ✅ Reminders on tasks at a time of day: set in the task, or type "Call the bank friday at 15:00" in quick add.
- ✅ Share to Axon: from any app's share sheet once installed, or a bookmarklet on a computer; save as a note or task, or keep the page's readable text as a literature note.
- ✅ A trash: whatever you delete stays 30 days with what went with it (a goal's milestones, a habit's check-ins), Undo right after deleting, and Account → Trash to restore.
- ✅ Bulk actions on tasks (status, context, due date, delete), and dragging tasks between days of the calendar or onto it from the undated next actions.
- ✅ A Today you arrange: choose and order its cards, with learning, habits, today's calendar, and the household among them.
- ✅ Only what you use: a new account says what it wants Axon for, and the menu and Today show just those pages; any page can be hidden or brought back in Preferences.
- ✅ Search by meaning, if you want it: choose an embedding service (a local model server, OpenAI, or Voyage AI) and give its own consent; notes are then found by what they mean, in the command center and under "Close in meaning". Off, related notes use local word statistics.
- ✅ Household sharing: people on the same Axon join with a one-time code and share lists (shopping, chores, anything), with items given to someone; nothing personal is shared.
- ✅ Imports from other apps: Todoist and TickTick tasks, Goodreads books, Letterboxd films, Firefly III and bank OFX/QFX transactions, Notion pages; preview first, and importing again adds nothing twice.
- ✅ A guided weekly review after Getting Things Done: empty your head, inbox to zero, fleeting notes, the week's highlights, the next two weeks, waiting for, stuck projects, goals, habits and learning, someday/maybe, and what matters next week; suggested on weekend mornings until done.
- ✅ Your other calendars in Axon, read only: follow Google, Nextcloud or any CalDAV server, Outlook, or any ICS address; repeating events, time zones, and exceptions handled; shown in the month and agenda views with their colour.
- ✅ Quick add for anything: press n and type `Call the plumber @phone tomorrow`, `book: Atomic Habits`, `habit: Read one page`, `goal: Run 10 km by 2027-05-01`, or `card: question :: answer #Deck`; Capture shows what it will add.
- ✅ Reminders: a morning digest (tasks, cards to review with the minutes they take, notes to recall, habits, the weekly test) and an evening nudge (habits still open, with their two-minute version), as notifications on your devices, encrypted end to end, and by email if your server can send it; once a day, only with something to say, off by default.
- ✅ Installable on phones and desktops; offline, Capture keeps tasks and notes on the device and sends them, once, when you are back online.
- ✅ A menu in five groups (Do, Think, Grow, Life, Reflect) plus Today, and one Menu sheet on phones.
- ✅ Agents where you work: every page offers the agents whose skills fit it (Task Manager on Tasks and Today, Goals, Habits, the Teacher on Notes, a note, Learn, and Flashcards, the Tutor and Resources on Roadmaps, Media, the Personal Trainer and Nutritionist on Health, Side-Projects and Code Review on Projects, Tech watch on the tech watch), with ready-made requests and what the page shows sent along; proposals are approved right there.
- ✅ Jarvis, the master agent: an opt-in ⚡ button opens a side panel where you say what you want in your own words, and Jarvis asks the right agents ("I'm reading The Martian" becomes the book, being read, with its note, waiting for your approval), on your provider or a local Ollama model.
- ✅ A model for all agents, or one for each, and the AI tokens each one used, this month and by month, with the measured saving of compacting what tools send the model.
- ✅ Anki packages (.apkg) per deck, pictures included, and automatic sync to your own `anki-sync-server` (for example on `srv-services`), by hand or right after you approve the Teacher's cards; Anki-only cards are never deleted.
- ✅ Films, series, books, and video games: want, in progress, finished, or dropped, with ratings out of 10, progress in pages, episodes, minutes, or hours played, and reviews. Once you allow lookups (they send the title you type), books come from Open Library, games from Wikipedia (with their cover, year, genre, and developer), and films and series from TMDB when the server has a key; anything can be added by hand. Covers are copied to Axon, so pages never load images from elsewhere. "Up next" ranks your list with the reasons (same creator or genre as what you rated 8 or more, or something you started), and a Media agent proposes new titles for your approval.

### 🥗 Health and sport
- ✅ Water: a glass or a bottle in one tap, against the drinks' share of your daily water need, with the week; health data under its consent.
- ✅ A daily journal: mood, energy, and stress on five faces, hours of sleep, three things you are grateful for, and a note, with the last 30 days as a chart; on Today in two taps if you like. Health data, under its consent.
- ✅ What helps you: which habits go with your better days (sleep, training, focus, habits, tasks, gratitude against mood, energy, and stress), with statistics that leave chance out, said as links, never as causes. Computed on your server, no AI.
- ✅ Food and training logs, only after you consent to keeping health data (turning it off deletes them): meals with energy and macronutrients, looked up in Open Food Facts (by name or barcode) or USDA FoodData Central once you allow lookups, or typed in; workouts with sets, minutes, and effort. Never in search or AI agents. A photo of a meal estimated by your own AI provider into foods, portions, and macronutrients that you check before logging (the photo is not kept; camera and location details are removed first), and a photo of a barcode read on your device. Daily needs (energy, protein, carbohydrates, fat, fibre, water) worked out from your age, sex, size, activity, and goal with published references, each explained, with a warning to check with a dietitian; a goal moves them within bounds only (lose gently: 15 % under, never under 1,200 kcal, and no loss when maintenance is already 1,200 kcal or less; gain: 10 % over), the day's bars and the Chef following them; none before 18 or in pregnancy.
- ✅ Personal Trainer and Nutritionist agents that read your logs only with a separate consent to share them with your AI provider, propose sessions or simple meal tasks for approval, hand red-flag topics (chest pain, fainting, eating disorders, pregnancy, medication…) to a professional without calling the model, and can never suggest under 1,200 kcal a day or multi-day fasts.

### ✨ Humane progress
- ✅ Opt-in progress rewards: points from every area (tasks, projects, habits, focus, flashcards, notes, roadmaps, goals), each traceable in a ledger, levels, badges, one daily streak with freezes bought only with earned points, and no shame or notifications.
- ✅ Insights: nine weekly metrics (tasks completed and captured, habit check-ins, milestones, focus minutes, notes, flashcards reviewed, recall rate, skills learned), each with its formula, its source, a trend in plain words, and the items behind every number; a weekly review against the week before, with what changed and open tasks to carry over; CSV export. All computed on your server.
- ✅ Ethical design: no dark patterns, no shame, no hidden consent, GDPR-aware controls.
- ✅ A profile picture, with a generated default (your initials on your own colours); uploads are re-encoded so no location data survives.
- ✅ Themed spaces: each area has its own original scenery, in light and dark: a Da Vinci notebook for notes, an Oxford library for learning, resources, and flashcards, a workshop blueprint for projects, a garden for habits, a contour map for goals and progress, a star chart for the calendar, a ledger for the budget, an old atlas for areas and the archive, and dotted paper for tasks and focus. Effects are slow, stop with reduced motion, and can be turned off in Preferences. Journal, Insights, the weekly review, and Agents have theirs too (Vermeer's evening light, an observatory, a lighthouse, clockwork automata); each page says what its picture shows and what it nods to, and each type of note has its own small drawing.
- ✅ A sky that follows the sun: the home page and Today show the sky as it is now in your time zone, the sun and the moon (with its phase) where they are, stars after dusk, over the landscape you choose: mountains, forest, ocean, desert, the Earth from orbit, or a monument on each continent (Eiffel Tower, Pyramids of Giza, Taj Mahal, Statue of Liberty, Christ the Redeemer, Sydney Opera House). The theme can follow it too: light from dawn to dusk.
- ✅ Move your whole Axon to another server: `axon instance export` and `axon instance import` carry every table and picture with a checksummed manifest, into a fresh installation, older versions included.
- ✅ Preferences in one place: English or French (every page, with dates in your language), theme, font, text size, icons, motion, templates for notes, projects, and resources, and your AI provider and budget. Edit them in the app or as a YAML file (keys are write-only).

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
| Chef & Nutritionist | Meal plans for the days you choose, within your budget, restrictions and allergies, constraints, and cooking time, close to your daily needs, with recipes, a shopping list, and a cost estimate; saved as a note once approved. | Own consent; strict restrictions, a reminder to check labels; no daily target under 1,200 kcal (enforced); red flags referred. | ✅ |

All agents share a constitution: simple, explicit, small diffs, read before write, think in data, test everything, do not be clever. Skills are versioned `SKILL.md` files. Each task ends with reflection, lessons learned, evaluation evidence, and—when needed—human-in-the-loop approval.

## Gamification

Progress systems use XP, levels, streaks, badges, variable rewards, commitment devices, and earned-point streak freezes. The design is opt-in and transparent: users can disable competitive surfaces, see how points are calculated, and recover from missed days without humiliation. Inspiration includes *Hooked*, Octalysis, and Self-Determination Theory; the ethical constraint is stronger than engagement metrics.

## Screenshots

[![The trailer: Axon in a minute](../web/static/teaser/axon.jpg)](../web/static/teaser/axon-trailer-4k.mp4)

**[Watch the trailer](../web/static/teaser/axon-trailer-4k.mp4)** (one minute, with music; also on the home page when the demo is on).

| Today | Tasks | Notes |
|---|---|---|
| ![Today: tasks, habits, goals, and the day's counts](screenshots/gallery/today.jpg) | ![Tasks on a Kanban board with contexts](screenshots/gallery/tasks.jpg) | ![The notes' knowledge graph with its communities](screenshots/gallery/notes.jpg) |

| Learn | Roadmap | Focus |
|---|---|---|
| ![Learn: flashcards, recall, the weekly test, and memory measured](screenshots/gallery/learn.jpg) | ![The roadmap's skills radar, hard and soft](screenshots/gallery/roadmap.jpg) | ![Focus: Pomodoro with music and the week's sessions](screenshots/gallery/focus.jpg) |

| Tech watch | News graph | Health |
|---|---|---|
| ![The tech watch: news, topics, and labels](screenshots/gallery/watch.jpg) | ![The news graph: concepts, links, and communities](screenshots/gallery/graph.jpg) | ![Water, coffee and tea, and the journal on Today](screenshots/gallery/health.jpg) |

| Decisions | Collection | Habits |
|---|---|---|
| ![The decision journal's calibration chart](screenshots/gallery/decisions.jpg) | ![The collection of trophies and cards](screenshots/gallery/collection.jpg) | ![Habits and Huberman's protocols, sourced](screenshots/gallery/habits.jpg) |

| Home | Today (dark, phone) |
|---|---|
| ![Home page with a night mountain scene and feature sections](screenshots/home.png) | ![Today view in dark mode on a phone](screenshots/today-mobile-dark.png) |

Captured from the demo account, with its sample data. `make teaser` takes them again and makes the trailer from a running Axon.

<details><summary>Design direction</summary>

Large typography, generous whitespace, dark mode, translucent glass over bundled scenery (ADR 0010). Motion should clarify state like a well-made instrument—not compete for attention. Respect reduced-motion preferences and keep every action usable without animation.

</details>
