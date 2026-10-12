# 0002 — Initial technology stack

- **Status:** Accepted (2026-10-10, maintainer delegated the decision)
- **Date:** 2026-10-10
- **Epic:** EP-00

## Context

The README names a stack but leaves several either/or choices open. EP-00's walking skeleton and EP-01–EP-04 cannot start until they are settled. Axon is a self-hosted product for one person or a household, so operational simplicity matters more than horizontal scale.

## Decision

Confirm the README's core stack:

| Layer | Choice |
|---|---|
| Backend | Go modular monolith, `chi` router |
| API | REST, OpenAPI 3.1 written first; no GraphQL until a measured need |
| Frontend | SvelteKit + TypeScript + Tailwind |
| Data | PostgreSQL 16 + pgvector (system of record) |
| Ephemeral | ~~Valkey for cache and job queue~~ — removed, see [ADR 0022](0022-no-valkey-yet.md); jobs stay idempotent in Postgres |

Resolve the open choices:

| Open choice | Proposal | Reason |
|---|---|---|
| Authentik vs built-in auth | Built-in WebAuthn + TOTP using maintained libraries | Authentik adds its own Postgres, Redis, and ~1 GB of RAM for a single-tenant app. We own the session code, so EP-04 must include an external security review. |
| Caddy vs Traefik | Caddy | Automatic TLS with a few lines of config; no container-label routing needed on one host. |
| GitOps vs Compose + Ansible | Docker Compose + Ansible | Fits a single host (see [0003](0003-single-host-first.md)); GitOps controllers assume Kubernetes. |
| D3 vs Sigma.js | Defer to EP-14 | Nothing needs a graph view before then. |

Decided for the walking skeleton (IS-00.06):

| Concern | Choice | Reason |
|---|---|---|
| Code forge and CI | GitHub (private repository) + GitHub Actions | The maintainer's existing repositories and CLI login are there. |
| Database driver | `pgx` v5 | The standard maintained PostgreSQL driver for Go. |
| Migrations | `goose` v3, SQL files embedded in the binary | Plain SQL, up and down in one file, runs as `axon migrate`. |
| Query layer | Hand-written SQL through `pgx` until EP-05; revisit `sqlc` then | No queries exist yet to justify a code generator. |
| Go checks | `gofmt`, `go vet`, `go test`; `golangci-lint` added in EP-02 | Built-in tools first. |
| Web checks | `svelte-check`, Prettier | The SvelteKit defaults. |
| Web runtime | SvelteKit 2 on Node 24 LTS, `adapter-node`; Vite 7, Vitest 4 | SvelteKit 3.0 was nine days old at decision time; upgrade once its ecosystem settles. npm 10 (Node 22) fails to resolve this dependency tree, npm 11 (Node 24) does. |
| Styling | Plain scoped CSS in the skeleton; Tailwind arrives with the design system (EP-07) | Nothing to style yet beyond one page. |

## Alternatives

- Authentik/Keycloak: mature, but heavy for one user; revisit if multi-tenant or SSO becomes a goal.
- Kubernetes + Argo CD/Flux: real GitOps, but large operating cost for one machine.

## Consequences

Fewer moving parts to run and back up. Owning authentication raises security risk, mitigated by vetted libraries, OWASP ASVS checks, and independent review. Any choice here can be superseded by a later ADR.
