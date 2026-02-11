<p align="center">
  <img src="https://github.com/dunamismax/images/blob/main/golang/go-logo.png" alt="Go Web Server Template Logo" width="400" />
</p>

<p align="center">
  <a href="https://github.com/dunamismax/go-web-server">
    <img src="https://readme-typing-svg.demolab.com/?font=Fira+Code&size=24&pause=1000&color=00ADD8&center=true&vCenter=true&width=900&lines=The+Modern+Go+Stack;Echo+v4+Framework+with+Type-Safe+Templates;HTMX+Dynamic+UX+without+JavaScript;SQLC+Generated+Queries+with+PostgreSQL;CSRF+Protection+and+Input+Sanitization;Structured+Error+Handling+and+Request+Tracing;Hot+Reload+Development+with+Mage+Automation;Single+Binary+Deployment+at+15MB;Production-Ready+Security+Middleware;Ubuntu+SystemD+Deployment" alt="Typing SVG" />
  </a>
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8.svg?logo=go" alt="Go Version"></a>
  <a href="https://echo.labstack.com/"><img src="https://img.shields.io/badge/Framework-Echo_v4-00ADD8.svg?logo=go" alt="Echo Framework"></a>
  <a href="https://templ.guide/"><img src="https://img.shields.io/badge/Templates-Templ-00ADD8.svg?logo=go" alt="Templ"></a>
  <a href="https://htmx.org/"><img src="https://img.shields.io/badge/Frontend-HTMX_2.x-3D72D7.svg?logo=htmx" alt="HTMX"></a>
  <a href="https://tailwindcss.com/"><img src="https://img.shields.io/badge/CSS-Tailwind_CSS-06B6D4.svg?logo=tailwindcss" alt="Tailwind CSS"></a>
  <a href="https://daisyui.com/"><img src="https://img.shields.io/badge/Components-DaisyUI-5A0EF8.svg" alt="DaisyUI"></a>
  <a href="https://sqlc.dev/"><img src="https://img.shields.io/badge/Queries-SQLC-00ADD8.svg?logo=go" alt="SQLC"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/Database-PostgreSQL-336791.svg?logo=postgresql" alt="PostgreSQL"></a>
  <a href="https://pkg.go.dev/github.com/jackc/pgx/v5"><img src="https://img.shields.io/badge/Driver-pgx_v5-00ADD8.svg?logo=go" alt="pgx PostgreSQL Driver"></a>
  <a href="https://pkg.go.dev/log/slog"><img src="https://img.shields.io/badge/Logging-slog-00ADD8.svg?logo=go" alt="Go slog"></a>
  <a href="https://github.com/knadh/koanf"><img src="https://img.shields.io/badge/Config-Koanf-00ADD8.svg?logo=go" alt="Koanf"></a>
  <a href="https://atlasgo.io/"><img src="https://img.shields.io/badge/Migrations-Atlas-FF6B6B.svg" alt="Atlas"></a>
  <a href="https://magefile.org/"><img src="https://img.shields.io/badge/Build-Mage-purple.svg?logo=go" alt="Mage"></a>
  <a href="https://github.com/air-verse/air"><img src="https://img.shields.io/badge/HotReload-Air-FF6B6B.svg?logo=go" alt="Air"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-green.svg" alt="MIT License"></a>
</p>

---

## Live Demo

