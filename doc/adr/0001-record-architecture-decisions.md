# 0001 — Record architecture decisions

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-00

## Context

Axon spans many domains and will be worked on by humans and AI agents. Decisions made in chat or in someone's head are lost, and agents cannot follow rules they cannot read.

## Decision

Record every significant architecture decision as an ADR in `doc/adr/NNNN-short-title.md`, using [0000-template.md](0000-template.md). Accepted ADRs are not rewritten; a new ADR supersedes them.

## Alternatives

- Wiki pages: drift from the code and are not reviewed in PRs.
- No records: cheapest now, most expensive later.

## Consequences

Decisions are reviewed with the code that implements them. Writing an ADR adds a small cost to each significant change.
