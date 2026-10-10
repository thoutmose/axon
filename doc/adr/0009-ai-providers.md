# 0009 — AI providers behind one wire format

- **Status:** Accepted
- **Date:** 2026-10-10
- **Epic:** EP-22

## Context

People want to run agents on the provider they already use, including a model on their own hardware. EP-19 (ADR 0008) built the tool loop, approval gate, and budget around the Anthropic Messages API. Adding providers must not weaken any of those guarantees or fork the loop.

## Decision

- The runtime keeps speaking the Messages API. A provider is anything that answers one Messages request.
- **Anthropic** uses the official SDK, with a person's own key, the server's key, or a server-side OAuth profile (`ant auth login`, enabled with `AXON_ANTHROPIC_OAUTH=1`), so no key has to be stored.
- **OpenAI, Mistral, Gemini, DeepSeek, and local servers** (Ollama, LM Studio, llama.cpp) use one adapter for the OpenAI-compatible chat-completions format they share. It translates system prompts, tool definitions, tool calls, tool results, stop reasons, and token usage in both directions. Cloud providers use fixed addresses; only `local` accepts an address, and only `http(s)`.
- Settings are per person. Keys are sealed with `AXON_ENCRYPTION_KEY` (the TOTP sealer) and never returned, logged, or exported. Settings can only be changed from a browser session.
- Budgets use the person's prices when set. Otherwise known Anthropic models use list prices and unknown models the dearest known price, so a budget errs towards stopping early. A local model is free unless the person sets prices.
- Consumer chat subscriptions (Claude Pro/Max, ChatGPT Plus) are not supported: providers do not allow third-party apps to use them.

## Alternatives

- **One SDK per provider:** more code and five times the surface to keep in step with the approval gate.
- **A third-party multi-provider library:** another dependency between Axon and its safety guarantees.
- **Server-wide provider only:** simpler, but forces one choice on every person and their data.

## Consequences

A new OpenAI-compatible provider is one line in `providerKinds`. Provider-specific features (effort, refusal fallback) apply only to Anthropic. Wire-format quirks of a given provider show up as failed runs, which are recorded and visible. The `local` address lets a person make the API server send requests on their local network. That is the purpose of the setting, and only the account owner can set it.
