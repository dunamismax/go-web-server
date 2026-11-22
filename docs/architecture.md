# Architecture

One Go process serves HTML rendered on the server and talks to one PostgreSQL
database. There is no separate frontend application and no API layer between
the pages and the data.

```text
browser
   │  HTML over HTTP (forms, links, htmx requests)
   ▼
net/http server ── middleware ── handlers ── templ components
                                    │
                                    ▼
                          store (pgx + sqlc) ── PostgreSQL
```

## Packages

| Package | Responsibility | Depends on |
| --- | --- | --- |
| `cmd/server` | Parse the command, load config, wire everything, run and shut down | all of the below |
| `internal/config` | Read and validate environment variables | — |
| `internal/store` | Connection pool, embedded migrations, generated queries | pgx, goose |
| `internal/auth` | Hash and verify passwords | `x/crypto` |
| `internal/assets` | Embed and serve static files with cache-busting URLs | — |
| `internal/view` | templ components and the plain structs that feed them | `store` (for the `User` type), `assets`, `auth` |
| `internal/web` | Routes, middleware, sessions, handlers | everything above |
| `internal/version` | Report the build's version from embedded VCS data | — |

Dependencies point one way: `web` uses `view` and `store`; neither knows
about `web`. `store` knows nothing about HTTP, and `view` performs no I/O.

## The life of a request

Routes and the middleware order are defined in one place,
[`internal/web/routes.go`](../internal/web/routes.go).

1. **`requestID`** assigns an ID, returned in `X-Request-Id` and attached to
   every log line for the request.
2. **`accessLog`** records method, path, status, size, duration and client
   address when the request finishes.
3. **`recoverPanic`** converts a panic into a logged 500 page.
4. **`secureHeaders`** sets the Content Security Policy and related headers.
5. **`crossOriginProtection`** rejects state-changing requests from other
   origins. This is the CSRF defense.
6. **`limitBody`** caps request bodies at 1 MB.
7. Static files and the health probes are served here, without a session.
8. **Session load** (scs) reads the session row for the cookie, if any.
9. **`loadUser`** looks up the session's user and puts it in the request
   context. A missing, deactivated or password-changed account ends the
   session on the spot.
10. The **route's handler** runs, wrapped in `requireUser`, `requireAdmin`,
    `guestOnly` or `rateLimited` as the route table says.

## Handlers

Handlers have the signature `func(w, r) error`. Returning an error sends the
request to one function, `handleError`, which decides the status, logs what
needs logging, and renders the error page. Handlers therefore read top to
bottom with no response-writing in their error branches.

A form handler follows the same shape every time:

1. Parse the form into a struct from `internal/view` (for example
   `view.RegisterForm`).
2. Validate, collecting messages per field in `view.Errors`.
3. On failure, re-render the same page with status 422. The struct carries
   the user's input and the messages back into the template.
4. On success, write to the database, set a one-time flash message, and
   redirect with 303.

## Rendering

Each page is a templ component that takes a `view.Page` (title, current
user, flash messages) plus whatever data it shows, and wraps itself in
`view.Layout`. Components render into a buffer before anything is written to
the client, so a failure partway through produces a clean error page.

Data reaches templates through explicit arguments, never through the request
context, so the compiler checks that every page gets what it needs.

## htmx as progressive enhancement

The app is built as ordinary pages and forms first. htmx is layered on in two
places on the users page:

- The search box requests `/users?q=...` as you type. The handler notices the
  `HX-Target` header and renders only the `UserResults` component.
- The Activate/Deactivate buttons post with htmx and receive the refreshed
  results region. Without JavaScript the same forms post normally and
  redirect.

The pattern to copy: write the full page, extract the part that changes into
its own component, and return that component alone when htmx asks for it.

Two details keep this robust:

- After a successful post, `redirect` answers htmx requests with an
  `HX-Redirect` header. htmx would otherwise follow the 303 itself and swap
  the destination page into the fragment's target.
- htmx is configured (in `view.htmxConfig`) to run without inline styles or
  `eval`, so it works under the Content Security Policy.

## Data access

`internal/store/migrations` is the single source of truth for the schema.
goose applies those files; sqlc reads the same files to learn the schema and
generates Go for the queries in `internal/store/queries`. The generated files
(`db.go`, `models.go`, `*.sql.go`) are checked in.

`store.Store` embeds the generated `Queries`, so handlers call
`s.store.GetUser(ctx, id)` directly. `Store.InTx` runs several queries in one
transaction.

## Sessions and authentication

Sessions live in the `sessions` table. A session stores only the user's ID
and the user's `password_changed_at` value from sign-in time. On every
request `loadUser` reads the user row and compares:

- no such user, or `is_active` is false → session destroyed;
- `password_changed_at` differs → session destroyed (the password was
  changed on another device, or reset by an admin);
- otherwise the user is attached to the request.

That one primary-key lookup per request is what makes deactivation, deletion
and password changes take effect immediately, with no session-revocation
list to maintain.

## Generated code is checked in

sqlc output, templ output and the compiled stylesheet are committed. A fresh
clone builds with only Go installed, the Docker build needs no extra tools,
and `go vet`, editors and CI work without a generate step. CI regenerates
all three with the pinned tool versions and fails if the result differs from
what is committed.
