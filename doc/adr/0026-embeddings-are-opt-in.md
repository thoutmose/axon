# 0026 — Embeddings are each person's choice, behind their own consent

- **Status:** Accepted
- **Date:** 2026-10-12
- **Epic:** EP-62 (closes IS-37.04)

## Context

Finding notes by meaning needs embeddings: vectors from a model that place texts close together when they mean the same thing. Axon has no embedding model of its own, and Anthropic does not offer one. EP-37 left this open because it needs a provider and a consent of its own; EP-49 proposed it again and the owner approved it.

## Decision

Each person may choose an embedding service: a local model server with an OpenAI-style `/v1/embeddings` (Ollama, LM Studio, llama.cpp), OpenAI, or Voyage AI, with their own key, sealed with `AXON_ENCRYPTION_KEY`. Nothing is sent until they also give the `ai_embeddings` consent; withdrawing it deletes every stored vector, and turning the service off deletes the service too. Vectors live in PostgreSQL with pgvector (already in the image since EP-01), one per note, tagged with the service and model so a change of model never mixes spaces; they are compared by cosine distance, without an index (a person's notes are few enough). A background indexer embeds new and changed notes, a few hundred a minute, and records failures for the person to see. Item notes kept by the vault mirror are left out, as from search.

## Consequences

Search by meaning appears in the command center and as "Close in meaning" on each note, only for those who set it up; everyone else keeps the local TF-IDF similarity. A local model server keeps notes on the person's network; a cloud service receives note text, which the consent says. An HNSW index can be added when a person's notes reach the tens of thousands.
