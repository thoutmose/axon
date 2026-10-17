# Alert runbooks

Alerts are defined in the fleet repository ([`terraform-proxmox`](https://github.com/thoutmose/terraform-proxmox), `ansible/roles/monitoring/templates/alert_rules.yml.j2`, group `axon`) and delivered through ntfy. Each alert's `instance` label is the host: `srv-staging` or `srv-prod`. Commands below use `srv-prod`; swap the host for staging. Deploy, rollback, and restore steps are in [operations.md](operations.md).

Useful everywhere:

```bash
ssh srv-prod 'cd /opt/axon && docker compose ps'
ssh srv-prod 'cd /opt/axon && docker compose logs --since 30m api'
curl -s http://192.168.1.38:9464/metrics | grep ^axon_      # from the LAN or tailnet
```

The Grafana dashboard **Axon → Axon** shows health, version, traffic, errors, latency, the database pool, backup age, and request logs.

## AxonDown

The host is up, but the API's metrics port (9464) does not answer.

1. `docker compose ps`: is `api` running or restarting?
2. If it is restarting, read `docker compose logs api` for the startup error (missing `DATABASE_URL`, port in use, crash).
3. If the last deployment caused it, roll back: `deploy/deploy.sh prod --rollback`.
4. If the container is running but the port does not answer, check that `deploy/compose.server.yaml` still publishes `9464:9464`.

## AxonDatabaseUnreachable

The API cannot ping PostgreSQL, so `/api/readyz` returns 503 and every data request fails.

1. `docker compose ps db`: is it healthy?
2. `docker compose logs --since 30m db`: look for disk-full, out-of-memory, or corruption messages.
3. Disk: `ssh srv-prod df -h /`. A full disk also stops backups; free space, then `docker compose up -d db`.
4. If the data volume is damaged, restore the latest backup ([operations.md → Restore](operations.md#restore)).

## AxonHighErrorRate

More than 5% of requests returned 5xx for 10 minutes.

1. Find the failing route in the dashboard's *Requests by route* panel.
2. Read the matching request logs: `docker compose logs --since 30m api | grep '"status":5'`. Logs carry `request_id`, `route`, and `status` only, never request content.
3. If it started with a deployment, roll back. Otherwise check `AxonDatabaseUnreachable` and host resources.

## AxonSlowRequests

p95 latency (excluding health checks) is above 500 ms for 15 minutes.

1. Check the *Database pool* panel: in-use connections at the maximum means requests are queueing for the database.
2. Check host CPU and memory on the host's **Nodes** dashboard, and container usage on **Containers**.
3. Look for slow queries: `docker compose exec db psql -U axon -d axon -c "select pid, now()-query_start as age, left(query,80) from pg_stat_activity where state <> 'idle' order by age desc"`.

## AxonBackupStale

No successful nightly backup for more than 26 hours while the host was up.

1. `docker compose logs --since 30h backup`: look for `backup FAILED` and the error before it.
2. Run one by hand: `docker compose exec -T backup axon-backup now`.
3. Off-site copy errors (`rsync`/`ssh`): check that `srv-services` is up and that its `~/.ssh/authorized_keys` still holds the host's backup key; re-run `deploy/setup-offsite.sh prod` if not.
4. Local dumps are kept even when the off-site copy fails: `ls -l /opt/axon/backups`.

## AxonWorkerFailing

A background worker (`watch`, `reminders`, `vault-mirror`, `semantic-index`, `video-summaries`, `calendars`, `maintenance`) failed more than five runs in 30 minutes.

1. Read why: `docker compose logs --since 1h api | grep '"level":"ERROR"'` on the host; each worker logs its failure with its name ("tech watch step failed", "reminders step failed", …).
2. The **Axon** dashboard's *Background workers* panels show when it started and how long runs take.
3. A source, a mail server, or the AI provider being down is retried on the next run; a database error points to [AxonDatabaseUnreachable](#axondatabaseunreachable).

## AxonContainerMemoryHigh

One of Axon's containers has used more than 1.5 GB for 15 minutes.

1. The **Axon** dashboard's *Memory by container* says which one; *Time serving each feature* and *Background workers* say what it was doing.
2. `api`: look at *Go heap and GC*; a long tech watch fetch or a large import can raise it for a while. `db`: PostgreSQL uses its shared buffers and the page cache by design; act only if the host runs short (`node_exporter`'s memory alerts).
3. Restarting the container (`docker compose restart api`) frees it; if it climbs again, keep the dashboard's view of the hour before and open an issue.
