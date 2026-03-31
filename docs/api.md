# API And Route Behavior

This repo now ships an embedded Astro + Vue browser frontend for the main page routes. The frontend talks to the Go backend through explicit same-origin JSON contracts under `/api/*`. The only non-API browser mutation endpoints left are the redirect-oriented auth form posts.

## CSRF Expectations

- CSRF protection applies to state-changing requests.
- The frontend bootstraps the current token from `/api/auth/state` and/or the `X-CSRF-Token` response header.
- JSON writes should send `X-CSRF-Token`.
- Browser auth form posts can also submit the token through the existing form middleware path.

## Browser Pages

| Method | Path | Response | Notes |
| --- | --- | --- | --- |
| `GET` | `/` | HTML | Embedded Astro home page |
| `GET` | `/auth/login` | HTML | Embedded Astro login page |
| `GET` | `/auth/register` | HTML | Embedded Astro registration page |
| `GET` | `/auth/logout` | HTML | Embedded Astro logout page |
| `GET` | `/profile` | HTML | Embedded Astro profile page, requires auth |
| `GET` | `/users` | HTML | Embedded Astro users page, requires auth |
| `GET` | `/_astro/*` | Static assets | Embedded frontend asset files |

## Utility Endpoints

| Method | Path | Response | Notes |
| --- | --- | --- | --- |
| `GET` | `/demo` | JSON | Simple backend connectivity payload |
| `GET` | `/health` | JSON | Health check payload |

## Browser Auth Fallback Submits

These are simple browser-oriented endpoints that redirect after success:

| Method | Path | Success behavior |
| --- | --- | --- |
| `POST` | `/auth/login` | `302` redirect to `/` |
| `POST` | `/auth/register` | `302` redirect to `/` |
| `POST` | `/auth/logout` | `302` redirect to `/auth/login` |

## JSON Auth Contracts

### `GET /api/auth/state`

Purpose: bootstrap current session state and CSRF details for the frontend.

Example response:

```json
{
  "authenticated": true,
  "user": {
    "id": 7,
    "email": "user@example.com",
    "name": "Example User",
    "is_active": true
  },
  "csrf": {
    "header": "X-CSRF-Token",
    "token": "csrf-token-value"
  }
}
```

### `POST /api/auth/login`

Request body:

```json
{
  "email": "user@example.com",
  "password": "Password1"
}
```

Response:

```json
{
  "message": "Login successful",
  "user": {
    "id": 7,
    "email": "user@example.com",
    "name": "Example User",
    "is_active": true
  }
}
```

### `POST /api/auth/register`

Request body:

```json
{
  "email": "user@example.com",
  "name": "Example User",
  "password": "Password1",
  "confirm_password": "Password1",
  "bio": "Optional bio",
  "avatar_url": "https://example.com/avatar.png"
}
```

Success returns `201 Created` plus the same response shape as login.

### `POST /api/auth/logout`

Success response:

```json
{
  "message": "Logout successful",
  "user": null
}
```

## JSON User Contracts

### `GET /api/users`

Returns active managed users.

```json
{
  "users": [
    {
      "id": 7,
      "email": "user@example.com",
      "name": "Example User",
      "bio": "Optional bio",
      "avatar_url": "https://example.com/avatar.png",
      "is_active": true,
      "created_at": "2026-03-30T20:15:00Z",
      "updated_at": "2026-03-30T20:15:00Z"
    }
  ],
  "count": 1
}
```

### `GET /api/users/count`

```json
{
  "count": 1
}
```

### `GET /api/users/:id`

Returns a single managed user record for edit flows.

### `POST /api/users`

Creates a managed user.

### `PUT /api/users/:id`

Updates a managed user.

### `PATCH /api/users/:id/deactivate`

Deactivates a managed user.

### `DELETE /api/users/:id`

Deletes a managed user.

All successful mutation responses share this shape:

```json
{
  "message": "User updated successfully",
  "user": {
    "id": 7,
    "email": "user@example.com",
    "name": "Example User",
    "bio": "Optional bio",
    "avatar_url": "https://example.com/avatar.png",
    "is_active": true,
    "created_at": "2026-03-30T20:15:00Z",
    "updated_at": "2026-03-30T20:20:00Z"
  }
}
```

## Error Shape

Validation and application errors use the shared structured error middleware. Typical JSON responses include:

```json
{
  "error": "Validation failed",
  "type": "validation",
  "details": [
    {
      "field": "email",
      "message": "Email is required"
    }
  ]
}
```
