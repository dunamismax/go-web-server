# Frontend migration inventory

This file is the current migration surface map for the Astro + Vue frontend work.

It exists so the migration can move route by route without pretending the legacy browser path is already gone.

## Status

- **Migration lane:** web-only
- **Current primary browser path:** Templ + HTMX
- **New staged workspace:** `web/`
- **Local frontend proxy prefix:** `/_backend/*`
- **Phase 2 result:** explicit JSON contracts now exist under `/api/*` for auth state and managed-user CRUD
- **Phase 3 result:** the staged Astro workspace now covers home, login, registration, logout, and profile flows
- **Phase 4 result:** the staged Astro workspace now also covers the protected `/users` CRUD surface through the JSON contracts
- **Next required phase after this doc:** retire the legacy browser stack only after the Astro CRUD path has enough verification

## Route inventory

| Route | Auth | Current mode | Notes for migration |
| --- | --- | --- | --- |
| `GET /` | public | page or HTMX fragment | Astro parity now exists in `web/`; Go still ships the legacy page for the primary browser path |
| `GET /demo` | public | JSON or HTMX fragment | Still useful as a backend connectivity check while the legacy browser path still exists |
| `GET /health` | public | JSON or HTMX fragment | The Astro home page reads this through the local proxy |
| `GET /auth/login` | public | page or HTMX fragment | Astro parity now exists in `web/` against `/api/auth/login` |
| `GET /auth/register` | public | page or HTMX fragment | Astro parity now exists in `web/` against `/api/auth/register` |
| `POST /auth/login` | public | redirect or HTMX redirect payload | Legacy submit stays until the primary browser path flips |
| `POST /auth/register` | public | redirect or HTMX redirect payload | Legacy submit stays until the primary browser path flips |
| `POST /auth/logout` | protected session in practice | redirect or HTMX redirect payload | Legacy submit stays until the primary browser path flips |
| `GET /profile` | protected | page or HTMX fragment | Astro parity now exists in `web/` with client-side unauthenticated redirect handling |
| `GET /users` | protected | page or HTMX fragment | Astro parity now exists in `web/` against the `/api/users/*` contracts |
| `GET /users/list` | protected | HTMX fragment | Legacy fragment surface, no longer needed by the staged Astro CRUD path |
| `GET /users/count` | protected | HTMX fragment | Legacy count fragment kept for the old screen; Astro now uses `/api/users/count` |
| `GET /users/form` | protected | HTMX fragment | Legacy fragment surface, no longer needed by the staged Astro CRUD path |
| `GET /users/:id/edit` | protected | HTMX fragment | Legacy fragment surface, no longer needed by the staged Astro CRUD path |
| `POST /users` | protected | HTMX fragment | Legacy submit kept in place until the legacy browser path is retired |
| `PUT /users/:id` | protected | HTMX fragment | Legacy submit kept in place until the legacy browser path is retired |
| `PATCH /users/:id/deactivate` | protected | HTMX fragment | Legacy submit kept in place until the legacy browser path is retired |
| `DELETE /users/:id` | protected | empty `200 OK` | Legacy delete submit kept in place until the legacy browser path is retired |
| `GET /api/auth/state` | public | JSON | Frontend bootstrap contract for session and CSRF state |
| `POST /api/auth/login` | public | JSON | Explicit login contract for later Astro auth flows |
| `POST /api/auth/register` | public | JSON | Explicit registration contract for later Astro auth flows |
| `POST /api/auth/logout` | public | JSON | Explicit logout contract for later Astro auth flows |
| `GET /api/users` | protected | JSON | Active managed-user list contract |
| `GET /api/users/count` | protected | JSON | Active managed-user count contract |
| `GET /api/users/:id` | protected | JSON | Single-user fetch contract for edit flows |
| `POST /api/users` | protected | JSON | Managed-user create contract |
| `PUT /api/users/:id` | protected | JSON | Managed-user update contract |
| `PATCH /api/users/:id/deactivate` | protected | JSON | Managed-user deactivate contract |
| `DELETE /api/users/:id` | protected | JSON | Managed-user delete contract |
| `GET /static/*` | public | embedded assets | Legacy assets remain backend-owned until retirement phase |

## Current development topology

For now, local development stays split across two processes:

1. `mage dev` runs the Go app on `http://127.0.0.1:8080`
2. `mage frontendDev` runs Astro on `http://127.0.0.1:4321`
3. Astro proxies `/_backend/*` to the Go app so browser requests stay same-origin from the frontend's point of view

That keeps session-cookie and CSRF work on a sane path without claiming the real route migration is done.
