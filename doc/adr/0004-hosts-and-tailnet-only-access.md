# 0004 — Hosts, existing fleet IaC, and tailnet-only access

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-01
- **Supersedes:** the staging placement in [0003](0003-single-host-first.md); the TLS role of Caddy in [0002](0002-initial-technology-stack.md)

## Context

The maintainer assigned hosts: production on `srv-prod`, staging on `srv-staging`, Anki sync on `srv-services`, and the Markdown notes on `srv-secondbrain`. These hosts already exist and are managed by the maintainer's fleet repository, [`terraform-proxmox`](https://github.com/thoutmose/terraform-proxmox): Terraform creates the VMs on Proxmox and Ansible configures them (base packages, SSH, Docker, `node_exporter`, Promtail → Loki, cAdvisor, Prometheus/Grafana on `srv-monitoring`, ntfy alerts). All hosts are on the maintainer's Tailscale network (tailnet).

## Decision

- **Host configuration stays in `terraform-proxmox`.** This repository does not provision or harden hosts; it only deploys the Axon application onto them. Host-level changes Axon needs are proposed there.
- **Production** runs on `srv-prod`, **staging** on `srv-staging`. Each runs the full Compose stack, including its own PostgreSQL. A separate database host (`srv-db`) waits for a measured need, as in 0003.
- **No public ingress.** Axon is reachable only over the tailnet: Tailscale Serve terminates HTTPS with the host's `*.ts.net` certificate and forwards to Caddy on `127.0.0.1:3000`. Caddy only routes `/api` to the API and the rest to the web app.
- **Observability reuses the fleet stack** on `srv-monitoring` (EP-03 adds Axon's own metrics and dashboards to it).
- **Images** are built on the development host and copied with `docker save | ssh … docker load` until EP-02 introduces a registry. No registry credentials live on the servers.

## Alternatives

- A separate Axon host playbook (tried in an early EP-01 draft): duplicates the fleet repository and conflicts with its SSH settings.
- Public HTTPS through Nginx Proxy Manager on `srv-npm`: needed only for access outside the tailnet, and adds an internet-facing attack surface.

## Consequences

Every device that uses Axon must be on the tailnet. Passkeys (EP-04) bind to the `*.ts.net` hostname, so changing hostnames later means re-registering them. EP-01 and EP-03 shrink to application-level work.
