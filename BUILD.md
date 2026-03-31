# BUILD.md

## Decision

- [x] Target frontend remains **web-only**.
  - [x] Keep the backend in Go.
  - [x] Replace the long-term browser path with **TypeScript + Bun + Astro + Vue**.
  - [x] Do **not** add an OpenTUI frontend here.

Why this still stands:

- this repo is a browser app starter, not an operator console
- the product surface is session-authenticated web CRUD, not terminal workbench UX
- a TUI would add maintenance cost without improving the reference use case

## Repo Role

This repo is still meant to become the **Go-backed reference starter for Stephen's default web frontend lane**:

- **backend:** Go + Echo + PostgreSQL + SQLC + Atlas
- **frontend:** Astro + Vue + TypeScript + Bun
- **auth:** same-origin session cookies + CSRF
- **deployment shape:** one Go service, one Postgres database, one built web asset set

This is a frontend migration plan, not a backend rewrite plan.

## Current State

Today the repo is:

- one Go binary with Echo handlers and middleware
- PostgreSQL-backed with `pgx/v5`, SQLC, Atlas migrations, and session auth
- still shipping a **Templ + HTMX** browser path
- using **Tailwind + DaisyUI + Pico CSS** built via **Bun** for the legacy frontend
- embedding frontend assets from `internal/ui/static/`
- carrying a staged **Astro + Vue + Bun** workspace under `web/`
- returning a mix of full HTML pages, HTMX fragments, redirects, and a little JSON

### Current user-facing surfaces

Public:

- `/`
- `/demo`
- `/health`
- `/auth/login`
- `/auth/register`
- `/auth/logout`
- `/api/auth/state`
- `/api/auth/login`
- `/api/auth/register`
- `/api/auth/logout`
- `/static/*`

Authenticated:

- `/profile`
- `/users`
- `/users/form`
- `/users/:id/edit`
- `/users` `POST`
- `/users/:id` `PUT`
- `/users/:id/deactivate` `PATCH`
- `/users/:id` `DELETE`
- `/api/users`
- `/api/users/count`
- `/api/users/:id`
- `/api/users` `POST`
- `/api/users/:id` `PUT`
- `/api/users/:id/deactivate` `PATCH`
- `/api/users/:id` `DELETE`

### Current-state truths that must not get lost

- the shipped browser path for auth, profile, and user CRUD is still tightly coupled to Templ and HTMX response shapes
- Phase 2 now adds parallel JSON contracts under `/api/*` for auth state and managed-user operations
- session cookies are already same-origin friendly and protected by CSRF middleware
- generated backend artifacts and built frontend assets are currently checked in
- local development now uses Bun for both the staged Astro workspace and the repo-root legacy CSS asset build

## Target State

The target shape is still:

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

That is still the correct backend fit because this repo is still:

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
- explicit frontend-facing JSON or other boring same-origin HTTP contracts

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

## Phase Plan And Status

- [x] **Phase 0 - Freeze scope and map the UI surface**
  - [x] Route inventory is mapped to page, fragment, redirect, JSON, or mixed behavior in `docs/frontend-migration-inventory.md`.
  - [x] Frontend workspace location is fixed as `web/`.
  - [x] This repo is explicitly recorded as **web-only**.
  - [x] Outcome: later work can point to a finite migration surface instead of reverse-engineering routes mid-build.

- [x] **Phase 1 - Add the new frontend workspace**
  - [x] `web/` contains Astro, Vue, TypeScript, Bun, Biome, Bun tests, mocked Playwright coverage, and a real browser smoke path.
  - [x] Local development can route frontend requests to the Go app through the `/_backend/*` proxy path.
  - [x] Root Mage wiring exists for Bun-based frontend install, dev, check, build, preview, unit-test, and e2e commands.
  - [x] The current Templ frontend still exists as the shipped browser path.
  - [x] The staged frontend can boot as a migration shell and is wired to reach backend health through the frontend proxy.

- [x] **Phase 2 - Normalize backend contracts**
  - [x] Explicit frontend-facing contracts exist for auth state, user list, create, edit, deactivate, delete, and count.
  - [x] `/api/*` naming reflects reality.
    - `/api/users/count` now returns JSON. The temporary legacy HTML fragment path used during the transition has since been retired.
  - [x] Current CSRF expectations are documented for browser requests in `docs/api.md`.
  - [x] Endpoint docs are complete enough that later frontend agents can work without reading Templ templates or handler code.
    - `docs/api.md` now documents the Phase 2 JSON contract surface and the remaining legacy routes explicitly.

