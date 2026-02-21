# AGENTS.md

> Runtime operations source of truth for this repository.
> This file defines how scry should operate in `go-web-server`.

---

## First Rule

Read `SOUL.md` first, then this file, then inspect `README.md` and the touched code paths before editing.

---

## Repo Scope

- Repo: `go-web-server`
- Purpose: production-ready Go web server template.
- Runtime shape:
  - Echo HTTP server entrypoint in `cmd/web/main.go`
  - App internals in `internal/` (handlers, middleware, config, store, views)
  - SQLC/DB schema and migrations under `internal/store/` and `migrations/`
  - Build/developer automation in `magefile.go`

---

## Owner

- Name: Stephen
- Alias: `dunamismax`
- Home: `/Users/sawyer`
- Projects root: `/Users/sawyer/github`

---

## Stack Contract (Strict)

Use the actual current stack unless Stephen explicitly approves changes:

- Language/runtime: **Go 1.25**
- Web framework: **Echo v4**
- Templates/UI rendering: **Templ**
- Frontend interaction: **HTMX**
- Styling: **Tailwind CSS + DaisyUI**
- Database: **PostgreSQL** with **pgx/v5**
- Query layer: **SQLC**
- Config: **Koanf**
- Build/task runner: **Mage**
- Hot reload: **Air**

---

## Operating Contract

- Keep handlers thin and move persistence logic into the store layer.
- Preserve security middleware posture (auth, CSRF, sanitization, validation, headers, rate limiting).
- Prefer explicit SQL and generated SQLC methods over ad-hoc DB access.
- Keep docs and code in sync when architecture or commands change.
- Keep changes minimal and reviewable.

---

## Workflow

`Wake -> Explore -> Plan -> Code -> Verify -> Report`

- **Explore**: read touched files end-to-end first.
- **Plan**: smallest safe diff that satisfies the request.
- **Code**: preserve existing project conventions.
- **Verify**: run concrete commands and report outcomes.
- **Report**: changed files, verification evidence, and remaining risks.

---

## Command Policy

- Prefer Mage commands as canonical workflow entrypoints.
- Use native Go tools where needed (`go test`, `go vet`, etc.) but keep Mage as default orchestration.
- Avoid destructive operations without explicit approval.

### Canonical commands

```bash
# setup and development
mage setup
mage dev
mage run

# code generation and formatting
mage generate
mage fmt

# quality gates
mage vet
mage lint
mage vulncheck
mage quality
mage ci

# database
mage migrate
mage migrateDown
mage migrateStatus

# build and release
mage build
mage release
mage snapshot
```

---

## Git Remote Sync Policy

- Use `origin` as working remote.
- `origin` fetch URL:
  - `git@github.com-dunamismax:dunamismax/go-web-server.git`
- `origin` push URLs:
  - `git@github.com-dunamismax:dunamismax/go-web-server.git`
  - `git@codeberg.org-dunamismax:dunamismax/go-web-server.git`
- `git push origin main` must publish to both.
- Never force-push `main` unless Stephen explicitly asks.

---

## Done Criteria

A task is done when all are true:

- Requirements are implemented.
- Relevant verification commands were run and reported.
- Docs reflect behavior/tooling changes.
- Diff is focused and reviewable.

---

## Living Document Protocol

- Keep current-state only.
- Update immediately when stack/workflow/contracts change.

---

## Platform Baseline (Strict)

- Primary and only local development OS is **macOS**.
- Assume `zsh`, BSD userland, and macOS filesystem paths by default.
- Do not provide or prioritize Windows/PowerShell/WSL instructions.
- If cross-platform guidance is requested, keep macOS as source of truth and treat Windows as out of scope unless Stephen explicitly asks for it.
- Linux deployment targets may exist per repo requirements; this does not change local workstation assumptions.
