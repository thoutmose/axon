# 0011 — Notes workspace without an editor framework

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-25

## Context

The owner wants note taking on par with dedicated apps (Alexandrie, Obsidian), built for Zettelkasten, PARA, and a Second Brain. Notes must stay plain Markdown, so export, import, and two-way vault sync round-trip byte for byte (EP-11). The renderer must never let a note inject markup.

## Decision

- **Editor:** a native `textarea` with a formatting toolbar, shortcuts (Ctrl+B/I/E/L, Ctrl+Shift+X/H), list continuation, and `[[` suggestions from an in-memory note index, next to a live preview. Text autosaves 1.2 s after typing stops. On a version conflict, nothing is overwritten and the person chooses to keep their version or take the saved one. Moving a file is a separate, explicit action.
- **Renderer:** Axon's own escape-first renderer gains tables, callouts (`> [!NOTE]`), `==highlight==`, `~~strike~~`, `#tags`, and heading anchors. Every construct is unit-tested against injection.
- **Organisation:** a sidebar tree built from paths, grouped by PARA folder. Templates write front matter for fleeting, literature, permanent (with a `YYYYMMDDHHmm` Zettelkasten id), project, area, resource, and meeting notes, plus a daily note at `Daily/YYYY-MM-DD.md`.
- **Tags** are parsed in Go from `tags:` front matter and body `#tags` outside code, stored in `notes.tags` (GIN index), and backfilled by `axon migrate up`.
- **Search** returns highlighted snippets (`ts_headline`, with U+0002/U+0003 markers that clients escape around). A global Ctrl/Cmd+K command center searches everything and runs commands.

## Alternatives

- **CodeMirror or ProseMirror:** richer editing, but a large dependency. A WYSIWYG model risks rewriting Markdown, which breaks byte-for-byte sync.
- **A Markdown library (markdown-it, marked) with a sanitizer:** broader syntax, but it moves the injection boundary into third-party code and a sanitizer configuration.

## Consequences

The editor stays small and accessible, and notes are never rewritten behind the person's back. LaTeX, diagrams, and embedded media are not rendered yet; they can be added to the renderer one tested construct at a time.
