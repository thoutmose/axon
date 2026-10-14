#!/bin/sh
# Keeps a folder of Markdown files on this server in step with Axon (EP-69).
#
#   curl -fsSL {{.Base}}/api/v1/vault/install.sh | sh -s -- --code ABCD-EFGH --dir ~/SecondBrain
#
# It downloads the axon program to ~/.local/bin, pairs with the one-time
# code from Account → Preferences → Markdown vault (the token it receives is
# saved to ~/.config/axon/vault-token, readable by you only), syncs once, and
# installs a user service that keeps syncing. Nothing runs as root; nothing
# outside the folder and those two paths is written.
set -eu

base='{{.Base}}'
code=
dir=
while [ $# -gt 0 ]; do
  case "$1" in
    --code) code=${2:-}; shift 2 ;;
    --dir) dir=${2:-}; shift 2 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done
if [ -z "$code" ] || [ -z "$dir" ]; then
  echo "usage: sh install.sh --code ABCD-EFGH --dir ~/SecondBrain" >&2
  exit 2
fi
case "$dir" in
  "~") dir=$HOME ;;
  "~/"*) dir="$HOME/${dir#"~/"}" ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) arch=$(uname -m) ;;
esac
os=$(uname -s | tr '[:upper:]' '[:lower:]')

bin="$HOME/.local/bin/axon"
config="$HOME/.config/axon"
umask 077
mkdir -p "$HOME/.local/bin" "$config" "$dir"

echo "Downloading axon for $os/$arch from $base"
curl -fsSL -o "$bin.tmp" "$base/api/v1/vault/agent?os=$os&arch=$arch" || {
  echo "This Axon cannot hand out a program for $os/$arch: use the Docker command shown in Axon instead." >&2
  rm -f "$bin.tmp"
  exit 1
}
chmod 0755 "$bin.tmp"
mv "$bin.tmp" "$bin"

set -- notes-sync -dir "$dir" -url "$base" -token-file "$config/vault-token" -state "$config/vault-sync.json"
echo "Pairing and syncing $dir once"
"$bin" "$@" -pair "$code" -two-way

if command -v systemctl >/dev/null 2>&1 && systemctl --user show-environment >/dev/null 2>&1; then
  unit="$HOME/.config/systemd/user/axon-vault.service"
  mkdir -p "$(dirname "$unit")"
  cat >"$unit" <<UNIT
[Unit]
Description=Axon vault sync for $dir
After=network-online.target

[Service]
ExecStart=$bin notes-sync -dir "$dir" -url $base -token-file $config/vault-token -state $config/vault-sync.json -watch
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
UNIT
  systemctl --user daemon-reload
  systemctl --user enable --now axon-vault.service
  if ! loginctl show-user "$(id -un)" -p Linger 2>/dev/null | grep -q yes; then
    echo "To keep syncing after you log out, run once: sudo loginctl enable-linger $(id -un)"
  fi
  echo "Done: the vault syncs continuously (systemctl --user status axon-vault)."
else
  echo "Done. No systemd user session here: keep this running, for example in tmux:"
  echo "  $bin $* -watch"
fi