**[View Live Demo →](https://go.dunamismax.com/)** - Self-hosted production deployment showcasing the complete Modern Go Stack in action.

---

## About

A production-ready template for modern web applications using **The Modern Go Stack** - a cohesive technology stack for building high-performance, maintainable applications. Creates single, self-contained binaries with zero external dependencies.

**Key Features:**

- **Echo v4 + Templ + HTMX**: High-performance web framework with type-safe templates and dynamic UX
- **SQLC + PostgreSQL + pgx Driver**: Type-safe database operations with high performance and connection pooling
- **Session Authentication**: Secure session-based authentication with Argon2id password hashing
- **Tailwind CSS + DaisyUI**: Modern utility-first CSS framework with comprehensive component library
- **Enterprise Security**: CSRF protection, input sanitization, XSS/SQL injection prevention, structured error handling
- **Atlas Migrations**: Declarative schema management with automatic migration generation
- **Mage Build System**: Go-based automation with comprehensive quality checks and vulnerability scanning
- **Production Ready**: Rate limiting, CORS, security headers, graceful shutdown, and embedded static assets
- **Developer Experience**: Hot reload with Air, schema migrations with Atlas, multi-source config with Koanf

## Tech Stack

| Layer          | Technology                                                  | Purpose                                |
| -------------- | ----------------------------------------------------------- | -------------------------------------- |
| **Language**   | [Go 1.25+](https://go.dev/doc/)                             | Latest performance & language features |
| **Framework**  | [Echo v4](https://echo.labstack.com/)                       | High-performance web framework         |
| **Templates**  | [Templ](https://templ.guide/)                      | Type-safe Go HTML components           |
| **Frontend**   | [HTMX](https://htmx.org/)                             | Dynamic interactions with smooth UX    |
| **CSS**        | [Tailwind CSS](https://tailwindcss.com/) + [DaisyUI](https://daisyui.com/) | Utility-first CSS with component library |
| **Authentication** | [Session-based](https://github.com/alexedwards/scs) + [Argon2id](https://pkg.go.dev/golang.org/x/crypto/argon2) | Secure session auth with password hashing |
| **Logging**    | [slog](https://pkg.go.dev/log/slog)                         | Structured logging with JSON output    |
| **Database**   | [PostgreSQL](https://www.postgresql.org/)                   | Enterprise-grade relational database   |
| **Queries**    | [SQLC](https://sqlc.dev/)                           | Generate type-safe Go from SQL         |
| **Validation** | [go-playground/validator](https://github.com/go-playground/validator) | Comprehensive input validation |
| **DB Driver**  | [pgx v5](https://pkg.go.dev/github.com/jackc/pgx/v5)       | High-performance PostgreSQL driver with pooling |
| **Assets**     | [Go Embed](https://pkg.go.dev/embed)                        | Single binary with embedded resources  |
| **Config**     | [Koanf](https://github.com/knadh/koanf)                     | Multi-source configuration management  |
| **Migrations** | [Atlas](https://atlasgo.io/)                   | Declarative schema management          |
| **Build**      | [Mage](https://magefile.org/)                               | Go-based build automation              |
| **Hot Reload** | [Air](https://github.com/air-verse/air)                     | Development server with live reload    |

---

## Quick Start

### Ubuntu Production Deployment (Recommended)

```bash
# Clone repository
git clone https://github.com/dunamismax/go-web-server.git
cd go-web-server

# Install PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Create database and user
sudo -u postgres createdb gowebserver
sudo -u postgres createuser -P gowebserver  # Set password when prompted

# Create your environment file
cp .env.example .env
# Edit .env with your database credentials (DATABASE_USER, DATABASE_PASSWORD, etc.)

# Install Go dependencies and build
mage setup
mage build

# Run database migrations
mage migrate

# Server binary available at: bin/server
```

**Requirements:** Ubuntu 20.04+, PostgreSQL, Go 1.25+

### Local Development

```bash
# Clone and setup
git clone https://github.com/dunamismax/go-web-server.git
cd go-web-server
go mod tidy

# Create your environment file
cp .env.example .env
# Edit .env with your database credentials (DATABASE_USER, DATABASE_PASSWORD, etc.)

# Install development tools and dependencies
mage setup

# Ensure PostgreSQL is running locally
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Start development server with hot reload
mage dev

# Server starts at http://localhost:8080
```

**Requirements:**

- Go 1.25+
- Mage build tool (`go install github.com/magefile/mage@latest`)
- PostgreSQL database (local installation)
- Node.js + npm (for Tailwind CSS build)

**Note:** First run of `mage setup` installs all development tools automatically.

## Documentation

**[Complete Documentation](docs/)** - Comprehensive guides for development, deployment, security, and architecture.

| Guide | Description |
|-------|-------------|
| **[Development Guide](docs/development.md)** | Local setup, hot reload, database management, and daily workflow |
| **[API Reference](docs/api.md)** | HTTP endpoints, HTMX integration, and CSRF protection |
| **[Architecture](docs/architecture.md)** | System design, components, and technology decisions |
| **[Security Guide](docs/security.md)** | CSRF, sanitization, headers, rate limiting, and monitoring |
| **[Deployment Guide](docs/deployment.md)** | Traditional production deployment and configuration |

---

<p align="center">
  <img src="https://github.com/dunamismax/images/blob/main/golang/gopher-mage.svg" alt="Gopher Mage" width="150" />
</p>

## Mage Commands

Run `mage help` to see all available commands and their aliases.

**Development:**

```bash
mage setup (s)        # Install tools and dependencies
mage generate (g)     # Generate sqlc and templ code
mage dev (d)          # Start development server with hot reload
mage run (r)          # Build and run server
mage build (b)        # Build production binary
```

**Database:**

```bash
mage migrate (m)      # Run database migrations up
mage migrateDown      # Roll back last migration
mage migrateStatus    # Show migration status
```

**Quality & Production:**

```bash
mage fmt (f)          # Format code with goimports and tidy modules
mage vet (v)          # Run go vet static analysis
mage lint (l)         # Run golangci-lint comprehensive linting
mage vulncheck (vc)   # Check for security vulnerabilities
mage quality (q)      # Run all quality checks
mage ci               # Complete CI pipeline
mage clean (c)        # Clean build artifacts
```

**Observability & Monitoring:**

```bash
# Enable Prometheus metrics (via environment variables)
FEATURES_ENABLE_METRICS=true mage run
# Then access metrics at: http://localhost:8080/metrics

# Enhanced health checks with database connectivity
curl http://localhost:8080/health

# Test JWT authentication endpoints
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","name":"Test User","password":"StrongPass123"}'

curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"StrongPass123"}'

# Note: Demo mode currently bypasses password validation for existing users.
# For production, implement proper password hashing and validation in auth handlers.
```

## Live Demo

### Web Application (`localhost:8080`)

Interactive user management application demonstrating:

- **Session Authentication**: Login/register system with secure session-based auth and Argon2id hashing
- **CRUD Operations**: Type-safe database queries with CSRF protection
- **Real-time Updates**: HTMX interactions with smooth page transitions
- **Responsive Design**: Modern Tailwind CSS styling with DaisyUI components and multiple themes
- **Enterprise Security**: Input sanitization, XSS/SQL injection prevention, and structured error handling

<p align="center">
  <img src="https://github.com/dunamismax/images/blob/main/golang/go-web-server-screenshot.png" alt="Go Web Server Screenshot" width="800" />
</p>

> **Easter Egg**: The default user database comes pre-populated with Robert Griesemer, Rob Pike, and Ken Thompson - the three brilliant minds who created the Go programming language at Google starting in 2007. A small tribute to the creators of the language that powers this entire stack!

## Project Structure

```sh
go-web-server/
├── cmd/web/              # Application entry point with main.go
├── docs/                 # Complete documentation
├── internal/
│   ├── config/           # Viper configuration management
│   ├── handler/          # HTTP handlers (auth, home, user, routes)
│   ├── middleware/       # Security, auth, CSRF, validation, error handling, metrics
│   ├── store/            # Database layer with SQLC (models, queries, migrations)
│   │   └── migrations/   # Goose database migrations
│   ├── ui/               # Static assets (embedded CSS, JS, favicon)
│   └── view/             # Templ templates and components
├── scripts/              # Deployment scripts and systemd service
├── bin/                  # Compiled binaries
├── tmp/                  # Development hot reload directory  
├── magefile.go          # Mage build automation with comprehensive commands
├── .golangci.yml        # Linter configuration
├── sqlc.yaml            # SQLC configuration
├── go.mod/go.sum        # Go module dependencies
└── .env.example         # Environment configuration template

```

---

<p align="center">
  <img src="https://github.com/dunamismax/images/blob/main/golang/gopher-aviator.jpg" alt="Go Gopher" width="400" />
</p>

## Ubuntu SystemD Deployment

```bash
# Build optimized binary for production deployment
mage build  # Creates optimized binary in bin/server (~15MB)

# Create systemd service file
sudo cp scripts/gowebserver.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable gowebserver
sudo systemctl start gowebserver
```

The binary includes embedded Tailwind CSS, DaisyUI, HTMX, and Templ templates. **Single binary deployment** with local PostgreSQL backend. Perfect for traditional Ubuntu servers behind Caddy reverse proxy with Cloudflare DNS.

## Key Features Demonstrated

**Modern Web Stack:**

- Echo framework with comprehensive middleware stack (recovery, CORS, rate limiting, timeouts)
- Session-based authentication with Argon2id password hashing and PostgreSQL session store
- Type-safe Templ templates with reusable components and embedded static assets
- HTMX dynamic interactions with smooth page transitions and custom events
- Tailwind CSS + DaisyUI styling with comprehensive component library and multiple themes
- SQLC type-safe database queries with high-performance pgx driver and connection pooling
- Structured logging with slog and configurable JSON/text output

**Developer Experience:**

- Hot reloading with Air for rapid development
- Comprehensive error handling with custom error types and structured logging
- Static analysis suite (golangci-lint, govulncheck, go vet)
- Mage build automation with goimports, templ formatting, and vulnerability scanning
- Single-command CI pipeline with quality checks and linting
- Environment-based configuration with sensible defaults

**Production Ready:**

- Enterprise security with CSRF protection, input sanitization, and XSS/SQL injection prevention
- Session-based authentication with Argon2id password hashing and PostgreSQL session store
- Structured error handling with request tracing, correlation IDs, and monitoring
- Multi-source configuration with Koanf supporting JSON, YAML, ENV, and .env files
- Atlas declarative schema management with automatic migration generation
- Single binary deployment (~15MB) with embedded assets (CSS, JS, templates)
- Comprehensive middleware stack with rate limiting, CORS, security headers, and timeouts

---

<p align="center">
  <a href="https://buymeacoffee.com/dunamismax" target="_blank">
    <img src="https://github.com/dunamismax/images/blob/main/golang/buy-coffee-go.gif" alt="Buy Me A Coffee" style="height: 150px !important;" />
  </a>
</p>

<p align="center">
  <a href="https://twitter.com/dunamismax" target="_blank"><img src="https://img.shields.io/badge/Twitter-%231DA1F2.svg?&style=for-the-badge&logo=twitter&logoColor=white" alt="Twitter"></a>
  <a href="https://bsky.app/profile/dunamismax.bsky.social" target="_blank"><img src="https://img.shields.io/badge/Bluesky-blue?style=for-the-badge&logo=bluesky&logoColor=white" alt="Bluesky"></a>
  <a href="https://reddit.com/user/dunamismax" target="_blank"><img src="https://img.shields.io/badge/Reddit-%23FF4500.svg?&style=for-the-badge&logo=reddit&logoColor=white" alt="Reddit"></a>
  <a href="https://discord.com/users/dunamismax" target="_blank"><img src="https://img.shields.io/badge/Discord-dunamismax-7289DA.svg?style=for-the-badge&logo=discord&logoColor=white" alt="Discord"></a>
  <a href="https://signal.me/#p/+dunamismax.66" target="_blank"><img src="https://img.shields.io/badge/Signal-dunamismax.66-3A76F0.svg?style=for-the-badge&logo=signal&logoColor=white" alt="Signal"></a>
</p>

## License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>The Modern Go Stack</strong><br>
  <sub>Echo • Templ • HTMX • Sessions • SQLC • PostgreSQL • pgx • Tailwind • DaisyUI • slog • Koanf • Atlas • Mage • Air</sub>
</p>

<p align="center">
  <img src="https://github.com/dunamismax/images/blob/main/golang/gopher-running-jumping.gif" alt="Gopher Running and Jumping" width="600" />
</p>

---

"The "Modern Go Stack" is a powerful and elegant solution that aligns beautifully with Go's core principles. It is an excellent starting point for many new projects, and any decision to deviate from it should be driven by specific, demanding requirements." - Me

---
