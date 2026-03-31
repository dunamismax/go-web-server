# Development Guide

## Tooling

Required for normal development:

- Go
- PostgreSQL
- Mage
- Bun for the `web/` frontend workspace

Optional but commonly useful:

- Docker for smoke scripts that self-start PostgreSQL
- Air for Go hot reload
- Playwright browsers for frontend smoke validation

## First-Time Setup

```bash
cp .env.example .env
mage setup
mage generate
```

Then start the app:

```bash
mage dev
```

## Core Commands

| Command | Purpose |
| --- | --- |
| `mage setup` | Install Go tools and download dependencies |
| `mage dev` | Run the Go app with Air |
| `mage run` | Build and run once |
| `mage generate` | Regenerate SQLC output |
| `mage fmt` | Format Go code and tidy modules |
| `mage vet` | Run `go vet ./...` |
| `mage test` | Run `go test ./...` |
| `mage lint` | Run `golangci-lint` |
| `mage quality` | Run vet, test, lint, and `govulncheck` |
| `mage ci` | Run the main local CI-style pipeline |
| `mage migrate` | Apply Atlas migrations |
| `mage migrateStatus` | Show Atlas migration status |

### Frontend commands

| Command | Purpose |
| --- | --- |
| `mage frontendInstall` | Install Bun dependencies in `web/` |
| `mage frontendDev` | Run Astro dev on port `4321` |
| `mage frontendCheck` | Run Biome, `astro check`, and Bun tests |
| `mage frontendBuild` | Build the frontend into `web/dist` |
| `mage frontendSmoke` | Run the Astro browser smoke flow |
| `mage smoke` | Run the Docker-backed Go-served runtime smoke flow |

## Generated Files And Checked-In Artifacts

The repo intentionally checks in:

- SQLC-generated Go output under `internal/store/`
- the built frontend under `web/dist/`

`mage generate` only covers SQLC generation now. Frontend artifact drift is caught by the frontend build and smoke checks.

## What Requires Regeneration

Run `mage generate` after changing:

- `internal/store/queries.sql`
- `internal/store/schema.sql`
- SQLC config that affects generated output

Run frontend checks and rebuild after changing:

- files under `web/src/`
- frontend config under `web/`
- content that should change the committed `web/dist/` output

## Verification

Start narrow and broaden when needed:

```bash
go build ./...
go vet ./...
go test ./...
```

Frontend checks:

```bash
mage frontendCheck
mage frontendBuild
```

Smoke validation:

```bash
./scripts/frontend-smoke.sh
./scripts/runtime-smoke.sh
```

## Current Frontend Boundary

- `web/` is the only active browser frontend workspace.
- The Go server embeds `web/dist` and serves it for the shipped browser pages.
- The frontend uses `/_backend/*` as its same-origin bridge and the Go server strips that prefix in shipped builds.
- Managed-user mutations now go through `/api/users/*` only.
- Browser auth form fallback submits remain at `POST /auth/*` for simple non-JavaScript flows and smoke coverage.
