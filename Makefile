SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------
# Pinned tool versions. Local runs, CI and Docker all go through this file, so
# a bump here moves every environment at once. templ is pinned in go.mod (as a
# "tool" dependency) because its generator must match its runtime library.
# ---------------------------------------------------------------------------
SQLC_VERSION          := v1.31.1
GOLANGCI_LINT_VERSION := v2.15.0
GOVULNCHECK_VERSION   := v1.10.0
TAILWIND_VERSION      := v4.3.5
HTMX_VERSION          := 2.0.11
HTMX_SHA256           := d6fdc75f204e6bdefa99b69bf1e6d4ac69b8a364f77929f45c13476b4000f717

# SHA-256 of each Tailwind standalone binary, from the release's sha256sums.txt.
TAILWIND_SHA256_linux-arm64 := ffcd0d21dbd7c605dc12743ff0c0b47debc963c86b399674084c1c8f2c96ecad
TAILWIND_SHA256_linux-x64   := e47695cee3f0d90623a91f3939b08bdee610eb5ad99340adff9d4145b284d6cc
TAILWIND_SHA256_macos-arm64 := f3f4f5d1727e2b693b8f1c9a1492b5dd87008c86ea286bee1589cb458e6235fc
TAILWIND_SHA256_macos-x64   := a63702686a3ed048b3135d84a73be49ea9d47f880c9b20e54b3ef60934ad8cdf

MODULE := $(shell go list -m)

SQLC          := go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOVULNCHECK   := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
TEMPL         := go tool templ

# sha256sum on Linux, shasum on macOS.
SHA256SUM := $(shell command -v sha256sum >/dev/null 2>&1 && echo sha256sum || echo shasum -a 256)

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
TAILWIND_OS   := $(if $(filter Darwin,$(UNAME_S)),macos,linux)
TAILWIND_ARCH := $(if $(filter arm64 aarch64,$(UNAME_M)),arm64,x64)
TAILWIND_PLATFORM := $(TAILWIND_OS)-$(TAILWIND_ARCH)
TAILWIND := bin/tailwindcss-$(TAILWIND_VERSION)

CSS_IN  := internal/assets/tailwind.css
CSS_OUT := internal/assets/static/app.css
HTMX    := internal/assets/static/htmx.min.js

# The database "make db-up" starts. Override either variable to point the
# tests at a different server.
DEV_DATABASE_URL  := postgres://gowebserver:gowebserver@localhost:5432/gowebserver?sslmode=disable
TEST_DATABASE_URL ?= $(DEV_DATABASE_URL)

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# --- Development -----------------------------------------------------------

.PHONY: setup
setup: $(TAILWIND) ## Download Go modules and the Tailwind CLI
	go mod download
	@test -f .env || { cp .env.example .env && echo "Created .env from .env.example"; }

.PHONY: dev
dev: $(TAILWIND) ## Run with live reload at http://localhost:7331
	@trap 'kill 0' INT TERM EXIT; \
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --minify --watch=always & \
	$(TEMPL) generate --watch --proxy="http://localhost:8080" --open-browser=false \
		--cmd="go run ./cmd/server" & \
	wait

.PHONY: run
run: ## Build and run the server once
	go run ./cmd/server

# --- Code generation -------------------------------------------------------

.PHONY: generate
generate: sqlc templ css ## Regenerate all checked-in generated code

.PHONY: sqlc
sqlc: ## Generate Go from internal/store/queries and migrations
	$(SQLC) generate

.PHONY: templ
templ: ## Generate Go from .templ files
	$(TEMPL) generate

.PHONY: css
css: $(TAILWIND) ## Compile the Tailwind stylesheet
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --minify

.PHONY: htmx
htmx: ## Re-download the pinned htmx build
	curl -fsSL -o $(HTMX) https://unpkg.com/htmx.org@$(HTMX_VERSION)/dist/htmx.min.js
	echo "$(HTMX_SHA256)  $(HTMX)" | $(SHA256SUM) -c -

