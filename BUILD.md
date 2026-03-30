# BUILD.md

## Decision

**Target frontend: web-only.**

Keep the backend in Go. Replace the current browser layer with **TypeScript + Bun + Astro + Vue**. Do **not** add an OpenTUI frontend here.

Why:

- this repo is a browser app starter, not an operator console
- the existing product surface is session-authenticated web CRUD, not terminal workbench UX
- a TUI would add maintenance cost without improving the reference use case

## Repo Role

This repo should become the **Go-backed reference starter for Stephen's default web frontend lane**:

- **backend:** Go + Echo + PostgreSQL + SQLC + Atlas
- **frontend:** Astro + Vue + TypeScript + Bun
- **auth:** same-origin session cookies + CSRF
- **deployment shape:** one Go service, one Postgres database, one built web asset set

This is a frontend migration plan, not a backend rewrite plan.

## Current State

Today the repo is:

- one Go binary with Echo handlers and middleware
- PostgreSQL-backed with `pgx/v5`, SQLC, Atlas migrations, and session auth
- server-rendered with **Templ + HTMX**
- using **Tailwind + DaisyUI + Pico CSS** built via **npm**
- embedding frontend assets from `internal/ui/static/`
- returning a mix of full HTML pages, HTMX fragments, redirects, and a little JSON

### Current user-facing surfaces

Public:

- `/`
- `/demo`
- `/health`
- `/auth/login`
- `/auth/register`
- `/auth/logout`
- `/static/*`

Authenticated:

- `/profile`
- `/users`
- `/users/list`
- `/users/form`
- `/users/:id/edit`
- `/users` `POST`
- `/users/:id` `PUT`
- `/users/:id/deactivate` `PATCH`
- `/users/:id` `DELETE`
- `/api/users/count`

### Current-state truths that must not get lost

- auth, profile, and user CRUD flows are tightly coupled to Templ and HTMX response shapes
- `/api/users/count` is not a real API contract today. It returns an HTML fragment
- session cookies are already same-origin friendly and protected by CSRF middleware
- generated backend artifacts and built frontend assets are currently checked in
- local development currently assumes Node/npm for CSS generation

## Target State

The target shape is:

- **Go stays the backend**
- **`web/` becomes the browser frontend workspace**
- **Astro owns pages, routing, and page composition**
- **Vue owns interactive widgets and CRUD interactions**
- **Bun owns package management, scripts, tests, and frontend builds**
- **PostgreSQL, Atlas, SQLC, and session auth stay in place**

Hard end state:

- no Templ in the primary frontend path
- no HTMX in the primary frontend path
- no npm/Node dependency for the active web build
- no fragment-shaped pseudo-API contract for frontend integration
- no checked-in generated CSS blobs as the long-term frontend delivery model

## Backend Notes

Keep Go.

That is the correct backend fit because this repo is still:

- a long-running web service
- a session-authenticated app starter
- a Postgres-backed backend with middleware, routing, and deployment concerns
- a repo where single-service operational simplicity matters

Do not turn this into a Python rewrite.

Do not rewrite Echo, SQLC, Atlas, session storage, or the database model unless the frontend migration exposes a concrete problem that forces it.

Backend responsibilities after migration:

- auth and session lifecycle
- CSRF issuance and validation
- persistence and migrations
- user CRUD and profile logic
- health and operational endpoints
- explicit frontend-facing JSON or same-origin HTTP contracts

## Data and Runtime Constraints

1. **Same-origin auth is non-negotiable**
   - production should keep one origin for app traffic
   - local development should use a proxy path, not cross-origin cookie hacks

2. **Session + CSRF stay real**
   - the Astro frontend must work with existing cookie auth
   - do not weaken CSRF protections to make frontend wiring easier

3. **PostgreSQL remains canonical**
   - no database change is part of this plan
   - Atlas migrations and SQLC stay as the data path

4. **Contract cleanup is required**
   - HTMX fragment routes are implementation details, not a durable app contract
   - the new frontend should talk to explicit JSON or other boring same-origin endpoints

5. **Release shape stays boring**
   - prefer one Go service plus built frontend assets
   - do not split this into extra services unless the repo later earns it

6. **BUILD.md is temporary**
   - once the migration is complete, fold durable truth into `README.md` and `docs/`, then delete this file

## Phase Plan

### Phase 0 - Freeze scope and map the UI surface

Goal: define exactly what the new frontend must cover.

Deliver:

- route inventory mapped to page, fragment, redirect, JSON, or mixed behavior
- frontend workspace location fixed as `web/`
- explicit decision recorded that this repo is **web-only**