- [x] **Phase 3 - Port the app shell and auth flows**
  - [x] A staged Astro layout, page shell, and Vue status card exist in `web/`.
  - [x] Home, login, registration, logout, and profile flows run through Astro + Vue instead of Templ.
    - `web/` now ships Astro routes for `/`, `/auth/login`, `/auth/register`, `/auth/logout`, and `/profile` backed by the Phase 2 JSON auth contracts.
  - [x] Success, error, redirect, and unauthenticated states are handled by the new frontend.
  - [x] A user can complete the auth journey without touching Templ pages.

- [x] **Phase 4 - Port user management**
  - [x] `/users` list view is ported to Astro + Vue.
    - `web/src/pages/users.astro` now mounts a Vue CRUD dashboard instead of pointing normal frontend work at the legacy HTMX screen.
  - [x] Create, edit, deactivate, delete, and count flows use explicit frontend contracts instead of HTMX fragments.
    - The staged frontend now talks to `/api/users`, `/api/users/:id`, `/api/users/:id/deactivate`, and `/api/users/count` through `web/src/lib/backend.ts`.
  - [x] The authenticated CRUD experience works end to end through Astro + Vue.
    - Playwright now covers the staged users route for create, edit, deactivate, and delete flows with mocked backend contracts.
  - [x] HTMX fragments are no longer required for normal user management.
    - Legacy fragment routes still exist for the shipped browser path, but the staged Astro `/users` flow no longer depends on them.

- [ ] **Phase 5 - Retire the legacy frontend stack**
  - [ ] Templ views and handlers that only existed for the old browser UI are removed.
    - `/users/list` and `/users/count` are now gone. The remaining legacy-only `/users` surface is the inline HTMX page plus `/users/form`, `/users/:id/edit`, and the legacy mutation submits.
  - [ ] HTMX is removed from shipped browser behavior.
  - [ ] npm and Tailwind legacy build steps are removed from the active frontend path.
  - [ ] `internal/ui/static/` is simplified to backend-owned assets only.
    - Today `mage generate` still runs `bun run build-css`, legacy Templ views still exist, and checked-in CSS assets are still part of the shipped path.

- [ ] **Phase 6 - Rewrite docs, CI, and release flow around the new truth**
  - [x] Repo docs now acknowledge the staged `web/` workspace and the migration inventory.
  - [ ] `README.md`, `docs/architecture.md`, `docs/development.md`, and `docs/api.md` describe Astro + Vue + Bun as the primary browser truth.
    - Current docs still say the shipped browser path is Templ + HTMX, which is accurate today but means this phase is not done.
  - [x] CI validates backend and frontend together.
    - `.github/workflows/ci.yml` now installs Bun, runs frontend install/check/build, exercises mocked Playwright coverage, and keeps the Go quality gates in the same pipeline.
  - [x] Smoke checks prove the new browser path actually works.
    - `scripts/frontend-smoke.sh` now drives Astro preview plus the real Go backend through registration, profile, `/users`, and logout.
  - [ ] `BUILD.md` is deleted once the migration is complete.

## Recommended Execution Order From Here

- [x] Phase 0
- [x] Phase 1
- [x] Phase 2
- [x] Phase 3
- [x] Phase 4
- [ ] Parity pass against current behavior
- [ ] Phase 5
- [ ] Phase 6

Non-negotiables:

- do not delete Templ before the Astro shell and auth flows work
- do not delete HTMX routes before replacement contracts exist
- do not rewrite the backend just because the frontend changed
- do not add a TUI unless the repo purpose changes completely

## Risks

- **contract drift:** current handlers mix full pages, fragments, redirects, and JSON
- **auth regressions:** session-cookie and CSRF behavior are easy to break during frontend rewrites
- **scope creep:** agents may try to rewrite the backend when the job is really boundary cleanup
- **toolchain churn:** legacy CSS assumptions need deliberate removal, not accidental breakage
- **docs lag:** if docs and CI do not move with the code, the repo will lie about what it is

## Acceptance Criteria Status

- [ ] The repo clearly ships **Go backend + Astro/Vue web frontend on Bun** as the primary browser path.
- [x] The frontend decision remains **web-only**.
- [x] Login, registration, logout, profile, and user CRUD work through the new frontend.
- [x] Backend and frontend integration uses explicit documented contracts.
- [x] PostgreSQL, Atlas, SQLC, sessions, and CSRF still form the backend foundation.
- [x] CI validates the combined backend + web shape.
- [ ] Repo docs match the final migrated reality.
- [ ] Legacy Templ + HTMX frontend machinery is gone.
  - The legacy `/users` screen no longer bootstraps through `/users/list` and `/users/count`, but the remaining HTMX form and mutation path still exists.
- [ ] `BUILD.md` can be removed because the migration is over.

## Final Guidance

Be conservative with backend churn and aggressive about cleaning up the frontend boundary.

This repo does not need dual frontends.
It needs a disciplined migration from an old Go-rendered browser stack to Stephen's default browser stack while preserving the boring strengths of the existing Go backend.
