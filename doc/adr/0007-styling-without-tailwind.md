# 0007 — Styling with design tokens and scoped CSS

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-07
- **Supersedes:** the Tailwind choice in [0002](0002-initial-technology-stack.md) and the README tech-stack table

## Context

The README planned Tailwind. Svelte already scopes each component's CSS, and Axon's design direction (large type, quiet surfaces, restrained motion) needs a small, consistent set of values more than a large utility vocabulary.

## Decision

- **Design tokens** are CSS custom properties in `web/src/app.css`: type scale, spacing scale, radii, colours, elevation, and motion durations, each defined for light and dark themes.
- **Components** in `web/src/lib/ui/` use only tokens, in Svelte's scoped `<style>` blocks. Pages compose components and add little CSS of their own.
- **Themes:** the system preference by default; the user can force light or dark (stored in the browser).
- **Motion** uses `--motion-*` durations that drop to `0ms` under `prefers-reduced-motion`.

## Alternatives

- Tailwind: fast to write, but adds a build step and a second styling vocabulary, and markup gets long.
- A component library: faster start, but heavy, opinionated, and hard to keep calm.

## Consequences

No CSS framework dependency. A new component must use tokens; hard-coded colours and sizes are review findings.
