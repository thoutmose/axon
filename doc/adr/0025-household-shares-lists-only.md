# 0025 — A household shares lists, and nothing else

- **Status:** Accepted
- **Date:** 2026-10-12
- **Epic:** EP-61

## Context

Household sharing was among the features proposed in the EP-49 audit and approved by the owner. Everything in Axon is owned by one person (ADR 0006): every query is scoped to the owner, and the vault mirror, agents, and exports rely on it. Opening tasks, notes, or money to other people would cut across all of that, and would put one person's private data in front of another by default.

## Decision

A household is a separate, small domain: households, members, invite codes, lists, and items. Members share only the household's lists and their items (shopping, chores, anything), and see each other's display names. A person joins only with a one-time code a member made (stored as a hash, valid for a week), belongs to at most one household, and can leave at any time; the last one out deletes it, and an owner who leaves hands it to the longest-standing member. The owner can remove members; what a departing member was given to do is unassigned. Personal items are never shared or linked to household lists.

## Consequences

Sharing is explicit and reversible, and needs no change to personal data's ownership rules. A personal data export holds the household's name and the items the person added or was given, not other members' names. Shared projects, budgets, or calendars would each need their own decision.
