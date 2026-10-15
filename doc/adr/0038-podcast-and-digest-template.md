# 0038 — The podcast is read by the device; digests follow a template

- **Status:** Accepted
- **Date:** 2026-10-14
- **Epic:** EP-113

## Context

The owner asked for a podcast of the tech watch, off by default, with a choice of voice, language, and format, and for the digest to be templated yet customisable and to carry the tech watch (2026-10-14).

## Decision

- **An episode is a script made in Axon, not by a model.** `GET /v1/watch/podcast` picks the period's news (the last day or week) by topic fit, then how many sources carried the story, then recency, until the target length is filled at 150 words a minute. Each story is "From <source>: <title>" and the sources' own summary: a sentence (briefing), a few (digest), or more from the article, grouped by topic with each topic announced (deep dive). Links, Markdown marks, and sentences that only point to a link are left out so a voice reads it well. An introduction gives the date, the number of stories, and the length; the closing says the episode was put together by Axon Project. The spoken links are English or French; the news keep their language. Nothing is generated, so nothing is invented, and nothing leaves Axon.
- **The device reads it** with the Web Speech API: the person's own voices for that language, their pace, one utterance per chapter so they can skip and see where they are. A browser that cannot speak offers the script as a text file. A server voice (an MP3 per episode, a private podcast feed) would need a speech service and a place to keep audio; it is left for later.
- **Off by default.** The settings key `podcast` (enabled, format, lang, minutes, days, voice, rate) starts disabled; the endpoint answers 409 until it is turned on.
- **Digest template.** The settings key `digest` lists each digest's sections in order (daily: tasks, habits, coming, learn, watch; weekly: open, week, reflect, watch; monthly: goals, month, watch) and whether the quote and the picture come along; all on by default. The digest builds every section, then keeps and orders them as listed and numbers the items again, so a reply's numbers still follow reading order. Every channel follows it: no quote line, and no picture in the email, Discord, Telegram, or the preview.
- **The Tech watch agent** (`tech-watch`) reads the person's topics (with their news of the last 30 days), sources (with their health), news (at most 50, summaries cut to 300 characters, wrapped as data), and the trends of their news graph. It proposes sources (https only, added as if imported, with the origin `agent`, fetched within a minute) and topics; nothing changes before approval, like every agent's change.
- **Scenery:** four spaces drawn by `scripts/spaces.py` like the others, light and dark: a radio dish (`signal`) for the reader and the sources, a star chart for the news graph, a studio for the podcast, and paper planes at sunrise for the digest. The existing art is unchanged.
- **The tech watch in the digests:** the five best stories of the period (a day, a week, or 30 days), linking to the reader. They carry no number, as a reply cannot act on news.

## Consequences

The podcast sounds like the device's voice, and better on systems with good voices. The digest and the podcast share the ranking, so the same stories lead both. A person who leaves a section out of the template no longer receives it anywhere; a digest with only news is still sent.
