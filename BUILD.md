# BUILD.md

**This is the primary operational handoff document for `go-web-server`. It is a living document. Every future agent or developer who touches this repository is responsible for keeping it accurate, current, and up to date. If you verify, break, fix, rename, add, or remove anything that affects setup, build, runtime, deployment, testing, migrations, or source-of-truth ownership, update this file in the same change.**

Last reviewed: 2026-03-23
Reviewer: Claude
Repository: `/Users/sawyer/github/go-web-server`

## 1. Project Baseline

### What the application currently does

`go-web-server` is a small server-rendered Go monolith built around:

- Echo for routing and middleware
- Templ for HTML rendering
- HTMX for partial-page interactions
- PostgreSQL for app data and session storage
- SQLC for typed query generation
- Mage for local task orchestration
- Tailwind + DaisyUI + Pico CSS for styling

Current user-visible behavior:

- Public routes:
  - `/` home page
  - `/demo` HTMX/JSON demo endpoint
  - `/health` health endpoint with a database check
  - `/auth/login`
  - `/auth/register`
- Protected routes:
  - `/profile`
  - `/users` CRUD screen for users
  - `/api/users/count` returns an HTML fragment despite the `/api` prefix
- Authentication:
  - Session-based auth backed by PostgreSQL via SCS
  - Registration hashes passwords with Argon2id
  - Login rejects users missing a usable password hash

### Major components, services, modules, and entry points

- `cmd/web/main.go`
  - Main application entry point
  - Loads config, opens the PostgreSQL pool, initializes Echo, installs middleware, wires session auth, and starts the server
- `internal/config/config.go`
  - Runtime configuration source of truth
  - Config precedence: defaults -> `.env` -> `config.yaml` / `config/config.yaml` -> environment variables
- `internal/handler/`
  - Route handlers for home, auth, and users
- `internal/middleware/`
  - Recovery, security headers, sanitization, CSRF, validation, timeout, auth/session helpers, structured errors
- `internal/store/`
  - `schema.sql`: canonical schema definition for SQLC and Atlas
  - `queries.sql`: SQLC query definitions
  - `store.go`: pool wiring plus a startup bootstrap schema path
- `internal/view/`
  - Templ source files and checked-in generated `*_templ.go` files
- `internal/ui/`
  - Embedded static assets served from `/static/*`
- `magefile.go`
  - Primary repo task runner for setup, generation, build, lint, vet, migrations, and release helpers
- `package.json`
  - CSS build/watch commands
- `.air.toml`
  - Hot-reload config used by `mage dev`
- `atlas.hcl`
  - Atlas migration environment config
- `.github/workflows/ci.yml`
  - Current GitHub Actions CI workflow

### Current implemented state

This repo is a working starter app, not a finished product. The implemented system is:

- One Go binary
- One PostgreSQL database
- One main domain model: `users`
- One auth mode: database-backed sessions
- One protected CRUD surface: `/users`

It does not currently implement:

- Roles or per-record authorization
- Password reset, email verification, or account recovery
- Metrics or `pprof` endpoints, despite config flags existing for them
- A JSON-first API surface
- Background jobs or async workflows
- A polished deployment platform beyond simple single-host scripts

## 2. Verified Build and Run Workflow

### Verification environment used for this review

Verified on 2026-03-18 in this workspace with:

- `go version` -> `go1.26.1 darwin/arm64`
- `node -v` -> `v24.13.1`
- `npm -v` -> `11.8.0`
- `templ version` -> `v0.3.1001`
- `sqlc version` -> `v1.30.0`
- `atlas version` -> not installed (`command not found`)
- `pg_isready` -> failed (`/tmp:5432 - no response`)

Important implication: I verified build/test/generation paths, but I did **not** verify a successful app boot against a live PostgreSQL instance in this environment.

### Verified commands

These commands were run successfully from the repo root unless noted otherwise:

