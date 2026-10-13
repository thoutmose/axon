# Privacy register

What Axon stores, why, for how long, and who else sees it. Update this file in the same PR as any change that adds a table, an external call, or a new kind of personal data.

## Data inventory

| Category | Where | Classification | Purpose | Retention |
|---|---|---|---|---|
| Account (name, creation time) | `users` | Personal | Identify the account owner | Until account erasure |
| Skills assessments (role, experience, goals, hours per week, preferred formats, self-ratings, check results, levels) | `assessments` | Personal | Place the person on roadmaps; read by the Tutor and Resources agents with `ai_agents` consent | Until deleted or erasure |
| Preferences (language, appearance, note/project/resource templates) | `user_settings` | Personal | Apply the person's choices on every device | Until changed or erasure; included in exports |
| Profile picture (256×256 PNG, re-encoded without metadata) | `users.avatar` | Personal | Show who is signed in | Until replaced, removed, or erasure; included in exports as `account/avatar.png` |
| Habits scorecard (behaviours of the day with good, neutral, or bad ratings) and habit identity, intention, and contract fields | `habit_scorecards`, `habits` | Personal, possibly sensitive | Atomic Habits design (EP-34) | Until changed or erasure; included in exports; nothing is sent to an accountability partner |
| Focus sessions (what was focused on, start, minutes worked, rounds) and Pomodoro preferences, including music links | `focus_sessions`, `user_settings` | Personal | Count focus time and show it on the dashboard and linked items | Until deleted or erasure; included in exports |
| Flashcards (question, answer, deck, optional picture, FSRS schedule) and review history (rating, time, duration) | `cards`, `card_reviews` | Personal | Retrieval practice and scheduling | Until deleted or erasure; both exported |
| Anki sync account (sync server address, user name, password sealed with `AXON_ENCRYPTION_KEY`, last sync status) and the bridge's copy of the person's Anki collection | `anki_accounts`; `ankidata` volume | Secret (password), personal | Send flashcards to the person's own anki-sync-server | Until disconnected or erasure; the password is never exported; the collection copy is derived and rebuilt from the sync server |
| Streak freezes (bought and used dates) and the progress-rewards switch; points, levels, and badges are computed, not stored | `streak_freezes`, `user_settings` | Personal | Opt-in progress rewards (EP-15) | Until erasure; freezes exported |
| Demo account and its sample data (no personal data; whatever visitors type is visible to other visitors) | `users` with `demo = true`, and the rows it owns | Sample, shared | Let people try Axon (ADR 0015) | Reset every night |
| Passkey public keys, names, last use | `passkeys` | Personal, security | Sign-in | Until removed or erasure |
| Sessions (token hash, user agent, times) | `sessions` | Personal, security | Keep the user signed in | 7 days idle, 30 days absolute, or sign-out |
| Recovery codes (SHA-256 hashes) | `recovery_codes` | Security | Account recovery | Until used, reissued, or erasure |
| TOTP secret (AES-256-GCM sealed) | `users.totp_secret` | Security | Second factor for recovery | Until turned off or erasure |
| Audit events (action, IP, user agent, time) | `audit_events` | Personal, security | Let the owner review account activity | Until erasure |
| Consent records | `consents` | Personal | Prove and honour the owner's choices | Until erasure |
| WebAuthn challenges (and, while registering, the browser's time zone) | `webauthn_ceremonies` | Transient | Complete a passkey ceremony | 5 minutes |
| Personal access tokens (SHA-256 hashes, names, use times) | `api_tokens` | Security | Script and sync access | Until expiry (≤ 365 days) or revocation |
| Idempotency records (stored responses) | `idempotency_keys` | Personal (copies of responses) | Safe retries | 24 hours |
| Tasks (titles, notes, dates, contexts, recurrence) and check-ins | `tasks`, `checkins` | Personal | Planning and doing | Until deleted or erasure |
| Account time zone (from the browser that created the account, then as the owner sets it) | `users.timezone` | Personal | Decide what “today” means | Until changed or erasure |
| Notes (Markdown exactly as written, path, derived title/type/PARA folder) | `notes` | Personal | Knowledge capture and vault sync | Until deleted or erasure |
| Goals (title, outcome, status, target date, measure) and milestones | `goals`, `milestones` | Personal | Planning towards outcomes | Until deleted or erasure |
| Calendar feeds (name, chosen types, SHA-256 of the URL token, last use) | `calendar_feeds` | Security | Let a calendar app read dated items | Until revoked or erasure |
| AI provider settings (provider, model, local address, prices; key sealed with `AXON_ENCRYPTION_KEY`) | `agent_providers` | Secret (key), personal | Run agents on the person's chosen provider | Until changed, removed, or erasure; the key is never exported |
| Agent budget | `agent_settings` | Personal | Cap AI spending | Until changed or erasure |
| Agent run records (agent, skill version, model, tool names, tokens, cost, outcome; no content) | `agent_runs` | Personal, operational | Audit and budget | Until erasure |
| Agent proposals (proposed change and its summary) | `agent_proposals` | Personal | Let the user approve or reject | Until erasure |
| Agent memories (facts the user approved) | `agent_memories` | Personal | Context for later runs | Until deleted or erasure |
| Agent replies and conversation transcripts | API process memory only | Personal | Show the run's answer and let the person answer the agent | 1 hour; never stored or logged |
| Habits (design, schedule, status) and habit check-ins (day, status, note) | `habits`, `habit_checkins` | Personal | Building habits | Until deleted or erasure |
| Projects (title, outcome, status, WIP limit, docs folder) and board cards (column, order, kind, parent) | `projects`, `project_cards` | Personal | Running projects | Until deleted or erasure |
| Repository connections (owner/name, token sealed with `AXON_ENCRYPTION_KEY`, comment switch, project) | `repo_connections` | Secret (token), personal | Let delivery agents read repositories | Until disconnected or erasure; the token is never exported |
| Roadmaps and skills (titles, sections, links, status) | `roadmaps`, `skills` | Personal | Planning what to learn | Until deleted or erasure |
| Paths of deleted notes | `note_tombstones` | Personal | Let a sync client delete the file too | 30 days, or erasure |
| Sync state (path → content hash; no content) | `.axon-sync-<vault>.json` next to the vault, on the vault host | Personal | Detect changes on both sides | Until the file is deleted |
| Entities, links, events | `entities`, `links`, `events` | Personal | The universal graph | Until deleted or erasure; events keep field names only |
| Health, food, training data (EP-18), water drunk (EP-101), and the profile daily needs are worked out from: birth year, sex, height, weight, activity, pregnancy (EP-88) | `food_entries`, `workouts`, `health_profiles`, and the journal's mood, energy, stress, sleep, gratitude, and notes (EP-91) in `journal_entries` | **Special category (GDPR Art. 9)** | Health tracking, and What helps you, worked out on request and not stored (EP-92) | Not stored before explicit `health_data` consent; deleted when it is withdrawn |
| Request logs (method, route pattern, status, duration, request ID) | Loki on `srv-monitoring` | Operational, no personal data | Operations | Fleet Loki retention |
| Database backups (encrypted) | `/opt/axon/backups`, `srv-services` | Contains all of the above | Disaster recovery | 14 days on host, 30 days off-host |

Erased accounts can survive in encrypted backups for up to 30 days; a restore from such a backup must re-apply erasures made since the backup was taken.

## Consent purposes

All off by default, revocable at any time in **Account → Privacy choices**, and recorded with a timestamp and an audit event.

| Purpose | What it allows |
|---|---|
| `ai_agents` | Sending selected data to the configured AI provider (Anthropic) so agents can draft suggestions |
| `external_metadata` | Looking up films, series, and books with TMDB and Open Library |
| `health_data` | Storing health data at all |
| `food_photos` | Sending a photo of a meal to the person's AI provider to estimate it (EP-89); the photo is re-encoded without camera or location details and never stored |

Features must call `privacy.Granted` before acting on a purpose.

## Processors and external services

| Service | Receives | When | Status |
|---|---|---|---|
| GitHub | Requests to read the repositories the person connected; approved comments | Only for connected repositories, during runs the person starts | Available (EP-26) |
| Allowlisted educational sites (Wikipedia, MDN, MIT OCW, Coursera, edX, Khan Academy, arXiv, language docs, freeCodeCamp, OpenStax, Gutenberg, OSSU) | A GET request for a public page the Tutor or Resource agent chose, with Axon's User-Agent; no user data beyond what the URL itself contains | Only during runs the person starts, at most 15 pages per run, robots.txt honoured (ADR 0014) | Available (EP-27) |
| The person's own anki-sync-server (for example on `srv-services`) | Their flashcards (questions, answers, pictures) and their Anki credentials, through the Anki bridge | Only when they press "Sync to Anki now", or after approving Teacher cards with automatic sync on | Available (EP-42) |
| Tailscale | Connection metadata between devices and hosts; TLS certificate requests for `*.ts.net` | Always (network layer) | In use |
| Let's Encrypt (via Tailscale) | Host names (`srv-*.tail781d53.ts.net`) | Certificate issuance | In use |
| GitHub | Source code, CI logs (no user data) | Development | In use |
| The AI provider the person chose: Anthropic, OpenAI, Mistral, Google (Gemini), DeepSeek, or their own local server | The instruction, and the tasks, goals, habits, and notes an agent reads through its tools during a run (the Teacher reads only literature and permanent notes) | Only with `ai_agents` consent, during a run they start | Available (EP-19, EP-22) |
| TMDB, Open Library | Search terms for media | Only with `external_metadata` consent | Planned (EP-17) |
| Open Food Facts, USDA FoodData Central | Food search terms | Only with `health_data` consent | Planned (EP-18) |

No analytics or telemetry leaves the deployment.

## Rights

| Right | How |
|---|---|
| Access and portability | **Account → Download everything**: ZIP with `axon-export.json` and a Markdown summary |
| Erasure | **Account → Delete account**: rows are deleted immediately; a test proves no table keeps them |
| Objection / withdrawal | **Account → Privacy choices** |
| Rectification | Edit the data in place (Account → Name) |
