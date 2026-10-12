# 0018 — An Anki bridge sidecar on the official Anki library

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-42
- **Builds on:** [0017](0017-flashcards-fsrs-and-anki.md)

## Context

Approved flashcards should reach the owner's `anki-sync-server` on `srv-services` automatically, by hand or after a Teacher proposal is approved. Anki's sync protocol is private to the Anki codebase: a client must hold a real collection, negotiate normal or full syncs, and upload media. Reimplementing it in Go would be fragile.

## Decision

- **A sidecar, not a reimplementation.** `deploy/anki` is a ~150-line Python service built on the official `anki` library (26.9.3, pinned). It is reachable only on the Compose network and exposes one endpoint:
  - `POST /sync` receives the person's cards as an `.apkg`, their sync address, user name, and password;
  - it keeps one collection per person in a volume;
  - it imports the package with notes updated **in place by GUID** (the Axon card id), so nothing is duplicated;
  - it never deletes anything, so Anki-only cards survive;
  - then it syncs.
- **Safe full syncs.** A new local collection always downloads the server's copy. An upload only happens when the server is empty. Any other full-sync request is refused with a message asking the person to sync from Anki desktop once. The person's Anki is never overwritten.
- **Credentials:**
  - each person connects their own sync account under Learn → Flashcards; the password is sealed with `AXON_ENCRYPTION_KEY`, never returned, and never exported;
  - the bridge receives it per request and does not log request bodies;
  - `ANKI_SYNC_URL` only suggests an address.
- **Triggers.** "Sync to Anki now", or automatically after the person approves the Teacher's `propose_cards` (a hook on proposal approval, run after the transaction). The outcome (time, ok or error, message) is shown on the page.
- **Tested end to end.** `scripts/check-anki-sync.sh`, also run in CI, starts a disposable `anki-sync-server` (from the same library) and the bridge, then checks three things: cards arrive, a second sync updates without duplicates, and an Anki-only note survives. API tests use a fake bridge.

## Consequences

- One more image to build, scan, and ship (`axon-anki`); deploy and CI handle it.
- The bridge's collections are derived data: if lost, they are rebuilt from the sync server, so they are not backed up.
- Scheduling stays separate: Axon schedules with its FSRS, and Anki with its own. Review history is not merged.
