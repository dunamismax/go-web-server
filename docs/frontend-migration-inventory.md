# Frontend migration inventory

This file is the current migration surface map for the Astro + Vue frontend work.

It exists so Phase 1 can add the `web/` workspace without pretending the legacy browser path is already migrated.

## Status

- **Migration lane:** web-only
- **Current primary browser path:** Templ + HTMX
- **New staged workspace:** `web/`
- **Local frontend proxy prefix:** `/_backend/*`
- **Phase 2 result:** explicit JSON contracts now exist under `/api/*` for auth state and managed-user CRUD
- **Next required phase after this doc:** port the app shell and auth/user flows onto Astro + Vue

## Route inventory

| Route | Auth | Current mode | Notes for migration |
| --- | --- | --- | --- |
| `GET /` | public | page or HTMX fragment | Home page must move to Astro page shell later |
| `GET /demo` | public | JSON or HTMX fragment | Good temporary connectivity check for the new frontend |
| `GET /health` | public | JSON or HTMX fragment | Current Phase 1 shell reads this through the local proxy |
| `GET /auth/login` | public | page or HTMX fragment | Phase 2 login contract exists under `/api/auth/login`; Astro page parity is still pending |
| `GET /auth/register` | public | page or HTMX fragment | Phase 2 registration contract exists under `/api/auth/register`; Astro page parity is still pending |
| `POST /auth/login` | public | redirect or HTMX redirect payload | Session + CSRF behavior must stay intact |
| `POST /auth/register` | public | redirect or HTMX redirect payload | Session + CSRF behavior must stay intact |
| `POST /auth/logout` | protected session in practice | redirect or HTMX redirect payload | Same-origin cookie handling is the important constraint |
| `GET /profile` | protected | page or HTMX fragment | Astro parity is still pending, but auth/session bootstrap contracts now exist |
| `GET /users` | protected | page or HTMX fragment | Later Astro page shell target |
| `GET /users/list` | protected | HTMX fragment | Legacy fragment surface, not a durable frontend contract |
| `GET /users/count` | protected | HTMX fragment | Legacy count fragment kept for the current `/users` page |
| `GET /users/form` | protected | HTMX fragment | Legacy fragment surface, not a durable frontend contract |
| `GET /users/:id/edit` | protected | HTMX fragment | Legacy fragment surface, not a durable frontend contract |
| `POST /users` | protected | HTMX fragment | Legacy submit kept in place until the Astro port is done |
| `PUT /users/:id` | protected | HTMX fragment | Legacy submit kept in place until the Astro port is done |
| `PATCH /users/:id/deactivate` | protected | HTMX fragment | Legacy submit kept in place until the Astro port is done |
| `DELETE /users/:id` | protected | empty `200 OK` | Legacy delete submit kept in place until the Astro port is done |
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

## Phase 1 development topology

For now, local development stays split across two processes:

1. `mage dev` runs the Go app on `http://127.0.0.1:8080`
2. `mage frontendDev` runs Astro on `http://127.0.0.1:4321`
3. Astro proxies `/_backend/*` to the Go app so browser requests stay same-origin from the frontend's point of view

That keeps session-cookie and CSRF work on a sane path without claiming the real route migration is done.
