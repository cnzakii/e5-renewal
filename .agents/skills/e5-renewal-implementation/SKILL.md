---
name: e5-renewal-implementation
description: Implement requested E5 Renewal behavior, tests, configuration, packaging, CI, or repository documentation. Use for creating, changing, fixing, or refactoring repository files; exclude read-only research and audits.
---

# E5 Renewal Implementation

Deliver the smallest complete change that satisfies the request within
[AGENTS.md](../../../AGENTS.md). Implement the requested outcome without adding unrelated features.

## Workflow

1. Read the affected implementation, callers, tests, configuration, and public
   contracts. The legacy reference is available through the pinned commit in
   `AGENTS.md`; do not restore its tree as a starting application.
2. Establish the requested outcome and explicit non-goals. Keep adjacent
   features, extra adapters, compatibility layers, and new tooling out unless
   the current request needs them.
3. Identify ownership of credentials, token refresh, execution, scheduling,
   storage, HTTP requests, and UI state before changing that contract. A
   request lifetime must not accidentally own a persistent background task.
4. Use `e5-renewal-knowledge` for external requirements or tooling facts.
   Separate verified sources, legacy observations, and project choices.
5. Implement the smallest coherent change and update every affected caller,
   model, API response, UI interaction, configuration, example, and test.
   Keep public documentation and runnable examples consistent with behavior.
6. Add the smallest meaningful check for changed behavior. Use integration
   checks for database dialects and OAuth boundaries, browser checks for UI,
   build checks for distribution, and measurements for resource claims.
   Do not make live Graph calls using real credentials without authorization.
7. Run focused checks first, followed by the required aggregate gate when it
   exists. For a documentation-only foundation, validate skills, metadata,
   links, notices, and the Git changes instead of inventing an application
   test suite.
8. Review the final artifact with `e5-renewal-audit`. Fix blocking findings
   through implementation and rerun the checks affected by those fixes.

## Boundaries

- Keep the rewrite independent of the legacy feature list and data schema.
- Use standard facilities and mature dependencies before local machinery.
- Preserve input validation, error handling, credential redaction, and
  management authentication when code is moved or simplified.
- Do not claim database interchangeability, OAuth compatibility, UI
  accessibility, or resource suitability without matching evidence.
- Keep application credentials distinct from administrator login credentials.
  Respect the agreed storage threat model without weakening Web access checks.
- Retain the applicable notices for third-party code or assets actually used.
- Report verification limits clearly. Do not bypass failing checks or a
  genuine external blocker to claim completion.