| Command | Result | Notes |
| --- | --- | --- |
| `go test ./...` | Passed | Coverage exists only in `internal/config` and `internal/middleware` tests. |
| `go build -o /tmp/go-web-server-review ./cmd/web` | Passed | Confirms plain Go build works. |
| `npm ci` | Passed | `package-lock.json` is now present locally and should be tracked to keep frontend installs reproducible. |
| `npm run build-css` | Passed | Emits a Browserslist warning about outdated `caniuse-lite`. |
| `go run github.com/magefile/mage -l` | Passed | Listed Mage targets correctly. |
| `go run github.com/magefile/mage generate` | Passed | Clean with pinned `templ`/`sqlc` versions installed locally. |
| `go run github.com/magefile/mage vet` | Passed | Wrapper around `go vet ./...`. |
| `go run github.com/magefile/mage lint` | Passed | The prior `goconst` failure in `internal/middleware/csrf_test.go` is fixed. |
| `go run github.com/magefile/mage build` | Passed | Runs generation + CSS build + Go build successfully after the lint/tooling fixes. |

### Verified commands that currently fail

No build/test/lint command failures were reproduced in this pass.

Still unverified because of missing local dependencies:

- Runtime boot against PostgreSQL
- Atlas-backed migration commands
- End-to-end auth and `/users` CRUD in a live browser

### Unverified but likely commands

These are present in repo code/docs but were **not** verified in this review:

| Command | Why unverified | What it likely does |
| --- | --- | --- |
| `mage setup` | Not run directly in this pass because equivalent tool installs were run manually | Installs pinned `templ`/`sqlc`, plus `govulncheck`, `air`, `goimports`, then downloads Go modules |
| `mage dev` | Requires app runtime env and reachable Postgres | Runs Air using `.air.toml` |
| `mage run` | Requires app runtime env and reachable Postgres | Builds and runs `bin/server` once |
| `mage migrate` | `atlas` missing locally and no verified DB | Runs `atlas migrate apply --env dev` |
| `mage migrateStatus` | `atlas` missing locally and no verified DB | Runs `atlas migrate status --env dev` |
| `mage vulnCheck` | Not run | Runs `govulncheck ./...` |
| `mage quality` | Not run end-to-end in this pass | Runs vet + lint + vulncheck |
| `mage ci` | Not run end-to-end in this pass | Runs generate + fmt + vet + lint + build + build-info |
| `docker build .` | Not run | Likely works only if committed generated assets are already current |
| `goreleaser build --snapshot --clean` / `mage snapshot` | Not run | Release packaging path defined in `.goreleaser.yaml` |
| `./scripts/deploy.sh` | Not run | Ubuntu/systemd deployment helper for `/opt/gowebserver` |

### Practical local bring-up sequence

This is the safest intended local flow, based on code + docs, but runtime boot was not verified here because PostgreSQL was unavailable:

1. Ensure PostgreSQL is running and reachable.
2. Create `.env` from `.env.example`.
3. Set at minimum:
   - `DATABASE_URL=postgres://...`
   - `AUTH_COOKIE_SECURE=false` for plain localhost HTTP
4. Install tooling:
   - `mage setup`
5. Generate derived files:
   - `mage generate`
6. Start the app:
   - `mage dev` for hot reload
   - or `mage run` for a one-shot run

## 3. Source-of-Truth Notes

### Treat these as authoritative first

- Runtime wiring:
  - `cmd/web/main.go`
- Config schema and precedence:
  - `internal/config/config.go`
- Route surface:
  - `internal/handler/routes.go`
- Database schema for SQLC + Atlas:
  - `internal/store/schema.sql`
- Canonical Atlas migration history:
  - top-level `migrations/`
- Query definitions:
  - `internal/store/queries.sql`
- Templ source:
  - `internal/view/**/*.templ`
- Static asset source:
  - `input.css`
  - `tailwind.config.js`
  - `package.json`
- Embedded static asset serving:
  - `internal/ui/embed.go`
- Dev/build orchestration:
  - `magefile.go`
  - `.air.toml`
  - `.golangci.yml`
  - `atlas.hcl`
- CI truth:
  - `.github/workflows/ci.yml`

### Generated files that are checked in

These are committed artifacts, not hand-edited source:

- `internal/view/*_templ.go`
- `internal/store/db.go`
- `internal/store/models.go`
- `internal/store/queries.sql.go`
- `internal/ui/static/css/styles.css`
- `internal/ui/static/css/pico.min.css`

Important: some build/release paths rely on these files already being current.

### Conflicts, drift, and ambiguous areas

