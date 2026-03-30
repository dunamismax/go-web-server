# Frontend migration inventory

This file is the current migration surface map for the Astro + Vue frontend work.

It exists so Phase 1 can add the `web/` workspace without pretending the legacy browser path is already migrated.

## Status

- **Migration lane:** web-only
- **Current primary browser path:** Templ + HTMX
- **New staged workspace:** `web/`
- **Local frontend proxy prefix:** `/_backend/*`
- **Next required phase after this doc:** backend contract cleanup

## Route inventory

| Route | Auth | Current mode | Notes for migration |
| --- | --- | --- | --- |
| `GET /` | public | page or HTMX fragment | Home page must move to Astro page shell later |
| `GET /demo` | public | JSON or HTMX fragment | Good temporary connectivity check for the new frontend |
| `GET /health` | public | JSON or HTMX fragment | Current Phase 1 shell reads this through the local proxy |
| `GET /auth/login` | public | page or HTMX fragment | Needs explicit frontend contract in Phase 2 before Astro parity work |
| `GET /auth/register` | public | page or HTMX fragment | Same constraint as login |
| `POST /auth/login` | public | redirect or HTMX redirect payload | Session + CSRF behavior must stay intact |
| `POST /auth/register` | public | redirect or HTMX redirect payload | Session + CSRF behavior must stay intact |
| `POST /auth/logout` | protected session in practice | redirect or HTMX redirect payload | Same-origin cookie handling is the important constraint |
| `GET /profile` | protected | page or HTMX fragment | Needs Phase 2 contract cleanup before Astro parity |
| `GET /users` | protected | page or HTMX fragment | Later Astro page shell target |
| `GET /users/list` | protected | HTMX fragment | Legacy fragment surface, not a durable frontend contract |
| `GET /users/form` | protected | HTMX fragment | Legacy fragment surface, not a durable frontend contract |
| `GET /users/:id/edit` | protected | HTMX fragment | Legacy fragment surface, not a durable frontend contract |
| `POST /users` | protected | HTMX fragment | Must become an explicit contract in Phase 2 |
| `PUT /users/:id` | protected | HTMX fragment | Must become an explicit contract in Phase 2 |
| `PATCH /users/:id/deactivate` | protected | HTMX fragment | Must become an explicit contract in Phase 2 |
| `DELETE /users/:id` | protected | empty `200 OK` | Durable enough to keep close to this shape if documented |
| `GET /api/users/count` | protected | HTMX fragment despite `/api` prefix | Rename or normalize in Phase 2 so the path reflects reality |
| `GET /static/*` | public | embedded assets | Legacy assets remain backend-owned until retirement phase |

## Phase 1 development topology

For now, local development stays split across two processes:

1. `mage dev` runs the Go app on `http://127.0.0.1:8080`
2. `mage frontendDev` runs Astro on `http://127.0.0.1:4321`
3. Astro proxies `/_backend/*` to the Go app so browser requests stay same-origin from the frontend's point of view

That keeps session-cookie and CSRF work on a sane path without claiming the real route migration is done.
