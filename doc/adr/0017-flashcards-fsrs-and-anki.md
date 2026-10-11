# 0017 — Flashcards with FSRS-5, a Teacher limited to processed notes, and Anki packages

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-16

## Context

The owner wants to view, add, edit, and remove illustrated flashcards on the website, practise and test with them, have a Teacher agent write them, and get them into the `anki-sync-server` on `srv-services`. The backlog asks that scheduling match the FSRS reference, that the Teacher never read fleeting notes, and that exported decks import into Anki.

## Decision

- **FSRS-5, ported and proven.**
  - `internal/fsrs` is a line-by-line port of py-fsrs 4.1.2: FSRS-5, 19 default weights, 1 and 10 minute learning steps, a 10 minute relearning step, 90% retention, fuzzing off.
  - `testdata/reference.json` holds the reference implementation's output for 40 cards × 12 reviews, including late and early reviews. The Go port matches all 480 exactly (stability and difficulty to 1e-9, same due times).
- **Cards are entities.** They link to their source with `derived_from`, so a note shows the cards made from it. Reviews are logged in `card_reviews`.
- **Pictures:** re-encoded to PNG, fitted to 1200 px (package `images`, shared with avatars).
- **Uploads** use their own route group with a 12 MB body limit. This also fixes avatar uploads, which the 64 KB JSON limit used to reject.
- **Teacher boundary:**
  - The Teacher's tools list and read only notes whose type is `literature` or `permanent`.
  - `propose_cards` refuses any other source.
  - The integration test checks that a fleeting note's text never reaches the model.
  - Cards are proposals, applied on approval (ADR 0008).
- **Anki packages.**
  - `GET /v1/flashcards/export.apkg` writes Anki's collection schema 11 (the format genanki produces, which every current Anki imports) with `modernc.org/sqlite`. This pure-Go driver was already in the module graph through goose; it is now a direct dependency.
  - Cards arrive in Anki as new cards; Axon keeps its own FSRS schedule.
  - A CI job imports a generated package with the official `anki` library (26.9.3) and checks the notes, decks, and media.
- **Sync to `anki-sync-server` is a separate Epic (EP-42).** Pushing into a sync server means acting as an Anki client (collection, media, conflicts, and credentials for `srv-services`), so it gets its own design and tests. The plan is a small sidecar built on the official `anki` library.

## Consequences

- The FSRS port is pinned to the reference version. A newer FSRS (more weights) means regenerating the fixtures and updating the port together.
- Practice tests are client-side and never change the schedule, so self-testing cannot distort FSRS.
- Until EP-42, getting cards into Anki means downloading the `.apkg` and importing it, or syncing it from Anki desktop.
