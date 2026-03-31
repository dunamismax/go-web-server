# API

Base URL: `http://localhost:8080`

This repo still ships a server-rendered Templ + HTMX browser path. Phase 2 adds a parallel JSON contract under `/api/*` so later Astro + Vue work can use stable backend contracts without reading handler or template code.

## Contract rules

- Session auth is same-origin cookie auth. The authenticated API routes use the same session as the legacy pages.
- Safe requests (`GET`, `HEAD`, `OPTIONS`) expose the current CSRF token through the `X-CSRF-Token` response header.
- State-changing requests (`POST`, `PUT`, `PATCH`, `DELETE`) must send the CSRF token through the `X-CSRF-Token` header or a `csrf_token` form field.
- The CSRF token rotates after every successful state-changing request. Frontend code should replace its cached token with the latest `X-CSRF-Token` response header.
- API requests without a valid session return `401 Unauthorized` JSON. Legacy browser page requests still redirect to `/auth/login`.
- Error responses use the shared JSON envelope from the Echo error handler.

## Error shape

All API errors use this JSON structure:

```json
{
  "type": "validation",
  "error": "Bad Request",
  "message": "Validation failed",
  "details": [
    {
      "field": "email",
      "message": "invalid email format",
      "tag": "email"
    }
  ],
  "code": 400,
  "path": "/api/auth/login",
  "method": "POST",
  "request_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "timestamp": "1774898342"
}
```

Notes:

- `type` is one of `validation`, `authentication`, `not_found`, `conflict`, `csrf`, `internal`, and the other middleware error categories.
- `details` is omitted when there is nothing useful to return.
- `timestamp` is currently emitted as a Unix-seconds string by the shared error handler.

## JSON data models

`SessionUser`

```json
{
  "id": 1,
  "email": "user@example.com",
  "name": "Example User",
  "is_active": true
}
```

`ManagedUser`

```json
{
  "id": 12,
  "email": "user@example.com",
  "name": "Example User",
  "avatar_url": null,
  "bio": null,
  "is_active": true,
  "created_at": "2026-03-30T12:00:00Z",
  "updated_at": "2026-03-30T12:00:00Z"
}
```

Notes:

- `created_at` and `updated_at` are UTC RFC3339 timestamps.
- `/api/users` and `/api/users/count` only cover active users.
- `/api/users/:id` can return inactive users when they still exist in the database. That is intentional so edit and post-deactivate views can inspect the record that was changed.

## Public JSON routes

### `GET /api/auth/state`

Purpose: bootstrap frontend auth state and CSRF handling.

Auth required: no.

Response `200 OK`:

```json
{
  "authenticated": false,
  "user": null,
  "csrf": {
    "header": "X-CSRF-Token",
    "form_field": "csrf_token",
    "token": "current-csrf-token"
  }
}
```

When a session exists, `authenticated` becomes `true` and `user` contains `SessionUser`.

### `POST /api/auth/login`

Purpose: create a session without using redirect or HTMX response shapes.

Auth required: no.

Request body:

```json
{
  "email": "user@example.com",
  "password": "Password1"
}
```

Response `200 OK`:

```json
{
  "message": "Login successful",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "Example User",
    "is_active": true
  }
}
```

Error cases:

- `400` invalid JSON or validation failure
- `401` invalid credentials or inactive account
- `403` invalid CSRF token

### `POST /api/auth/register`

Purpose: create a user account and immediately create a session.

Auth required: no.

Request body:

```json
{
  "email": "user@example.com",
  "name": "Example User",
  "password": "Password1",
  "confirm_password": "Password1",
  "bio": "Optional short bio",
  "avatar_url": "https://example.com/avatar.png"
}
```

Optional fields: `bio`, `avatar_url`.

Response `201 Created`:

```json
{
  "message": "Registration successful",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "Example User",
    "is_active": true
  }
}
```

Error cases:

- `400` validation failure
- `409` duplicate email
- `403` invalid CSRF token

### `POST /api/auth/logout`

Purpose: destroy the current session without redirect behavior.

Auth required: no. The endpoint is safe to call even if the session is already gone.

Response `200 OK`:

```json
{
  "message": "Logout successful"
}
```

## Protected JSON routes

These routes require an authenticated active session.

### `GET /api/users`

Purpose: fetch the active user list for the new frontend.

Response `200 OK`:

```json
{
  "users": [
    {
      "id": 12,
      "email": "user@example.com",
      "name": "Example User",
      "avatar_url": null,
      "bio": null,
      "is_active": true,
      "created_at": "2026-03-30T12:00:00Z",
      "updated_at": "2026-03-30T12:00:00Z"
    }
  ],
  "count": 1
}
```

### `GET /api/users/count`

Purpose: fetch the active user count as JSON.

Response `200 OK`:

```json
{
  "count": 1
}
```

