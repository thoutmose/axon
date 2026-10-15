# 0032 — Digests on chat channels, drawn and written by Axon

- **Status:** Accepted
- **Date:** 2026-10-14
- **Epic:** EP-107

## Context

The owner asked for a digest each morning, each week, and each month, by email, Discord, Slack, Teams, and the like: what to do that day, the next important or due tasks, projects, and milestones, what to learn and recall, aesthetic, illustrated, with a motivating quote fitting the day or the person (2026-10-14). Reminders (ADR 0024) already pushed a one-line morning digest and emailed it as plain text.

## Decision

- **One digest, many renderings.** `internal/digest` builds the content as data (sections of items, stats for the period that ended, a quote, a theme); renderers turn it into an HTML email (inline styles, a dark variant), plain text, Slack Block Kit, Discord embeds, a Teams Adaptive Card, Telegram HTML, and ntfy Markdown. Items a person can act on are numbered in reading order, so a reply can name them (EP-108). Push notifications keep their one-line form.
- **Kinds and times.** Daily at the morning time; weekly on a chosen day and time (default Sunday 18:00), with last week's numbers and the week ahead; monthly on a day from 1 to 28 (default the 1st at 09:00), with last month's numbers, goals with progress, and the month ahead. Each goes out once (`reminders_sent`), only when there is something to say, under the same master switch as reminders, in the person's language (English or French).
- **The picture is drawn by Axon.** A landscape PNG (sky, sun or moon, four ranges of hills, birds or stars) in the colours of the day's theme, seeded by the person and the date: no image service, no network, the same image in the email (inline `cid:`), Discord (attachment), and Telegram (photo). The theme also tints the email.
- **Quotes are curated, not generated.** About fifty short lines with sources we are confident of, each with Axon's French translation and tags: the day's theme (overdue tasks → persistence, Monday → a start, weekend → rest, a learning day → learning, weekly → reflection, monthly → progress) and the person's domains, guessed from words in their active goals, areas, and projects (craft, writing, money, health). The best matches are chosen between by a hash of the date. No AI is involved, so it works without a provider and cannot invent a quote.
- **Channels.** A person connects Slack, Discord, Teams, Telegram, ntfy, or a webhook (`notify_channels`), chooses which digests each receives and whether reminders go there too. Addresses are checked against each service's own hosts; ntfy and webhooks go through `netsafe` (no private networks unless the operator allows them), and webhooks can be signed (`X-Axon-Signature: sha256=HMAC`). Addresses and tokens are sealed with `AXON_ENCRYPTION_KEY`, never returned or exported, and kept out of error messages. A failing channel records its error and does not stop the others.

## Consequences

Digests reach people where they already are, and the content stays the same everywhere. Connecting a channel sends task titles and similar personal data to that service: the person chooses it, and the privacy register lists it. Teams receives digests but cannot answer (its replies need Axon to be reachable from Microsoft); Slack, Discord, and Telegram bots can (EP-108).
