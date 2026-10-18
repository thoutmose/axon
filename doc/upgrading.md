# Upgrading Axon

Axon upgrades in place: pull the new release, rebuild, and apply migrations. Migrations only add or transform; they never drop data that is still in use. `axon integrity` checks the graph afterwards.

## Every upgrade

```bash
docker compose exec -T api axon instance export > axon-before-upgrade.tar.gz   # a restore point
git fetch --tags && git checkout vX.Y.Z
docker compose up -d --build --wait
docker compose exec api axon migrate up
docker compose exec api axon integrity
```

On the maintainer's servers, `deploy/deploy.sh staging` then `deploy/deploy.sh prod` do all of this, refuse a commit without green CI, and refuse a production version staging is not serving ([operations.md](operations.md#deploy)).

To go back: check out the previous release, start it on an empty database, and import the restore point ([install.md](install.md#7-back-up)). `deploy/deploy.sh prod --rollback` returns the maintainer's production to its previous images; it does not undo migrations.

## From v0.1.0 to 1.0

v0.1.0 (schema 5: accounts, the graph, the API platform, and tasks) is what production ran before 1.0. 1.0 adds migrations 6 to 71: notes, goals, habits, projects, roadmaps, agents, settings, focus, flashcards, Anki, progress, PARA, budget and finance, media, health, savings pots, and insights indexes up to 35 (the release candidate), then the journal, the vault server, the tech watch and the news, decisions, invitations, video games, models per agent, and how much agents' tool results are compacted. Nothing is removed.

What to expect:

- Accounts, passkeys, sessions, personal access tokens, tasks, links, and the event log carry over unchanged.
- Everything new starts empty, and every consent starts off (AI agents, catalogue lookups, health data).
- New optional settings: `AXON_CONTACT`, `TMDB_API_KEY`, `USDA_API_KEY`, `ANKI_SYNC_URL` (see [install.md](install.md#3-configure)), and since the release candidate sign-in with other accounts, email reminders, and speech and transcription servers, all listed under **Configuration** in the [architecture guide](architecture.md#configuration).

How it was tested:

- **Automated** (`TestUpgradeFromThePreviousReleaseKeepsData`, run in CI): a database filled at schema 5 (an account, a passkey, a session, a token, a done and a recurring task, a link, an event) is migrated to the latest schema. Every row is kept, the integrity check is clean, and the new API serves the old task to the old token.
- **With the real images** (2026-10-11): the v0.1.0 stack, built from `main`, created three tasks through its own API with a personal access token; the 1.0 images then started on the same volume, migrated (83 s including builds), passed `axon integrity`, and returned the three tasks to the same token, which also reached the new Insights endpoint.
