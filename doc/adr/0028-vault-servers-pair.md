# 0028 — A vault server pairs with a one-time code

- **Status:** Accepted
- **Date:** 2026-10-14
- **Epic:** EP-69

## Context

The owner asked to name, in Axon, the server that holds the Markdown vault; to set up the synchronisation with a new server automatically; to start an empty vault with its structure and templates; and to keep the database as the default home of every note (2026-10-14). Until now a vault server needed an access token made by hand, a binary copied out of an image, and a command assembled from the runbook.

## Decision

- Without a vault server, notes live in the database only. That is the default and needs nothing.
- Account → Preferences → Markdown vault names one server and folder per person (`vault_sources`). Saving makes a pairing code: 40 random bits shown as `ABCD-EFGH`, stored hashed, valid 30 minutes, used once, tried through the sign-in rate limit.
- `axon notes-sync -pair CODE` exchanges it at `POST /v1/vault/pair` (no session, no Origin: the code is the credential) for a write token valid a year, saved readable by its owner only. Pairing again replaces the token; removing the server revokes it.
- Axon hands out two ready commands: a one-line `curl … | sh` whose script downloads Axon's own program (when the server has the same system and architecture), pairs, syncs once, and installs a **user** systemd service; or `docker run` with the `axon-api` image. Neither needs root. The host and folder are checked so they cannot break out of the shell quoting.
- When the folder holds no note and the person kept the option on, pairing adds `Home.md` and the built-in templates to Axon, and the server makes the folders (inbox, PARA, Zettelkasten, daily, templates, attachments); the first sync writes the files. A folder with notes is left as it is. The templates have one source, the web app's, and a test on each side keeps the API's copy equal.
- The sync reports after each run and every five minutes (`POST /v1/vault/heartbeat`), so Axon can show in step, silent (nothing for 15 minutes), or failing, with the conflicts kept.

## Consequences

Setting up a vault server takes one command and no token handling. Axon serves its own executable and an install script publicly; both are the same for everyone and carry no secret. Fleet hosts can still run the same command from the fleet repository (ADR 0004). A server of another architecture uses the Docker command or builds the program.
