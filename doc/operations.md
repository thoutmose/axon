# Operations runbook

How Axon runs on its servers. Host configuration (SSH, Docker, monitoring, logs) belongs to the fleet repository [`terraform-proxmox`](https://github.com/thoutmose/terraform-proxmox); this runbook covers only the application ([ADR 0004](adr/0004-hosts-and-tailnet-only-access.md)).

## Environments

| Environment | Host | URL (tailnet only) |
|---|---|---|
| Staging | `srv-staging` | https://srv-staging.tail781d53.ts.net |
| Production | `srv-prod` | https://srv-prod.tail781d53.ts.net |

Each host runs the Compose stack in `/opt/axon`: Caddy on `127.0.0.1:3000`, API, web, PostgreSQL 16 + pgvector, Valkey, and the backup service. Tailscale Serve provides HTTPS in front of Caddy. Nothing listens publicly.

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

The script refuses to deploy a commit without a green `ci` run on GitHub, and refuses a production version that staging is not serving. After migrating it runs `axon integrity`, which fails the deployment if any entity lacks its domain row or has an unregistered type. It then builds the images, copies them over SSH, copies the configuration, runs pending migrations, starts the stack, enables Tailscale Serve, and fails unless `/api/readyz` answers within 30 seconds.

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
