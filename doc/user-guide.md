# Axon user guide

Axon keeps your notes, tasks, goals, habits, learning, focus, money, media, health logs, and reflection in one place, on a server you or someone you trust runs. This guide is a tour; every page also explains itself. Press `?` anywhere for keyboard shortcuts.

## Getting in

- **Create your account** with a passkey (your device's fingerprint, face, or PIN, or a security key). Save the recovery codes: each works once if you lose your devices. The account starts in your device's time zone, which decides what "today" means; change it under **Account**. Add more passkeys and an authenticator app under **Account → Security**.
- **Say what you want Axon for.** A new account is asked first, on Today: tasks and projects, notes, learning, goals and habits, health, money, media, or a shared home. The menu and Today then show just those pages; the others stay one link or search away. Change it under **Preferences → Pages in the menu**, or choose **Show me everything**.
- **Try the demo** (when offered) opens a shared account with sample data that resets every night.

- **Invite someone.** On a private Axon, only the first account can register. As its owner, open **Account → Invitations**, add a note, choose how long the link lasts and whether it brings the person into your household, and **Create a link**. Send it privately: it creates one account, then stops working. Revoke a link you no longer want.

## Search

**Search**, in the menu bar (or Ctrl+K, or /), finds your tasks, notes, goals, habits, news, and the rest by their words, and notes close in meaning when search by meaning is set up; type a page's name to open it. Arrow keys choose, Enter opens.

## Today

Your day at a glance: overdue and today's tasks, your inbox, active goals, recent notes, habits to check in, and focus time. Below, the dashboard compares this period with the previous one, charts daily activity and the year, lists what needs attention, and draws the network of your hard and soft skills. Every number opens the items behind it. Use the **daily check-in** to note how the day went.

**Your roadmap today** sits under the counts once you start a roadmap: the phase under way and its week, the days to its milestone, today's study block, the phase's project with its board, and the next certification, with **Focus on today's block**, **Review cards**, **Practice**, and **Open the plan**. Before a start it points to the skills assessment and the roadmap. **Customise Today** turns cards on and off; a card new in a release shows on your Today until you turn it off.

## Capture and do: Tasks

A task you check off stays in its list, crossed out, until the next day, so you see what you did; uncheck it to put it back where it was. The same goes for habits on Today and on Habits.

- Press `n` anywhere to capture a task. It lands in the **inbox**.
- Tasks open on a **GTD board**: inbox, next, waiting, someday. Clarify each inbox item, give it a context (`@phone`) and a date, or move it to someday. **Kanban** (to do, doing, done) is one click away.
- Tasks can repeat (every Saturday, the first of the month) and belong to projects and goals.
- The **calendar** shows tasks and milestones on a month grid; subscribe to its iCal feed from your phone.

## Think: Notes (Zettelkasten, PARA, Second Brain)

**Templates** (Notes → Templates) are those of the SecondBrain vault: literature and permanent notes, PARA project, area, and resource notes, daily, weekly, and monthly notes, roadmap, goal, milestone, habit, and media notes in `Roadmaps/` and `Tracker/`, and the six retrieval tests (five Ws, blank page, flashcards, cloze, monthly, Feynman) in `Reviews/`. Their placeholders are filled when you use them: `{{title}}`, `{{created}}` (date and time), `{{date}}`, `{{slug}}`, and the note a test is about. A template of the same name in your vault's `Templates/` folder replaces the built-in one.

- Notes are Markdown with `[[wikilinks]]`, backlinks, and `#tags`, in PARA folders (Projects, Areas, Resources, Archive) plus Inbox and Zettel.
- **New note** offers templates: fleeting, literature, permanent, project, area, resource, archive, and daily. They live as files in your vault's `Templates/` folder, so edit them there.
- Your **tasks, goals, milestones, projects, areas, resources, habits, and media are notes too**, under `Axon/` in your vault: edit a note's front matter or sections and the item follows, write a new note with `type: task` (or goal, habit, book…) to create one, delete the file to delete it, move it into `Axon/Archive/` to archive it. Turn it off, change the folder, or add your food and training logs in **Preferences → Markdown vault**.
- **Notes → Templates** shows every template with its sections; the strip on the Notes page starts one in a click.
- **Where your notes live**, at the top of Notes, says and sets it: in Axon only, or in Axon and a Markdown vault (for example Obsidian on your SecondBrain server) kept in step both ways, through the vault server or `axon notes-sync -watch`. Add, edit, rename, or delete on either side and it reaches the other within seconds; a note changed on both sides keeps both versions.
- **Maths**: write LaTeX between `$…$` in a line or `$$…$$` on lines of their own, as in Obsidian; **∑ Write maths** in the toolbar offers fractions, sums, integrals, matrices, symbols, and Greek letters with a live preview.
- **Draw** (✎ in the editor's toolbar) opens a sheet: pen, marker, line, arrow, box, ellipse, text, and eraser, in six colours and three thicknesses, with undo and redo. **Save in the note** stores an SVG picture in `Attachments/` and embeds it, so the drawing shows in Axon and in the vault alike (Obsidian shows SVG). Put the caret on the drawing's line and press Draw again to go on with it. Pictures drawn elsewhere (Excalidraw's SVG export, a tablet) embed like any picture.
- **Explore** filters notes by PARA, type, tag, date, or unconnected notes, charts them, and draws a knowledge graph where every link says why.
- The **knowledge graph** at the top of Notes connects your literature and permanent notes by what they say, groups them into communities with names, and marks central notes and bridges; select a note to see why it connects.
- **Analysis** looks at your literature and permanent notes: key phrases, readability, topics, and a concept graph. Each note has an **Analysis** panel with related ideas and why.
- **Vim mode** for the editor is in **Preferences → Keyboard**.

In the notes' graph and the news graph, scroll or pinch to zoom, drag to move, **Fit** to see everything, and **Full screen**; type in **Find in the graph** to select a note or a concept and centre on it. Labels that would overlap are left out until you zoom in.

## Aim: Goals, Projects, Areas, Resources

- **Goals** have an outcome, a date, milestones, and linked tasks and habits. Progress shows exactly how it was worked out.
- **Projects** have Kanban boards with WIP limits, epics, issues, and project docs.
- **Areas** (standards to keep: health, home, money) and **Resources** (topics you collect) start from templates; finished things go to the **Archive**, where they can be restored.

## Grow: Habits

Designed with James Clear's *Atomic Habits*:

- Write who you want to become (identity), then make the habit **obvious** (an implementation intention, "I will … at … in …", or a stack, "After I …, I will …"), **attractive**, **easy** (a two-minute version for hard days), and **satisfying** (a reward).
- To break a habit, invert the laws: make it invisible, unattractive, difficult, unsatisfying.
- Check in with the full or the two-minute version. Missing a day is normal; never miss twice is a kindness, not a rule. The **scorecard** lists your current habits as helpful, neutral, or harmful.

**Protocols for health, after Andrew Huberman** lists the habits he recommends on the Huberman Lab podcast. Open one to read what to do and why, how strong the evidence is (strong, some, or his advice not tested by any study), the studies behind it, and a caution where one is due; **Make it a habit** fills in the form for you to adjust. They are information, not medical advice.

## Learn: Roadmaps and flashcards

- Start with the **assessment**: your situation, job, goals, and current level. It places you on roadmaps.
- **Roadmaps** are skill graphs (competencies, with the books, courses, and certifications under each). Axon ships one: **Top-5% in 37 engineering domains**, 18 phases from Monday 19 October 2026 with a project each and a certification for every domain (45 in all, from PCPP1 to the Google Professional Data Engineer; Rust, which has no individual exam yet, takes a course from a Rust Foundation accredited trainer), and the skills assessment retaken at the end of each phase. The roadmap's page shows the plan itself, before you start it too (a certification the plan gained after you started has **Plan it**): each phase as a milestone with its dates, project, exit check, and certifications. **Start this roadmap** creates the whole plan: the goal (measured in domains proven), a milestone and a project per phase whose board holds the exit check as issues and subtasks, a *42 cursus* side project, the four habits (daily study block, daily review, Saturday project session, Sunday reflection), the first week's tasks, the areas *Engineering mastery* and *Maths*, resources, and the roadmap's note. Started while phase 1 runs, it keeps its own dates; later, it starts on the coming Monday. The calendar shows its weekly rhythm as **study blocks**: Monday theory, Tuesday the core track, Wednesday practice, Thursday whichever is behind, Friday spaced tests and DSA, and the weekend's project days, with the tracks of the phase each day falls in; each opens the roadmap, and iCal feeds can include them. Each phase project also has its note in `1-Projects/<project>/`, in the vault's project template. **Start at my level** marks the domains your skills assessment rates 3 of 5 or more as being learnt. Create, edit, and remove your own roadmaps by hand or with the Tutor agent. "Skills acquired" draws your hard and soft skills as two radar charts: one spoke per roadmap (or per section when you follow fewer than three), the filled shape for skills done, the dashed one adding those you are learning.
- **Practice** (Learn → Practice) logs problems, wargame levels, and labs: paste a link (LeetCode, Codeforces, Advent of Code, OverTheWire, PortSwigger, Kaggle…) and the platform and title fill in. Mark each attempt honestly, solved unaided, with help, or not solved: a problem comes back tomorrow after a miss, in three days after help, a week after a first unaided solve (easy ones are done), then at 2.5 times the last gap until it would pass 90 days. **Re-solve today** lists what is due; **Your practice** counts problems solved by difficulty and topic and by week, the measure behind the roadmap's "60 LeetCode (≥ 30 medium)".
- **Certifications** (Learn → Certifications) keeps the exams you plan, book, pass, or fail: provider, code, cost, exam date with a countdown, score, credential link, and expiry, worked out from how long it stays valid; a warning shows 90 days before one expires. Booked exams and expiries are on the calendar. Starting the Top-5% roadmap plans its 45 certifications for the end of their phases (one you already have, by its code, is kept rather than planned twice), with their price; the summary adds up what is paid of what is planned.
- **Study hours**, on a roadmap's page, add up the focus sessions that were for it: the roadmap, one of its skills, its goal and milestones, or the projects, habits, and tasks of its plan. This week shows day by day beside the plan's block, against the plan's 22 hours, with the last 12 weeks and the time by skill and project, for the monthly roadmap check. **Focus on today's block** opens Focus aimed at the roadmap, so the time counts.
- **Flashcards** (with pictures) are reviewed with FSRS, the Anki scheduler; take practice tests. Export Anki packages or sync to your own `anki-sync-server`. The Teacher agent can draft cards from your notes for your approval.

## Focus

Pomodoro sessions (25 minutes of focus and 5-minute breaks, four rounds, then a 15-minute break, all adjustable) for a task, project, skill, or note. Focus sounds play in the browser: noise, rain, or music Axon composes as you work, without words and never the same twice. "Matched to what you focus on" plays lo-fi beats for a task or project, ambient pads for a note, and calm piano for a skill or flashcards; change it during the session from the timer. Music stops during breaks. You can also keep your own music links per activity.

## Money: Budget

Firefly III–style personal finance: accounts and balances, income, expenses, transfers, categories with monthly budgets, tags, recurring transactions, rules, CSV import from your bank, reports, and **savings pots** (*tirelire*) for something to buy later. Amounts are exact; finance data stays out of search and agents. The Budget page opens on the month's counts, then two pies, where the month's money went and where it came from by category, and the bars of income and expenses for each month of the year, January to December; **Reports** does the same for 3, 6, or 12 months.

## Journal

**Reflect → Journal** takes a minute a day: mood, energy, and stress on five faces, hours of sleep, up to three things you are grateful for, and a note. The last 30 days show as a chart and a list; open any day to change it. Mood and sleep are health data, so the journal is kept under the same consent as the health logs and deleted with it. Add "How are you today?" to Today (Customise Today) to log mood and energy in two taps.

After two weeks of entries, **What helps you** on the same page shows which of your habits go with your better days: sleeping 7 hours or more, training, a focus session, habits kept, tasks finished, or writing what you are grateful for, against mood, energy, and stress over the last 90 days. A link is shown only when it is unlikely to be chance, as the two averages, with how it was worked out. It is a link, not a cause: take it as something to try. It is computed on your server; no AI is involved.

If your vault mirror writes health logs (Preferences → Vault), each journal day is also a daily note in `Axon/Health/Journal/`, with mood, energy, stress, and sleep in the front matter, ready for queries in Obsidian.

## Decisions

**Reflect → Decisions** is a decision journal. When you make a decision, write down what you chose, what you expect to happen, and how likely you think it is (5 to 95 percent), and when to look back: in a month, three, six, or twelve, or on a day. The forecast cannot be changed afterwards. On the review day the decision appears under **To review**: say what happened and what you learned, then whether what you expected came true. **Your calibration** scores your reviewed forecasts with the Brier score (0 is perfect; always saying 50 percent scores 0.25) and, after ten reviews, tells you whether you are over- or under-confident; the chart puts the chance you gave against the share that came true.

## Media and health

- **Media**: films, series, books, and video games you want, are on, have finished, or dropped, with ratings, progress (pages, episodes, minutes, or hours played), and reviews. **Up next** explains each suggestion. Searching sends only the typed title to Open Library, TMDB, or Wikipedia (games), after you allow lookups. Each item can open a literature note from its template (film, series, book, or game note: how it plays, the moments that stay, its design ideas, a play log), and is a note under `Axon/Media/` in the vault.
- **Health**: food and training logs, only after you consent to keeping health data. Turning that consent off deletes them. **Your daily needs** works out, from your birth year, sex, height, weight, and activity, the energy, protein, carbohydrates, fat, fibre, and water a healthy adult needs to keep their weight, shows the day's log against them, and says how each figure is worked out and from which reference. They are estimates, not advice: check them with a registered dietitian or nutritionist. Axon gives no numbers before 18 or during pregnancy or breastfeeding, and sets no weight-loss target (ADR 0028). **Photo of a meal** (with an AI provider set up) sends the photo, without its camera and location details, to your provider once you allow it; you get a line per food with its weight and nutrients to correct before logging, and the photo is not kept. **Photo of a barcode** (Chrome, Edge, Android) reads the barcode on your device and looks the product up. **Plan my meals with the chef** asks the Chef & Nutritionist agent for meals over the days you choose, for a number of people, within a budget, dietary restrictions and allergies, other constraints, and a cooking time; it aims close to your daily needs and writes recipes, a shopping list, and a cost estimate. The plan is saved in Notes under "Meal plans" only when you approve it. Check labels for allergens; for a medical condition, pregnancy, or an eating disorder, plan with a dietitian.
- **Water**: on Health (and on Today, if you add it), log a glass of 250 ml or a bottle of 500 ml in one tap, or any amount, and remove a drink you logged by mistake. The day shows as glasses against your target, the drinks' share (about 80 %) of the water your daily needs give, as EFSA counts it, with the last seven days; without a profile, under 18, or during pregnancy there is no target. **What helps you** can link the days you drank 1.5 L or more to how they felt.

**Coffee and tea** (on Health, and as a Today card) logs each drink in one tap with its usual caffeine (an espresso 63 mg, a coffee 95 mg, a black tea 47 mg…), or your own amount. It shows the day against EFSA's limit (400 mg, or 200 mg if your profile says pregnant or breastfeeding), how much is still in your body, and tonight: until when a coffee is still fine, or that it is past that time. That time is when a coffee would leave about 32 mg at your bedtime, what one leaves 8.8 hours later, the gap a 2023 meta-analysis found keeps sleep intact. Set your bedtime and how fast you clear caffeine (fast, average, slow) in **Preferences → Coffee and tea**. It is information, not medical advice.

## Insights

Nine weekly metrics with their formulas, trends in plain words, and the items behind every number; a **weekly review** against the week before, with what changed and open tasks to carry over; CSV export.

## Progress (optional)

Turn on progress rewards in Preferences: points from every area, each traceable, levels, badges, and one daily streak with freezes bought with earned points. No shame, no notifications.

**Your collection** gathers cards to raise from bronze to silver to gold: a trophy for each skill of your roadmap (started, 10 hours of focus on it, learned), a card for each habit by its best run of scheduled days kept (7, 21, then 66 days, about the time a habit takes to become automatic), and milestones for problems solved by difficulty, focus hours, flashcard reviews, your longest daily streak, and decisions reviewed. A locked card shows what it takes.

**Share** (on your level, an earned card, a goal, or a habit) draws a picture of it on your device: share it from there, download or copy it, or open a post on X, LinkedIn, or Bluesky with its words and attach the picture. Nothing is sent until you do.

## AI agents (optional)

Agents draft changes; **nothing changes until you approve**. Each person picks a provider (Anthropic, OpenAI, Mistral, Gemini, DeepSeek, or a local model) and a monthly budget under **Agents**, and turns on **AI agents** in Privacy. Agents: Task Manager, Goals, Habits, Tutor, Resource, Teacher, Media, Side projects, Code review, and, with their own consent, Personal Trainer and Nutritionist.

**AI tokens**, under Preferences → AI, add up every agent run, meal photo, video summary, and Jarvis answer: tokens in and out and cost this month, by month, and by agent. Axon compacts what its tools hand the model (empty fields out, times to the minute, records as rows) and shows the saving it measured.

**Models**: under Preferences → AI, choose one model for all agents (for example `claude-sonnet-5-5`), or one for an agent (a deep model for the Tutor, a quick one for summaries); costs follow the model. **Summarise videos and podcast episodes in prose** has your model write each one's 5 to 10 lines and keywords once, within the budget; without it, Axon summarises them itself from their own sentences.

## Your data

- **Account → Privacy**: consents (all off until you turn them on), the full export (JSON plus your notes as Markdown), and erasure.
- **Preferences**: language (English or French), theme (including **Follow the sun and the moon**: light from dawn to dusk in your time zone), the landscape under the sky on the home page and Today (with a slider to preview it later in the day), font, icons, motion, templates, Pomodoro, keyboard, and progress, in the app or as YAML.
- Personal access tokens (**Account → Tokens**) let your own scripts use the API (`/api-docs`).