1. Duplicate migration directories are not just leftover; they have drifted.
   - Top-level `migrations/` contains Atlas-style files and `atlas.sum`.
   - `internal/store/migrations/` contains older Goose-style `Up`/`Down` files.
   - The files are not identical.
   - Treat top-level `migrations/` as canonical for Atlas.
   - Treat `internal/store/migrations/` as legacy/confusing until it is removed or clearly quarantined.

2. There are effectively three schema definitions to keep aligned.
   - `internal/store/schema.sql`
   - top-level `migrations/`
   - `internal/store/store.go` -> `InitSchema()`
   This is a real maintenance risk.

3. `.env.example` was realigned in this pass, but compatibility fields can still confuse readers.
   - `SECURITY_TRUSTED_PROXIES` now defaults to empty, which matches the docs/runtime intent.
   - `FEATURES_ENABLE_METRICS` now defaults to `false`, which matches the live app.
   - JWT-related env vars still exist only because config still exposes those fields.

4. Config contains dead or not-yet-live fields.
   - `auth.jwt_secret`
   - `auth.token_duration`
   - `auth.refresh_duration`
   - `features.enable_metrics`
   - `features.enable_pprof`
   These exist in config, but the live app uses session auth and does not expose metrics or pprof endpoints.

5. Generator version drift was reduced, but not completely eliminated.
   - `mage setup` now pins `templ v0.3.1001`.
   - `mage setup` now pins `sqlc v1.30.0`.
   - `cmd/web/main.go` `go:generate` directives now use those same explicit versions.
   - `mage lint` now installs the same `golangci-lint v2.11.3` version used in CI.
   - `mage generate` still uses whatever `templ`/`sqlc` binaries are currently on `PATH`, so rerun `mage setup` if generation output looks unexpectedly noisy.

6. Release/build paths are inconsistent about generation.
   - `mage build` runs SQLC + Templ + CSS generation
   - `Dockerfile` does a plain `go build` and assumes generated assets are already current
   - `.goreleaser.yaml` runs `go generate ./...`, which does not run the Node/Tailwind CSS build
   If `input.css` changes and generated CSS is not committed, Docker/GoReleaser may package stale assets.

7. Frontend reproducibility is improved, but only once the lockfile is committed.
   - `.gitignore` no longer ignores `package-lock.json`.
   - `mage` now prefers `npm ci` over `npm install` when a lockfile is present and `node_modules` is missing.

8. Obvious tracked artifacts were removed in this pass.
   - The root `web` Mach-O binary was deleted.
   - `output/playwright/*.png` screenshots were deleted.
   - `.gitignore` now explicitly ignores those generated artifact paths.

## 4. Current Gaps and Known Issues

### Verified issues

- Local Postgres was unavailable in this review environment.
  - Runtime bring-up, login flow, migrations, and `/health` against a real DB were not verified
- Atlas CLI was not installed locally.
  - Migration commands were not verified
- CSS builds emit a Browserslist maintenance warning.
  - `caniuse-lite` is outdated, but this did not block the build

### Obvious codebase gaps

- No tests for:
  - handler flows
  - auth session lifecycle
  - SQLC/store behavior against a real database
  - end-to-end user CRUD
- No authorization model beyond authenticated vs unauthenticated
- In-memory rate limiting only; no shared/distributed store
- CORS defaults are permissive unless tightened in config
- No metrics, pprof, or observability endpoints despite config flags
- No audit trail for auth/user-management actions
- No password reset or email-based account recovery

### Risk areas

- Schema drift between `schema.sql`, `migrations/`, and `InitSchema()`
- Legacy duplicate migration directory may mislead future edits
- Docker/GoReleaser rely on committed generated assets being current
- `mage ci` includes `Fmt`, which mutates the working tree; that is unusual for CI-style validation

## 5. Phase Dashboard

### Phase 0 — starter app foundation

**Status:** done / checked

Checklist:

- [x] server-rendered Go app exists with Echo, Templ, HTMX, PostgreSQL, SQLC, and Mage
- [x] auth routes and `/users` CRUD exist
- [x] CSS build path works
- [x] core build/test/vet/lint paths are green in this environment
- [x] BUILD records what was and was not actually verified

### Phase 1 — source-of-truth and schema consolidation

