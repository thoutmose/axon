<!-- Title: <type>(<area>): <imperative outcome> [EP-xx/IS-xx.xx] -->

## Purpose

What user or operator outcome this delivers, and which issue it closes.

## Evidence

Screenshots, API examples, or terminal output showing it works.

## Tests

Which tests were added or run, and their result.

## Migrations

None / describe, including how to roll back.

## Security and privacy impact

None / describe data touched, new external calls, consent changes.

## Rollback

How to undo this change in production.

## Checklist

- [ ] Stays inside one Epic; no unrelated changes.
- [ ] Simplest design; failure paths are explicit.
- [ ] Authorization, secrets, injection, and audit covered where relevant.
- [ ] Loading, empty, error, keyboard, and reduced-motion states covered where relevant.
- [ ] Queries bounded; retries idempotent.
- [ ] Docs, OpenAPI, and ADRs updated.
