# BUILD.md

Last reviewed: 2026-03-31

## Purpose

This file is the active execution manual for shaping `go-web-server` into the right long-term starter for Stephen's current portfolio direction.

Future agents are expected to:

- read this file before making substantial repo changes
- work through the phases in order unless a later phase is explicitly unblocked first
- keep this document current as reality changes
- check boxes only when the work is actually done and verified
- keep stable docs such as `README.md` and `docs/*.md` current-state oriented, while using this file for forward-looking execution tracking

If the plan changes, rewrite this file to match the new truth instead of appending stale history.

## Reference Context

Read these before making major direction changes:

- `README.md`
- `docs/architecture.md`
- `docs/development.md`
- `docs/api.md`
- `/Users/sawyer/github/dunamismax/tech-stacks/web-fullstack-tech-stack.md`
- `/Users/sawyer/github/dunamismax/README.md`

## Repo Fit Boundaries

This repo should **not** pretend to be Stephen's default Bun-native full-stack TypeScript app starter.

That full-stack lane is for products where Bun owns both the web frontend and backend service layer.

`go-web-server` has a different and legitimate job:

- Go remains the backend lane
- PostgreSQL remains the database lane
- Astro + Vue + Bun remain the frontend lane
- the shipped product shape stays boring and self-hostable
- the browser talks to explicit same-origin JSON contracts
- the frontend is embedded into the Go service for the default shipped experience

In other words: this repo should evolve into a **Go-backed modern web starter**, not a fake TypeScript monorepo rewrite.

## Non-Goals

Do not turn this repo into any of the following unless Stephen explicitly asks:

- a Bun + Elysia backend rewrite
- a full Bun workspace monorepo just to match the TypeScript web stack document more literally
- a client-heavy SPA-first starter
- a multi-service platform with Redis, queues, or extra infrastructure before the starter clearly earns it
- a generic kitchen-sink template that hides the actual request, auth, and data flow

## Current State Snapshot

Today the repo already has a real foundation:

- Echo-based Go backend with session auth, CSRF, PostgreSQL, SQLC, and Mage
- Astro + Vue + Bun frontend under `web/`
- committed `web/dist` output embedded into the Go binary
- explicit `/api/auth/*` and `/api/users/*` JSON contracts
- browser smoke coverage for both Astro-dev and Go-served embedded flows
- single-host deployment docs based on a binary and reverse proxy

The real gaps are mostly about productizing the starter, tightening the story, and aligning the operational shape with Stephen's current standards.

## Target State

The target is a portfolio-ready starter with this shape:

- Go owns backend logic, sessions, CSRF, persistence, and operational endpoints
- Astro owns page composition and server-first delivery
- Vue is used only where interactivity clearly earns it
- PostgreSQL is the default system of record
- local development and single-host deployment are easy to understand and verify
- docs make the boundaries obvious: when to use this repo, when to use the Bun full-stack lane instead, and what is intentionally out of scope

## Working Rules For Execution

When implementing phases below:

1. Prefer current-state doc edits over historical storytelling.
2. Keep the starter boring and inspectable.
3. Do not add abstraction or infrastructure without a clear starter-level payoff.
4. Verify the smallest relevant slice after each meaningful change.
5. If a phase reveals a better plan, update this file in the same change.

## Phase 1: Position The Repo Correctly

Goal: make the repo's identity unambiguous inside the portfolio and inside its own docs.

### Work

- [ ] Audit root and docs wording so the repo is consistently described as a Go-backed web starter with an Astro + Vue frontend, not as a generic full-stack TypeScript starter.
- [ ] Add a concise "use this repo when" and "do not use this repo when" framing to the stable docs.
- [ ] Make the relationship to Stephen's broader stack explicit: Go backend lane, Astro + Vue frontend lane, PostgreSQL default.
- [ ] Remove or rewrite any wording that implies legacy UI technology is still active.
- [ ] Keep `BUILD.md` as the only forward-looking phase tracker while the repo is still being actively shaped.

### Acceptance Criteria

- [ ] A new contributor can read `README.md` and understand this repo's exact lane in under two minutes.
- [ ] Stable docs describe current behavior only.
- [ ] No stable doc implies a Bun backend rewrite is planned unless that becomes true and this file is updated first.

## Phase 2: Harden The Go Starter Core

Goal: make the backend starter path feel intentional, reproducible, and easy to extend.

### Work

