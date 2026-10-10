# Privacy register

What Axon stores, why, for how long, and who else sees it. Update this file in the same PR as any change that adds a table, an external call, or a new kind of personal data.

## Data inventory

| Category | Where | Classification | Purpose | Retention |
|---|---|---|---|---|
| Account (name, creation time) | `users` | Personal | Identify the account owner | Until account erasure |
| Passkey public keys, names, last use | `passkeys` | Personal, security | Sign-in | Until removed or erasure |
| Sessions (token hash, user agent, times) | `sessions` | Personal, security | Keep the user signed in | 7 days idle, 30 days absolute, or sign-out |
| Recovery codes (SHA-256 hashes) | `recovery_codes` | Security | Account recovery | Until used, reissued, or erasure |
| TOTP secret (AES-256-GCM sealed) | `users.totp_secret` | Security | Second factor for recovery | Until turned off or erasure |
| Audit events (action, IP, user agent, time) | `audit_events` | Personal, security | Let the owner review account activity | Until erasure |
| Consent records | `consents` | Personal | Prove and honour the owner's choices | Until erasure |
| WebAuthn challenges | `webauthn_ceremonies` | Transient | Complete a passkey ceremony | 5 minutes |
| Personal access tokens (SHA-256 hashes, names, use times) | `api_tokens` | Security | Script and sync access | Until expiry (≤ 365 days) or revocation |
| Idempotency records (stored responses) | `idempotency_keys` | Personal (copies of responses) | Safe retries | 24 hours |
| Tasks (titles, notes, dates, contexts, recurrence) and check-ins | `tasks`, `checkins` | Personal | Planning and doing | Until deleted or erasure |
| Account time zone | `users.timezone` | Personal | Decide what “today” means | Until changed or erasure |
| Notes (Markdown exactly as written, path, derived title/type/PARA folder) | `notes` | Personal | Knowledge capture and vault sync | Until deleted or erasure |
| Goals (title, outcome, status, target date, measure) and milestones | `goals`, `milestones` | Personal | Planning towards outcomes | Until deleted or erasure |
| Calendar feeds (name, chosen types, SHA-256 of the URL token, last use) | `calendar_feeds` | Security | Let a calendar app read dated items | Until revoked or erasure |
| Paths of deleted notes | `note_tombstones` | Personal | Let a sync client delete the file too | 30 days, or erasure |
| Sync state (path → content hash; no content) | `.axon-sync-<vault>.json` next to the vault, on the vault host | Personal | Detect changes on both sides | Until the file is deleted |
| Entities, links, events | `entities`, `links`, `events` | Personal | The universal graph | Until deleted or erasure; events keep field names only |
| Health, food, training data (EP-18) | — | **Special category (GDPR Art. 9)** | Health tracking | Not stored before explicit `health_data` consent |
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

Features must call `privacy.Granted` before acting on a purpose.

## Processors and external services

| Service | Receives | When | Status |
|---|---|---|---|
| Tailscale | Connection metadata between devices and hosts; TLS certificate requests for `*.ts.net` | Always (network layer) | In use |
| Let's Encrypt (via Tailscale) | Host names (`srv-*.tail781d53.ts.net`) | Certificate issuance | In use |
| GitHub | Source code, CI logs (no user data) | Development | In use |
| Anthropic | Data the user selects for an agent task | Only with `ai_agents` consent | Planned (EP-19) |
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
