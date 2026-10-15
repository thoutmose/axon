# 0036 — The news has its own knowledge graph, bridged to the notes

- **Status:** Accepted
- **Date:** 2026-10-14
- **Epic:** EP-111

## Context

The owner asked for a knowledge graph of the tech watch built with NLP (stems, stopwords, outliers, a thesaurus, some machine learning), where hovering a node shows a short summary closely tied to it, and that is either distinct from the Zettelkasten graph or integrated with a clear line between feed notes and Zettelkasten notes (2026-10-14). Feedly's "Leo" trends and FreshRSS's tag clouds were the reference: both rank topics by frequency, neither shows how topics relate or why.

## Decision

- **A separate graph.** `GET /v1/watch/graph` reads `watch_items` only; the Zettelkasten graph (EP-47) reads literature and permanent notes only. The news would outnumber a person's notes a hundred to one and turn their own thinking into noise. The two meet through **bridges**: a concept whose stems are also a key phrase of a literature or permanent note says so and lists the notes (a ringed star). The page is drawn as a night sky so it never reads as the notes' graph, and links to it.
- **Nodes are concepts**, not news: the TextRank key phrases of the period's news (EP-110), keyed by Snowball stems, single words merged with their thesaurus synonyms. A news item carries a concept when its stemmed text contains the phrase, not only when the phrase was among its own top key phrases. A concept needs two news items; a shorter one that only ever appears inside a longer one ("kernel" in "linux kernel") is dropped; the 120 most frequent are kept.
- **Edges are NPMI** (normalised pointwise mutual information) of two concepts appearing in the same news, at least twice, above 0.15 by default (adjustable), at most 400. NPMI, unlike raw counts, does not tie every concept to the most frequent one.
- **Communities** come from Louvain and **centrality** from PageRank, the same code as the Zettelkasten graph. A concept with no link above the threshold, and news set aside as outliers, form the **fringe**, hidden by default.
- **The hover summary is extractive**: of the sentences in the concept's own news that contain all its stems (6 to 45 words, no links), the two closest by cosine to the TF-IDF centroid of those news, the second not repeating the first (Jaccard of stems at most 0.5). Nothing is generated, so nothing is invented, and it runs without a model.
- **Trend** is the last week's count against the weekly average of the period before, as a z-like score (above 1.5 rising, below -1.5 fading); with no news older than a week there is nothing to compare and every trend is 0.
- Separate **story clusters** are left out: the same story from several sources is already stored once (EP-110), and communities group related stories.

## Consequences

The graph is computed on request from at most 2,000 news items (about 2 seconds at that cap, mostly stemming), with no table of its own, so it is always current and nothing extra is stored. A period with fewer than two news items per concept shows an empty state. Exports, the podcast, and the agent (EP-112, EP-113) can reuse the concepts.