**Status:** in progress

Checklist:

- [ ] resolve whether top-level `migrations/` is the only canonical migration history
- [ ] decide whether `internal/store/migrations/` is deleted, archived, or explicitly legacy
- [ ] decide whether startup keeps `InitSchema()` bootstrap behavior or moves to Atlas-only expectations
- [ ] align Mage, Docker, and GoReleaser around one generation and asset-build story

Exit criteria:

- one migration story is canonical
- one schema source of truth is canonical
- generation/build/release paths no longer depend on undocumented assumptions

### Phase 2 — DB-backed runtime verification

**Status:** planned

Checklist:

- [ ] boot the app successfully against a live PostgreSQL instance
- [ ] verify `/health` against a live DB
- [ ] verify register/login/logout end to end
- [ ] verify protected `/users` CRUD end to end
- [ ] add at least one real DB-backed integration path to the automated suite

### Phase 3 — product hardening

**Status:** planned

Checklist:

- [ ] either wire metrics/pprof for real or remove misleading config claims
- [ ] remove dead JWT or unused config surfaces if they are not part of the product
- [ ] keep deployment helpers honest about generated assets and migration expectations
- [ ] preserve one-binary boring-default operator ergonomics

### Phase 4 — tech stack alignment

**Status:** planned

The Go full-stack and backend tech stacks define the canonical choices for this workspace. This phase closes the gaps between what those stacks specify and what the repo currently implements.

**HTTP layer**

The tech stack defaults to `net/http` with explicit middleware and says to reach for `chi` only when route grouping and middleware composition become materially easier. It explicitly lists "reflection-driven web frameworks" under "Avoid By Default". The repo currently uses Echo, which is a reflection-heavy framework.

- [ ] Replace Echo with `net/http` and `chi`; wire middleware explicitly as the tech stack describes
- [ ] Migrate all route definitions in `internal/handler/routes.go` to `chi` router groups
- [ ] Replace Echo-specific middleware (`echomiddleware.*`) with stdlib or `chi`-compatible equivalents
- [ ] Remove `github.com/labstack/echo/v4` and `github.com/labstack/gommon` from `go.mod`

**Migrations**

The tech stack specifies `goose` as the default migration tool. The repo uses Atlas CLI, which was not installed in the review environment and adds external dependency friction. There is already a legacy `internal/store/migrations/` directory with goose-style files.

- [ ] Replace Atlas with `goose` as the canonical migration tool
- [ ] Add `github.com/pressly/goose/v3` to `go.mod`
- [ ] Convert top-level `migrations/` Atlas files to goose-compatible `.sql` files with `-- +goose Up` / `-- +goose Down` markers
- [ ] Remove `atlas.hcl` and the Atlas CLI dependency from `magefile.go` and docs
- [ ] Delete `internal/store/migrations/` after verifying the goose migration set is complete
- [ ] Update `mage migrate` and `mage migrateStatus` targets to use `goose` instead of `atlas`
- [ ] Update `cmd/web/main.go` and `internal/store/store.go` to run goose migrations at startup or as an explicit CLI flag

**Observability**

The tech stack requires a `/metrics` Prometheus endpoint and `pprof` on an admin-only path as part of the observability baseline. Both are stubbed in config but never wired.

- [ ] Add `github.com/prometheus/client_golang` to `go.mod`
- [ ] Wire a `/metrics` handler when `features.enable_metrics` is true, or remove the config field and make the endpoint always-on
- [ ] Wire `net/http/pprof` handlers on an admin-only path (e.g. `/debug/pprof/`) when `features.enable_pprof` is true, or always-on behind a localhost check
- [ ] Add structured log fields at startup confirming which observability surfaces are active

**CI security scan**

The tech stack requires `govulncheck` in CI. It exists in Mage as `mage vulnCheck` but the CI workflow does not call it.

- [ ] Add a `govulncheck ./...` step to `.github/workflows/ci.yml`

**CSS dependencies**

The tech stack defaults to hand-written CSS or Tailwind CSS v4 via the standalone CLI. The repo adds DaisyUI and Pico CSS on top of Tailwind.

- [ ] Decide whether DaisyUI and Pico CSS are intentional product choices or incidental pull-ins
- [ ] If incidental, remove them from `package.json` and `tailwind.config.js` and replace any component usage with plain Tailwind utility classes or hand-written CSS

