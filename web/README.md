# Web frontend workspace

This directory is the staged Astro + Vue + Bun workspace for the `go-web-server` frontend migration.

Current truth:

- the shipped browser path is still the legacy Templ + HTMX app
- the Astro workspace now covers `/`, `/auth/login`, `/auth/register`, `/auth/logout`, `/profile`, and `/users`
- those routes talk to the Go backend through the explicit `/api/auth/*` and `/api/users/*` contracts
- local frontend development still talks to the Go app through a same-origin-style proxy prefix: `/_backend/*`
- CI now runs Bun install, frontend check and build steps, mocked Playwright e2e coverage, and a real Astro browser smoke flow against the Go backend
- the next migration target is retiring the legacy browser stack once the new CRUD path has enough verification

## Local development

Run the backend and frontend in separate terminals from the repo root:

```bash
mage dev
mage frontendDev
```

By default the Astro frontend runs at `http://127.0.0.1:4321` and proxies `/_backend/*` requests to `http://127.0.0.1:8080`.

If your Go app is on a different origin, set `FRONTEND_BACKEND_ORIGIN` before starting the frontend:

```bash
FRONTEND_BACKEND_ORIGIN=http://127.0.0.1:9090 mage frontendDev
```

## Commands

```bash
mage frontendInstall
mage frontendCheck
mage frontendBuild
mage frontendPreview
mage frontendTest
mage frontendE2E
mage frontendSmoke
```

The root Mage targets are the supported entrypoints for this workspace during the migration.

`mage frontendSmoke` builds the Astro app, starts the Go backend against a disposable PostgreSQL instance when needed, and runs a real browser flow through Astro preview with Playwright.
