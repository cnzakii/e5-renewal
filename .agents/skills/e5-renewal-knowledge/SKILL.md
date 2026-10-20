---
name: e5-renewal-knowledge
description: Retrieve and verify source-based facts for E5 Renewal about OAuth, Microsoft Graph, frontend tooling, databases, authentication, resource use, CI, and licenses. Maintain docs/knowledge only when requested; keep project decisions and code-specific audit findings outside it.
---

# E5 Renewal Knowledge

Use `docs/knowledge` as a factual reference layer, not a product plan. Read
[the repository guidance](../../../AGENTS.md) and
[the knowledge conventions](../../../docs/knowledge/README.md).

## Retrieve And Verify

1. Search topic metadata before loading full documents:

   ```console
   rg -n '^(title|description|topics|checked_at):' docs/knowledge --glob '*.md'
   ```

2. Read only matching topics and follow citations when wording, provenance,
   current behavior, or a security boundary matters.
3. Distinguish normative sources, authoritative registries, official guidance,
   pinned implementation observations, and methodological synthesis.
4. Verify live primary sources for mutable provider behavior, supported OAuth
   flows, Graph permissions, package metadata, framework behavior, or current
   security guidance. A working legacy flow is an observation, not proof of a
   provider guarantee or subscription renewal outcome.
5. Report unavailable, superseded, conflicting, or changed evidence. Separate
   source facts from recommendations and measured results.

## Maintain The Collection

Modify `docs/knowledge` only when knowledge maintenance is requested.

1. Update an existing owner document before creating another topic.
2. Keep facts independent of this codebase, audits, plans, release decisions,
   benchmark results, and temporary recommendations.
3. Include the required topic frontmatter and citations. Change `checked_at`
   only after reviewing every external factual claim in the document.
4. Pin implementation observations, replace stale claims, and update the
   collection index when its document set changes.
5. Verify metadata, relative links, provenance, and absence of personal paths
   or credentials. Do not create archives, generated indexes, reports, or
   monitoring machinery for ordinary research.
