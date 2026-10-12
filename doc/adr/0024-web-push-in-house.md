# 0024 — Reminders by Web Push, implemented with the standard library

- **Status:** Accepted
- **Date:** 2026-10-12
- **Epic:** EP-56

## Context

Reminders were among the features proposed in the EP-49 audit and approved by the owner. A self-hosted instance has no app store and no notification service of its own; browsers deliver push messages only through their vendor's push service (Google, Mozilla, Apple, Microsoft). Email needs a mail server the operator already has.

## Decision

1. **Web Push, end-to-end encrypted.** Payloads are encrypted for each browser (RFC 8291, aes128gcm) and requests signed with an instance VAPID key (RFC 8292). The vendor's service relays bytes it cannot read. The implementation uses only the Go standard library (`crypto/ecdh`, `crypto/hkdf`, AES-GCM, ECDSA); a unit test checks it against the RFC 8291 example byte for byte, and the integration test decrypts as a browser would.
2. **The VAPID key is the instance's own.** It is made on first use, stored in `instance_keys`, sealed with `AXON_ENCRYPTION_KEY` when set. No configuration.
3. **Only browser push services.** A subscription's endpoint must be HTTPS on a known push service host, so a subscription cannot make Axon call another server.
4. **Two reminders a day, at local times, only with something to say.** A morning digest (tasks, cards with minutes, notes to recall, habits, the weekly test) and an evening nudge (habits still open, with their two-minute version; cards still due). Each is recorded in `reminders_sent`, so it goes out once; a reminder more than three hours late is skipped. Off by default.
5. **Email is optional.** With `AXON_SMTP_URL` and `AXON_SMTP_FROM`, the same text can also be emailed. No third-party email service.

## Consequences

Push works on desktop browsers and Android, and on iPhone and iPad once Axon is installed (EP-55). The person's browser vendor learns that a notification was sent to that browser, not its content. Dropped subscriptions (404/410) are forgotten. The shared demo never sends reminders.
