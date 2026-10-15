# 0037 — Exports and share links for the tech watch

- **Status:** Accepted
- **Date:** 2026-10-14
- **Epic:** EP-112

## Context

The owner asked to export the news by type, source, label, and period, and to share them with other people, as PDF, ODS, CSV, Markdown and the like, each saying it comes from Axon Project (2026-10-14). Feedly shares boards by link and exports to a few integrations; FreshRSS exports OPML and starred items as JSON. Neither marks where an export comes from.

## Decision

- **One filter for the reader, exports, and links**: tab, kind (article, video from YouTube or Vimeo, podcast episode with an audio enclosure), source, topic, label, words, and period. What the reader shows is what is exported.
- **Formats, written in Go with no new dependency:** CSV (header first, the attribution as the last row, cells that start like a formula prefixed with `'` against CSV injection); an OpenDocument spreadsheet (a stored `mimetype` first, dates typed and shown as dates, links as links, the attribution under the title and in its metadata); Markdown grouped by day; a zip of feed notes, the same files **Keep as a note** writes, with that page as `README.md`; JSON; and PDF (A4, the standard Helvetica fonts, WinAnsi text, clickable titles, the attribution on every page). OPML of the sources stays as in EP-110. Each export says "Exported from Axon Project (https://github.com/thoutmose/axon)" with its date and the filter in words, and holds at most 5,000 news, newest first.
- **Share links**, like calendar feeds: a 256-bit random token in the link, only its SHA-256 stored, made and revoked from a signed-in session only (not an API token), audited (`watch_share.created`, `watch_share.revoked`), and expiring after 1 to 365 days (30 by default). A link stores the filter, not the news, so it shows what matches at the moment it is opened. Of the statuses, only Saved is kept: a link to Unread would empty as its owner reads.
- **A shared page shows what the sources published**: title, link, source, author, date, summary, kind, labels, reading time; never the owner's name, status, notes, or ids. It counts views, is not indexed (`X-Robots-Tag: noindex`, `Cache-Control: no-store`), sends no referrer to the sites it links to, and has its own rate limit (30 at once, then one every 2 seconds per address), apart from sign-in's.

## Consequences

Anyone who has a link can read and download that selection until it expires or is revoked; the owner sees each link's views and last view. News carry what the source published, so a shared summary is the publisher's text with its link. A share link to a source or topic later deleted simply shows less.
