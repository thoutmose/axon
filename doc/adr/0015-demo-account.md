# 0015 — A shared demo account, walled off from credentials and the outside

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-28

## Context

The owner wants a demo account to try the website on staging and production without creating a passkey, and without touching real data. Axon signs in with passkeys only, and only the first account may register.

## Decision

- **One demo user**, marked by `users.demo` (a unique partial index allows at most one). It does not count for the "first account registers" rule, so a fresh install with the demo still lets the owner register.
- **Sign-in without a passkey:**
  - `POST /v1/auth/demo` starts a 12-hour session, only when the server sets `AXON_DEMO=1`. Otherwise it returns 404.
  - It is rate-limited like other sign-ins, and the landing page shows "Try the demo".
  - The demo is reachable only by people who can reach the server: the tailnet (ADR 0004).
- **Walled off.** `auth.NotDemo` refuses, with 403:
  - personal access tokens, passkeys, recovery codes, and TOTP;
  - AI provider keys and tests, and repository connections;
  - account erasure.
- **Agents off.** Agents never run for the demo (reason `demo`), so it spends no one's AI credit. Seeded proposals let visitors try the approval queue.
- **Ownership checks unchanged.** The demo is an ordinary user to every data query, so the existing authorization model isolates it from real accounts, and the integration test checks both directions.
- **Reset.** `axon demo reset` deletes the demo user (everything cascades from `users`) and recreates it in one transaction with sample data in every area. The server creates it once migrations have run, then resets it nightly after 03:00 UTC. A reset signs visitors out.
- **Non-secret server settings** go in `deploy/env/{staging,prod}.settings.env`, committed and reviewed in the open. `deploy.sh` appends them to the decrypted secrets, so switching the demo on or off does not need the secrets key.

## Consequences

- Anyone on the tailnet can read and change the demo data until the next reset; the banner says so.
- The demo sees exports of its own data only, which contain nothing real.
- If a future endpoint adds credentials or outside reach, it must use `auth.NotDemo`; the demo integration test lists the refused routes.
