---
title: Microsoft OAuth and Graph request boundaries
description: Redirect registration, PKCE, refresh-token replacement, and throttling.
topics: [oauth, microsoft, graph, credentials, retries]
checked_at: 2026-10-16
---

# Microsoft OAuth and Graph request boundaries

- **Official guidance:** authorization-code exchange must use the registered
  redirect URI. Microsoft recommends PKCE, including for confidential clients.
  Request `offline_access` when a refresh token is required.
  [Authorization-code flow](https://learn.microsoft.com/en-us/entra/identity-platform/v2-oauth2-auth-code-flow)
- **Official guidance:** refresh responses can contain replacement refresh
  tokens; persist the replacement and discard the old token. Tokens may be
  revoked before their expected expiration, so clients need a reauthorization
  path. [Refresh tokens](https://learn.microsoft.com/en-us/entra/identity-platform/refresh-tokens)
- **Official guidance:** Microsoft Graph throttling can return HTTP 429 and
  `Retry-After`. Respect that delay; when absent, use a backoff strategy.
  [Graph throttling](https://learn.microsoft.com/en-us/graph/throttling)
- **Methodological synthesis:** requesting an endpoint is separate from
  proving that an application's permissions and tenant policy permit it.
  Request status is not evidence of a subscription-renewal guarantee.
