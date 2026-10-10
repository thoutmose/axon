# 0004 — Hosts and tailnet-only access

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-01
- **Supersedes:** the staging placement in [0003](0003-single-host-first.md); the TLS role of Caddy in [0002](0002-initial-technology-stack.md)

## Context

The maintainer assigned hosts: production on `srv-prod`, staging on `srv-staging`, Anki sync on `srv-services`, and the Markdown notes on `srv-secondbrain`. All hosts are on the maintainer's Tailscale network (tailnet). Axon holds personal data and has a single user, so nothing requires it to be reachable from the public internet.

## Decision

- **Production** runs on `srv-prod`, **staging** on `srv-staging`. Each runs the full Compose stack, including its own PostgreSQL. A separate database host waits for a measured need, as in 0003.
- **No public ingress.** Axon is reachable only over the tailnet. Tailscale Serve terminates HTTPS with the host's `*.ts.net` certificate and forwards to Caddy on `127.0.0.1:3000`. Caddy only routes `/api` to the API and the rest to the web app.
- **Host firewall** (ufw) denies inbound traffic except from the tailnet interface and the home LAN, which existing monitoring uses.
- **Images** are built on the development host and copied to the target with `docker save | ssh … docker load` until EP-02 introduces a registry. No registry credentials live on the servers.

## Alternatives

- Public HTTPS through a reverse proxy with Let's Encrypt: needed only for access outside the tailnet, and adds an internet-facing attack surface.
- Staging on `srv-prod` (0003): one less host, but the maintainer already has `srv-staging`.

## Consequences

Every device that uses Axon must be on the tailnet. Passkeys (EP-04) bind to the `*.ts.net` hostname, so changing hostnames later means re-registering passkeys. A public deployment would need a new ADR.
