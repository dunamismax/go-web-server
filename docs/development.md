# Development

Run all commands in this guide from the repo root.

## Prerequisites

- Go `1.26.1`
- PostgreSQL
- Bun for both the repo-root legacy CSS asset build and the staged Astro + Vue frontend workspace under `web/`
- Atlas CLI if you want explicit migration state locally

## First-Time Local Setup

1. Make sure PostgreSQL is running.
2. Create a local database and user, or point the app at an existing Postgres instance:

```bash
createuser -P gowebserver
createdb -O gowebserver gowebserver
```

3. Copy the sample environment file:

```bash
cp .env.example .env
```

4. Update at least these values in `.env`:

- `DATABASE_URL=postgres://gowebserver:your-password@localhost:5432/gowebserver?sslmode=disable`
- `AUTH_COOKIE_SECURE=false`
- `APP_ENVIRONMENT=development`

5. Install local tools and generate the derived files:

```bash
mage setup
mage generate
```

6. Start the app:

```bash
mage dev
```

Use `mage run` if you want a plain build-and-run without Air.

7. Optional frontend migration workflow: run the staged Astro routes in a second terminal.

```bash
mage frontendInstall
mage frontendDev
```

By default the Astro dev server listens on `http://127.0.0.1:4321` and proxies `/_backend/*` to the Go app on `http://127.0.0.1:8080`. The staged frontend now includes home plus the login, registration, logout, profile, and users flows.

8. Optional but recommended: prove the Astro browser path end to end with the real frontend smoke check:

```bash
mage frontendSmoke
```

9. If you also want coverage for the current shipped Go-served browser path, run the Docker-backed runtime smoke check:

```bash
mage smoke
```

## Configuration Sources

Configuration loads in this order:

1. Built-in defaults
2. `.env`
3. `config.yaml` or `config/config.yaml`
4. Environment variables

Environment variables win last. If `DATABASE_URL` is empty, the app tries to build it from `DATABASE_USER`, `DATABASE_PASSWORD`, `DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_NAME`, and `DATABASE_SSLMODE`.

See:

- [`.env.example`](../.env.example)
- [`config.example.yaml`](config.example.yaml)
- [`internal/config/config.go`](../internal/config/config.go)

## Daily Commands

| Command | Purpose |
| --- | --- |
| `mage dev` | Run the current Go app with Air hot reload |
| `mage frontendInstall` | Install Bun-managed frontend dependencies in `web/` |
| `mage frontendDev` | Run the staged Astro frontend shell |
| `mage frontendCheck` | Run Biome, `astro check`, and Bun tests in `web/` |
| `mage frontendBuild` | Build the staged Astro frontend |
| `mage frontendSmoke` | Run the Astro browser smoke flow against the real Go backend |
| `mage run` | Build and run once |
| `mage generate` | Regenerate SQLC, Templ, and CSS output |
| `mage fmt` | Format Go and Templ files and tidy modules |
| `mage vet` | Run `go vet ./...` |
| `mage test` | Run `go test ./...` |
| `mage smoke` | Run the Docker-backed Go-served runtime smoke validation |
| `mage lint` | Run `golangci-lint` |
| `mage quality` | Run vet, test, lint, and `govulncheck` |
| `mage ci` | Run the main local CI pipeline |
| `mage build` | Build `bin/server` |

## Database Workflow

- The app calls [`store.InitSchema()`](../internal/store/store.go) on startup, so a fresh local database can boot even if you have not run Atlas yet.
- That bootstrap path now executes the canonical [`internal/store/schema.sql`](../internal/store/schema.sql) first, then applies a small compatibility patch for older local databases that predate `password_hash` enforcement.
- The canonical schema file is [`internal/store/schema.sql`](../internal/store/schema.sql).
- The canonical migration directory is [`migrations/`](../migrations/).
- [`internal/store/migrations/`](../internal/store/migrations/) still exists as legacy history and should not be treated as the source of truth.

Use Atlas when you want explicit migration state:

```bash
mage migrate
mage migrateStatus
```

`mage migrateDown` does not roll back changes. It prints guidance only.

## Generated Files and Assets

Generated Go artifacts, the shipped Astro dist in `web/dist`, and legacy built frontend assets are checked in for reproducible builds and releases. After changing SQL, Templ views, CSS source, or the shipped frontend pages/components, regenerate the relevant outputs and commit the resulting updates.

The Astro frontend under `web/` uses Bun and keeps its own lockfile. Its built `web/dist` output is now embedded into the Go binary for the shipped browser GET routes. The repo root also tracks `bun.lock` for the legacy CSS asset build that still runs through `mage generate`.

CI reruns `mage generate` and fails if tracked generated files or built assets drift. It also installs Bun dependencies, runs frontend check and build steps, runs mocked Playwright e2e coverage, and then validates both the Astro browser smoke flow and the Go-served runtime smoke flow.

## What Requires Regeneration

Run `mage generate` after changing:

- [`internal/store/queries.sql`](../internal/store/queries.sql)
- Templ files under [`internal/view/`](../internal/view/)
- [`input.css`](../input.css)

## Verification

For most code changes, the baseline checks are:

```bash
go build ./...
go vet ./...
go test ./...
```

Useful repo-native follow-ups:

```bash
mage frontendCheck
mage frontendBuild
mage lint
bun run build-css
```

When Bun and Playwright are available, run the Astro browser smoke check for real frontend verification:

```bash
./scripts/frontend-smoke.sh
```

When Docker is available, also run the runtime smoke check for the shipped Go-served embedded Astro browser path:

```bash
./scripts/runtime-smoke.sh
```

If Atlas is part of the change, also run `mage migrateStatus`.

The repo still has light automated coverage, so manual UI checks remain important when you touch auth, sessions, or the `/users` flow.
