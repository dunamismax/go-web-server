# BUILD.md

## Decision

**Target frontend: web-only.**

This repo should become a **Go backend + Astro/Vue web starter on Bun**.
Do **not** add an OpenTUI frontend here.

Why:

- The product surface is a browser app starter, not an operator console.
- A TUI would be ceremonial and would not improve the core reference workflow.
- The repo name and current backend shape still make Go the right backend fit.

## Repo Role After Migration

This repo should stop being a reference for Go server-rendered Templ + HTMX apps and become a reference for:

- Go for backend, auth, persistence, jobs, and runtime concerns
- Astro for pages and delivery
- Vue for interactive browser UI
- Bun for frontend runtime, package management, scripts, and tests
- PostgreSQL for durable app state

If this migration lands, the repo becomes a **Go-backed implementation of Stephen's default web frontend lane**, not an exception frozen in the old UI stack.

## Current State Summary

Today the repo is:

- one Go binary built around Echo
- PostgreSQL-backed with `pgx/v5`, SQLC, Atlas migrations, and session auth
- server-rendered with Templ views plus HTMX fragments
- styled via Tailwind CSS with DaisyUI and bundled Pico CSS
- using `npm` for CSS asset generation
- embedding static assets from `internal/ui/static/`
- returning a mix of full HTML pages, HTML fragments, and a small amount of JSON

Important current frontend/runtime truths:

- auth, users, profile, and home flows are tightly coupled to Templ + HTMX responses
- `/api/users/count` is not really an API route today because it returns an HTML fragment
- CSRF and session auth are already cookie-based and same-origin friendly
- generated backend artifacts are checked in; built frontend assets are also checked in today

## Target State Summary

The target shape is:

- **backend:** Go stays
- **web frontend:** `web/` Astro + Vue + TypeScript + Bun
- **database:** PostgreSQL stays
- **auth model:** same-origin session cookies stay
- **backend contract:** explicit JSON or boring same-origin HTTP boundaries for the web app
- **build orchestration:** Mage can stay as the root task entrypoint, but it should call Bun for web tasks
- **asset policy:** built web artifacts should be produced in CI/release flows, not committed as generated CSS blobs

Hard target:

- no Templ in the final frontend path
- no HTMX in the final frontend path
- no Node/npm dependency for the web frontend
- no Tailwind-only asset build living as the primary UI pipeline in the Go app root

## Backend Notes

Keep the backend in Go.

That is the correct fit here because:

- the repo is explicitly a Go web server reference
- the runtime is a long-lived web service with auth, middleware, and DB access
- deployment simplicity and single-service operation matter more than switching languages

Do **not** plan a Python rewrite in this repo.
The migration is a **frontend rewrite around a stable Go backend**, not a full stack language churn exercise.

Backend responsibilities after migration:

- session auth
- CSRF issuance and validation
- persistence and migrations
- user CRUD and profile logic
- JSON/web contract endpoints
- health checks and operational middleware

## Data And Runtime Constraints

These constraints should shape every phase:

1. **Same-origin auth first**
   - Keep Astro and the Go backend on the same origin in production.
   - In development, use a proxy setup rather than inventing cross-origin auth complexity.

2. **Session + CSRF stay real**
   - The frontend must work with the existing session-cookie model.
   - Do not regress CSRF protections just to make frontend integration easier.

3. **PostgreSQL stays canonical**
   - No database change is part of this plan.
   - Atlas migrations and SQLC-backed data access remain in scope.

4. **Backend routes need clearer contracts**
   - HTMX fragment endpoints are not a good long-term frontend contract.
   - JSON endpoints should replace fragment-shaped pseudo-API routes where the new frontend needs them.

5. **Release shape should stay boring**
   - Prefer one Go service plus one built web app artifact set.
   - Avoid introducing extra services unless the repo later earns that complexity.

6. **BUILD.md is temporary**
   - Once the migration is complete, fold stable guidance into `README.md` and `docs/`, then delete this file.

## Phase Plan

### Phase 0 - Lock the migration boundary

Goal: define exactly what the new frontend must cover before code starts moving.

Do:

- inventory current user-facing pages and HTMX fragments
- classify routes as page, fragment, JSON, or mixed
- identify which routes should survive, which should become JSON, and which should disappear
- decide the new web app location as `web/`

Done when:

- there is a route and screen inventory future agents can execute against
- the repo has a written decision that the target is web-only
- no implementation has started yet

### Phase 1 - Introduce the new web workspace

Goal: create the web frontend skeleton without breaking the current app.

Do:

- add `web/` with Astro + Vue + TypeScript + Bun
- use Biome, `astro check`, `bun test`, and Playwright as the frontend quality bar
- define dev commands so the web app can talk to the Go backend cleanly
- keep the existing Templ frontend live while the new app is scaffolded