### `GET /api/users/:id`

Purpose: fetch a single user record for edit flows.

Response `200 OK`:

```json
{
  "user": {
    "id": 12,
    "email": "user@example.com",
    "name": "Example User",
    "avatar_url": null,
    "bio": null,
    "is_active": true,
    "created_at": "2026-03-30T12:00:00Z",
    "updated_at": "2026-03-30T12:00:00Z"
  }
}
```

Error cases:

- `400` invalid `:id`
- `404` missing user

### `POST /api/users`

Purpose: create a managed user from the protected CRUD surface.

Request body:

```json
{
  "email": "user@example.com",
  "name": "Example User",
  "password": "Password1",
  "confirm_password": "Password1",
  "bio": "Optional short bio",
  "avatar_url": "https://example.com/avatar.png"
}
```

Response `201 Created`:

```json
{
  "message": "User created successfully",
  "user": {
    "id": 12,
    "email": "user@example.com",
    "name": "Example User",
    "avatar_url": null,
    "bio": null,
    "is_active": true,
    "created_at": "2026-03-30T12:00:00Z",
    "updated_at": "2026-03-30T12:00:00Z"
  }
}
```

Error cases:

- `400` validation failure
- `409` duplicate email
- `403` invalid CSRF token

### `PUT /api/users/:id`

Purpose: update a managed user.

Request body:

```json
{
  "email": "user@example.com",
  "name": "Updated User",
  "password": "Password2",
  "confirm_password": "Password2",
  "bio": "Optional short bio",
  "avatar_url": "https://example.com/avatar.png"
}
```

Notes:

- `email` and `name` are required.
- `password` and `confirm_password` are both optional.
- If either password field is supplied, both are required and must match.

Response `200 OK`:

```json
{
  "message": "User updated successfully",
  "user": {
    "id": 12,
    "email": "user@example.com",
    "name": "Updated User",
    "avatar_url": null,
    "bio": null,
    "is_active": true,
    "created_at": "2026-03-30T12:00:00Z",
    "updated_at": "2026-03-30T12:10:00Z"
  }
}
```

Error cases:

- `400` validation failure or invalid `:id`
- `404` missing user
- `409` duplicate email
- `403` invalid CSRF token

### `PATCH /api/users/:id/deactivate`

Purpose: soft deactivate a user.

Request body: none.

Response `200 OK`:

```json
{
  "message": "User deactivated successfully",
  "user": {
    "id": 12,
    "email": "user@example.com",
    "name": "Updated User",
    "avatar_url": null,
    "bio": null,
    "is_active": false,
    "created_at": "2026-03-30T12:00:00Z",
    "updated_at": "2026-03-30T12:15:00Z"
  }
}
```

Error cases:

- `400` invalid `:id`
- `404` missing user
- `403` invalid CSRF token

### `DELETE /api/users/:id`

Purpose: hard delete a user.

Request body: none.

Response `200 OK`:

```json
{
  "id": 12,
  "deleted": true,
  "message": "User deleted successfully"
}
```

Error cases:

- `400` invalid `:id`
- `404` missing user
- `403` invalid CSRF token

## Legacy HTML and HTMX routes that still exist

These routes still power the shipped browser UI. They remain in place while the remaining legacy HTMX path is retired. The `/users` page now owns its inline create and edit form state and renders its current count and list inline, so the old `/users/list`, `/users/count`, `/users/form`, and `/users/:id/edit` fragments are gone.

| Method | Path | Response shape | Notes |
| --- | --- | --- | --- |
| `GET` | `/` | HTML page or HTMX fragment | Home page |
| `GET` | `/demo` | JSON or HTMX fragment | Utility demo endpoint |
| `GET` | `/health` | JSON or HTMX fragment | Health endpoint |
| `GET` | `/auth/login` | HTML page or HTMX fragment | Legacy login page |
| `GET` | `/auth/register` | HTML page or HTMX fragment | Legacy registration page |
| `POST` | `/auth/login` | Redirect or HTMX redirect payload | Legacy login submit |
| `POST` | `/auth/register` | Redirect or HTMX redirect payload | Legacy registration submit |
| `POST` | `/auth/logout` | Redirect or HTMX redirect payload | Legacy logout submit |
| `GET` | `/profile` | HTML page or HTMX fragment | Legacy profile page |
| `GET` | `/users` | HTML page or HTMX fragment | Legacy users screen with inline list, count, and inline form state rendering |
| `POST` | `/users` | HTML fragment | Legacy create submit |
| `PUT` | `/users/:id` | HTML fragment | Legacy update submit |
| `PATCH` | `/users/:id/deactivate` | HTML fragment | Legacy deactivate submit |
| `DELETE` | `/users/:id` | Empty `200 OK` | Legacy delete submit |
| `GET` | `/static/*` | Static files | Embedded legacy assets |

## Health endpoint

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
