# 0019 — Progress computed from source data, with an opt-in switch

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-15

## Context

Gamification must connect every area (tasks, projects, habits, goals, notes, roadmaps, flashcards, focus, and later media), show exactly how each point was earned, be fully switchable off, sell freezes only for earned points, and never shame. A separate points table written by every feature would drift: undo a check-in, delete a task, and the points would stay.

## Decision

- **No stored points.** A single SQL ledger (`internal/progress`) unions the source tables:
  - completed tasks, with an on-time bonus;
  - habit check-ins;
  - focus sessions (1 point per 5 minutes);
  - flashcard reviews;
  - notes, with a permanent-note bonus;
  - learned skills, reached milestones, achieved goals, and finished projects.

  Totals, levels, today's points, the active days, and the ledger are all queries over it, so undoing an action removes its points, and the ledger always sums to the total (tested). New areas (media, EP-17) add one `UNION` branch.
- **Rules are data.** `GET /v1/progress/rules` lists every rule and its points. Nothing else gives points.
- **Levels** start at 50·L·(L−1) points: 100, 300, 600, 1,000, and so on.
- **One daily streak** comes from the ledger's active local days.
  - The state machine (inactive, active, at risk, frozen, broken) is a pure function on local dates.
  - Property tests run over 3,000 random histories: bounds; freezes used once and never before they were bought; acting today never lowers the streak; stable once freezes are saved. A time-zone test covers six zones around a DST change.
- **Freezes** cost 100 earned points, with at most 2 in stock; there is no money path. A freeze covers a missed day only when there is a run behind it. It is used automatically, and its use is saved.
- **Badges** are computed: the nth ledger entry of a rule, or the first day a run of 7, 30, or 100 days was completed.
- **Opt-in.** `settings.gamification.enabled` is false by default. When it is off, `/v1/progress` returns only `{"enabled": false}`, the ledger and freezes refuse, and no surface shows anything. There are no notifications at all. Messages never use loss framing ("Fresh start: one small action today begins a new run").

## Consequences

- Points for a habit check-in follow the check-in: unchecking removes them. This is by design.
- The ledger query grows with history; it is bounded by period for the page and indexed through each source table. If it ever becomes slow, a materialized daily rollup can be added without changing the API.
- Shared experiences (sharing a badge with another person on the same Axon) are deferred: production has a single account, and sharing needs a person-to-person model that does not exist yet.
