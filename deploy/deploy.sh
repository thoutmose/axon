#!/usr/bin/env bash
# Deploy the current commit to a server, or switch a server back to its previous version.
#
#   deploy/deploy.sh staging|prod             build, copy, migrate, start, smoke-test
#   deploy/deploy.sh staging|prod --rollback  restart the previously deployed version
#   deploy/deploy.sh dev [--skip-ci|--rollback]
#                     the same on srv-dev (or AXON_DEV_HOST), for a branch under
#                     review; --skip-ci deploys without a green CI run, on dev only
#
# Needs Docker, age, the private key in ~/.config/axon/age.key (or AXON_AGE_KEY),
# and SSH access to the host. See doc/operations.md.
set -euo pipefail

usage() { echo "usage: $0 staging|prod [--rollback] | dev [--skip-ci|--rollback]" >&2; exit 2; }

case "${1:-}" in
  prod) host=srv-prod ;;
  staging) host=srv-staging ;;
  dev) host=${AXON_DEV_HOST:-srv-dev} ;;
  *) usage ;;
esac
env=$1
flag=${2:-}
[[ $# -le 2 ]] || usage
case "$flag" in
  '' | --rollback) ;;
  --skip-ci) [[ "$env" == dev ]] || { echo "--skip-ci is for dev only: staging and production need a green CI run" >&2; exit 2; } ;;
  *) usage ;;
esac
remote_dir=/opt/axon
key=${AXON_AGE_KEY:-$HOME/.config/axon/age.key}
cd "$(dirname "$0")/.."

# dev has its own secrets once deploy/env/dev.env.age exists, staging's until then.
secrets=deploy/env/$env.env.age
if [[ "$env" == dev && ! -f "$secrets" ]]; then
  secrets=deploy/env/staging.env.age
  echo "dev: no deploy/env/dev.env.age yet, using staging's secrets"
fi

remote() { ssh "$host" "cd $remote_dir && $1"; }

smoke_test() {
  remote 'for i in $(seq 1 30); do curl -fsS http://127.0.0.1:3000/api/readyz >/dev/null && exit 0; sleep 1; done; exit 1' \
    || { echo "smoke test failed on $host" >&2; exit 1; }
  echo "$env ($host) is serving $(remote 'curl -fsS http://127.0.0.1:3000/api/healthz')"
}

if [[ "$flag" == "--rollback" ]]; then
  previous=$(remote 'cat previous-version')
  echo "rolling back $env to $previous (database migrations are not reverted)"
  remote "sed -i 's/^AXON_VERSION=.*/AXON_VERSION=$previous/' .env && docker compose up -d --wait api web"
  smoke_test
  exit 0
fi
if [[ -n "$(git status --porcelain)" ]]; then
  echo "commit your changes first: a deployment must match a commit" >&2
  exit 1
fi
version=$(git rev-parse --short HEAD)

# Promotion rules: CI must have passed for this commit, and production only
# receives a version that staging is already serving.
if [[ "$flag" == "--skip-ci" ]]; then
  echo "dev: deploying $version without checking CI"
else
  # dev's pull request run is skipped (it repeats the push run): ignore it.
  ci=$(gh run list --commit "$(git rev-parse HEAD)" --workflow ci --json status,conclusion \
    --jq '[.[] | select(.status == "completed" and .conclusion != "skipped")][0].conclusion // "missing"')
  if [[ "$ci" != "success" ]]; then
    echo "CI for $version is '$ci': push the commit and wait for a green CI run" >&2
    exit 1
  fi
fi
if [[ "$env" == prod ]]; then
  staging=$(ssh srv-staging 'curl -fsS http://127.0.0.1:3000/api/healthz' | sed -n 's/.*"version":"\([^"]*\)".*/\1/p')
  if [[ "$staging" != "$version" ]]; then
    echo "deploy $version to staging first (staging serves '${staging:-nothing}')" >&2
    exit 1
  fi
fi

echo "building $version"
AXON_VERSION=$version docker compose build api web anki caddy db
build_net=$(sed -n 's/^AXON_BUILD_NETWORK=//p' .env 2>/dev/null)
docker build -q --network "${build_net:-default}" -t "axon-backup:$version" deploy/backup >/dev/null

echo "copying images to $host"
docker save "axon-api:$version" "axon-web:$version" "axon-anki:$version" "axon-backup:$version" "axon-caddy:$version" "axon-db:$version" | ssh "$host" docker load

echo "copying configuration"
ssh "$host" "sudo install -d -o \$(id -un) -g \$(id -gn) -m 0750 $remote_dir && mkdir -p -m 0700 $remote_dir/backups"
rsync -R compose.yaml deploy/compose.server.yaml deploy/Caddyfile deploy/env/recipients.txt "$host:$remote_dir/"
# Key for copying backups off-host; authorized on srv-services by deploy/setup-offsite.sh.
remote '[ -f backup_ed25519 ] || ssh-keygen -q -t ed25519 -N "" -C "axon-backup@$(hostname)" -f backup_ed25519'
# Backup metrics file for node_exporter's textfile collector (directory owned by root).
ssh "$host" 'f=/var/lib/prometheus/node-exporter/axon_backup.prom; [ -f $f ] || sudo install -o "$(id -un)" -m 0644 /dev/null $f'
{
  # Borrowed secrets carry their server's own address: srv-dev declares its own
  # in dev.settings.env, or its sign-in and every write would be refused there.
  if [[ "$secrets" != "deploy/env/$env.env.age" ]]; then
    age --decrypt -i "$key" "$secrets" | grep -v '^\(AXON_BASE_URL\|WEBAUTHN_RP_ID\)='
  else
    age --decrypt -i "$key" "$secrets"
  fi
  # Non-secret settings, reviewed in the open (they must not repeat a secret's name).
  grep -v '^\s*\(#\|$\)' "deploy/env/$env.settings.env"
  echo "COMPOSE_FILE=compose.yaml:deploy/compose.server.yaml"
  echo "AXON_VERSION=$version"
} | ssh "$host" "umask 077 && cat > $remote_dir/.env.next"

echo "migrating and starting"
remote '
  set -e
  if [ -f .env ]; then grep "^AXON_VERSION=" .env | cut -d= -f2 > previous-version; fi
  mv .env.next .env
  docker compose up -d --wait db
  docker compose run --rm --no-deps api migrate up
  docker compose run --rm --no-deps api integrity
  docker compose up -d --wait --remove-orphans
  # rsync replaces the Caddyfile with a new file, and a single-file bind mount
  # keeps the old one: recreate Caddy so header and route changes apply.
  docker compose up -d --wait --no-deps --force-recreate caddy
  # Keep only the running and previous versions for rollback.
  keep="$(grep "^AXON_VERSION=" .env | cut -d= -f2)|$(cat previous-version 2>/dev/null || echo none)"
  docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "^axon-(api|web|anki|backup|caddy|db):" \
    | grep -vE ":($keep)\$" | xargs -r docker rmi >/dev/null
'
# Locally, only the version just deployed is worth keeping.
docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "^axon-(api|web|anki|backup|caddy|db):" \
  | grep -vE ":($version|dev)\$" | xargs -r docker rmi >/dev/null || true

echo "serving on the tailnet"
ssh "$host" 'sudo tailscale serve status --json | grep -q "127.0.0.1:3000" || sudo tailscale serve --bg --https=443 http://127.0.0.1:3000 >/dev/null'

smoke_test
