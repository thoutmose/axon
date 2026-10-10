# Axon — agent operating rules

Axon is a self-hosted life OS. Product and architecture: [README.md](README.md). Work items and delivery rules: [doc/BACKLOG.md](doc/BACKLOG.md). Decisions: [doc/adr/](doc/adr/).

## Current state

EP-00 (project foundation) in progress. No application code yet.

## Rules

- Work inside the active Epic only. Stop at the Epic boundary and wait for human review.
- Read the target files, tests, and docs before editing. State assumptions; ask when one could affect user data or security.
- Make the smallest coherent diff. Simple, explicit code; no speculative abstractions.
- Run focused checks first, then the relevant suite. Report failures as they are.
- Inspect your diff for scope creep, secrets, generated noise, and unsafe defaults.
- Never merge your own work, mutate production data, rotate credentials, or expand scope silently.
- AI-generated plans and mutations are drafts until a human approves them.
- If a requirement conflicts with privacy, safety, or consent, stop and ask.

Conventions (branches, commits, PRs, ADRs) are in [CONTRIBUTING.md](CONTRIBUTING.md).
