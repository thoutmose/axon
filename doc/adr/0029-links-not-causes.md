# 0029 — What helps you shows links, not causes, and is computed without AI

- **Status:** Accepted
- **Date:** 2026-10-13
- **Epic:** EP-92

## Context

With the daily journal (EP-91) next to the logs of sleep, training, focus, habits, and tasks, Axon can say which habits go with a person's better days. Two failures are easy: noise read as a pattern (six habits against three feelings is eighteen tests, and one in twenty passes by chance), and a pattern read as a cause ("sleep more and your mood will rise"). Both would turn a mirror into advice the data cannot support, on health data (GDPR art. 9).

## Decision

The analysis runs in Go on the server, never through a model:

- **Spearman rank correlation** for each habit and feeling, over the last 90 days: the scales are ordinal and the counts skewed, so ranks, not values.
- **A permutation test** (2,000 shuffles) for each pair's p-value: no assumption of normality on 14 to 90 points, and seeded by the pair, so the same data always gives the same answer.
- **The Benjamini–Hochberg correction** over every pair tested, and a link is shown only when q ≤ 0.1 and |ρ| ≥ 0.3, with at least 14 days and 5 on each side of the split.
- Each link is said in plain words, as two averages ("on days you slept 7 h or more, your mood averaged 3.6 out of 5; on the other days, 2.8"), never as a recommendation, with "a link is not a cause" beside it and the whole method one click away.

## Consequences

Few links are shown, and some real ones are missed; that is the price of not showing chance. Nothing leaves the deployment, nothing costs AI credit, and the result can be checked by hand. The thresholds are conventions, written in `journal/helps.go` and in the method text.
