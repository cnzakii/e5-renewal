---
name: e5-renewal-audit
description: Review E5 Renewal code, changes, APIs, architecture, UI, tests, storage, authentication, packaging, CI, or documentation for evidence-backed defects and unnecessary complexity. Use after implementation or for a requested read-only review; never modify files.
---

# E5 Renewal Audit

Review the current artifact independently and remain read-only. Read
[AGENTS.md](../../../AGENTS.md) before reviewing.

## Workflow

1. Read the requested scope, changed files, affected callers, relevant tests,
   and public contracts. Inspect current facts rather than trusting a previous
   report or the implementation author's conclusion.
2. Select relevant lenses: OAuth and credential lifecycle, Graph behavior,
   task ownership, database correctness, management authentication, UI
   usability, resource use, packaging, CI, documentation, or maintenance cost.
3. Use `e5-renewal-knowledge` when a finding depends on an external fact.
   Distinguish standards, official guidance, pinned observations, project
   contracts, and engineering judgment.
4. Challenge new machinery: identify its actual caller and owner; check who
   handles state, validation, errors, and lifecycle; verify whether a direct
   existing path already supplies the required behavior.
5. Match evidence to claims. A unit test does not prove real OAuth success,
   a single database test does not prove all dialects work, a screenshot does
   not prove keyboard accessibility, and a build does not prove low memory.
6. For UI and documentation, follow the actions a user is asked to perform.
   Check loading, errors, empty data, initialization, and recovery when those
   paths are within scope. Do not assume legacy behavior is a requirement.
7. Report findings first, ordered by demonstrated consequence, then genuine
   open questions or evidence gaps. Return fixes to implementation.

## Finding Standard

Report a finding only when all four are present:

1. a precise fact in the current artifact;
2. a governing source, explicit contract, reproduction, or labelled judgment;
3. a concrete correctness, usability, security, data-loss, resource, delivery,
   or maintenance consequence;
4. the smallest credible resolution direction.

Set severity from consequence and likelihood. Do not report stylistic taste,
hypothetical threats outside the accepted boundary, unmeasured performance
concerns, missing abstractions where direct code is clear, or compatibility
concerns for an unreleased surface without a compatibility requirement.

If no finding meets this standard, say so and identify residual verification
gaps. Do not modify files, add a standing audit report, or require another
agent process solely to perform this role.
