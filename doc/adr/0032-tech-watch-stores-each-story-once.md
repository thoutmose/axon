# 0032 — The tech watch stores each story once, in its own tables

- **Status:** Accepted
- **Date:** 2026-10-14
- **Epic:** EP-73

## Context

The owner asked for a tech watch: domains, sources, and feeds defined by hand, by a normalised CSV, or by an agent; checked by a schedule that defaults to every midnight; new items stored in a relational database with no duplicates or near-duplicates; NLP labels; each news item a feed note with its own template; access to Medium, Substack, and the like; and easy monitoring (2026-10-14). Feedly, Inoreader, and FreshRSS were the reference: OPML import and export, folders as topics, conditional fetching, full text for excerpt-only feeds, mark-as-read flows, and per-feed health.

## Decision

- **Its own tables** (`watch_topics`, `watch_sources`, `watch_items`, `watch_seen`). A news item is an entity (`feed_item`), so it can be linked to tasks, projects, goals, and notes and found by search, but it is not a note: hundreds of items a day would bury the vault and the Zettelkasten. Keeping one (**Keep as a note**) writes a feed note to `Feeds/<year>/<month>/` with front matter `type: feed`, so it never joins the literature/permanent graph. Items made by the watch are recorded with the actor `watch`, which activity charts leave out.
- **Feeds Axon reads:** RSS 2.0, RSS 1.0, Atom, and JSON Feed, any encoding. **Discovery** rewrites known services without a request (Medium authors and publications, `*.medium.com`, Substack, YouTube channels and playlists, GitHub releases and users, subreddits, dev.to, Hacker News), then reads a page's `<link rel="alternate">`, then common paths. Every request goes through `netsafe` (no private networks unless the operator allows them), with Axon's user agent, a 10 MB cap, and `If-None-Match`/`If-Modified-Since`. With **full text** on, the article pages of excerpt-only entries are read with the clipper (EP-67), ten per fetch.
- **Schedules** are five-field cron expressions in the person's time zone, `0 0 * * *` by default, at least 15 minutes apart; a failing source is retried after 15 minutes, then 30, up to 8 hours, never later than its next run. The worker runs four fetches at a time and purges, after 180 days, news nobody saved, kept, or linked.
- **No duplicates.** An entry seen before from the same source is skipped (`watch_seen`). A link is canonicalised (scheme, `www.`, tracking parameters, fragment, query order, trailing slash) and hashed; a match adds the source to the stored item (`also_in`). Otherwise the text is compared with the last ten days: SimHash over stemmed words and word pairs within 3 bits for long texts, or a Jaccard overlap of at least 0.85 on titles or 0.8 on short texts. Only the first copy is stored; the source counts as merged. One ingest per person at a time (an advisory lock) keeps two sources from storing the same story concurrently.
- **Labels by local NLP:** language, Snowball stems, stopwords, TextRank key phrases, the feed's categories, and the person's topics. A topic applies when one of its keywords (all its words, each as itself or a WordNet/Axon thesaurus synonym, each through a different word) is in the title, or two distinct words of evidence are in the text. Outliers (too short, mostly links, shouting title) are set aside, not deleted. Changing a topic relabels the last month.
- **Imports:** the normalised CSV `url,title,topics,schedule,full_text`; OPML (folders become topics); FreshRSS through its Google Reader API, the API password used once and not stored. Imported sources are fetched within a minute and discovered then. OPML export keeps every reader one click away.
- **Monitoring:** per source, health (ok, failing, waiting, off), last error, next run, news this week, and stories merged; an overview of the last 30 days; Prometheus counters `axon_watch_fetches_total`, `axon_watch_items_total`, and the histogram `axon_watch_fetch_seconds`.

## Consequences

The vault stays the person's writing; the watch can grow to thousands of items without slowing notes. A paywalled article is stored as far as its page shows it. A source on the person's own network needs `AXON_CALENDAR_PRIVATE=1`, as for calendars. The analysis, the knowledge graph, exports, sharing, the podcast, and the agent build on these tables (EP-74 to EP-76).
