# Security Notes

This repo has a solid starter baseline, but it is still a starter. The current security posture is good enough to build on, not complete enough to call finished.

## Current Baseline

- Session-cookie auth with server-side session storage
- CSRF protection on state-changing requests
- Rate limiting and security headers middleware
- Structured application errors
- Password hashing with Argon2id
- PostgreSQL-backed persistence with SQLC-generated parameterized queries
- Same-origin browser architecture for the shipped frontend

## Output And Query Safety

- SQLC keeps the data path parameterized by default.
- Astro, Vue, and normal JSON serialization escape content safely in their standard render paths.
- Avoid introducing raw HTML rendering without a clear sanitization policy.

## Gaps Still Worth Closing

- No roles or per-record authorization model
- No password reset or account recovery flow
- No audit trail for sensitive account actions
- No dedicated metrics or intrusion visibility path
- No production-secret rotation story beyond normal env/config discipline

## Operational Guidance

- Keep `AUTH_COOKIE_SECURE=true` outside plain HTTP localhost work.
- Leave `security.trusted_proxies` empty unless the app is actually behind reverse proxies you control.
- Treat the built frontend under `web/dist` as a shipped artifact and verify it in CI.
- Keep database access boring and explicit. Do not bypass SQLC casually.
