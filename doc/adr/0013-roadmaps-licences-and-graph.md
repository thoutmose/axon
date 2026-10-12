# 0013 — Roadmap sources by licence, and a skill graph without a library

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-14
- **Settles:** the graph-library question deferred by [0002](0002-initial-technology-stack.md)

## Context

EP-14 asked for imports from roadmap.sh and OSSU, with licences checked before any import is built, and a skill graph that stays responsive with 500 nodes and has an accessible list alternative.

## Decision

- **OSSU Computer Science is imported.** It is MIT licensed (checked through GitHub's licence API and the `LICENSE` file). Axon bundles a snapshot of its curriculum (`api/internal/roadmaps/ossu.json`: 63 courses with links, effort, sections, and 24 prerequisite edges parsed from the README). Its attribution and licence link are shown before import and on the imported roadmap.
- **roadmap.sh is not imported.** Its licence allows personal use and sharing links, but forbids taking content out of the repository and sharing it through other media. Shipping it inside Axon would be redistribution. Axon links to roadmap.sh instead, and the person adds the skills they choose by hand.
- **Prerequisites are graph links** (`requires`, skill → skill), so skills and roadmaps also connect to goals, notes, and everything else (EP-24).
- **The graph uses no library.** A layered layout (each skill one column right of the furthest skill it needs, with cycles broken) is about 40 lines. It is linear in nodes plus edges, unit-tested at 500 nodes in under 50 ms, and rendered as SVG with links that are named for screen readers. The list view, grouped by section with a status control per skill, is the default and the accessible alternative.

## Alternatives

- **Cytoscape.js, D3-force, or ELK:** richer layouts, but a large dependency for a left-to-right prerequisite chart. A force layout also moves on every load, which hurts orientation.
- **Fetching roadmap.sh at runtime for "personal use":** still copies the content into another product, which the licence does not allow.

## Consequences

More curricula can be added as reviewed, openly licensed snapshots. The Tutor agent (cited, draft roadmaps) and the Resource agent moved to EP-27, because the backlog asked for this Epic to be split.
