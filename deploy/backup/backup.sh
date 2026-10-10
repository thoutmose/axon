#!/bin/sh
# axon-backup now       dump, encrypt, prune, and copy off-host once
# axon-backup schedule  do that every day at BACKUP_AT (UTC, HH:MM)
#
# Env: PGHOST PGUSER PGPASSWORD PGDATABASE, BACKUP_OFFSITE (user@host, optional),
# BACKUP_KEEP_DAYS (default 14). Recipients: /etc/axon/recipients.txt.
# Metrics (optional): /textfile/axon_backup.prom.
# Off-site key: /run/secrets/backup_ed25519. Output: /backups.
set -eu

run_once() {
	stamp=$(date -u +%Y%m%dT%H%M%SZ)
	out="/backups/axon-$stamp.dump.age"
	pg_dump --format=custom | age --encrypt -R /etc/axon/recipients.txt > "$out.partial"
	mv "$out.partial" "$out"
	find /backups -name 'axon-*.dump.age' -mtime +"${BACKUP_KEEP_DAYS:-14}" -delete
	if [ -n "${BACKUP_OFFSITE:-}" ]; then
		# rsync splits -e on spaces only, so keep the ssh command on one line.
		ssh_cmd="ssh -i /run/secrets/backup_ed25519 -o IdentitiesOnly=yes -o BatchMode=yes"
		ssh_cmd="$ssh_cmd -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/backups/.known_hosts"
		rsync -e "$ssh_cmd" "$out" "$BACKUP_OFFSITE:"
	fi
	date -u +%FT%TZ > /backups/last-success
	# Picked up by the host's node_exporter textfile collector (fleet monitoring).
	if [ -w /textfile/axon_backup.prom ]; then
		printf 'axon_backup_last_success_timestamp_seconds %s\naxon_backup_last_size_bytes %s\n' \
			"$(date -u +%s)" "$(wc -c < "$out")" > /textfile/axon_backup.prom
	fi
	echo "backup ok: $out"
}

case "${1:-}" in
now)
	run_once
	;;
schedule)
	at=${BACKUP_AT:-03:15}
	while true; do
		now=$(date -u +%s)
		next=$(date -u -d "$(date -u +%Y-%m-%d) $at" +%s)
		[ "$next" -gt "$now" ] || next=$((next + 86400))
		echo "next backup at $(date -u -d "@$next" +%FT%TZ)"
		sleep $((next - now))
		run_once || echo "backup FAILED" >&2
	done
	;;
*)
	echo "usage: axon-backup now|schedule" >&2
	exit 2
	;;
esac
