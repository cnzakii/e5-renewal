---
title: Browser authentication and localized failures
description: Session cookies, CSRF protection, error identifiers, translation fallback, and rendering recovery.
topics: [authentication, sessions, csrf, errors, i18n, react]
checked_at: 2026-10-16
---

# Browser authentication and localized failures

- **Official guidance:** session identifiers should be unpredictable and renewed
  after authentication. `HttpOnly` limits script access to cookies; `Secure`
  restricts their transport to HTTPS. Idle and absolute expiration bound session
  lifetimes. [OWASP session management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- **Official guidance:** `SameSite` is a defense in depth. For cookie-authenticated
  state changes, CSRF tokens and origin checks provide additional request
  verification. [OWASP CSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- **Official guidance:** OWASP lists Argon2id with at least 19 MiB memory, two
  iterations, and parallelism one among its password-hashing configurations.
  [OWASP password storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- **Normative source:** RFC 9457 separates problem identification from human
  explanation. Clients should not parse the `detail` text for machine decisions.
  This does not require every API to adopt its complete response format.
  [RFC 9457, section 3.1](https://datatracker.ietf.org/doc/html/rfc9457#section-3.1)
- **Official guidance:** i18next supports ordered fallback keys for dynamic error
  codes. Sentence-as-key translation is possible, but its documentation warns
  that changing fallback text makes catalogs harder to maintain.
  [i18next fallback](https://www.i18next.com/principles/fallback)
- **Official guidance:** React error boundaries can show recovery UI when child
  rendering fails. They do not catch event-handler or ordinary asynchronous
  errors; those operations need their own error handling.
  [React error boundaries](https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary)
- **Official guidance:** translations change text length and line height.
  W3C recommends flexible layouts and room for reflow rather than tightly
  fitted labels or indiscriminate abbreviation.
  [Text size in translation](https://www.w3.org/International/articles/article-text-size)

**Methodological synthesis:** use stable identifiers for program decisions and
localized interface copy. Keep diagnostic messages separate, provide a generic
translated fallback for unknown failures, and offer recovery without silently
replaying a state-changing request.
