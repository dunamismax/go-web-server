# Frontend Migration Inventory

This document started as the route-by-route migration inventory for the Templ and HTMX browser stack. The migration is now complete. It remains as a historical map of what changed and what survived the cleanup.

## Final Outcome

- Embedded Astro pages now own the shipped browser surface for `/`, `/auth/login`, `/auth/register`, `/auth/logout`, `/profile`, and `/users`.
- Managed-user browser mutations no longer use legacy fragment endpoints.
- The Go backend now exposes the durable browser integration surface through `/api/auth/*` and `/api/users/*`.
- The only browser fallback submit endpoints left outside `/api/*` are the redirect-oriented auth form posts.
- Templ, HTMX, and the repo-root legacy CSS pipeline are gone.

## Final Route Inventory

### Browser pages

| Route | Auth | Shape | Notes |
| --- | --- | --- | --- |
| `GET /` | public | embedded Astro HTML | Shipped home page |
| `GET /auth/login` | public | embedded Astro HTML | Sign-in page |
| `GET /auth/register` | public | embedded Astro HTML | Registration page |
| `GET /auth/logout` | public | embedded Astro HTML | Logout flow page |
| `GET /profile` | protected | embedded Astro HTML | Protected profile page |
| `GET /users` | protected | embedded Astro HTML | Protected CRUD page |
| `GET /_astro/*` | public | static asset files | Embedded frontend assets |

### Browser fallback submits

| Route | Auth | Shape | Notes |
| --- | --- | --- | --- |
| `POST /auth/login` | public | redirect | Browser fallback submit |
| `POST /auth/register` | public | redirect | Browser fallback submit |
| `POST /auth/logout` | protected session in practice | redirect | Browser fallback submit |

### Utility endpoints

| Route | Auth | Shape | Notes |
| --- | --- | --- | --- |
| `GET /demo` | public | JSON | Backend connectivity check |
| `GET /health` | public | JSON | Operational health payload |

### JSON contracts

| Route family | Auth | Shape | Notes |
| --- | --- | --- | --- |
| `/api/auth/*` | mixed | JSON | Auth bootstrap and session mutations |
| `/api/users/*` | protected | JSON | Managed-user list, count, fetch, create, update, deactivate, delete |

## Retired Surfaces

The following legacy browser surfaces were removed:

- Templ-rendered browser pages and layouts
- HTMX fragment rendering and HX-specific mutation responses
- `/users/list`
- `/users/count` HTML fragment behavior
- `/users/form`
- `/users/:id/edit`
- browser mutation routes for `POST /users`, `PUT /users/:id`, `PATCH /users/:id/deactivate`, and `DELETE /users/:id`
- the repo-root CSS asset pipeline and checked-in CSS/HTMX blobs under `internal/ui/static/`
