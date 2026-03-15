# CLAUDE.md

## Project Overview

E5 Renewal is a self-hosted Microsoft 365 E5 developer subscription renewal tool. It automates Graph API calls on a configurable schedule to keep E5 subscriptions active.

- **Backend:** Go 1.25 + Gin framework + SQLite (GORM)
- **Frontend:** Vue 3 + TypeScript + Tailwind CSS + Element Plus
- **Deployment:** Single Docker image with embedded frontend (distroless)

## Project Structure

```
e5-renewal/
├── backend/                 # Go backend
│   ├── main.go              # Entry point, wires all components
│   ├── config/              # YAML + env config loader
│   ├── database/            # GORM models and repositories
│   ├── handlers/            # Gin HTTP handlers (REST API)
│   ├── middleware/          # Auth (JWT), rate limiting, logging
│   ├── models/              # Shared data structures
│   ├── services/
│   │   ├── executor/        # Graph API call execution
│   │   ├── graph/           # Microsoft Graph API client
│   │   ├── login/           # Login key management (bcrypt)
│   │   ├── notifier/        # Shoutrrr push notifications
│   │   ├── oauth/           # OAuth 2.0 flow (authorization code)
│   │   ├── scheduler/       # Periodic task scheduling with timers
│   │   └── security/        # JWT, bcrypt, AES encryption
│   ├── spa/                 # SPA static file handler
│   └── static/dist/         # Embedded frontend build (generated)
├── frontend/                # Vue 3 SPA
│   ├── src/
│   │   ├── components/      # Reusable Vue components
│   │   ├── views/           # Page-level views
│   │   ├── api/             # API client (axios)
│   │   ├── i18n/            # Internationalization (zh-CN, en)
│   │   ├── router/          # Vue Router config
│   │   └── stores/          # Pinia stores (auth)
│   └── vite.config.ts
├── Dockerfile               # Multi-stage build (node → go → distroless)
├── docker-compose.yml       # Local deployment template
└── e5-renewal.yaml.example  # Config file template
```

## Key Conventions

- **Language:** Go code uses standard library patterns; frontend uses Vue 3 Composition API with `<script setup>`
- **Database:** All DB access goes through repository structs in `database/` package, never raw SQL in handlers
- **Auth:** JWT tokens for API auth; login key stored as bcrypt hash in DB
- **Secrets:** Client secrets and refresh tokens encrypted with AES-GCM (encryption_key in config)
- **Config precedence:** Environment variables > config file > defaults
- **Error responses:** Always `gin.H{"error": "message"}` format
- **i18n:** Frontend supports zh-CN and en; notification messages support both via `notifier/messages.go`

## Development Commands

```bash
# Backend
cd backend
go build ./...              # Build
go test ./...               # Run tests
go test -race ./...         # Run tests with race detector
golangci-lint run           # Lint (see .golangci.yml for config)

# Frontend
cd frontend
npm ci                      # Install dependencies
npm run dev                 # Dev server
npm run build               # Production build
npx eslint src/             # Lint
npx vitest run              # Run tests

# Docker
docker build -t e5-renewal:latest .
```

## CI Requirements

- Backend coverage >= 80%
- Frontend coverage >= 80%
- ESLint warnings <= 20
- golangci-lint must pass (strict config in .golangci.yml)
- Docker build must succeed

## Important Notes

- The scheduler uses `time.AfterFunc` with context cancellation for clean shutdown; do not add WaitGroup tracking to timer callbacks
- `rateLimiter` has a `Stop()` method for cleanup goroutine; it's exported for testability
- OAuth `/callback` endpoint must remain unauthenticated (Microsoft redirect target)
- The `maskSecret` function is used to detect if a client sent back masked values on update; compare exact masked output, not pattern matching
- Frontend build output is embedded into the Go binary via `embed.FS`
