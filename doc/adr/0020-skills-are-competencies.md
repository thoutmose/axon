# 0020 — Skills are competencies; resources teach them

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-42

## Context

ADR 0013 and ADR 0014 imported OSSU and the curated library as one skill per course, book, or site. The skill graph, the skills network on Today, and the assessment placement then showed resource names ("Fluent Python, 2nd edition") instead of what a person can do ("Idiomatic Python"). The owner reported that the nodes were resources, not skills.

## Decision

- `library.json` and `ossu.json` name the skills of each roadmap (title, section, description, prerequisites), and every resource names the skill it teaches. A unit test checks that each skill has a resource and that prerequisites point to earlier skills.
- An imported skill stores its resources in `skills.resources` (JSON array: title, kind, by, free, url, note). The skill links to its first resource.
- Roadmaps imported before this change are not rewritten silently. `POST /v1/roadmaps/{id}/regroup` (a button on the roadmap) groups them: statuses are combined (done when every part is done, learning once any part is started), connections move to the skill that absorbs them, hand-added skills and their prerequisites stay, and running it twice changes nothing.

## Alternatives

- **A global skills table shared by roadmaps.** Better for deduplication across roadmaps, but a large migration and a new ownership model; not needed to fix what the graph shows.
- **Resources as entities linked to skills.** Lets resources carry status and connections, at the cost of hundreds of entities per import and a noisier graph. The JSON list is enough until resources need their own progress.
- **A migration that regroups every roadmap.** It would change people's data without asking; the explicit action keeps the person in control.

## Consequences

Skill graphs and the skills network show competencies, with fewer, more meaningful nodes. Resources are not editable through the API yet (they come from imports). Reverting means dropping `skills.resources` and re-importing; regrouped roadmaps keep their skills.