Exit criteria:

- HTTP layer is `net/http` plus `chi` with no Echo dependency in `go.mod`
- `goose` is the only migration tool; Atlas and the legacy migration directory are gone
- `/metrics` and `/debug/pprof` are wired and documented, or the config fields that claim to enable them are removed
- `govulncheck` runs in CI
- CSS dependency list matches what the tech stack specifies

## 6. Next-Pass Priorities

### Highest impact, dependency-ordered

1. Resolve source-of-truth ambiguity around database schema and migrations.
   - Decide whether `internal/store/migrations/` should be deleted, archived, or clearly marked as legacy-only.
   - Decide whether startup should keep `InitSchema()` bootstrap behavior or move to Atlas-only expectations.
   - Update docs and build paths to reflect the chosen approach.

2. Stabilize the toolchain and generated-file workflow.
   - Decide whether `mage generate` itself should enforce pinned CLI versions, not just `mage setup`
   - Decide whether Docker/GoReleaser should explicitly run CSS generation

3. Add at least one real DB-backed integration path.
   - Auth registration/login
   - Protected `/users` CRUD
   - Health endpoint against a live DB

### Quick wins

Completed in this pass:

- Fixed the `goconst` lint failure in `internal/middleware/csrf_test.go`
- Updated `.env.example` to stop implying trusted proxies and metrics are enabled by default
- Added a README note about tracking `package-lock.json`
- Made `templ`, `sqlc`, and local `golangci-lint` versions explicit in build tooling
- Removed tracked binary/screenshot artifacts and ignored those paths going forward

### Deeper refactors

- Collapse schema ownership to one migration story plus one canonical schema definition
- Remove dead JWT/metrics/pprof config if the app will stay session-only and minimal
- Align Docker, Mage, and GoReleaser so they all build from the same generation assumptions

## 7. Next-Agent Checklist

Follow this order to minimize confusion:

1. Read this file first.
2. Read the runtime and routing entry points:
   - `cmd/web/main.go`
   - `internal/config/config.go`
   - `internal/handler/routes.go`
3. Read the auth and user flows:
   - `internal/handler/auth.go`
   - `internal/handler/user.go`
   - `internal/middleware/auth.go`
   - `internal/middleware/csrf.go`
4. Read the data ownership files:
   - `internal/store/schema.sql`
   - top-level `migrations/`
   - `internal/store/store.go`
   - `internal/store/queries.sql`
5. Check local tool versions before generating anything:
   - `go version`
   - `templ version`
   - `sqlc version`
   - `node -v`
   - `npm -v`
   - `atlas version`
6. Create local runtime config:
   - `cp .env.example .env`
   - set a real `DATABASE_URL`
   - keep `AUTH_COOKIE_SECURE=false` for local HTTP
7. Confirm PostgreSQL is reachable before trying to boot:
   - `pg_isready`
8. Run the lowest-risk validation commands first:
   - `go test ./...`
   - `go run github.com/magefile/mage vet`
   - `npm run build-css`
9. Run `go run github.com/magefile/mage lint`.
10. If you change `internal/view/**/*.templ`, `internal/store/queries.sql`, `internal/store/schema.sql`, or `input.css`, run:
    - `go run github.com/magefile/mage generate`
11. Inspect generated diffs carefully.
    - If output looks unexpectedly noisy, rerun `mage setup` to restore the pinned tool versions first
12. Once the DB is live, verify the app bring-up path:
    - `go run github.com/magefile/mage dev`
    - or `go run github.com/magefile/mage run`
13. Manually verify:
    - `/health`
    - register a user
    - login/logout
    - `/users` CRUD
14. If you touch schema or migrations, update this file before finishing.

## 8. Short Truth Summary

If you need the shortest honest summary before working:

- The repo builds, tests, vets, and generates successfully in this environment.
- Local lint is green again after fixing the small test-file constant issue.
- Runtime and migration bring-up were not verified because local PostgreSQL was unavailable and Atlas was not installed.
- The biggest repo hygiene problem is source-of-truth drift: schema/bootstrap/migrations/generated artifacts are all close enough to work, but not unified enough to be low-risk.
