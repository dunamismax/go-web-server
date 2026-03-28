# Development

Run all commands in this guide from the repo root.

## Prerequisites

- Go `1.26.1`
- PostgreSQL
- Node.js and `npm` for the Tailwind build
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
| `mage dev` | Run the app with Air hot reload |
| `mage run` | Build and run once |
| `mage generate` | Regenerate SQLC, Templ, and CSS output |
| `mage fmt` | Format Go and Templ files and tidy modules |
| `mage vet` | Run `go vet ./...` |
| `mage test` | Run `go test ./...` |
| `mage lint` | Run `golangci-lint` |
| `mage quality` | Run vet, test, lint, and `govulncheck` |
| `mage ci` | Run the main local CI pipeline |
| `mage build` | Build `bin/server` |

## Database Workflow

- The app calls [`store.InitSchema()`](../internal/store/store.go) on startup, so a fresh local database can boot even if you have not run Atlas yet.
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

Generated artifacts are checked in for reproducible builds and releases. After changing SQL, Templ views, or CSS source, run `mage generate` and commit the resulting updates.

CI reruns `mage generate` and fails if tracked generated files or built assets drift.

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
mage lint
npm run build-css
```

If Atlas is part of the change, also run `mage migrateStatus`.

The repo still has light automated coverage, so manual UI checks remain important when you touch auth, sessions, or the `/users` flow.
