# Documentation

This docs set stays close to the repo that exists today. Start with the development guide if you are bringing the project up locally for the first time.

## Start Here

- [Development guide](development.md): prerequisites, local setup, configuration, and daily commands
- [Architecture overview](architecture.md): repo layout, request flow, config loading, and schema ownership

## App Behavior

- [API and route behavior](api.md): public routes, protected routes, HTMX fragments, and auth behavior
- [Frontend migration inventory](frontend-migration-inventory.md): Phase 1 route map, current response modes, and the staged Astro workspace
- [Security notes](security.md): actual controls in the code today, plus current gaps and risks

## Deployment

- [Deployment notes](deployment.md): the current single-host deployment story and its limits
- [Ubuntu deployment walkthrough](ubuntu-deployment.md): the repo's `systemd`-based path
- [Example YAML config](config.example.yaml): sample config file for non-`.env` usage

## Current-State Notes

- Top-level [`migrations/`](../migrations/) is the canonical Atlas migration directory.
- [`internal/store/migrations/`](../internal/store/migrations/) is legacy history and should not be used as the source of truth.
- Generated Go code and legacy built frontend assets are checked in. Run `mage generate` after source changes and commit the resulting artifacts.
- `web/` is now the Astro + Vue + Bun workspace for the frontend migration, and its committed `dist/` output is embedded into the Go binary for the primary GET browser routes.
- CI reruns generation, fails if tracked generated files drift, runs frontend install, check, build, and mocked Playwright e2e coverage, then runs both the Astro browser smoke flow and the Go-served runtime smoke check.

## Naming Note

The repo directory and Go module path use `go-web-server`. Deployment examples still use `gowebserver` for the service user, systemd unit, and sample database name.
