# OWASP ASVS 4.0.3 Level 2 — authentication and sessions

Status of the chapters EP-04 touches. ✅ met · ➖ not applicable · ⏳ later Epic.

## V2 Authentication

| Requirement | Status | How |
|---|---|---|
| 2.1 Password security (2.1.1–2.1.12) | ➖ | No passwords; passkeys only |
| 2.2.1 Anti-automation against brute force | ✅ | Per-IP rate limit on every sign-in endpoint |
| 2.2.2 Weak authenticators (SMS/email) not used | ✅ | Passkeys, TOTP, single-use recovery codes only |
| 2.2.3 Notify users after authentication changes | ✅ | Audit log in Account → Recent activity (no email by design) |
| 2.2.4 Phishing resistance (L3) | ✅ | WebAuthn is origin-bound |
| 2.3.1 Initial secrets random and expiring | ✅ | Recovery codes 80-bit random; WebAuthn challenges expire after 5 minutes |
| 2.5.1–2.5.4 Credential recovery | ✅ | Recovery codes shown once, hashed, single use; optional TOTP alongside |
| 2.5.6 Recovery uses a secure mechanism | ✅ | Codes + optional TOTP, audited |
| 2.6 Look-up secrets (recovery codes) | ✅ | Single use, ≥ 80 bits, stored hashed |
| 2.7 Out-of-band verifiers | ➖ | None |
| 2.8.1–2.8.4 TOTP | ✅ | RFC 6238 via `pquerna/otp`, secrets encrypted at rest, confirmation before activation, each code accepted once |
| 2.9 Cryptographic verifiers | ✅ | WebAuthn (`go-webauthn`), sign-count clone detection |
| 2.10 Service authentication | ✅ | Personal access tokens: 256-bit, hashed, scoped read/write, expiring, revocable, never able to manage the account |

## V3 Session management

| Requirement | Status | How |
|---|---|---|
| 3.1.1 Tokens never in URLs | ✅ | Cookie only |
| 3.2.1 New token on authentication | ✅ | Every sign-in creates a new session |
| 3.2.2 ≥ 64 bits of entropy | ✅ | 256-bit random tokens |
| 3.2.3 Stored securely in the browser | ✅ | `HttpOnly`, `Secure` (HTTPS), `SameSite=Lax` cookies |
| 3.3.1 Logout invalidates the session | ✅ | Row deleted server-side |
| 3.3.2 Re-authentication / idle timeout | ✅ | 7 days idle, 30 days absolute |
| 3.3.3 Option to end all sessions after changing authenticators | ✅ | Sign out everywhere |
| 3.3.4 Users can view and log out active sessions | ✅ | Account → Signed-in devices |
| 3.4.1–3.4.5 Cookie attributes | ✅ | `Secure`, `HttpOnly`, `SameSite`, host-only, `Path=/` |
| 3.5 Token-based sessions | ✅ | Bearer tokens checked server-side on every request; revocation is immediate |
| 3.7.1 Re-authentication for sensitive actions | ⏳ | Account deletion asks for typed confirmation; step-up passkey check planned with EP-07 |

## V4 Access control (relevant parts)

| Requirement | Status | How |
|---|---|---|
| 4.1.1 Enforced on a trusted service layer | ✅ | API middleware and SQL `user_id` filters |
| 4.1.3 Least privilege / deny by default | ✅ | All `/v1/me` routes require a session |
| 4.2.1 Protection against IDOR | ✅ | Cross-user test harness |
| 4.2.2 CSRF protection | ✅ | Origin check + `SameSite` |
