# Operations runbook

How Axon runs on its servers. Host configuration (SSH, Docker, monitoring, logs) belongs to the fleet repository [`terraform-proxmox`](https://github.com/thoutmose/terraform-proxmox); this runbook covers only the application ([ADR 0004](adr/0004-hosts-and-tailnet-only-access.md)).

## Environments

| Environment | Host | URL (tailnet only) |
|---|---|---|
| Staging | `srv-staging` | https://srv-staging.tail781d53.ts.net |
| Production | `srv-prod` | https://srv-prod.tail781d53.ts.net |
| Dev (a branch under review) | `srv-dev` (or `AXON_DEV_HOST`) | https://srv-dev.tail781d53.ts.net |

Each host runs the Compose stack in `/opt/axon`: Caddy on `127.0.0.1:3000`, the API (with its background workers), web, PostgreSQL 16 + pgvector, the Anki bridge, and the backup service. Tailscale Serve provides HTTPS in front of Caddy. Nothing listens publicly.

## Prerequisites on the operator machine

- Docker, `age`, `rsync`, and SSH entries for the hosts in `~/.ssh/config`.
- The age private key at `~/.config/axon/age.key` (or set `AXON_AGE_KEY`).

## Secrets

Server settings live in `deploy/env/{staging,prod}.env.age`, encrypted to the public keys in `deploy/env/recipients.txt`. Backups are encrypted to the same keys.

```bash
age -d -i ~/.config/axon/age.key deploy/env/staging.env.age            # read
age -d -i ~/.config/axon/age.key deploy/env/staging.env.age > /tmp/s    # edit…
age -e -R deploy/env/recipients.txt -o deploy/env/staging.env.age /tmp/s && rm /tmp/s
```

Settings that are not secret (for example `AXON_DEMO`) live in plain text in `deploy/env/{staging,prod}.settings.env`. `deploy.sh` appends them to the decrypted secrets, so they are reviewed in pull requests like code. Never repeat a secret's name there.

> [!CAUTION]
> Losing every private key listed in `recipients.txt` makes the env files and **all backups** unreadable. Keep a copy of `~/.config/axon/age.key` in a password manager. To add a key holder, append their public key to `recipients.txt` and re-encrypt both env files.

## Deploy

Deployments use the committed state of the current checkout. Always deploy to staging first.

```bash
deploy/deploy.sh staging
deploy/deploy.sh prod
```

The script refuses to deploy a commit without a green `ci` run on GitHub, and refuses a production version that staging is not serving. Deploy from `main`: pull request runs leave out the image build, vulnerability scan, and SBOM to save Actions minutes, and skip documentation-only changes entirely. After migrating it runs `axon integrity`, which fails the deployment if any entity lacks its domain row or has an unregistered type. It then builds the images, copies them over SSH, copies the configuration, runs pending migrations, starts the stack, enables Tailscale Serve, and fails unless `/api/readyz` answers within 30 seconds.

A branch under review goes to dev, from its checkout: `deploy/deploy.sh dev`. While CI cannot run (GitHub Actions minutes spent until 1 November 2026), `deploy/deploy.sh dev --skip-ci` deploys without it; the flag is refused for staging and production, whose rules do not change. Dev reads `deploy/env/dev.settings.env`, and staging's secrets until `deploy/env/dev.env.age` exists. Dev's backup key is not authorized on `srv-services`, so its off-site copies fail until `deploy/setup-offsite.sh` covers it.

## CI runner

CI runs on a self-hosted GitHub Actions runner on srv-dev (`~/actions-runner`, systemd unit `actions.runner.thoutmose-axon.srv-dev`, label `srv-dev`), so it uses no Actions minutes; pushes to `main` and `dev` and pull requests run there one job at a time. Its Docker bridge networks have no internet, so builds and scans use the host's network; the e2e job runs its stack as compose project `axon-ci` on port 3200, beside the demo on 3000, and removes it afterwards. Watch the disk (`df -h /`): each job prunes dangling images; `docker builder prune -af` frees the build cache. Status: `sudo ~/actions-runner/svc.sh status`; to remove it, `sudo ~/actions-runner/svc.sh uninstall` and `./config.sh remove` with a removal token (`gh api -X POST repos/thoutmose/axon/actions/runners/remove-token`).

## Roll back

```bash
deploy/deploy.sh prod --rollback
```

Restarts the API and web app on the previously deployed version. **Migrations are not reverted.** If the bad release added a migration that the old version cannot work with, roll back the schema first:

```bash
ssh srv-prod 'cd /opt/axon && docker compose run --rm --no-deps api migrate down'
```

## Backups

- The `backup` service dumps the database every day at 03:15 UTC, encrypts it with age, and keeps 14 days in `/opt/axon/backups`.
- Each dump is copied to `srv-services:~/backups/axon-<env>/`, kept there for 30 days. The server's key can only write into that folder; it cannot list, read, or delete.
- `/opt/axon/backups/last-success` holds the time of the last good backup.
- Erased user data can survive in backups for up to 30 days.

Run a backup now:

```bash
ssh srv-prod 'cd /opt/axon && docker compose exec -T backup axon-backup now'
```

First deployment of a new environment only: authorize its backup key on `srv-services` with `deploy/setup-offsite.sh staging|prod`.

## Restore

Targets: RPO ≤ 24 h, RTO ≤ 4 h. Drill on 2026-10-10 (staging, empty schema): 4 seconds from download to restored database.

```bash
f=$(ssh srv-services 'ls -t ~/backups/axon-prod/*.dump.age | head -1')
scp "srv-services:$f" latest.dump.age
age -d -i ~/.config/axon/age.key -o latest.dump latest.dump.age

# Into the running server (replaces current data — stop the API first):
scp latest.dump srv-prod:/tmp/latest.dump
ssh srv-prod 'cd /opt/axon && docker compose stop api web \
  && docker compose exec -T db sh -c "pg_restore -U \$POSTGRES_USER -d \$POSTGRES_DB --clean --if-exists --no-owner" < /tmp/latest.dump \
  && docker compose start api web && rm /tmp/latest.dump'
rm latest.dump latest.dump.age
```

To rehearse without touching a server, restore into a throwaway `pgvector/pgvector:pg16` container and check `goose_db_version` and the `vector` extension.

## Move an instance (EP-40)

`axon instance export` writes the whole installation to one gzipped tar archive. It contains every table, uploaded pictures included (they are stored in the database), plus a `manifest.json` with the Axon version, the schema version, row counts, a SHA-256 per table, and a fingerprint of `AXON_ENCRYPTION_KEY`. The archive holds personal data and sealed secrets, so keep it like a backup: encrypt it at rest and delete it after use.

`axon instance import`:

- loads an archive only into a **fresh installation** (no accounts, every table empty);
- moves the schema to the archive's version and copies every table in one transaction, checking each count and checksum;
- then applies newer migrations, so an archive from an older Axon goes into a newer one;
- refuses an archive from a newer Axon, a damaged archive, or a different `AXON_ENCRYPTION_KEY`, because TOTP secrets, AI provider keys, and tokens could not be opened without the same key;
- needs the database's superuser (the compose `POSTGRES_USER` is one).

The Anki bridge volume is a cache rebuilt from the sync server, so it is not part of the archive.

```bash
# On the old server (read-only, the API keeps running):
ssh old 'cd /opt/axon && docker compose exec -T api axon instance export' > axon-instance.tar.gz

# On the new server, after a normal install with the same AXON_ENCRYPTION_KEY:
ssh new 'cd /opt/axon && docker compose exec -T api axon migrate up \
  && docker compose stop web && docker compose exec -T api axon instance import \
  && docker compose start web && docker compose exec -T api axon integrity' < axon-instance.tar.gz
rm axon-instance.tar.gz
```

Drill on 2026-10-11 (local stack, demo data, 47 tables): export, `docker compose down -v`, a fresh install, and import took 42 seconds. Integrity was ok, the demo account signed in with its notes, and a second import was refused. `internal/instance` runs the same drill in the integration tests, with an older-schema archive, a wrong key, and a damaged table.

## Tech watch (EP-110)

A worker in the API fetches each person's sources when their schedule comes (cron in the person's time zone, midnight by default), four at a time, with `If-None-Match`/`If-Modified-Since`. It needs outbound HTTPS to the sites people follow; sources on a private network need `AXON_CALENDAR_PRIVATE=1`. A failing source shows its error in **Tech watch → Sources and health** and is retried after 15 minutes, then up to 8 hours. Metrics on the metrics port: `axon_watch_fetches_total{result="ok|not_modified|error"}`, `axon_watch_items_total{outcome="new|merged|skipped"}`, `axon_watch_fetch_seconds`. A rising error rate in Grafana usually means one site changed its feed; logs say `tech watch fetch failed` with the source id. News nobody saved, kept, or linked are purged after 180 days. The shared demo fetches nothing. YouTube videos (EP-123) get their words after each round, three at a time: the API asks `www.youtube.com` for the video's captions (the author's, else the automatic ones); a video with none is sent, as its smallest audio stream (at most 60 MB and 3 hours), to `AXON_WHISPER_URL` when it is set, a server on your own network being fine. Each video is tried once (`watch_items.transcript_tried_at`); **Read the transcript** on an item tries again by hand. YouTube changes its player now and then: if no video gets captions any more, the logs say `transcript failed` and the rest of the tech watch is unaffected. Podcasts from Audiobookshelf (EP-124): people connect their own server in Preferences; the API calls it with their API token (sealed with `AXON_ENCRYPTION_KEY`, so set it), and a server on the LAN or tailnet needs `AXON_CALENDAR_PRIVATE=1`. Shows are read on the tech watch's schedule; episodes are transcribed from the person's server when `AXON_WHISPER_URL` is set. The podcast (EP-132) speaks with natural voices when `AXON_TTS_URL` points to a server with the OpenAI speech API: the bundled Kokoro (`docker compose --profile tts up -d`, then `AXON_TTS_URL=http://tts:8880`; about 4 GB on disk, a part of a minute takes a few seconds on four cores) or your own (`AXON_TTS_MODEL`, `AXON_TTS_TOKEN`). Episodes are spoken part by part as people play them, nothing stored; without the server, the browser's voices read them. Share links (EP-112) are served at `/api/v1/shared/watch/{token}` without a session, rate-limited per address (30 at once, then one every 2 seconds); creating and revoking one is in the audit log as `watch_share.created` and `watch_share.revoked`.

