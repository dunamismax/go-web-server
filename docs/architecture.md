# Architecture Overview

## Current Shape

The app now ships as one Go service plus one committed frontend build:

```text
browser
  -> embedded Astro pages served by Echo
  -> same-origin /_backend/* bridge in shipped builds
  -> explicit JSON contracts under /api/*
  -> session cookie + CSRF middleware in Go
  -> PostgreSQL via pgx/sqlc
```

The shipped browser frontend lives in `web/` and is embedded from `web/dist`. Go remains responsible for routing, auth, CSRF, persistence, and operational endpoints.

## Runtime Boundaries

### Browser pages

- Public pages: `/`, `/auth/login`, `/auth/register`, `/auth/logout`
- Protected pages: `/profile`, `/users`
- Static frontend assets: `/_astro/*`

These pages are served from the embedded Astro build.

### Browser form fallback submits

These remain as plain redirect-oriented browser endpoints for auth flows:

- `POST /auth/login`
- `POST /auth/register`
- `POST /auth/logout`

### JSON API contracts

- Auth: `/api/auth/state`, `/api/auth/login`, `/api/auth/register`, `/api/auth/logout`
- Users: `/api/users`, `/api/users/count`, `/api/users/:id`, plus create, update, deactivate, and delete

### Utility endpoints

- `GET /demo`
- `GET /health`

These now return JSON only.

## Repo Layout

| Path | Purpose |
| --- | --- |
| [`cmd/web/`](../cmd/web/) | App entrypoint and startup wiring |
| [`internal/handler/`](../internal/handler/) | Echo handlers, API contracts, frontend serving, and route registration |
| [`internal/middleware/`](../internal/middleware/) | Sessions, CSRF, auth, timeouts, errors, security headers |
| [`internal/store/`](../internal/store/) | Database access, SQLC output, schema bootstrap, and migrations wiring |
| [`internal/ui/`](../internal/ui/) | Embedded backend-owned static assets such as the favicon |
| [`web/`](../web/) | Astro + Vue + TypeScript + Bun frontend workspace |
| [`web/dist/`](../web/dist/) | Built frontend committed for embedding into the Go binary |
| [`migrations/`](../migrations/) | Canonical Atlas migrations |
| [`scripts/`](../scripts/) | Browser smoke and runtime smoke validation |
| [`magefile.go`](../magefile.go) | Developer and CI task runner |

## Important Constraints

- Same-origin session cookies are the auth model.
- CSRF stays enforced on state-changing requests.
- PostgreSQL remains canonical.
- The browser frontend talks to boring JSON contracts instead of fragment-shaped UI responses.
- Release shape stays boring: one Go service, one database, one embedded frontend build.
