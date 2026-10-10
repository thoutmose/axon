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

> [!CAUTION]
> Losing every private key listed in `recipients.txt` makes the env files and **all backups** unreadable. Keep a copy of `~/.config/axon/age.key` in a password manager. To add a key holder, append their public key to `recipients.txt` and re-encrypt both env files.

## Deploy

Deployments use the committed state of the current checkout. Always deploy to staging first.

```bash
deploy/deploy.sh staging
deploy/deploy.sh prod
```

The script refuses to deploy a commit without a green `ci` run on GitHub, and refuses a production version that staging is not serving. It then builds the images, copies them over SSH, copies the configuration, runs pending migrations, starts the stack, enables Tailscale Serve, and fails unless `/api/readyz` answers within 30 seconds.

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
