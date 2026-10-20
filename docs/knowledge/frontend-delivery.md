---
title: Static frontend builds with Bun and Go
description: Bun script execution, reproducible dependencies, and multi-platform Docker builds.
topics: [bun, vite, frontend, docker, ci, packaging]
checked_at: 2026-10-16
---

# Static frontend builds with Bun and Go

- **Official guidance:** Bun can install dependencies and run Vite. Running a
  script with Bun does not automatically override a CLI's Node shebang; use
  `--bun` or the corresponding run configuration when Bun must be the runtime.
  [Vite with Bun](https://bun.com/guides/ecosystem/vite),
  [Bun script execution](https://bun.com/docs/runtime)
- **Official guidance:** commit `bun.lock` and use
  `bun install --frozen-lockfile` for reproducible installations. Bun can migrate
  npm lockfiles. [Bun install](https://bun.com/docs/pm/cli/install)
- **Official guidance:** `bun run test` invokes a package script; `bun test`
  selects Bun's own test runner. Projects using Vitest should distinguish them.
  [Vitest getting started](https://vitest.dev/guide/)
- **Official guidance:** Docker supports multi-stage builds and Go cross
  compilation with `BUILDPLATFORM`, `TARGETOS`, and `TARGETARCH`. Native build
  stages avoid emulating the compiler for each target architecture.
  [Docker multi-platform builds](https://docs.docker.com/build/building/multi-platform/)
- **Methodological synthesis:** a static browser bundle needs its build tools
  during compilation. Serving the completed bundle from another application
  does not require installing those tools in the production image.