## Digests and chat channels (EP-107)

People choose times and channels in Account → Preferences → Reminders and digests; nothing needs configuring on the server beyond what reminders use (`AXON_SMTP_URL`/`AXON_SMTP_FROM` for email, `AXON_ENCRYPTION_KEY` so channel secrets are sealed). The API calls the chat services' own hosts (`hooks.slack.com`, `slack.com`, `discord.com`, `*.webhook.office.com` or `*.logic.azure.com`, `api.telegram.org`) and the ntfy servers and webhooks people chose (public addresses only, unless `AXON_CALENDAR_PRIVATE=1`), so the server needs outbound HTTPS to them. A channel that fails shows its last error in Preferences; the logs say `digest to a channel failed` with the channel id.

Answers (EP-108) come through connections the API opens itself: Telegram long polling (`api.telegram.org`), Slack Socket Mode (`slack.com`, then a WebSocket to `wss-primary.slack.com`), and the Discord gateway (`gateway.discord.gg`). No inbound port is needed. Each bot has one listener, refreshed every 30 seconds; a lost connection is retried with back-off (logs: `chat listener stopped; reconnecting`). The demo instance starts none.

## Vault server set up from Axon (EP-106)

Notes live in the database unless a person names a vault server. To name one: Account → Preferences → Markdown vault → **Also as files on a server**, enter the host and folder, **Set up this server**, and run the line shown on that server within 30 minutes. It downloads `axon` from this Axon (same OS and architecture only; otherwise use the Docker command shown), pairs with the one-time code, saves a write token to `~/.config/axon/vault-token` (0600), syncs once, and installs the user service `axon-vault` (`systemctl --user status axon-vault`; `sudo loginctl enable-linger <user>` keeps it running after logout). An empty folder gets the inbox, PARA, Zettelkasten, daily, template, and attachment folders, `Home.md`, and the templates. Preferences shows whether the vault is in step (a report at least every 5 minutes), silent, or failing. **Stop syncing** revokes the token. For fleet hosts, the fleet repository runs the same `notes-sync -pair` command (ADR 0004).

