# 0034 — Sign in with other accounts, linked by their owner

- **Status:** Accepted
- **Date:** 2026-10-14
- **Epic:** EP-109

## Context

The owner asked to allow other types of connection: Google, Microsoft, Discord, Slack, Apple (2026-10-14). Axon signs in with passkeys, recovery codes, and optional authenticator codes (EP-04). Self-hosters often run their own identity provider (Authentik, Keycloak, GitLab).

## Decision

- **Providers are the operator's choice.** Each is on when its client id and secret are set: `AXON_GOOGLE_*`, `AXON_MICROSOFT_*` (and `AXON_MICROSOFT_TENANT`), `AXON_DISCORD_*`, `AXON_SLACK_*`, Apple with `AXON_APPLE_CLIENT_ID`, `AXON_APPLE_TEAM_ID`, `AXON_APPLE_KEY_ID`, and the `.p8` key, and any OpenID Connect provider with `AXON_OIDC_ISSUER` (read from its discovery document at start). The redirect address is `AXON_BASE_URL/api/v1/auth/oauth/<provider>/callback`.
- **Authorization code flow with PKCE (S256)**, a state stored hashed and bound to the browser by a cookie (10 minutes, used once), and a nonce for OpenID Connect. The ID token comes straight from the provider's token endpoint over TLS, so the issuer is the server Axon called (OpenID Connect Core 3.1.3.7); Axon still checks `iss` (Microsoft: a tenant's issuer), `aud` and `azp`, `exp`, and the nonce. Discord, plain OAuth 2, is asked who signed in. Apple's client secret is a five-minute ES256 JWT signed with the operator's key; with no scopes, Apple answers with a plain redirect, so the session cookie stays `SameSite=Lax`.
- **Linking is explicit.** An identity is linked by its owner while signed in (Account → Sign-in & security), or created with a new account when registration is open. It is never matched by email, so an account at another service cannot take over an Axon account with the same address. One identity belongs to one account. The last way to sign in cannot be removed; passkeys stay the recommended way, and a new account made with a provider is asked to add one.
- Starts are rate limited on their own; sign-ins, links, and unlinks are in the audit log; linked accounts are in the personal export.

## Consequences

People can sign in with an account they already have. Someone who controls a linked account can sign in as its owner, as with any federated sign-in; owners choose to link, see when each was last used, and can unlink. The demo account cannot link.
