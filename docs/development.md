# Development

## Requirements

- Go 1.27 or newer
- `make` and `curl`
- PostgreSQL 14 or newer, or Docker to run it for you

Everything else is fetched on demand at a pinned version:

| Tool | How it is run | Where the version lives |
| --- | --- | --- |
| templ | `go tool templ` | `go.mod` (a `tool` dependency, so the generator always matches the runtime library) |
| sqlc, golangci-lint, govulncheck | `go run <pkg>@<version>` | top of the `Makefile` |
| Tailwind CLI | downloaded to `bin/`, SHA-256 verified | top of the `Makefile` |
| htmx | vendored at `internal/assets/static/htmx.min.js` | top of the `Makefile` |

The first `make lint` and `make sqlc` compile their tools, which takes a
minute or two. After that the Go build cache makes them instant.

## First run

```bash
make setup
make db-up
make dev
```

Then open <http://localhost:7331>.

`make dev` runs two watchers:

- `templ generate --watch` regenerates Go from `.templ` files, restarts the
  server when Go code changes, and reloads the browser through its proxy on
  port 7331.
- The Tailwind CLI rebuilds `internal/assets/static/app.css` when the classes
  used in the templates change.

In development the server reads static files from disk, so a CSS rebuild
shows up on the next page load without a restart.

## What to run after changing what

| You changed | Run |
| --- | --- |
| A `.templ` file | `make templ` (or let `make dev` do it) |
| `internal/assets/tailwind.css`, or classes in a template | `make css` |
| A file in `internal/store/queries/` or `migrations/` | `make sqlc` |
| Any of the above, before committing | `make generate` |
| Go code | `make fmt lint test` |

`make check` runs what CI runs.

## Adding a database table

1. Create the migration:

   ```bash
   make migration name=create_widgets
   ```

   Fill in both sections of the new file. Always write the `Down` section;
   a test rolls every migration back and reapplies it.

   ```sql
   -- +goose Up
   CREATE TABLE widgets (
       id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
       owner_id   BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
       name       TEXT NOT NULL,
       created_at TIMESTAMPTZ NOT NULL DEFAULT now()
   );

   -- +goose Down
   DROP TABLE widgets;
   ```

2. Write queries in `internal/store/queries/widgets.sql`:

   ```sql
   -- name: ListWidgetsByOwner :many
   SELECT * FROM widgets WHERE owner_id = $1 ORDER BY created_at DESC;
   ```

3. Run `make sqlc`. The new methods appear on `store.Store`.

4. Restart the app (or run `make migrate`) to apply the migration.

## Adding a page

1. **View.** Add a component to a `.templ` file in `internal/view` that
   takes a `Page` and wraps its content in `@Layout(p) { ... }`.
2. **Handler.** Add a method on `*Server` in `internal/web` with the
   signature `func(w http.ResponseWriter, r *http.Request) error`, ending in
   `return render(w, r, http.StatusOK, view.YourPage(s.page(r, "Title")))`.
3. **Route.** Register it in `internal/web/routes.go`, wrapped in
   `s.requireUser(...)` or `s.requireAdmin(...)` if it needs a signed-in
   user.

For a form, copy the shape of `register` in
`internal/web/handlers_auth.go`: a form struct in `internal/view/view.go`,
validation into `view.Errors`, re-render with 422 on failure, flash and
redirect on success.

## Tests

```bash
make test         # everything, with the race detector
make test-short   # only tests that need no database
```

`make test` expects the `make db-up` database. To use another server:

```bash
TEST_DATABASE_URL=postgres://user:pass@localhost:5433/dbname make test
```

The role needs permission to create schemas in that database. Each test
calls `storetest.New(t)`, which creates a uniquely named schema, migrates
it, and drops it when the test ends. Tests are isolated from each other and
from any other data in the database, and run in parallel.

The end-to-end tests in `internal/web` start the complete handler stack on
an `httptest.Server` and drive it with a cookie-carrying HTTP client, the
way a browser would. `newApp(t)` and `a.newBrowser()` in
`internal/web/web_test.go` are the helpers to reuse.

## Upgrading tools and dependencies

- **Go modules:** `go get -u ./... && go mod tidy`, then `make check`.
- **templ:** `go get -tool github.com/a-h/templ/cmd/templ@latest`, then
  `make templ`.
- **sqlc, golangci-lint, govulncheck:** change the version at the top of the
  `Makefile`, then `make generate` or `make lint`.
- **Tailwind:** change `TAILWIND_VERSION` and the four `TAILWIND_SHA256_*`
  values (from the release's `sha256sums.txt`), then `make css`.
- **htmx:** change `HTMX_VERSION` and `HTMX_SHA256`, then `make htmx`.
  htmx 4 is published under the `next` tag and changes how requests are
  made; this template stays on the stable 2.x line.
- **Go itself:** change the `go` and `toolchain` lines in `go.mod` and the
  image tag in the `Dockerfile`.

After any upgrade, `make check` is the gate: it regenerates the checked-in
code, lints, runs the full test suite and scans for known vulnerabilities.
