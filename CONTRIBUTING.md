# Contributing to Axon

Contributions are welcome when they improve user agency, accessibility, privacy, or maintainability. Keep changes small.

## Workflow

1. Pick the next ready issue in [doc/BACKLOG.md](doc/BACKLOG.md). Work on one Epic at a time.
2. Branch from `main`: `feat/EP-09-task-planner`, `fix/EP-04-session-rotation`, `docs/EP-00-agent-constitution`.
3. Write tests or acceptance examples first, then the smallest vertical slice that passes them.
4. Keep migrations reversible and APIs documented in OpenAPI.
5. Record significant decisions as ADRs in [doc/adr/](doc/adr/).
6. Open a PR titled `<type>(<area>): <imperative outcome> [EP-xx/IS-xx.xx]`.

## Shared helpers

Use these instead of writing your own; each one fixed a bug that its copies had.

| Need | API (Go) | Web (TypeScript) |
|---|---|---|
| The person's time zone, and their "today" | `clock.Zone`, `clock.Today`, `clock.Day` | the API's `today`, or `localDate(new Date())` from `$lib/dates` |
| Date arithmetic on `YYYY-MM-DD` | `time.Time` at UTC midnight | `addDays`, `daysBetween`, `mondayOf`, `isoWeekday` from `$lib/dates` |
| Showing a `YYYY-MM-DD` date | — | `formatDate` from `$lib/dates`, never `new Date(day).toLocaleDateString()` (it shows the day before, west of UTC) |
| Storing text from outside (headers, files, pages) | `textx.Cut`, or `strings.ToValidUTF8`: PostgreSQL refuses invalid UTF-8 | — |
| Fetching an address a person gave | `netsafe.Client` (refuses private networks) | — |

The e2e suite runs the browser in a time zone whose date is not UTC's, so code that mixes the two fails there.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), for example `feat(tasks): add recurring task schedule`. Put the issue ID in the body.

## Pull requests

Each PR states its purpose, screenshots or API examples, tests run, migrations, security/privacy impact, and rollback plan. A reviewer who did not author the change must approve it, and a human approves every merge. The review checklist is in [doc/BACKLOG.md](doc/BACKLOG.md#pr-review-checklist).

## Never commit

Secrets, credentials, `.env` files, personal or health data, or another person's data — not in code, issues, prompts, screenshots, or logs.

## Security

Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md), not in a public issue. Everyone taking part follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

By contributing you agree that your contribution is licensed under [AGPL-3.0-or-later](LICENSE).
