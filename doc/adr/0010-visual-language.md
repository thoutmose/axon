# 0010 — Visual language: glass over bundled scenery

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-23
- **Builds on:** [0007](0007-styling-without-tailwind.md)

## Context

The owner asked for an interface that feels as considered as a flagship product page, with imagery, depth, and blur. Axon is self-hosted and private, so pictures and fonts cannot come from third-party hosts. It must also stay readable, accessible, and fast on a phone.

## Decision

- **Scenery** is an inline SVG mountain landscape shipped with the app (`lib/ui/Scenery.svelte`), with dawn, day, dusk, and night palettes picked from the local hour. It sits behind full-bleed heroes on the home page and Today, under a dark gradient that keeps white text readable.
- **Glass:** surfaces are translucent (`--surface`) with `backdrop-filter: saturate(1.8) blur(24px)` over a fixed, soft colour field (`body::before`). Elements with the class `card`, `section`, or a `glass` class get the blur globally.
- **Neutral tokens** close to platform conventions (`#f5f5f7` / `#000` backgrounds, `#1d1d1f` text, one blue accent), pill buttons, larger radii, a display size for hero titles, the system font stack.
- **Motion** is a 6 px settle with no opacity change, so text is never shown below full contrast. It is off with reduced motion.
- **Reduced transparency** swaps every glass surface for a solid one.

## Alternatives

- **Photographs:** heavy, need licences, and either a third-party host or large files in the image.
- **A web font:** another request, and the system stack already reads well on every platform.

## Consequences

Pages get the new look through tokens without per-page rewrites. The e2e suite checks that the home page makes no cross-origin request and that axe finds no contrast problems. Backdrop blur costs some GPU time on old devices; reduced transparency turns it off.