## Live notes sync with SecondBrain (EP-36)

Run this once on `srv-secondbrain` to import `~/SecondBrain` (notes and its `Templates/` folder) and keep it in step both ways. Create a write token first under Account → Access tokens, and save it to `~/.config/axon/notes-token` (mode 0600):

```bash
docker run -d --name axon-notes-sync --restart unless-stopped \
  -v "$HOME/SecondBrain:/vault" -v "$HOME/.config/axon/notes-token:/token:ro" \
  axon-api:<version> notes-sync -dir /vault -state /vault/.axon-sync.json \
  -url https://srv-prod.tail781d53.ts.net -token-file /token -watch
```

- Create, edit, rename, or delete a Markdown file and the note follows within seconds, and the other way round (tested within 10 s).
- A note edited on both sides keeps both versions (`*.axon-conflict.md`).
- Pictures and PDFs (`.png`, `.jpg`, `.gif`, `.webp`, `.svg`, `.pdf`, up to 10 MB each, 500 MB per person) sync the same way, byte for byte; a conflict keeps Axon's copy as `<name>.axon-conflict.<ext>`. A file that is not what its name says is reported as skipped and stays local.
- Notes in `Templates/` appear as templates when you create a note in Axon; `{{title}}` and `{{date}}` are filled in.
- `-watch` replaces `-git`: commit the vault on your own schedule, for example with a nightly cron `git -C ~/SecondBrain commit -am sync && git push`.
- The service definition belongs to the fleet repository (ADR 0004); this command is what it should run.