Done when:

- the repo has a clean web workspace shape
- Bun replaces npm for active frontend work
- the new web app can render a shell and reach the backend in development

### Phase 2 - Normalize backend contracts for the new frontend

Goal: stop treating HTMX fragments as the app contract.

Do:

- define explicit JSON endpoints for auth state, users CRUD interactions, and dashboard widgets that the new UI needs
- rename or replace fragment-shaped `/api` routes that are not actually APIs
- keep same-origin cookie auth and CSRF flows explicit in the contract
- preserve existing business logic and store layer unless a small cleanup is required

Done when:

- frontend-facing endpoints are explicit and documented
- HTML fragment responses are no longer the primary integration boundary for new work
- a future frontend agent can build against stable backend contracts

### Phase 3 - Port the app shell and auth flows

Goal: establish real user-facing parity on the new stack.

Do:

- build Astro page structure and shared layouts
- port login, registration, logout, home, and profile flows
- implement flash/error/success handling in the new UI
- keep accessibility, redirect behavior, and auth failure states honest

Done when:

- a user can complete the auth journey entirely through Astro + Vue
- session and CSRF behavior match current protections
- the new frontend owns the primary page shell

### Phase 4 - Port user management and interactive views

Goal: replace the core HTMX CRUD experience with Vue-powered interactions.

Do:

- port `/users` list, create, edit, deactivate, delete, and count flows
- replace server-sent fragments with clear client/server interactions
- keep the UX simple and server-aligned, not SPA theater
- preserve protected-route behavior and failure handling

Done when:

- the full authenticated app experience works without Templ + HTMX
- the frontend no longer depends on fragment endpoints for normal operation

### Phase 5 - Retire the legacy frontend stack

Goal: remove the old UI path after parity exists.

Do:

- remove Templ-driven frontend routes and views that are no longer needed
- remove HTMX from shipped assets and page behavior
- remove the npm/Tailwind legacy build path if it is no longer part of the chosen frontend stack
- stop checking in generated frontend asset blobs that only exist for the retired stack
- simplify `internal/ui/static/` to only what the backend still truly owns

Done when:

- Astro + Vue is the only primary browser frontend
- the repo no longer markets or behaves like a Templ + HTMX starter
- frontend build responsibility is clearly Bun-based

### Phase 6 - Stabilize docs, CI, and release flow

Goal: make the new shape the documented truth.

Do:

- rewrite `README.md` around the new stack
- update `docs/architecture.md`, `docs/development.md`, and `docs/api.md`
- update Mage/CI/release flows to build and verify the web app
- replace old smoke assumptions with checks that prove the new frontend actually works
- delete this `BUILD.md` once the repo is out of migration mode

Done when:

- repo docs describe the Astro + Vue + Bun frontend truth
- CI validates the backend and web frontend together
- `BUILD.md` is no longer needed

## Recommended Execution Order

Follow this order and do not collapse it into one giant rewrite:

1. Phase 0
2. Phase 1
3. Phase 2
4. Phase 3
5. Phase 4
6. parity check against current behavior
7. Phase 5
8. Phase 6

Non-negotiable sequencing rules:

- do not delete Templ before the Astro/Vue shell and auth flows work
- do not delete HTMX endpoints before replacement contracts exist
- do not rewrite backend storage or auth unless the new frontend proves a real need
- do not add a TUI unless the repo's purpose changes dramatically

## Risks

Primary risks:

- **contract drift:** current handlers mix page HTML, fragments, and JSON, so migration work can get messy fast
- **auth regressions:** session and CSRF behavior are easy to weaken during frontend rewrites
- **scope creep:** this can turn into an unnecessary backend rewrite if future agents are not disciplined
- **tooling churn:** the repo currently assumes npm and checked-in built CSS; that operating model should change deliberately, not accidentally
- **docs lag:** if README/docs are not rewritten at the end, the repo will lie about what it is

## Acceptance Criteria

The migration is only done when all of this is true:

- the repo clearly targets **Go backend + Astro/Vue web frontend on Bun**
- the frontend decision remains **web-only**
- login, registration, logout, profile, and user CRUD work through the new frontend
- backend/frontend integration uses explicit, documented contracts
- PostgreSQL, Atlas, SQLC, and session auth still work
- CI validates the combined backend + web shape
- README and docs match reality
- legacy Templ + HTMX frontend machinery is removed
- this `BUILD.md` can be deleted because the migration is no longer active

## Final Guidance For Future Agents

Be conservative with backend churn and aggressive about frontend boundary cleanup.

This repo does **not** need a second frontend.
It needs a clean migration from an old Go-rendered browser stack to the new default browser stack while preserving the boring strengths of the existing Go backend.
