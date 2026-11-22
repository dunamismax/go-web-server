# Changelog

Notable changes to this project. Versions follow
[Semantic Versioning](https://semver.org/).

## 5.0.0 - 2026-11-22

Version 5 rebuilds the template on a Go-only toolchain. Pages are rendered on
the server, there is no Node.js build, and the result is one static binary.
It is a rewrite rather than an in-place upgrade: projects built on 4.x should
treat it as a reference, not merge it.

### Breaking changes

- **Frontend.** Pages are still [templ](https://templ.guide) components
  enhanced with [htmx](https://htmx.org), but rewritten. Tailwind CSS 4 is
  compiled by its standalone CLI, so the npm toolchain (`package.json`,
  `tailwind.config.js`, DaisyUI) is gone. The Astro + Vue + Bun frontend and
  JSON API that lived on `main` after 4.0.0 were never released and are
  removed; projects that tracked `main` lose `/api/auth/*`, `/api/users/*`
  and the `/_backend` bridge.
- **HTTP layer.** Echo v4 is replaced by the standard library's `net/http`
  router.
- **Database.** The schema starts over with a single
  [goose](https://github.com/pressly/goose) migration embedded in the binary.
  Atlas, `atlas.hcl` and the startup schema bootstrap are removed. The users
  table gains `role` and `password_changed_at`, drops `bio` and `avatar_url`,
  and stores emails lowercased. There is no automatic migration from a 4.x
  database.
- **Configuration.** Environment variables are the only source, and their
  names changed (`HTTP_ADDR`, `APP_ENV`, `LOG_LEVEL`, `SESSION_LIFETIME` and
  others; see `.env.example`). YAML config files and the unused JWT and
  feature-flag settings are gone.
- **Tooling.** Mage is replaced by a Makefile, Air by
  `templ generate --watch`, and the GoReleaser packaging by a Dockerfile and
  a systemd unit.
- **Routes.** Sign-in and sign-up moved from `/auth/login` and
  `/auth/register` to `/login` and `/register`, and `/profile` became
  `/account`. User management uses plain `POST` forms; the fragment routes
  (`/users/list`, `/users/form`) and the `PUT`, `PATCH` and `DELETE`
  endpoints are gone. `/health` became `/healthz` (liveness) and `/readyz`
  (readiness). `/demo` and `/api/users/count` are removed.
- **Passwords.** New passwords need 12 to 128 characters. The
  uppercase/lowercase/digit rule is gone.
- **Go.** Requires Go 1.27.

### Added

- Roles: `admin` and `member`. The first account on a fresh database becomes
  the administrator, user management is admin-only, and admins cannot
  demote, deactivate or delete themselves.
- The account page now lets you edit your profile and change your password.
  Changing a password, or having it reset by an admin, signs out every other
  session for that account.
- Search and pagination on the user list, with search-as-you-type and
  in-place activate/deactivate through htmx. Every page works without
  JavaScript.
- `server migrate up|down|status`, `server healthcheck` and `server version`
  subcommands.
- A test suite that runs against a real PostgreSQL, with each test in its own
  throwaway schema, including end-to-end tests of every flow and of the
  server's startup and graceful shutdown.
- A distroless, non-root container image with a built-in health check, a
  `compose.yaml` for local PostgreSQL, a sandboxed systemd unit, and a Caddy
  example.
- `make rename module=...` to change the Go module path after cloning.
- Automatic re-hashing of passwords at sign-in when the Argon2id parameters
  are raised.

### Changed

- CSRF protection uses Go's built-in `http.CrossOriginProtection` instead of
  a custom token middleware. Forms need no hidden token field.
- The Content Security Policy is `default-src 'self'` with nothing inline,
  replacing a policy that allowed `'unsafe-inline'` and `'unsafe-eval'`.
- The session cookie is `SameSite=Lax` and, in production, `Secure` with the
  `__Host-` prefix.
- Sign-in no longer reveals whether an email is registered, by message or by
  response time, and is rate-limited per client address.
- Every tool version is pinned in one place and used by local runs, CI and
  Docker alike. The Tailwind CLI and htmx downloads are checksum-verified.
- sqlc, templ and CSS output is checked in, so a fresh clone builds with only
  Go installed. CI fails if the checked-in copies are stale.

### Fixed

- A deactivated user stayed signed in until their session expired. Sessions
  now end on the user's next request after deactivation or deletion.
- Any signed-in user could edit, deactivate or delete every other user.
- `cookies.jar` and `cookies.txt` from a local `curl` session were committed
  to the repository.

## 4.0.0 - 2026-03-29

PostgreSQL-backed sessions, Argon2id authentication and a protected user
CRUD screen. See the
[release notes](https://github.com/dunamismax/go-web-server/releases/tag/4.0.0).
