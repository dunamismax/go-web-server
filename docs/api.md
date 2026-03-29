# API

Base URL: `http://localhost:8080`

This repo is mostly a server-rendered app, not a large JSON API. Many endpoints return full HTML pages or HTMX fragments instead of JSON.

## Response Modes

- Regular browser requests usually return full HTML pages.
- HTMX requests usually return HTML fragments.
- A few endpoints such as `/health` and `/demo` return JSON for non-HTMX clients.

## Public Routes

| Method | Path | Response | Notes |
| --- | --- | --- | --- |
| `GET` | `/` | HTML page or HTMX fragment | Home page |
| `GET` | `/demo` | HTMX fragment or JSON | Demo payload for UI interactions |
| `GET` | `/health` | HTMX fragment or JSON | Health response with database check |
| `GET` | `/auth/login` | HTML page or HTMX fragment | Login form |
| `GET` | `/auth/register` | HTML page or HTMX fragment | Registration form |
| `POST` | `/auth/login` | Redirect or HTMX redirect payload | Creates a session on success |
| `POST` | `/auth/register` | Redirect or HTMX redirect payload | Creates a user and session on success |
| `POST` | `/auth/logout` | Redirect or HTMX redirect payload | Destroys the current session if present |
| `GET` | `/static/*` | Static files | Embedded CSS, JS, images, favicon |

## Protected Routes

These routes require an authenticated session.

| Method | Path | Response | Notes |
| --- | --- | --- | --- |
| `GET` | `/profile` | HTML page or HTMX fragment | Profile page |
| `GET` | `/users` | HTML page or HTMX fragment | User management screen |
| `GET` | `/users/list` | HTML fragment | User list partial |
| `GET` | `/users/form` | HTML fragment | New-user form partial |
| `GET` | `/users/:id/edit` | HTML fragment | Edit-user form partial |
| `POST` | `/users` | HTML fragment | Create user and return refreshed list |
| `PUT` | `/users/:id` | HTML fragment | Update user and return refreshed list |
| `PATCH` | `/users/:id/deactivate` | HTML fragment | Soft deactivate user and return updated row |
| `DELETE` | `/users/:id` | Empty `200 OK` | Hard delete user |
| `GET` | `/api/users/count` | HTML fragment | Active user count widget, despite the `/api` prefix |

## Auth Behavior

- Browser requests without a session are redirected to `/auth/login`.
- HTMX or JSON-style requests without a session receive `401 Unauthorized`.
- Registered users get Argon2id password hashes.
- Accounts without a usable password hash are rejected during login.

## CSRF

- `POST`, `PUT`, `PATCH`, and `DELETE` require a CSRF token.
- Tokens are issued in the `_csrf` cookie.
- The middleware accepts the token from the `X-CSRF-Token` header or a `csrf_token` form field.
- The app refreshes the token after successful state-changing requests and sends the updated token back through response headers and rendered markup.

## Health Endpoint

`GET /health` returns JSON like:

```json
{
  "status": "ok",
  "timestamp": "2026-03-07T15:04:05Z",
  "service": "go-web-server",
  "version": "4.0.0",
  "uptime": "12m3s",
  "checks": {
    "database": "ok",
    "database_connections": "ok",
    "memory": "ok"
  }
}
```

Status codes:

- `200 OK`: healthy
- `206 Partial Content`: warning or degraded
- `503 Service Unavailable`: unhealthy
