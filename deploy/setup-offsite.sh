#!/usr/bin/env bash
# One-time: let a server's backup key write (only write) into its own folder on srv-services.
#   deploy/setup-offsite.sh staging|prod
set -euo pipefail

case "${1:-}" in
  prod) host=srv-prod ;;
  staging) host=srv-staging ;;
  *) echo "usage: $0 staging|prod" >&2; exit 2 ;;
esac
dir="backups/axon-$1"

pub=$(ssh "$host" cat /opt/axon/backup_ed25519.pub)
entry="restrict,command=\"/usr/bin/rrsync -wo /home/thoutmose/$dir\" $pub"

ssh srv-services "mkdir -p -m 0700 ~/$dir && touch ~/.ssh/authorized_keys && { grep -qxF '$entry' ~/.ssh/authorized_keys || echo '$entry' >> ~/.ssh/authorized_keys; }"

# Servers can only write, so retention runs on srv-services: keep 30 days.
job="0 5 * * * find /home/thoutmose/$dir -name 'axon-*.dump.age' -mtime +30 -delete"
ssh srv-services "crontab -l 2>/dev/null | grep -qxF \"$job\" || (crontab -l 2>/dev/null; echo '# axon-$1 backup retention'; echo \"$job\") | crontab -"
echo "off-site backups for $1 go to srv-services:~/$dir"
