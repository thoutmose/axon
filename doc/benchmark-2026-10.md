# Benchmark, October 2026: Axon against the best of each category

> Owner request (2026-10-20): "search like on SecondBrain, Zettelkasten, note-taking, Jira, Confluence, Pomodoro, productivity apps, 42 School, and try also to improve learning and the memorising process". This is what the research found, what Axon has today (read from the code, not the docs), and the Epic each gap became. Evidence tags: **strong** = meta-analysis or replicated; **moderate** = several studies; **engagement** = raises use of the app, no learning effect shown; **unverified** = reported by users or vendors only.

## What the evidence says about learning

| Practice | Evidence | Axon today | Epic |
|---|---|---|---|
| Retrieval practice, the testing effect (g ≈ 0.50 over 222 classroom studies; free recall beats concept maps) | strong | FSRS flashcards, free recall of notes, a weekly test | kept |
| Spacing (spaced beat massed in 259 of 271 comparisons) | strong | FSRS, re-solve schedule, recall intervals | kept; EP-190 personalises the parameters |
| Successive relearning (recall to a criterion, then relearn in spaced sessions: 80 % at a week against 20 %) | strong | learning steps only; test misses never come back | EP-183 |
| Self-explanation and "why" questions (g = 0.55) | strong | none in the app | **EP-181** (the explanation), EP-182 |
| Learning by teaching, the protégé effect (g = 0.56 when interactive) | strong | the "Explain one thing learnt" habit | EP-181, EP-182 |
| Generation: try before the worked example (d = 0.40) | strong | none | EP-183 (exam), later |
| Interleaving (g = 0.42, more for similar categories) | moderate | decks take turns | EP-183 (by skill) |
| Metacognition: judgements of learning after a retrieval attempt | moderate | the scheduler's calibration, not the learner's | EP-183 |
| Mastery gates with retries, not grades (Kulik 1990; 42; Khan; Math Academy) | moderate | skills ticked by hand | **EP-181** |
| One hard deadline that moves slowly, plus self-set ones (42's black hole) | moderate | a phase just turns "past" | **EP-181** ("days behind") |
| Sleep and exercise after study | moderate | health logs, unlinked to learning | later |
| Highlighting, rereading, summarising, learning styles | strong null | avoided | — |

## École 42, as a system

Peer-to-peer, no teachers, project-based. A project is **validated in a defense**: the student explains why they did it this way, two or three peers check an **evaluation sheet** (preliminaries, mandatory parts, bonus, one final flag), the mark is 0 to 100, **up to 125 with the bonus**, and any failing flag (crash, leaks, norm, cheat) is 0: a retry, not a penalty. Moulinette, the automated grader, zeroes everything after the first failing exercise. **XP** per project (Libft 462 … webserv 21,630 … ft_transcendence 24,360) scaled by the mark gives a **level** from a lookup table (level 1 at 462, level 10 at 95,000, level 21 at 457,632), displayed with decimals; the common core ends near level 10. The **black hole** starts at 77 days and only validation adds days; at 0 the student leaves. Exams are closed-book machine exams of 4 levels, failed back to level 1. The holy graph limits choice to 1–3 unlocked projects. Coalitions, a wallet, and achievements are the engagement layer. What makes it work: the explanation is mandatory, grading is a single global judgement on known criteria, failure is normal and bounded, and the deadline is global and slow. (Sources: 42.fr, 42 Wolfsburg, the Hevhove/42_projects README, the 42 API dumps in lucas-ht/42calculator, Norm v4.1, Laurie Mézard's 2015 gamification report.)

Axon's plan already has phases, exit checks, 42 projects, proof criteria per domain, and XP. What it lacked was the defense, the sheet, the mark, the retry, and the consequence of a missed phase: **EP-181** adds them in the person's own hands, and **EP-182** lets the Teacher agent play the evaluator.

## Notes, Zettelkasten, Second Brain

| Feature or practice | Who has it | Axon today | Epic |
|---|---|---|---|
| Wikilinks, backlinks, unlinked mentions, rename that rewrites links | table stakes (Obsidian, Logseq, Roam) | yes | — |
| Transclusion `![[note]]`, heading and block links | Obsidian, Logseq, Roam | no: `![[x]]` renders as "!" and a link; `#heading` is ignored | EP-184 |
| Aliases, backlinks with their context sentence | Obsidian | aliases ignored; backlinks without context | EP-184 |
| Outline, find and replace, clickable checkboxes, syntax highlighting | Obsidian | none | EP-184 |
| Typed properties and Bases-style views | Obsidian 1.10, Notion, Anytype, Tana | title, type, and tags only; `created:` ignored | later |
| One idea per note, the literature → permanent step, "why this link" (Ahrens, zettelkasten.de) | method consensus | promotion only rewrites `type:`; no extract; links carry no why | EP-185 |
| Structure, hub, and index notes (zettelkasten.de, LYT) | method consensus | one demo MOC | EP-185 |
| Inbox drained on a cadence, ageing of unprocessed notes (Matuschak) | consensus | fleeting notes older than 2 days are listed | EP-185 |
| Daily note as the hub, interstitial journaling | Roam, Logseq, Reflect, Capacities | a daily note, no yesterday/tomorrow, no timestamped entry | EP-185 |
| PARA, progressive summarisation, project kickoff and completion checklists (Forte) | method | PARA yes; no layers, no kickoff gathering related notes | EP-188 (kickoff), later |
| Highlights in, resurfaced as questions (Readwise, SuperMemo) | differentiating | none | EP-189 |
| Knowledge graph with reasons, communities, close in meaning | Axon's own | yes | — |
| Chat with notes citing sources, MCP for agents | table stakes by 2026 | agents and MCP yes | — |

## Jira, Confluence, Pomodoro, and personal planners

| Feature or practice | Who has it | Axon today | Epic |
|---|---|---|---|
| Priority, estimate, labels on tasks | table stakes (Todoist, Things, Linear, Jira) | none | EP-186 (priority, estimate), EP-187 (labels) |
| Morning planning with estimates and a capacity check; 1–3 most important tasks (Sunsama, Akiflow, Marvin, Ivy Lee) | consensus | an AI suggestion only | EP-186 |
| Shutdown ritual: done, rollover, a line of reflection, tomorrow's three (Newport, Sunsama) | consensus | the evening nudge lists habits | EP-186 |
| Intention before a focus session, reflection after, stats by project (Session, Forest, Rize) | table stakes among focus apps | sessions keep a target and minutes | EP-186 |
| Cycles with a goal, rollover, velocity, burndown (Linear, Jira, Shortcut) | table stakes | none | EP-187 |
| Blocked badge from relations; put an existing task on a board; change a card's kind | table stakes | links exist; the UI cannot | EP-187 |
| Quick add with priority, project, recurrence tokens | table stakes | context and dates only; the Tasks page field does not parse | EP-187 |
| Page tree, templates (meeting, decision, retro, PRD, runbook), page status, comments, versions (Confluence, Notion) | table stakes | a flat list of notes per project | EP-188 |
| Decision journal reviewed on a date, graded on process (Farnam Street, Duke) | practice | yes, but due decisions appear nowhere else | EP-188 |
| Monthly and quarterly reviews | practice | weekly only | EP-188 |
| Time blocking and auto-scheduling (Motion, Reclaim) | differentiating | dates only, no times | later |
| WIP limit, weekly review after GTD, Pomodoro with music, habits with recovery | — | yes | — |

## Learning apps

| Mechanic | Who | Evidence | Axon today | Epic |
|---|---|---|---|---|
| FSRS with parameters fitted to the person (20–30 % fewer reviews than SM-2) | Anki, RemNote | strong | FSRS-5 with default weights | EP-190 |
| Cloze with siblings, image occlusion, tags, `.apkg` import | Anki, RemNote | strong (formulation) | basic cards with why, example, image | EP-190 |
| Confidence before the answer, calibration of the learner | research | moderate | no | EP-183 |
| Timed cumulative quiz whose misses become review tasks | Math Academy, Khan | strong | a self-marked random test, misses dropped | EP-183 |
| Prerequisites reviewed inside new material, lessons locked until reviewed | Math Academy, Execute Program | strong | no | EP-183 (by skill), later |
| Mastery levels per skill with a mixed test and demotion | Khan | moderate | self-rated assessment, fixed bank | EP-183, later |
| Mentoring per exercise | Exercism | moderate | the Teacher agent | EP-182 |
| Streaks with freezes, daily goals, leagues | Duolingo, Boot.dev | engagement | streak and freezes yes | — |

## Sources

Learning science: Dunlosky et al. 2013 (*PSPI* 14:1); Rowland 2014, Adesope 2017, Yang 2021 (testing effect); Cepeda 2006 and 2008 (spacing); Rawson & Dunlosky (successive relearning); Bisra 2018 (self-explanation); Kobayashi 2019 (learning by teaching); Brunmair & Richter 2019 (interleaving); Bertsch 2007 (generation); Rhodes & Tauber 2011 (judgements of learning); Kulik 1990 (mastery learning); VanLehn 2011 (tutoring); Sailer & Homner 2020 (gamification); Pashler 2008 (learning styles); open-spaced-repetition/srs-benchmark (FSRS). École 42: 42.fr, 42wolfsburg.de, github.com/Hevhove/42_projects, github.com/lucas-ht/42calculator (API dumps), github.com/42School/norminette, rbean.io (Mézard 2015), codequoi.com. Notes: zettelkasten.de, notes.andymatuschak.org, fortelabs.com, obsidian.md/help, tana.inc, capacities.io, readwise.io. Work and focus: linear.app/docs, support.atlassian.com, help.sunsama.com, todoist.com/help, culturedcode.com, fs.blog/decision-journal, calnewport.com. Apps: docs.ankiweb.net, mathacademy.com/how-it-works, executeprogram.com, exercism.org/docs.
