# go-web-server

`go-web-server` is a small Go starter for server-rendered apps with Echo, Templ, HTMX, PostgreSQL, SQLC, and Mage. The goal is boring, legible defaults: one binary, one Postgres database, session auth, and enough structure to ship without dragging in a giant framework.

## What You Get

- Session-based login, registration, logout, and profile pages
- A protected `/users` CRUD screen backed by PostgreSQL
- Templ views with HTMX interactions and generated Tailwind CSS
- CSRF protection, security headers, request IDs, rate limiting, and structured errors
- Mage tasks for setup, generation, formatting, linting, testing, building, and release work
- Atlas migrations plus a schema bootstrap path for fresh local bring-up
- CI that regenerates derived files and verifies build, vet, test, lint, and vulnerability checks

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

3. Install the local toolchain and generate code/assets:

```bash
mage setup
mage generate
```

4. Start the app:

```bash
mage dev
```

Use `mage run` if you want a plain build-and-run without Air.

The app listens on [http://localhost:8080](http://localhost:8080). Open [http://localhost:8080/auth/register](http://localhost:8080/auth/register) to create the first account.

## Common Commands

| Command | Purpose |
| --- | --- |
| `mage setup` | Install Go tools and download dependencies |
| `mage dev` | Run the app with Air hot reload |
| `mage run` | Build and run the server once |
| `mage generate` | Regenerate SQLC, Templ, and CSS output |
| `mage fmt` | Format Go and Templ files and tidy modules |
| `mage vet` | Run `go vet ./...` |
| `mage test` | Run `go test ./...` |
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
mage lint
npm run build-css
```

`mage ci` now mirrors the main local validation flow without calling the formatting target.

## Documentation

- [Docs index](docs/README.md)
- [Development guide](docs/development.md)
- [API and route behavior](docs/api.md)
- [Security notes](docs/security.md)
- [Architecture overview](docs/architecture.md)
- [Deployment notes](docs/deployment.md)
- [Ubuntu deployment walkthrough](docs/ubuntu-deployment.md)
- [Example YAML config](docs/config.example.yaml)

## Current-State Notes

- The canonical Atlas migration directory is top-level [`migrations/`](migrations/).
- The duplicate `internal/store/migrations/` directory is legacy history, not the source of truth.
- [`internal/store/schema.sql`](internal/store/schema.sql) is the schema source used for SQLC and Atlas.
- The app still keeps a startup bootstrap path in [`internal/store/store.go`](internal/store/store.go) so a fresh local database can boot before explicit Atlas migration work.
- Generated files and built frontend assets are checked in. CI runs `mage generate` and fails if that changes tracked files.
- Local runtime verification still depends on a reachable PostgreSQL instance and Atlas CLI when you want explicit migration state checks.
- Leave `security.trusted_proxies` empty unless the app is actually behind reverse proxies you control.
- `package-lock.json` is tracked so frontend dependency resolution stays reproducible across contributors and CI.

## Naming Notes

This repo currently has two names in play:

- Repo, local checkout, and Go module path: `go-web-server`
- Deployment user/service/database examples: `gowebserver`

The repo and module naming are aligned again. The `gowebserver` deployment naming stays as a simple service/database slug.