Gate:

- no implementation yet
- future agents can point to a finite migration surface instead of reverse-engineering it mid-build

### Phase 1 - Add the new frontend workspace

Goal: introduce the Astro + Vue + Bun app without breaking the current one.

Deliver:

- `web/` with Astro, Vue, TypeScript, Bun, Biome, tests, and Playwright scaffolding
- dev commands that let the frontend talk to the Go app through a same-origin-friendly path
- root task wiring so Mage can call Bun-based frontend tasks

Gate:

- the current Templ frontend still works
- the new frontend can boot, render a shell, and reach the backend in development

### Phase 2 - Normalize backend contracts

Goal: stop treating HTMX fragments as the public integration surface.

Deliver:

- explicit contracts for auth state, user list, create, edit, deactivate, delete, and count
- `/api/*` naming that reflects reality
- CSRF expectations documented for browser requests
- endpoint docs future frontend agents can build against without reading Templ templates

Gate:

- new frontend work no longer depends on fragment HTML as the contract
- business logic mostly stays where it is

### Phase 3 - Port the app shell and auth flows

Goal: get the new stack handling real user entry paths.

Deliver:

- Astro layouts and page shells
- login, registration, logout, home, and profile flows
- success, error, redirect, and unauthenticated states handled honestly

Gate:

- a user can complete the auth journey without touching Templ pages
- session and CSRF behavior match current protections

### Phase 4 - Port user management

Goal: replace the current HTMX CRUD path with Vue-powered browser interactions.

Deliver:

- `/users` list view
- create, edit, deactivate, delete, and count flows
- simple client interactions without drifting into SPA theater

Gate:

- the authenticated app experience works end to end through Astro + Vue
- HTMX fragments are no longer required for normal use

### Phase 5 - Retire the legacy frontend stack

Goal: remove the old browser path after parity exists.

Deliver:

- remove Templ views and handlers that only existed for the old browser UI
- remove HTMX from shipped behavior
- remove npm/Tailwind legacy build steps that the new frontend no longer needs
- simplify `internal/ui/static/` to backend-owned assets only

Gate:

- Astro + Vue is the only primary browser frontend
- Bun is the active frontend toolchain

### Phase 6 - Rewrite docs, CI, and release flow around the new truth

Goal: make the migrated shape the documented and verified default.

Deliver:

- `README.md`, `docs/architecture.md`, `docs/development.md`, and `docs/api.md` updated to the Astro + Vue + Bun frontend truth
- CI updated to validate backend and frontend together
- smoke checks updated so they prove the new browser path actually works
- `BUILD.md` deleted once the repo is no longer in active migration

Gate:

- the repo docs stop describing a Templ + HTMX starter
- the new frontend is verified, not just implied

## Recommended Execution Order

Use this order. Do not collapse it into one giant rewrite.

1. Phase 0
2. Phase 1
3. Phase 2
4. Phase 3
5. Phase 4
6. parity pass against current behavior
7. Phase 5
8. Phase 6

Non-negotiables:

- do not delete Templ before the Astro shell and auth flows work
- do not delete HTMX routes before replacement contracts exist
- do not rewrite the backend just because the frontend changed
- do not add a TUI unless the repo purpose changes completely

## Risks

- **contract drift:** current handlers mix full pages, fragments, redirects, and JSON
- **auth regressions:** session-cookie and CSRF behavior are easy to break during frontend rewrites
- **scope creep:** agents may try to rewrite the backend when the job is really boundary cleanup
- **toolchain churn:** npm-era CSS assumptions need deliberate removal, not accidental breakage
- **docs lag:** if docs and CI do not move with the code, the repo will lie about what it is

## Acceptance Criteria

This migration is only done when all of this is true:

- the repo clearly targets **Go backend + Astro/Vue web frontend on Bun**
- the frontend decision remains **web-only**
- login, registration, logout, profile, and user CRUD work through the new frontend
- backend/frontend integration uses explicit documented contracts
- PostgreSQL, Atlas, SQLC, sessions, and CSRF still work
- CI validates the combined backend + web shape
- repo docs match reality
- legacy Templ + HTMX frontend machinery is gone
- `BUILD.md` can be removed because the migration is over

## Final Guidance

Be conservative with backend churn and aggressive about cleaning up the frontend boundary.

This repo does not need dual frontends.
It needs a disciplined migration from an old Go-rendered browser stack to Stephen's default browser stack while preserving the boring strengths of the existing Go backend.