## Notes sync with SecondBrain

`axon notes-sync` runs on the host that holds the vault (`srv-secondbrain`, `~/SecondBrain`) and talks to Axon over the tailnet with a personal access token. It only ever reads and writes inside the vault folder.

1. In Axon, under Account → Access tokens, create a **write** token named after the host. Save it on the vault host as `~/.axon-token` with mode `600`.
2. Copy the binary out of the release image (`docker create` the `axon-api` image, `docker cp <id>:/usr/local/bin/axon ./axon`) or build it with `go build ./cmd/axon`.
3. Start with one-way import and check the report:
   ```sh
   axon notes-sync -dir ~/SecondBrain -url https://srv-prod.tail781d53.ts.net -token-file ~/.axon-token
   ```
4. Templates come with the vault: notes under `Templates/` (or `_templates/`) are offered in Axon's New note dialog, with Obsidian and Templater placeholders filled. Check the report lists them, then open Notes → New note.
5. Once the import looks right, schedule it (systemd timer or cron, every 15 minutes). Add `-two-way -git` to write Axon's edits back and commit/push the repository.

Conflicts are never resolved automatically: Axon's version is written next to the file as `<name>.axon-conflict.md` and the run reports it. Merge into the original, delete the conflict file, and the next run sends the merged version. The sync state (hashes only) lives in `.axon-sync-SecondBrain.json` beside the vault; deleting it makes the next run compare everything again, which is safe.

## Demo account

With `AXON_DEMO=1` (on for staging and production in the settings files), the landing page offers **Try the demo**. It is a shared account with sample data in every area, and it needs no passkey (ADR 0015).

- The API creates the demo account once migrations have run, then resets it every night after 03:00 UTC.
- To reset it now: `docker compose exec api axon demo reset`.
- The demo cannot create tokens or passkeys, set AI providers, connect repositories, delete the account, or run agents. Real accounts never see its data.
- To turn it off, set `AXON_DEMO=` in the settings file and deploy. The demo user stays in the database, unused, until you run `axon demo reset` again or remove it with `DELETE FROM users WHERE demo`.


