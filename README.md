<p align="center">
  <img src="https://github.com/dunamismax/images/blob/main/golang/go-logo.png" alt="Go Web Server logo" width="260" />
</p>

# Go Web Server

Production-oriented Go web application template built on the **Modern Go Stack**: Echo, Templ, HTMX, SQLC, PostgreSQL, and Mage. It is designed for teams that want strong defaults for security, maintainability, and fast iteration without a heavy frontend framework.

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8.svg?logo=go" alt="Go Version"></a>
  <a href="https://echo.labstack.com/"><img src="https://img.shields.io/badge/Framework-Echo_v4-00ADD8.svg?logo=go" alt="Echo Framework"></a>
  <a href="https://templ.guide/"><img src="https://img.shields.io/badge/Templates-Templ-00ADD8.svg?logo=go" alt="Templ"></a>
  <a href="https://htmx.org/"><img src="https://img.shields.io/badge/Frontend-HTMX_2.x-3D72D7.svg?logo=htmx" alt="HTMX"></a>
  <a href="https://sqlc.dev/"><img src="https://img.shields.io/badge/Queries-SQLC-00ADD8.svg?logo=go" alt="SQLC"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/Database-PostgreSQL-336791.svg?logo=postgresql" alt="PostgreSQL"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-green.svg" alt="MIT License"></a>
</p>

## Quick Start

### Prerequisites