- [ ] Confirm there is one canonical schema and migration truth, and remove or clearly quarantine any remaining legacy duplication.
- [ ] Expand backend verification around auth, session, CSRF, and user CRUD behavior where coverage is still thin.
- [ ] Document the expected extension path for adding new protected resources, handlers, queries, and API contracts.
- [ ] Review whether Echo remains the right router and middleware layer for this starter or whether simplification is warranted. Only migrate if the payoff is clear and the resulting story gets simpler.
- [ ] Make fresh local bring-up from an empty PostgreSQL database fully reproducible from docs and commands already in the repo.

### Acceptance Criteria

- [ ] A clean machine can reach a working local app from the documented steps without hidden migration knowledge.
- [ ] Backend extension points are documented clearly enough that future agents do not need to reverse-engineer the pattern from multiple files.
- [ ] The backend lane stays boring, explicit, and PostgreSQL-first.

## Phase 3: Tighten The Frontend Starter Lane

Goal: make the frontend feel like Stephen's current web lane without pretending the repo is Bun-native end to end.

### Work

- [ ] Keep Astro as the page owner and use Vue only for genuinely interactive islands or dashboards.
- [ ] Review the current UI shell, tokens, and component structure so the starter looks intentional rather than merely functional.
- [ ] Consolidate frontend data access, CSRF bootstrapping, and error handling into a clear pattern that future features can follow.
- [ ] Document the `/_backend/*` development bridge and shipped embedded runtime behavior in one place that future agents will actually find.
- [ ] Decide whether any shared frontend utilities or contract helpers should be promoted into a clearer structure inside `web/` without introducing monorepo theater.

### Acceptance Criteria

- [ ] The Astro-dev flow and embedded Go-served flow behave consistently enough that frontend work does not depend on guesswork.
- [ ] The starter demonstrates a clean pattern for protected pages, mutations, and optimistic or refresh-based UI updates.
- [ ] The frontend aligns with Stephen's current Astro + Vue expectations while staying appropriate for a Go repo.

## Phase 4: Align Local And Deployment Operations

Goal: make the operational story closer to Stephen's current preferred web deployment shape.

### Work

- [ ] Decide on the canonical local orchestration story: pure host setup, Docker-backed PostgreSQL, or both with clear primary guidance.
- [ ] Add a simple container and reverse-proxy path only if it materially improves local parity or single-host deployment clarity.
- [ ] If Docker Compose is added, keep it minimal and centered on PostgreSQL, the app, and optional reverse proxy only.
- [ ] If Caddy is added, keep the config repo-local and aligned with same-origin cookie and proxy expectations.
- [ ] Tighten deployment docs so TLS, trusted proxies, cookie security, config inputs, and migration steps are obvious.

### Acceptance Criteria

- [ ] There is one clearly documented primary local setup path.
- [ ] There is one clearly documented primary single-host deployment path.
- [ ] Operational docs do not hand-wave over cookie, CSRF, proxy, or migration requirements.

## Phase 5: Make It Portfolio-Ready

Goal: ensure the repo reads like a deliberate starter Stephen would actually point people at.

### Work

- [ ] Tighten the README opening, feature framing, and screenshots or demo assets so the repo reads like a modern starter rather than a stitched demo.
- [ ] Add a short decision guide that explains when this repo is the right starting point versus Stephen's Bun full-stack lane or a pure Go service repo.
- [ ] Review naming, examples, and seeded content so the starter feels cohesive and not half-generic.
- [ ] Ensure verification commands called out in docs match what contributors can actually run today.
- [ ] Prune stale implementation notes once the replacement current-state docs are strong enough.

### Acceptance Criteria

- [ ] The repo's public story matches Stephen's current portfolio direction.
- [ ] The starter feels intentionally opinionated, not overloaded.
- [ ] A future agent can continue the work from this file without reconstructing repo history.

## Verification Rhythm

Use the narrowest useful checks first.

Backend baseline:

```bash
go build ./...
go vet ./...
go test ./...
```

Frontend and repo-native checks when relevant:

```bash
mage frontendCheck
mage frontendBuild
mage frontendSmoke
mage smoke
```

For docs-only changes, verify that the changed docs are internally consistent and that no stable doc now contradicts this file.

## Done Means

A phase is only done when:

- the code and docs agree
- the relevant checks were run
- the boxes in this file were updated to reflect reality
- the next agent can pick up cleanly without guessing intent
