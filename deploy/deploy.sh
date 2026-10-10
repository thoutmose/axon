#!/usr/bin/env bash
# Deploy the current commit to a server, or switch a server back to its previous version.
#
#   deploy/deploy.sh staging|prod             build, copy, migrate, start, smoke-test
#   deploy/deploy.sh staging|prod --rollback  restart the previously deployed version
#
# Needs Docker, age, the private key in ~/.config/axon/age.key (or AXON_AGE_KEY),
# and SSH access to the host. See doc/operations.md.
set -euo pipefail

usage() { echo "usage: $0 staging|prod [--rollback]" >&2; exit 2; }

case "${1:-}" in
  prod) host=srv-prod ;;
  staging) host=srv-staging ;;
  *) usage ;;
esac
env=$1
remote_dir=/opt/axon
key=${AXON_AGE_KEY:-$HOME/.config/axon/age.key}
cd "$(dirname "$0")/.."

remote() { ssh "$host" "cd $remote_dir && $1"; }

smoke_test() {
  remote 'for i in $(seq 1 30); do curl -fsS http://127.0.0.1:3000/api/readyz >/dev/null && exit 0; sleep 1; done; exit 1' \
    || { echo "smoke test failed on $host" >&2; exit 1; }
  echo "$env ($host) is serving $(remote 'curl -fsS http://127.0.0.1:3000/api/healthz')"
}

if [[ "${2:-}" == "--rollback" ]]; then
  previous=$(remote 'cat previous-version')
  echo "rolling back $env to $previous (database migrations are not reverted)"
  remote "sed -i 's/^AXON_VERSION=.*/AXON_VERSION=$previous/' .env && docker compose up -d --wait api web"
  smoke_test
  exit 0
fi
[[ $# -eq 1 ]] || usage

if [[ -n "$(git status --porcelain)" ]]; then
  echo "commit your changes first: a deployment must match a commit" >&2
  exit 1
fi
version=$(git rev-parse --short HEAD)

echo "building $version"
AXON_VERSION=$version docker compose build api web
docker build -q --network host -t "axon-backup:$version" deploy/backup >/dev/null

echo "copying images to $host"
docker save "axon-api:$version" "axon-web:$version" "axon-backup:$version" | ssh "$host" docker load

echo "copying configuration"
ssh "$host" "sudo install -d -o \$(id -un) -g \$(id -gn) -m 0750 $remote_dir && mkdir -p -m 0700 $remote_dir/backups"
rsync -R compose.yaml deploy/compose.server.yaml deploy/Caddyfile deploy/env/recipients.txt "$host:$remote_dir/"
# Key for copying backups off-host; authorized on srv-services by deploy/setup-offsite.sh.
remote '[ -f backup_ed25519 ] || ssh-keygen -q -t ed25519 -N "" -C "axon-backup@$(hostname)" -f backup_ed25519'
{
  age --decrypt -i "$key" "deploy/env/$env.env.age"
  echo "COMPOSE_FILE=compose.yaml:deploy/compose.server.yaml"
  echo "AXON_VERSION=$version"
} | ssh "$host" "umask 077 && cat > $remote_dir/.env.next"

echo "migrating and starting"
remote '
  set -e
  if [ -f .env ]; then grep "^AXON_VERSION=" .env | cut -d= -f2 > previous-version; fi
  mv .env.next .env
  docker compose up -d --wait db valkey
  docker compose run --rm --no-deps api migrate up
  docker compose up -d --wait --remove-orphans
'

echo "serving on the tailnet"
ssh "$host" 'sudo tailscale serve status --json | grep -q "127.0.0.1:3000" || sudo tailscale serve --bg --https=443 http://127.0.0.1:3000 >/dev/null'

smoke_test
