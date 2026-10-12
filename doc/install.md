# Installing Axon

This guide installs Axon on one Linux server for one person or a household. It needs no knowledge of how the maintainer's own servers are set up ([operations.md](operations.md) covers those). Every command is meant to be copied as it is.

Time: about 15 minutes, most of it building images.

## 1. What you need

- A Linux server or VM with 2 vCPU, 4 GB of memory, and 20 GB of disk (Axon itself uses about 200 MB of memory; see [ADR 0021](adr/0021-single-host-measured.md) for measurements).
- Docker Engine 24 or newer with Compose v2 (`docker compose version` prints v2), Git, and OpenSSL.
- **HTTPS**, because passkeys only work in a secure context. Choose one:
  - [Tailscale](https://tailscale.com/) on the server (private, nothing exposed to the internet); or
  - a domain name pointing at the server, if Axon should be reachable publicly.
  - Plain `http://localhost:3000` works for a trial on your own computer.

## 2. Get the code

```bash
git clone https://github.com/thoutmose/axon.git
cd axon
git checkout v1.0.0
```

Use the latest release tag; `main` is the development line.

## 3. Configure

```bash
cp .env.example .env
sed -i "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$(openssl rand -hex 24)|" .env
sed -i "s|^AXON_ENCRYPTION_KEY=.*|AXON_ENCRYPTION_KEY=$(openssl rand -hex 32)|" .env
sed -i "s|^DATABASE_URL=.*|DATABASE_URL=postgres://axon:$(grep ^POSTGRES_PASSWORD= .env | cut -d= -f2)@db:5432/axon?sslmode=disable|" .env
```

Then edit `.env` and set the address people will use:

| Setting | Example | Why |
|---|---|---|
| `AXON_BASE_URL` | `https://axon.example.org` or `https://myserver.tail1234.ts.net` | Links, cookies, and passkeys are tied to it. |
| `WEBAUTHN_RP_ID` | `axon.example.org` | The host name of `AXON_BASE_URL`, without scheme or port. |
| `AXON_REGISTRATION` | `first-user` | Only the first person can create an account. Use `open` for a household, then switch back. |

> [!CAUTION]
> Keep a copy of `AXON_ENCRYPTION_KEY` in a password manager. It seals two-factor secrets, AI provider keys, and repository tokens. Without it, an instance archive or a database backup cannot open them.

Optional settings (each feature works without them; see the README's configuration table):

| Setting | Turns on |
|---|---|
| `AXON_CONTACT` | Your email address, sent to Open Library and Open Food Facts as they ask. |
| `TMDB_API_KEY` | Film and series lookups (your own free, non-commercial TMDB key). |
| `USDA_API_KEY` | USDA FoodData Central food search (a free data.gov key). |
| `ANKI_SYNC_URL` | The address of your own `anki-sync-server`. |
| `AXON_SMTP_URL`, `AXON_SMTP_FROM` | Reminder emails, through your own mail server (`smtp://user:pass@host:587` or `smtps://user:pass@host:465`). Push notifications to phones and browsers need no setting. |
| `AXON_DEMO=1` | A shared demo account with sample data, reset every night. Do not use it on an instance with real people. |

AI providers are not configured here: each person adds their own key in **Agents**, or none.

## 4. Start

```bash
docker compose up -d --build --wait
docker compose exec api axon migrate up
docker compose exec api axon integrity
curl -s http://127.0.0.1:3000/api/readyz
```

`integrity ok` and `{"status":"ready"}` mean Axon is running. It listens on `127.0.0.1:3000` only.

## 5. HTTPS

**With Tailscale** (private to your tailnet):

```bash
sudo tailscale serve --bg --https=443 http://127.0.0.1:3000
```

**With a domain**, install Caddy on the host and give it this `/etc/caddy/Caddyfile`, then `sudo systemctl reload caddy`. Caddy gets the certificate by itself.

```caddyfile
axon.example.org {
	reverse_proxy 127.0.0.1:3000
}
```

## 6. Create your account

Open `AXON_BASE_URL`, choose **Create your account**, and save a passkey. Keep the recovery codes shown once. Then, under **Account**:

- **Privacy**: everything is off until you turn it on (AI agents, catalogue lookups, health data).
- **Preferences**: language, theme, Pomodoro, templates, or the same as YAML.

## 7. Back up

Axon keeps everything in PostgreSQL, uploaded pictures included. A nightly instance archive is the simplest backup ([operations.md](operations.md#move-an-instance-ep-40)):

```bash
sudo mkdir -p /var/backups/axon && sudo chown "$USER" /var/backups/axon
crontab -l 2>/dev/null | { cat; echo '17 3 * * * cd '"$PWD"' && docker compose exec -T api axon instance export > /var/backups/axon/axon-$(date +\%F).tar.gz && find /var/backups/axon -name "axon-*.tar.gz" -mtime +14 -delete'; } | crontab -
```

Copy `/var/backups/axon` to another machine, encrypted (for example with `age` or `restic`). The archive holds personal and health data.

To restore onto a fresh installation (steps 2 to 4 with the **same** `AXON_ENCRYPTION_KEY`):

```bash
docker compose exec -T api axon instance import < axon-2026-10-11.tar.gz
docker compose exec api axon integrity
```

## 8. Upgrade

Read the release notes in [CHANGELOG.md](../CHANGELOG.md) first, then:

```bash
docker compose exec -T api axon instance export > axon-before-upgrade.tar.gz
git fetch --tags && git checkout v1.0.1      # the new release
docker compose up -d --build --wait
docker compose exec api axon migrate up
docker compose exec api axon integrity
```

Migrations only move forward and keep data ([upgrading.md](upgrading.md)). If something goes wrong, check out the previous tag, start it on an empty database (`docker compose down -v` deletes the data volume), and import `axon-before-upgrade.tar.gz`.

## 9. Troubleshooting

| Symptom | Fix |
|---|---|
| "This passkey isn't for this site" or registration fails | `WEBAUTHN_RP_ID` must equal the host name in `AXON_BASE_URL`, and the page must be served over HTTPS (or `localhost`). |
| `readyz` says the database is down | `docker compose logs db`; check `DATABASE_URL` uses the same password as `POSTGRES_PASSWORD`. |
| Image builds cannot download packages | Set `AXON_BUILD_NETWORK=host` in `.env` (Docker bridge without internet). |
| Lookups say "cannot be reached" | The server needs outbound HTTPS to the catalogue hosts; items can always be added by hand. |
| Agents say "not set up" | Add a provider under **Agents**, and turn on **AI agents** in **Privacy**. |

Logs: `docker compose logs -f api`. Health: `/api/healthz` (process) and `/api/readyz` (database). Support and security reports: [support.md](support.md).
