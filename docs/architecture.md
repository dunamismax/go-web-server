# Architecture

## High-Level Shape

```text
Browser
  -> today: Echo routes + middleware + Templ/HTMX rendering
  -> staged migration lane: Astro + Vue workspace in web/
  -> handlers
  -> store (SQLC)
  -> PostgreSQL
```

The repo is a small monolith. There is one binary, one Postgres database, and one main demo domain model: users. Phase 1 of the frontend migration adds a staged Astro + Vue + Bun workspace under `web/`, but the shipped browser path is still the legacy Templ + HTMX app.

## Request Flow

1. Echo receives the request.
2. Middleware applies recovery, security headers, request normalization, CSRF, request IDs, logging, rate limiting, and timeout handling.
3. Session middleware loads the current user, if any.
4. Handlers validate input, call the store, and render Templ views or JSON.

## Route Split

- Public: home, demo, health, login, registration, static assets
- Protected: profile, user CRUD, user count API

## Configuration Flow

Configuration is loaded by [`internal/config/config.go`](../internal/config/config.go) in this order:

1. Built-in defaults
2. `.env`
3. `config.yaml` or `config/config.yaml`
4. Environment variables

Environment variables win last.

## Repo Layout

| Path | Purpose |
| --- | --- |
| [`cmd/web/main.go`](../cmd/web/main.go) | App bootstrap, middleware stack, config wiring, and graceful shutdown |
| [`internal/handler/`](../internal/handler/) | Route handlers and response helpers |
| [`internal/middleware/`](../internal/middleware/) | Auth, CSRF, error, validation, and normalization middleware |
| [`internal/store/`](../internal/store/) | Database pool setup, SQLC queries, schema, and store methods |
| [`internal/view/`](../internal/view/) | Templ components and layouts for the current shipped browser path |
| [`internal/ui/static/`](../internal/ui/static/) | Embedded CSS, JS, images, and favicon for the legacy frontend |
| [`web/`](../web/) | Staged Astro + Vue + Bun frontend workspace for the migration lane |
| [`migrations/`](../migrations/) | Atlas-managed SQL migrations |
| [`docs/`](./) | User-facing repo documentation |

## Schema and Migration Sources

- [`internal/store/schema.sql`](../internal/store/schema.sql) is the canonical schema definition used by Atlas.
- [`internal/store/store.go`](../internal/store/store.go) contains a matching bootstrap path for local startup.
- Top-level [`migrations/`](../migrations/) is the canonical migration directory.

The duplicate [`internal/store/migrations/`](../internal/store/migrations/) directory still exists, but the app and docs should treat top-level [`migrations/`](../migrations/) as the source of truth.
