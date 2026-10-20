---
title: E5 Renewal engineering knowledge
description: Source-based facts for OAuth, Microsoft Graph, web interfaces, storage, security, testing, and delivery.
topics: [index]
---

# Engineering Knowledge

This collection supplies source facts and evidence maps. It does not decide
what E5 Renewal should implement. Keep project choices in the implementation and maintained project guidance,
not in this collection.

Add factual topic documents when requested and needed. Keep research scoped
to reusable engineering questions.

## Evidence Labels

- **Normative source:** an applicable standard, RFC, or BCP and its updates.
- **Authoritative registry:** names or values maintained by the responsible
  registry or provider.
- **Official guidance:** documentation from a service, tool, or language owner.
- **Observed practice:** behavior in a named implementation pinned to a commit
  or exact release; it is not a universal requirement.
- **Methodological synthesis:** a labelled conclusion drawn across sources.

## Topic Documents

- [Go services and verification](go-engineering.md)
- [Frontend builds and delivery](frontend-delivery.md)
- [Microsoft OAuth and Graph](microsoft-oauth-graph.md)
- [Browser authentication and localized failures](web-auth-localization.md)

Use `title`, `description`, `topics`, and `checked_at` frontmatter. Search this
metadata before opening full documents. Cite primary sources beside their
claims and pin implementation observations to an immutable revision.

Keep reusable facts about OAuth, Graph, frontend behavior, databases,
authentication, testing, resource measurement, and delivery here when needed.
Do not store project audits, implementation plans, benchmark results, release
decisions, personal instructions, or conversation transcripts here.

## Freshness And Maintenance

`checked_at` means every external factual claim in a topic document was
reviewed against its cited source on that date. It does not guarantee that a
mutable source is still current.

- Check live primary sources when the answer depends on current provider
  behavior, package metadata, tool behavior, or a security recommendation.
- Keep a pinned observation tied to its source revision. Do not present it as
  the current behavior of a later release.
- Report missing, conflicting, superseded, or materially changed evidence.
  Read-only research does not authorize editing this collection.
- Change `checked_at` only after reviewing the whole document. Replace stale
  claims instead of appending a contradictory account.
- Update this index when the topic document set changes. Keep citations and
  relative links usable without personal filesystem paths.
