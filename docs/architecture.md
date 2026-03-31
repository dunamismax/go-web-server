# Architecture

## High-Level Shape

```text
Browser
  -> Echo routes + middleware + embedded Astro dist for primary GET pages
  -> temporary legacy mutation submits still return redirect or HTMX-oriented responses
  -> handlers
  -> store (SQLC)
  -> PostgreSQL
```

The repo is a small monolith. There is one binary, one Postgres database, and one main demo domain model: users. The Astro + Vue + Bun workspace under `web/` now owns the shipped GET browser path for home, auth, profile, and users through committed `web/dist` output that is embedded into the Go binary.

Phase 2 added a parallel JSON API surface under `/api/*` for auth state and user CRUD contracts. Those contracts now back both Astro development and the shipped embedded frontend. The remaining legacy browser surface is the old auth and `/users` mutation submit path plus the Templ rendering code that still supports it during the retirement phase.

## Request Flow

1. Echo receives the request.
2. Middleware applies recovery, security headers, request normalization, CSRF, request IDs, logging, rate limiting, and timeout handling.
3. Session middleware loads the current user, if any.
4. Handlers validate input, call the store, and return embedded Astro HTML, JSON, or temporary legacy Templ responses depending on the route surface.

## Route Split

- Public embedded Astro pages: `/`, `/auth/login`, `/auth/register`, `/auth/logout`, and `/_astro/*` assets
- Public backend utility routes: `/demo`, `/health`, and `/static/*`
- Public JSON API: auth state, login, registration, logout
- Protected embedded Astro pages: `/profile` and `/users`
- Temporary legacy mutation routes: `POST /auth/*`, `POST /users`, `PUT /users/:id`, `PATCH /users/:id/deactivate`, and `DELETE /users/:id`
- Protected JSON API: `/api/users`, `/api/users/count`, `/api/users/:id`, create, update, deactivate, and delete

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
| [`internal/view/`](../internal/view/) | Temporary Templ components and layouts that still back the legacy submit path |
| [`internal/ui/static/`](../internal/ui/static/) | Embedded CSS, JS, images, and favicon for the legacy frontend |
| [`web/`](../web/) | Astro + Vue + Bun frontend workspace plus the committed shipped `dist/` output |
| [`migrations/`](../migrations/) | Atlas-managed SQL migrations |
| [`docs/`](./) | User-facing repo documentation |

## Schema and Migration Sources

- [`internal/store/schema.sql`](../internal/store/schema.sql) is the canonical schema definition used by Atlas.
- [`internal/store/store.go`](../internal/store/store.go) contains a matching bootstrap path for local startup.
- Top-level [`migrations/`](../migrations/) is the canonical migration directory.

The duplicate [`internal/store/migrations/`](../internal/store/migrations/) directory still exists, but the app and docs should treat top-level [`migrations/`](../migrations/) as the source of truth.
