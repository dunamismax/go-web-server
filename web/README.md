# Frontend Workspace

`web/` is the primary browser frontend workspace for `go-web-server`.

## Stack

- Astro for pages and layout composition
- Vue for interactive client-side widgets
- TypeScript for app code
- Bun for package management, scripts, tests, and builds
- Biome for lint and formatting
- Playwright for mocked browser coverage

## Runtime Contract

- In local frontend development, Astro uses the `/_backend/*` proxy path to reach the Go app.
- In shipped builds, the Go server serves `web/dist` and strips the baked `/_backend` prefix in-process.
- Auth and managed-user interactions use the explicit `/api/auth/*` and `/api/users/*` JSON contracts.
- Browser auth fallback submits still exist on `POST /auth/*`, but the shipped interactive flows use the JSON API surface.

## Important Outputs

- `web/src/` is the source of truth.
- `web/dist/` is committed and embedded into the Go binary for shipped browser routes.
