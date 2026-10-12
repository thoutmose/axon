# 0016 — One settings document, editable in the app or as YAML, and a light i18n layer

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-31

## Context

The owner wants to set everything in one place — language, AI provider keys and limits, theme, fonts, icons, templates — either in the web app or by editing a YAML file. Keys must never leak, and the interface needs a French translation.

## Decision

- **One document, two views.**
  - `user_settings.doc` stores language, appearance, and templates. The AI provider and budget stay in their existing tables (ADR 0009).
  - `GET/PUT /v1/me/settings` serves JSON. `GET/PUT /v1/me/settings.yaml` serves the same document as a commented YAML file that includes the AI section.
  - Both use one validation, and a YAML upload is applied in one transaction.
- **Errors name the line.** YAML is parsed with `go.yaml.in/yaml/v4`, rejecting unknown keys. Validation errors carry a dotted path, like `appearance.theme`, which is mapped back to its line in the uploaded file.
- **Write-only secrets.** `ai.provider.api_key` is accepted on upload and sealed (ADR 0009). An export never contains it, and re-uploading an export keeps the stored key.
- **Session only.** The settings routes require a browser session, so a leaked access token cannot read or replace them. The demo account can change preferences but not AI providers.
- **Appearance on `<html>`.** Settings set `data-theme`, `data-font`, `data-size`, `data-icons`, and `data-motion`, plus `lang`. CSS tokens do the rest, so no component needs to know. They are cached in `localStorage` for the first paint, and the server copy wins once loaded.
- **i18n without a framework.** `t('English text')` looks up `i18n.fr.ts`, falling back to English, with `{name}` placeholders. English strings are the keys, so code stays readable and a missing translation shows English, never a blank. A unit test keeps placeholders intact. Pages are translated progressively.

## Consequences

- New UI strings should be wrapped in `t()`. Untranslated pages show English until they are covered.
- Adding a language means adding one dictionary file and an `enum` value.
- The YAML library is pinned at v4.0.0-rc.2, the version already in the module graph. It is to be upgraded to the stable release when one is published.