- Linux, macOS, or WSL2 (recommended). Native Windows PowerShell is not the primary target because Mage tasks use Unix utilities like `which`.
- Go 1.25+
- PostgreSQL 15+
- Node.js + npm (for Tailwind build)
- [Atlas CLI](https://atlasgo.io/getting-started)
- [Mage](https://magefile.org/) (`go install github.com/magefile/mage@latest`)

### 1. Clone and configure

```bash
git clone https://github.com/dunamismax/go-web-server.git
cd go-web-server
cp .env.example .env
```

Edit `.env` with real database credentials (at minimum `DATABASE_USER`, `DATABASE_PASSWORD`, or a full `DATABASE_URL`).

### 2. Prepare PostgreSQL

```bash
sudo -u postgres createuser -P gowebserver
sudo -u postgres createdb -O gowebserver gowebserver
```

### 3. Install tools and generate code

```bash
mage setup
mage generate
```

### 4. Apply database migrations

```bash
mage migrate
```

### 5. Run development server

```bash
mage dev
```

Open `http://localhost:8080`.

Expected results:
- Home page loads with HTMX-enabled UI
- `GET /health` returns JSON health data
- Login and registration routes are available at `/auth/login` and `/auth/register`

## Why This Template

- Fast server-rendered UX with HTMX, no SPA complexity required
- Type-safe database and template workflow (SQLC + Templ)
- Session authentication with Argon2id password hashing
- Security middleware included by default (CSRF, sanitization, headers, rate limiting)
- Single-binary deployment with embedded static assets
- Practical build and quality automation through Mage

## Feature Overview

- **Web stack**: Echo v4 + Templ + HTMX + Tailwind + DaisyUI
- **Data layer**: PostgreSQL + pgx/v5 + SQLC generated queries
- **Auth model**: Session-based auth using `scs` with PostgreSQL-backed sessions
- **Security controls**: CSRF protection, input sanitization, strict security headers, request IDs, structured errors
- **Ops readiness**: graceful shutdown, systemd service, deployment scripts, structured logging with `slog`
- **Dev workflow**: hot reload with Air, code generation, linting, vulnerability checks

## Tech Stack

| Layer | Technology | Purpose |
| --- | --- | --- |
| Language | Go 1.25+ | Application runtime |
| HTTP | Echo v4 | Routing and middleware |
| Templates | Templ | Type-safe server-rendered UI |
| Frontend behavior | HTMX 2.x | Progressive dynamic interactions |
| CSS | Tailwind CSS + DaisyUI | Styling and components |
| Database | PostgreSQL | Relational storage |
| DB driver | pgx/v5 | High-performance PostgreSQL driver |
| Query generation | SQLC | Compile-time safe SQL access |
| Auth/session | scs + pgxstore + Argon2id | Secure session auth and password hashing |
| Config | Koanf | Defaults + `.env` + file + env layering |
| Migrations | Atlas | Declarative schema migration workflow |
| Build/dev tooling | Mage + Air | Automation and hot reload |

## Project Structure

```text
go-web-server/
├── cmd/web/                    # Entry point
├── internal/
│   ├── config/                 # Configuration loading (Koanf)
│   ├── handler/                # HTTP handlers and route registration
│   ├── middleware/             # CSRF, auth, sanitization, error handling
│   ├── store/                  # SQLC queries, schema, and DB store
│   ├── ui/                     # Embedded static assets
│   └── view/                   # Templ components/pages
├── migrations/                 # Atlas migrations used by mage migrate
├── docs/                       # Development, API, architecture, deployment docs
├── scripts/                    # systemd service and deployment scripts
├── magefile.go                 # Build/dev/quality command definitions
├── sqlc.yaml                   # SQLC configuration
├── atlas.hcl                   # Atlas configuration
└── .env.example                # Environment template
```

## Development Workflow

### Daily commands

```bash
mage dev          # Run with hot reload
mage generate     # Regenerate SQLC + Templ + CSS
mage quality      # vet + lint + vulncheck
mage build        # Build production binary (bin/server)
mage ci           # generate + fmt + quality + build
```

### Full command reference

```bash
mage help
```

### Clean and reset

```bash
mage clean        # Remove build artifacts and temp outputs
mage reset        # Clean + regenerate + migrate for a fresh local state
```

## Configuration

Runtime config is loaded in this order:
1. Built-in defaults
2. `.env` (if present)
3. `config.yaml` / `config.yml` (optional)
4. Environment variables

Important settings (`.env`):

```env
APP_ENVIRONMENT=development
APP_DEBUG=true
APP_LOG_LEVEL=debug
APP_LOG_FORMAT=text

SERVER_PORT=8080

DATABASE_URL=postgres://username:password@localhost:5432/gowebserver?sslmode=disable
# or use DATABASE_USER / DATABASE_PASSWORD / DATABASE_HOST / DATABASE_PORT / DATABASE_NAME

AUTH_COOKIE_NAME=auth_token
AUTH_COOKIE_SECURE=false
```

For production, set:
- `APP_ENVIRONMENT=production`
- `APP_DEBUG=false`
- `AUTH_COOKIE_SECURE=true`
- restrictive `SECURITY_ALLOWED_ORIGINS`

## HTTP Surface (Snapshot)

Core routes currently registered:

- `GET /` home page
- `GET /demo` HTMX demo endpoint
- `GET /health` health check
- `GET|POST /auth/login` login
- `GET|POST /auth/register` registration
- `POST /auth/logout` logout
- `GET /profile` authenticated profile page
- `GET|POST|PUT|PATCH|DELETE /users...` user CRUD endpoints
- `GET /api/users/count` API example endpoint

## Security Model

Implemented in current codebase:

- Session-based authentication (`scs`) with PostgreSQL session store
- Argon2id password hashing
- CSRF middleware on state-changing requests
- Input sanitization middleware
- Security headers + CORS + rate limiting
- Structured error responses and request correlation IDs

Operational note:
- `/profile` is authenticated.
- The `/users` CRUD routes are currently not wrapped by auth middleware by default. Restrict these routes before internet-facing production deployment.

## Deployment (Ubuntu + systemd)

### Quick path

```bash
mage build
sudo ./scripts/deploy.sh
```

`scripts/deploy.sh` installs to `/opt/gowebserver`, copies `.env`, installs `scripts/gowebserver.service`, and restarts the service.

### Manual path

```bash
mage build
sudo cp bin/server /opt/gowebserver/bin/
sudo cp scripts/gowebserver.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable gowebserver
sudo systemctl restart gowebserver
sudo systemctl status gowebserver
```

## Documentation

- [Documentation Index](docs/README.md)
- [Development Guide](docs/development.md)
- [API Reference](docs/api.md)
- [Architecture](docs/architecture.md)
- [Security Guide](docs/security.md)
- [Deployment Guide](docs/deployment.md)
- [Ubuntu Deployment Guide](docs/ubuntu-deployment.md)

## Known Gaps

- Some docs still mention legacy JWT terminology; runtime auth in this codebase is session-based.
- Feature flags for metrics/pprof exist in config, but dedicated production endpoints are not fully wired in current routes.

## Contributing

1. Fork repository
2. Create a focused branch
3. Run `mage quality`
4. Open a pull request with clear implementation notes

## License

Licensed under the [MIT License](LICENSE).
