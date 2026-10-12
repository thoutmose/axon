# Support and deprecation policy

Axon is maintained by one person for themselves and their household, and shared as open source (AGPL-3.0-or-later). This policy says what you can rely on.

## Releases

- Versions follow [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`.
- The **latest minor release** receives fixes. Security fixes are also backported to the previous minor release for **90 days** after a new minor release.
- Upgrades between supported releases are tested with data ([upgrading.md](upgrading.md)). Skipping releases is supported: migrations run in order.

## API

- `/v1` is stable from 1.0. Within `/v1`, fields and endpoints are only **added**; nothing is removed or changes meaning.
- A field or endpoint that must go is first marked `deprecated: true` in the OpenAPI document and in the release notes, answers with a `Deprecation` header, and keeps working for **at least six months and one minor release** before removal in the next major version.
- Personal access tokens, the export format (`axon-export/1`), and the instance archive format (`axon-instance/1`) follow the same rule.

## Data

- Your data is yours: the account export (JSON and Markdown) and the instance archive always contain everything, in documented formats.
- Erasure deletes the account and everything in it at once; withdrawing a consent deletes the data only that consent allowed (health logs).

## Reporting problems

- **Bugs and questions:** open an issue on the repository with steps to reproduce. Never paste tokens, passwords, or personal data.
- **Security issues:** do not open a public issue. Use GitHub's private vulnerability reporting on the repository. You get an answer within 7 days, and a fix or mitigation for high-severity issues within 30 days, credited in the release notes if you wish.

## Dependencies and hosts

- Images are pinned by digest and rebuilt for security updates; CI fails on fixable high or critical vulnerabilities (Trivy) and on reachable Go vulnerabilities (govulncheck).
- The supported runtime is the Compose stack in this repository on Linux with Docker 24+.
