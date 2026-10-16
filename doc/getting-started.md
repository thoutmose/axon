# Your first afternoon with Axon

A new Axon is empty on purpose: no notes, tasks, goals, or habits of anyone else. What it does ship is the scaffolding: note templates (fleeting, literature, permanent, film, series, book, project, area, resource, daily, weekly, the retrieval tests…), a library roadmap with its courses, Huberman's protocols, and agents that propose and never act without you. In three to four hours you can make it yours. *Version française : [getting-started.fr.md](getting-started.fr.md).*

Every agent works the same way: it **proposes**, you **read the preview**, you **approve or reject**. Nothing changes until you approve ([ADR 0008](adr/0008-agents-propose-people-approve.md)).

## Before you start (15 minutes)

1. Install Axon ([install guide](install.md)) and register with a passkey. Keep the recovery codes.
2. **Account → Preferences**: language, time zone, theme, and the icons.
3. **Account → Preferences → AI provider** (or the **Agents** page): a key (Anthropic, OpenAI, Mistral, Gemini, DeepSeek) or a local model (Ollama, LM Studio, vLLM). Without one, Axon works fully; only the agents wait.
4. Say what you want Axon for on the welcome screen: the menu then shows just that, and grows as you use it.

## Hour 1 — Where you are going

1. **Grow → Roadmaps**: start the library's roadmap (a broad engineering plan with its courses), or ask the **Tutor** to draft your own ("I want to become a backend engineer in 18 months, 10 hours a week"). Take the skills assessment so the roadmap starts at your level.
2. **Goals**: ask the **Goals** agent to break the roadmap's outcome into milestones with dates you can keep.
3. **Projects**: create one project per outcome with a finish line, from a template (software, learning, event, home, writing). With a GitHub repository, the **Side-Projects** agent drafts epics and issues from it.

## Hour 2 — What you do every day

1. **Habits**: ask the **Habits** agent for two or three habits stacked on what you already do, each with its two-minute version. Link a reading habit to its book ("Read 20 pages" → the book): checking it in turns the pages.
2. **Tasks**: press `n` anywhere to capture; the **Task Manager** sorts the inbox into next actions by context. The GTD board shows five cards a column, the rest one click away.
3. **Focus**: set your Pomodoro rounds and the music.
4. **Health** (optional, with its own consent): your profile for daily needs; the **Chef & Nutritionist** plans meals within your budget and restrictions. The life web scores the day.

## Hour 3 — What you learn and keep

1. **Notes**: connect your vault (Obsidian or any folder of Markdown), or keep it in Axon. The templates land in `Templates/`; edit them like any note.
2. Write literature notes as you read or watch: from a film, series, or book, **Write a note** starts from its template. The **Teacher** turns literature and permanent notes into retrieval practice and flashcards (Anki sync if you like).
3. **Tech watch**: follow a few sources, YouTube channels, or your Audiobookshelf podcasts; the digest picks the best each morning.

## The last half hour — Staying in touch

1. **Digest**: choose its times and where it goes (email, Slack, Discord, Teams, Telegram, ntfy), and send a test.
2. **Budget** (optional): import a bank CSV, add rules for the payees you see every month; Reports shows where the money went.
3. Put the **weekly review** in your calendar: Axon walks you from an empty head to next week.

From then on, the agents on each page (**Ask an agent**) are the fastest way to set up anything new: describe what you want, read what they propose, keep what fits.
