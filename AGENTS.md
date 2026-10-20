# E5 Renewal Repository Guidance

## Product Boundary

- E5 Renewal is a self-hosted Microsoft OAuth account manager and Graph task runner.
- Keep credentials, token refresh, execution, scheduling, and storage in the Go
  backend. The browser is a statically built management interface.
- There is no legacy API or database compatibility requirement. OAuth and Graph
  behavior can be inspected at reference commit
  `b003df437fa5d7757681dffdff28a5875b0f6fbf` with `git show`.
- Keep resource use bounded and measure it before making performance claims.
- Preserve the MIT license and notices for dependencies or source actually used.

## Repository Map

- `cmd/e5-renewal`: executable startup and local administrator recovery.
- `internal/app`: configuration, persistence, authorization, HTTP API, execution,
  and scheduling.
- `frontend`: React management interface, public UI primitives, and translations.
- `web`: embedded production frontend assets.
- `docs/knowledge`: verified external engineering facts, not plans or audit logs.
- `.agents/skills`: knowledge, implementation, and read-only audit workflows.

## Development

Run `make fmt` after edits and focused behavior checks while changing a component.
Go checks require golangci-lint 2.14.0. Run `make check` for the complete
local gate and `make build` for the executable. Docker and external database
integration checks require a running container engine. Report unavailable
checks instead of treating them as passed.

## Change Rules

- Read affected files, callers, tests, configuration, and public contracts
  before editing. Complete every part of a changed contract together.
- Keep Microsoft credentials and token refresh ownership in the backend.
  Validate external input, redact credentials from logs, and authenticate
  management operations regardless of database choice or URL prefix.
- Use existing language and platform facilities before introducing another
  dependency, layer, configurable backend, or service. Implement the storage
  engines and deployment modes actually agreed for the release.
- Preserve the compact interface, existing UI primitives, typography, colors,
  and responsive behavior. Keep long diagnostics in details rather than table rows.
- Keep setup, usage, and contribution instructions in README.md.
- Match evidence to the claim: behavior tests for behavior, build checks for
  packaging, browser checks for UI, and reproducible measurements for CPU and
  memory. Do not claim a resource target is met without measurements.
- Run the relevant focused checks, then the aggregate gate once one exists.
  Read fresh results and explicitly report unavailable verification.
- Keep generated artifacts, credentials, local environments, editor state,
  temporary reports, and personal instructions out of version control.
- Write backend messages, logs, comments, and public documentation in English.
  Keep interface copy in the English and Chinese catalogs. API and persisted
  failures use stable error codes; do not translate by matching diagnostic prose.

## Project Skills

- Use `e5-renewal-knowledge` to retrieve and verify source facts. Change
  `docs/knowledge` only when knowledge maintenance is requested.
- Use `e5-renewal-implementation` for code, tests, configuration, CI, or
  repository documentation changes.
- Use `e5-renewal-audit` for read-only review after implementation or on request.

Completion requires the applicable checks followed by a read-only audit of the
final artifact. Auditing is a separate review pass; fixes return to the
implementation workflow. These roles do not require a separate agent process.

<!-- CODEGRAPH_START -->
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the
repo root), reach for it before grep/find or reading files to understand or
locate code. Use `codegraph_explore`, or
`codegraph explore "<symbol names or question>"`.

If there is no `.codegraph/` directory, skip CodeGraph. Indexing is the user's
decision.
<!-- CODEGRAPH_END -->
