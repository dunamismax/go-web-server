# go-web-server

`go-web-server` is a Go starter with an Echo backend, PostgreSQL, SQLC, Mage, and a shipped Astro + Vue browser frontend embedded into the Go binary from `web/dist`. Go still owns sessions, CSRF, routing, and persistence. The browser UI now uses explicit same-origin JSON contracts under `/api/*`, with only the browser auth form posts kept as redirect-oriented fallbacks.

## What You Get

- Session-based login, registration, logout, and profile pages
- A protected `/users` CRUD screen backed by PostgreSQL
- An embedded Astro + Vue + Bun frontend for `/`, `/auth/login`, `/auth/register`, `/auth/logout`, `/profile`, and `/users`
- Explicit JSON contracts for auth state and managed-user CRUD under `/api/*`
- CSRF protection, security headers, request IDs, rate limiting, and structured errors
- Mage tasks for setup, generation, formatting, linting, testing, building, and release work
- Atlas migrations plus a schema bootstrap path for fresh local bring-up
- CI that regenerates SQLC output, verifies frontend install/check/build, runs mocked Playwright coverage, and exercises both Astro-preview and Go-served browser smoke flows

## What You Do Not Get

- Roles, per-user authorization, or record ownership rules
- Password reset, email verification, or account recovery
- Metrics, tracing, or `pprof` endpoints wired into the app
- A polished design system or product-specific architecture
- A complete production platform story beyond simple single-host deployment

## Quick Start

1. Start PostgreSQL locally and create a database:

```bash
createuser -P gowebserver
createdb -O gowebserver gowebserver
```

2. Copy the sample environment file and update it for your machine:

```bash
cp .env.example .env
```

Set at least:

- `DATABASE_URL=postgres://gowebserver:your-password@localhost:5432/gowebserver?sslmode=disable`
- `AUTH_COOKIE_SECURE=false` for plain HTTP localhost development

3. Install the local toolchain and generate derived backend code:

```bash
mage setup
mage generate
```

4. Start the app:

```bash
mage dev
```

Use `mage run` if you want a plain build-and-run without Air.

5. Optional but recommended: run the Astro browser smoke check to prove the frontend can register, load protected pages, and exercise the real Go backend end to end:

```bash
mage frontendSmoke
```

6. If you also want coverage for the shipped Go-served browser path, run the Docker-backed runtime smoke check:

```bash
mage smoke
```

The app listens on [http://localhost:8080](http://localhost:8080). Open [http://localhost:8080/auth/register](http://localhost:8080/auth/register) to create the first account.

## Common Commands

| Command | Purpose |
| --- | --- |
| `mage setup` | Install Go tools and download dependencies |
| `mage dev` | Run the current Go app with Air hot reload |
| `mage frontendInstall` | Install Bun-managed frontend dependencies in `web/` |
| `mage frontendDev` | Run the Astro frontend on port `4321` |
| `mage frontendCheck` | Run Biome, `astro check`, and Bun tests for `web/` |
| `mage frontendBuild` | Build the Astro frontend |
| `mage frontendSmoke` | Run the Astro browser smoke flow against the real Go backend |
| `mage run` | Build and run the server once |
| `mage generate` | Regenerate SQLC output |
| `mage fmt` | Format Go code and tidy modules |
| `mage vet` | Run `go vet ./...` |
| `mage test` | Run `go test ./...` |
| `mage smoke` | Run the Docker-backed Go-served runtime smoke validation |
| `mage lint` | Run `golangci-lint` |
| `mage quality` | Run vet, test, lint, and `govulncheck` |
| `mage ci` | Run the main local CI-style pipeline |
| `mage migrate` | Apply Atlas migrations |
| `mage migrateStatus` | Show Atlas migration state |

`mage migrateDown` is informational only. Atlas does not auto-rollback this repo.

## Verification

These are the baseline checks for code changes:

```bash
go build ./...
go vet ./...
go test ./...
```

Cheap repo-native checks that are worth running when relevant:

```bash
mage frontendCheck
mage frontendBuild
mage lint
```

For real browser-path bring-up confidence, run the Astro browser smoke check when Bun and Playwright are available:

```bash
./scripts/frontend-smoke.sh
```

If you want coverage for the shipped Go-served browser path, run the runtime smoke check when Docker is available:

```bash
./scripts/runtime-smoke.sh
```

`mage ci` mirrors the main local validation flow without calling the formatting target.

## Documentation

- [BUILD.md](BUILD.md) for the active phased execution plan while this starter is still being shaped
- [Docs index](docs/README.md)
- [Development guide](docs/development.md)
- [API and route behavior](docs/api.md)
- [Frontend migration inventory](docs/frontend-migration-inventory.md)
- [Security notes](docs/security.md)
- [Architecture overview](docs/architecture.md)
- [Deployment notes](docs/deployment.md)
- [Ubuntu deployment walkthrough](docs/ubuntu-deployment.md)
- [Example YAML config](docs/config.example.yaml)

## Current-State Notes

- The canonical Atlas migration directory is top-level [`migrations/`](migrations/).
- The duplicate `internal/store/migrations/` directory is legacy history, not the source of truth.
- [`internal/store/schema.sql`](internal/store/schema.sql) is the schema source used for SQLC and Atlas.
- The app still keeps a startup bootstrap path in [`internal/store/store.go`](internal/store/store.go), but it now executes the canonical [`internal/store/schema.sql`](internal/store/schema.sql) before applying a small legacy reconciliation patch for older local databases.
- Generated SQLC output and the built `web/dist` frontend are checked in. CI reruns `mage generate` and frontend build checks to catch drift.
- `web/` is the Astro + Vue + Bun frontend workspace. Its built `web/dist` output is committed and embedded into the Go binary for the shipped browser routes.
- The embedded Astro build still talks to `/_backend/*`; the Go server strips that prefix in-process for shipped builds, while Astro dev keeps using it as a real proxy prefix.
- Browser auth fallback submits still exist at `POST /auth/login`, `POST /auth/register`, and `POST /auth/logout`, but managed-user browser mutations now go through `/api/users/*` only.
- CI installs Bun dependencies, runs frontend checks and builds, exercises the mocked Playwright suite, and then runs both the Astro browser smoke flow and the Go-served runtime smoke flow.
- [`scripts/frontend-smoke.sh`](scripts/frontend-smoke.sh) validates the Astro home, registration, profile, users create/edit/deactivate/delete, and logout flow against the real Go backend.
- [`scripts/runtime-smoke.sh`](scripts/runtime-smoke.sh) validates the Go-served embedded Astro shell, the same-origin `/_backend/*` bridge it uses at runtime, and the browser registration redirect path.
- Leave `security.trusted_proxies` empty unless the app is actually behind reverse proxies you control.

## Naming Notes

This repo currently has two names in play:

- Repo, local checkout, and Go module path: `go-web-server`
- Deployment user/service/database examples: `gowebserver`

The repo and module naming are aligned again. The `gowebserver` deployment naming stays as a simple service/database slug.