$(TAILWIND):
	@mkdir -p bin
	@test -n "$(TAILWIND_SHA256_$(TAILWIND_PLATFORM))" || \
		{ echo "No pinned Tailwind checksum for $(TAILWIND_PLATFORM)"; exit 1; }
	curl -fsSL -o $@.tmp \
		https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_PLATFORM)
	echo "$(TAILWIND_SHA256_$(TAILWIND_PLATFORM))  $@.tmp" | $(SHA256SUM) -c -
	chmod +x $@.tmp
	mv $@.tmp $@

# --- Quality ---------------------------------------------------------------

.PHONY: fmt
fmt: ## Format Go and templ sources
	$(GOLANGCI_LINT) fmt
	$(TEMPL) fmt internal

.PHONY: lint
lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: test
test: ## Run all tests, including the PostgreSQL-backed ones
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -race -count=1 ./...

.PHONY: test-short
test-short: ## Run only the tests that need no database
	TEST_DATABASE_URL= go test -count=1 ./...

.PHONY: vuln
vuln: ## Scan dependencies for known vulnerabilities
	$(GOVULNCHECK) ./...

.PHONY: check-generated
check-generated: generate ## Fail if generated code is out of date
	@git diff --exit-code -- internal/store internal/view $(CSS_OUT) || \
		{ echo; echo "Generated code is stale. Run 'make generate' and commit the result."; exit 1; }

.PHONY: check
check: check-generated lint test vuln build ## Everything CI runs

# --- Build -----------------------------------------------------------------

# The version reported by "server version" and logged at startup: the Git tag
# on a release commit, otherwise tag-commits-hash. Docker builds cannot see
# .git, so they receive it as a build argument.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null)
LDFLAGS := -s -w -X $(MODULE)/internal/version.override=$(VERSION)

.PHONY: build
build: ## Build the production binary into bin/server
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/server ./cmd/server

.PHONY: docker
docker: ## Build the container image
	docker build --build-arg VERSION="$(VERSION)" -t go-web-server .

.PHONY: clean
clean: ## Remove build output
	rm -rf bin/server tmp

# --- Database --------------------------------------------------------------

.PHONY: db-up
db-up: ## Start PostgreSQL in Docker and wait until it is ready
	docker compose up -d --wait db

.PHONY: db-down
db-down: ## Stop PostgreSQL (the data volume is kept)
	docker compose down

.PHONY: db-reset
db-reset: ## Destroy the development database and start a fresh one
	docker compose down --volumes
	docker compose up -d --wait db

.PHONY: migrate
migrate: ## Apply pending migrations
	go run ./cmd/server migrate up

.PHONY: migrate-down
migrate-down: ## Roll back the most recent migration
	go run ./cmd/server migrate down

.PHONY: migrate-status
migrate-status: ## Show which migrations are applied
	go run ./cmd/server migrate status

.PHONY: migration
migration: ## Create a migration: make migration name=add_widgets
	@test -n "$(name)" || { echo "usage: make migration name=add_widgets"; exit 1; }
	@file="internal/store/migrations/$$(date -u +%Y%m%d%H%M%S)_$(name).sql"; \
	printf -- '-- +goose Up\n\n-- +goose Down\n' > "$$file"; \
	echo "Created $$file"

# --- Template ---------------------------------------------------------------

.PHONY: rename
rename: ## Change the Go module path: make rename module=github.com/you/app
	@test -n "$(module)" || { echo "usage: make rename module=github.com/you/app"; exit 1; }
	grep -rlF "$(MODULE)" --include='*.go' --include='*.templ' --include=go.mod \
		--include=Dockerfile --include=.golangci.yml . \
		| xargs perl -pi -e 's{\Q$(MODULE)\E}{$(module)}g'
	go build ./...
	@echo "Module is now $(module)"