To give a real account what the demo is built on, without its invented history (the library roadmap started as a plan, the first course's literature note and flashcards, the learning notes, the universities and schools as resources, the roadmap's funds without money in them, the skills assessment, Huberman's protocols as habits), run on the server `docker compose exec -T api axon demo starter --user '<display name or id>'`. It skips what the account already has, so it can run again.

## Media catalogues (EP-17)

- **Books** come from Open Library with no key. Set `AXON_CONTACT` (an email address) so Open Library can reach you; it is sent in the User-Agent. Without it, Axon sends `AXON_BASE_URL`.
- **Films and series** need `TMDB_API_KEY`: your own key from your TMDB account settings (a v3 API key or a v4 read access token). TMDB's base licence is non-commercial, so a paid or revenue-generating Axon needs a separate agreement with TMDB. Keep the key in the encrypted env file, never in `*.settings.env`.
- Without a key, or when a catalogue is down, people add items by hand.
- Searches are cached for a week. Covers are copied into `media_covers` once and served by Axon. A daily job drops TMDB covers and descriptions older than six months, as TMDB's terms require.

## Food catalogues and health data (EP-18)

- Health logs need each person's `health_data` consent. Withdrawing it deletes their food entries and workouts at once; an operator cannot turn this off.
- Open Food Facts needs no key; Axon identifies itself with `AXON_CONTACT`. The server stays under Open Food Facts' 10 searches and 15 product reads a minute (beyond that, people see "busy" for a few seconds).
- `USDA_API_KEY` (free from data.gov, 1,000 requests an hour) turns USDA FoodData Central search on. Keep it in the encrypted env file.
- Foods are cached for 30 days in `foods`. They are shared and hold no personal data.

## Vault mirror (EP-48)

A background worker in the API (`internal/mirror`) reads the event log every 3 seconds and keeps each person's items and their notes under `Axon/` in step, both ways; vault sync then carries the notes to the Markdown vault. Health logs and journal days (daily notes under `Health/Journal/`, the scales in the front matter) are written only when the person turned them on, one way, and are removed when the health consent is withdrawn. The first run writes a note for every existing item. Tables: `vault_mirror` (item ↔ note), `vault_mirror_users`, `vault_mirror_queue` (habit check-ins and health logs), `vault_mirror_cursor`. Nothing to configure; people turn it off or change the folder in Preferences.

## Anki packages

`GET /api/v1/flashcards/export.apkg` downloads flashcards as an Anki package. To check that Anki still imports what Axon writes (CI does this on every change): `sh scripts/check-apkg.sh`. It needs Docker and pulls the official `anki` library into a throwaway container. 

### Anki bridge (EP-42)

The `anki` service (image `axon-anki`, deployed with the rest) syncs people's flashcards to their own anki-sync-server.
- Set `ANKI_SYNC_URL` in the settings file to suggest the address of the server on `srv-services`.
- Each person enters their own Anki user name and password under Learn → Flashcards → Anki.
- To check the bridge against a disposable sync server: `sh scripts/check-anki-sync.sh` (CI runs it too).
- If the `ankidata` volume is lost, the next sync downloads each person's collection from the sync server again; nothing is lost.

## Data backfills

`axon migrate up` (run by `deploy/deploy.sh`) also fills data that needs Go to compute for rows written before it existed, such as note tags. It is safe to repeat.

## AI agents

Each person can choose their own provider under **Agents → AI provider**: Anthropic, OpenAI, Mistral, Gemini, DeepSeek, or a local server such as Ollama (`http://<host>:11434/v1`). Keys they enter are sealed with `AXON_ENCRYPTION_KEY`, so that variable must be set to store keys.

The server can also offer a default Anthropic account to everyone:

- **With a key:** add `ANTHROPIC_API_KEY=…` to the encrypted env file (see [Secrets](#secrets)) and deploy.
- **Without storing a key:** on the host, run `ant auth login --no-browser` as the deploy user with `ANTHROPIC_CONFIG_DIR=/opt/axon/anthropic`, then add `AXON_ANTHROPIC_OAUTH=1` and `ANTHROPIC_CONFIG_DIR=/config/anthropic` to the env file, mount `/opt/axon/anthropic:/config/anthropic` into the `api` service in `deploy/compose.server.yaml`, and deploy. This bills the Anthropic Console organisation you signed in with. A Claude Pro or Max subscription cannot be used.

Each person must still grant the `ai_agents` consent, and their monthly budget (default $2) caps spending. Remove the default and every other feature keeps working.

Run outcomes are logged as `agent run finished` with status, tool count, tokens, and cost; prompts and replies are never logged. A run stuck as `running` after a restart is marked `failed` (`interrupted`) when the same person next starts one.

## Load testing (EP-21)

`api/cmd/axon-load` sends a mix of reads and writes (today, tasks, notes, search, dashboard overview, insights, habits, and 10 % task creation) with a write-scoped personal access token, and prints throughput and p50/p95/p99 per endpoint. It creates tasks, so run it only against a local stack or a throwaway instance:

```bash
cd api && AXON_TOKEN=axon_… go run ./cmd/axon-load -url http://localhost:3000/api -c 20 -d 60s
```

Measured on 2026-10-11 (4 vCPU, 7 GB; the demo account plus 5,000 tasks): 10 clients reach 481 requests/s at p95 79 ms, and 50 clients reach 713 requests/s at p95 197 ms, with no errors. The dashboard overview is the slowest endpoint. [ADR 0021](adr/0021-single-host-measured.md) keeps the single-host design on these numbers.

## Service level objectives

Measured only while the host is powered on (`srv-staging` and `srv-prod` are on-demand VMs in the fleet).

| Objective | Target | Measured by |
|---|---|---|
| Availability | 99% of minutes per 30 days with the API up and its database reachable | `up{job="axon"}`, `axon_db_up` |
| Errors | Fewer than 1% of requests return 5xx per 30 days | `axon_http_requests_total` |
| Latency | 95% of requests (excluding health checks) under 500 ms | `axon_http_request_duration_seconds` |
| Backups | A successful backup at least every 26 hours | `axon_backup_last_success_timestamp_seconds` |

Alerts and what to do about them: [runbooks.md](runbooks.md).

## Monitoring

Monitoring is the fleet stack on `srv-monitoring` (Prometheus, Grafana, Loki, Alertmanager → ntfy), configured in [`terraform-proxmox`](https://github.com/thoutmose/terraform-proxmox). Axon contributes:

- API metrics on port 9464 of each host (`/metrics`, separate from the public routes).
- Backup freshness through the host's node_exporter textfile collector (`axon_backup.prom`).
- JSON request logs (method, route pattern, status, duration, request ID; never paths, queries, headers, or bodies), shipped to Loki by the fleet's Promtail.
