# 0023 — A learning engine built on the evidence, local by default

- **Status:** Accepted
- **Date:** 2026-10-12
- **Epic:** EP-52

## Context

The owner asked for the learning, memorisation, and note-taking processes to be automated and to follow *Understanding How We Learn* (Weinstein, Sumeracki & Caviglioli: spaced practice, retrieval practice, interleaving, elaboration, concrete examples, dual coding) and *Moonwalking with Einstein* (Foer: vivid images, the memory palace, deliberate practice past the "OK plateau"). They also asked for AI-generated illustrated flashcards. Before EP-52, Axon had FSRS-scheduled cards, a practice test, and a Teacher agent; new cards all came due at once, decks were reviewed one after the other, and notes were never recalled as a whole.

## Decision

1. **The schedule does the work.** A day's review is known cards due plus a few new cards (20 a day by default), with decks interleaved across a review of everything. Retention is a setting (70–97%, 90% by default) passed to FSRS. A weekly mixed test is offered when none was taken in seven days.
2. **Free recall is scored locally.** A note's key ideas are its TextRank key phrases (EP-46); an idea counts as recalled when half its stemmed words, or thesaurus synonyms, appear in what the person wrote. Nothing is sent to an AI provider, so free recall works without one. The next recall follows expanding intervals (about ×2.5 when 80% or more is recalled, the same gap when half, tomorrow when less), not FSRS: a whole note is not a single fact, and the person sees the reason in plain words.
3. **Cards carry what makes them stick.** Elaboration, a concrete example, a mnemonic (vivid image), and a place in a memory palace are card fields. They travel to Anki in the answer field.
4. **AI drawings are SVG, sanitised by allowlist.** The Teacher may attach a drawing to a card it proposes or to an existing card (`illustrate_cards`), as every agent write, only after approval. The SVG is rebuilt from an allowlist of shapes, text, and gradients; scripts, links, external resources, styles, and animation are dropped. It is served with `nosniff` and a sandboxing CSP. Raster image generation was not chosen: it needs another provider and consent, and cannot be checked for content as an SVG can.
5. **Cards drafted from notes need no AI.** `question :: answer`, `Q:`/`A:`, `==highlight==` (cloze), and `**Term**: definition` become drafts the person adds; generating the card yourself is part of learning.
6. **Deliberate practice leaves the schedule alone.** Leeches (6 lapses or more) and the cards closest to being forgotten can be practised; those ratings are not recorded, so practice cannot distort FSRS.

## Consequences

The daily load is steady and visible (Learn → Today, with minutes from the person's own pace). Recall scoring is approximate: it rewards the note's words and their synonyms, not understanding; the page says so by showing exactly which ideas were found. Learning settings are read by package `cards` straight from the settings document, since package `settings` depends on the agents that depend on `cards`.
