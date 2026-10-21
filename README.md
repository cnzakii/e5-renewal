# E5 Renewal

[![CI](https://github.com/cnzakii/e5-renewal/actions/workflows/ci.yml/badge.svg)](https://github.com/cnzakii/e5-renewal/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A self-hosted tool for connecting Microsoft accounts, scheduling read-only Graph
requests, and checking their results from a browser.

- Connect through Microsoft sign-in or application credentials.
- Set execution intervals, view account activity, and inspect failed requests.
- Run with Docker or one executable, with SQLite included.
- Chinese and English, light and dark themes.
- Optional PostgreSQL or MySQL storage.

Successful Graph requests do not guarantee Microsoft subscription renewal.

## Run with Docker

```console
docker run -d --name e5-renewal \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v e5-data:/data \
  -e E5_PUBLIC_URL=http://localhost:8080 \
  ghcr.io/cnzakii/e5-renewal:latest
```

This command allows local access on the Docker host. Open
`http://localhost:8080` there. First startup initializes the database and prints
an administrator password once:

```console
docker logs e5-renewal
```

Change it after signing in. You can instead supply `E5_INITIAL_PASSWORD` on
first startup; supplied passwords are not printed. Restarts preserve the password.

For Compose, clone this repository, set `E5_PUBLIC_URL` in `.env`, and run
`docker compose up -d`. To build this checkout locally, run
`docker build -t e5-renewal .` and use `e5-renewal` in the command above.

Set `E5_PUBLIC_URL` to the full HTTP or HTTPS address people use in their browser,
including any port, for example `https://e5.example.com`. Behind a reverse proxy,
forward traffic to the container's HTTP port. For direct network access, publish
`8080:8080` instead of the loopback mapping.

Behind a reverse proxy, set `E5_TRUSTED_PROXIES` to its source IP address as seen
by the application, or its dedicated CIDR range. For example, a native process
behind a local proxy can use `127.0.0.1,::1`; Docker may use a bridge gateway
address instead. The proxy must set or append the actual client IP in
`X-Forwarded-For`. Only explicitly trusted proxies can supply client addresses
for login limits; otherwise the connection's source IP is used. Trust only your
proxy addresses, since trusting arbitrary clients lets them bypass these limits.

To use a subpath, set `E5_PUBLIC_URL=https://e5.example.com/tools/e5` and restart.
Open `https://e5.example.com/tools/e5/login`; pages, assets, APIs, session cookies,
and OAuth callbacks follow that prefix. The same executable and image work at
the domain root or under a prefix. Forward the complete path through your reverse
proxy; do not strip or rewrite the prefix there. Without a path in `E5_PUBLIC_URL`,
the application runs at the domain root.

Use plain path segments such as `/e5` or `/tools/e5`. Trailing slashes are
normalized; percent-encoded segments, dot segments, backslashes, semicolons, and
duplicate interior slashes are rejected. With a prefix, URLs outside it return
404, including the root `/api` and `/healthz`; use `/tools/e5/healthz` instead.
After changing the prefix, update the Web redirect URI in Entra to the callback
address shown in **Settings**, then authorize again if an authorization was in progress.

For browser authorization, Microsoft accepts HTTP callbacks on `localhost`;
use HTTPS for a remote callback. See Microsoft's
[redirect URI requirements](https://learn.microsoft.com/en-us/entra/identity-platform/reply-url).

The container runs as UID/GID `65532`; bind-mounted directories must be writable
by that user. Run one application instance per database.

## Run a native executable

Download the archive matching your system from
[Releases](https://github.com/cnzakii/e5-renewal/releases):

| System | Archive |
| --- | --- |
| Linux x86-64 | `e5-renewal-linux-amd64.tar.gz` |
| Linux ARM64 | `e5-renewal-linux-arm64.tar.gz` |
| macOS Intel | `e5-renewal-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `e5-renewal-darwin-arm64.tar.gz` |

Extract the archive and run:

```console
E5_PUBLIC_URL=http://localhost:8080 ./e5-renewal
```

Open `http://localhost:8080` on the same machine and sign in with the password
printed in the terminal.

By default, data is saved in `./data` relative to your working directory. Keep
using that directory, or set `E5_DATA_DIR` to a persistent absolute path. For
access from another machine, set `E5_LISTEN=0.0.0.0:8080` and `E5_PUBLIC_URL` to
the address you will open in your browser.

## Connect Microsoft accounts

Choose **delegated authorization** to sign in an account and read its mail,
OneDrive, calendars, and contacts. Choose **application authorization** for
directory and SharePoint reads using administrator-approved tenant permissions.

1. In [Microsoft Entra](https://entra.microsoft.com), open **App registrations**
   and register an application for the accounts you intend to connect.
2. From its **Overview**, copy **Application (client) ID** and
   **Directory (tenant) ID**. For a single-tenant application, use this tenant ID
   in E5 Renewal instead of the default `common`.
3. Under **Certificates & secrets**, create a client secret. Copy its **Value**
   immediately; **Secret ID** is a different field.
4. Under **API permissions → Add a permission → Microsoft Graph**, select the
   matching delegated or application permissions listed below. Application
   permissions require **Grant admin consent**.
5. In E5 Renewal, add an account with those IDs and the secret value. For browser
   authorization, copy the callback address from **Settings**, register it as a
   **Web** redirect URI in Entra, then select **Save and authorize**.
6. Sign in to Microsoft and approve the permissions. After returning, select
   **Run now** and inspect the result. Then edit the account to enable automatic
   execution. Application authorization skips browser sign-in.

| Authorization | Required Microsoft Graph permissions |
| --- | --- |
| Delegated | `User.Read`, `Files.Read`, `Mail.Read`, `Calendars.Read`, `Contacts.Read`, plus `offline_access` |
| Application | `User.Read.All`, `Organization.Read.All`, `Sites.Read.All` |

The shared interface pool covers profile, OneDrive, mail, calendars, contacts,
directory, licensing, and SharePoint reads: 14 delegated and 10 application
interfaces. Each run samples between one and all matching interfaces and shuffles
their order. Microsoft services must be provisioned and the required permissions
consented to. Imported tokens must cover those permissions.

For manual callback, enter a registered Web redirect URI, open the generated
sign-in link, then paste the full returned address containing `code` and `state`.
Credentials and token exchange stay in the backend. Stored secrets are never
returned by the management API. Changing delegated credentials requires authorization again.

## Scheduling and history

Set minimum and maximum intervals in minutes. The next interval is chosen at
random after each run finishes; equal values give a fixed interval. Manual and
automatic runs share one execution slot. Overdue accounts run once after restart.
An invalid Microsoft grant disables automatic execution. Reauthorize the account,
test **Run now**, then select **Edit → Enable automatic execution → Save account**
to resume its schedule.

History is kept for 30 days and cleaned daily. Filter by account, result, trigger,
and time range. Charts and date filters use the browser's system time zone.
Each Graph call records its interface, time, duration, HTTP status, attempts, and
error code. Response bodies are discarded. Token failures are shown separately
and do not count as Graph calls.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `E5_LISTEN` | `127.0.0.1:8080` | Listen address; Docker uses `0.0.0.0:8080` |
| `E5_PUBLIC_URL` | `http://127.0.0.1:8080` | Browser-facing HTTP or HTTPS URL, with an optional path prefix |
| `E5_TRUSTED_PROXIES` | unset | Comma-separated proxy IPs or CIDRs trusted to supply `X-Forwarded-For` |
| `E5_DATA_DIR` | `data` | Data directory; `/data` in Docker |
| `E5_DB_DRIVER` | `sqlite` | `sqlite`, `postgres`, or `mysql` |
| `E5_DB_DSN` | `<data directory>/e5.db` | SQLite filename or external database connection string |
| `E5_INITIAL_PASSWORD` | generated | Administrator password on first initialization |
| `E5_LOG_LEVEL` | `INFO` | `DEBUG`, `INFO`, `WARN`, or `ERROR` |
| `E5_LOG_FORMAT` | `text` | Structured `text` or `json` logs |
| `E5_LOG_FILE` | unset | Optional additional rotating log file |

External databases must already exist; startup creates tables and indexes.
Use a dedicated database account with schema-management permissions. Example DSNs:

```text
host=db.example.com user=e5 password=REPLACE_ME dbname=e5 sslmode=require
e5:REPLACE_ME@tcp(db.example.com:3306)/e5?charset=utf8mb4&parseTime=true&loc=UTC&tls=true
```

Administrator passwords are hashed with Argon2id. Microsoft credentials are
stored without field encryption, so protect the database and its backups.
Changing database configuration does not move existing data.

## Password recovery

Stop the application, run `--reset-password` with the same database configuration,
then start it again. For Compose:

```console
docker compose stop e5-renewal
docker compose run --rm --no-deps e5-renewal --reset-password
docker compose up -d e5-renewal
```

For the container created above:

```console
docker stop e5-renewal
docker run --rm --volumes-from e5-renewal "$(docker inspect --format '{{.Image}}' e5-renewal)" --reset-password
docker start e5-renewal
```

The reset command reuses the installed image. If you configured an external
database or a custom data path with environment variables, pass those same
variables to `docker run`; `--volumes-from` only shares mounted storage.

For a native installation, run `./e5-renewal --reset-password` while the server
is stopped. The new password is printed once. Accounts and history remain intact.
Web password changes revoke existing sessions immediately; restarting also ends
browser sessions.

Login limits allow five immediate attempts per source IP, then restore one
attempt every 12 seconds. Password changes have a separate allowance. Password
verification is limited to two concurrent requests; excess requests receive
HTTP 429 with a `Retry-After` header. These limits are held in bounded process
memory and reset on restart; the administrator password and stored accounts do
not change. Clients sharing a public IP also share its login allowance.

## Logs

Logs go to stdout by default. Use `docker logs e5-renewal` or your service journal.
Set `E5_LOG_FORMAT=json` for a log collector. Set `E5_LOG_FILE=/data/e5-renewal.log`
to also save logs, rotating at 10 MiB with three backups.

Generated password notices go only to the console, but Docker, service journals,
and shell redirection can retain console output. Requests include `X-Request-ID`
for matching API errors with logs. Credentials, request bodies, and OAuth query
strings are excluded from application logs.

## Development and contributions

Install Go 1.26+, Bun 1.4.3, and golangci-lint 2.14.0. CI, Docker, and native
release packaging use the build toolchain recorded in `go.mod`.
When updating Go, keep its `toolchain` directive and the pinned Go image in
`Dockerfile` on the same version.

```console
cd frontend && bun install --frozen-lockfile && cd ..
make fmt
make check
make build
./bin/e5-renewal
```

`make check` runs Go formatting/static analysis, race-enabled tests, Prettier,
ESLint, frontend behavior tests, TypeScript checking, and a production UI build.
`make fmt` also applies Go block-spacing rules and explicit frontend branch braces.
`make bench` runs API benchmarks; `make fuzz` checks account-input invariants.
Use `GOLANGCI_LINT=/path/to/golangci-lint` if needed.

For hot reload, start the backend with `E5_PUBLIC_URL=http://localhost:5173`,
then run `cd frontend && bun run dev`. Register that development callback in Entra.
Tests use simulated Microsoft responses. External database tests use dedicated
`E5_TEST_POSTGRES_DSN` and `E5_TEST_MYSQL_DSN` databases; CI provides disposable services.

Issues and pull requests are welcome. Describe the behavior being changed and
include a focused regression test for bug fixes. Keep Go diagnostics and comments
in English, UI copy in both translation catalogs, and credentials in the backend.

## Releases

PRs and changes to `main` run checks, verify native macOS builds and tests, and
build Linux `amd64`/`arm64` container images.
Monthly Dependabot updates cover Go, Bun, Docker, and GitHub Actions.

CI also packages all four native targets and starts the container with fresh
storage, checking the embedded UI, sign-in, persistence across restarts, and
graceful shutdown. Install [GoReleaser](https://goreleaser.com/install/), then run
`make dist VERSION=v1.2.3` to build the archives locally without publishing.
`scripts/container-smoke.sh IMAGE [BASE_PATH]` checks a local container image using
Python 3, including authentication, execution failures, and persistent history.

Pushing a version tag such as `v1.2.3` runs the checks, publishes
`ghcr.io/cnzakii/e5-renewal:1.2.3`, and prepares a draft GitHub release with Linux
and macOS archives with SHA-256 checksums. Review the draft before publishing it.
Publishing the current stable release promotes its existing image to `latest`;
prereleases such as `v1.2.3-rc.1` do not. Published releases cannot be overwritten
by rerunning the workflow; fixes need a new version tag.

## Acknowledgements

Thanks to [Beszel](https://github.com/henrygd/beszel) for the UI inspiration.

## License

[MIT](LICENSE).
