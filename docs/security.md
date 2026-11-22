# Security

This is what the template does out of the box, where each piece lives, and
what it leaves for you to add.

## What is in place

### Passwords

- Hashed with Argon2id (19 MiB, 2 iterations, 1 thread: the OWASP baseline),
  stored in PHC string format. See
  [`internal/auth`](../internal/auth/password.go).
- The parameters travel with each hash. Raise `auth.DefaultParams` and
  existing users are re-hashed transparently the next time they sign in.
- Length is the only rule: 12 to 128 characters, following NIST SP 800-63B.
  There are no character-class requirements.

### Sign-in

- A wrong password and an unknown email produce the same message and cost
  the same work (a decoy hash is verified when no account matches), so the
  form does not reveal which emails are registered.
- Sign-in, sign-up and password-change posts are limited to 10 per minute
  per client address.
- The session token is regenerated at sign-in, which prevents session
  fixation.
- The post-login `next` redirect only accepts local paths.

Sign-up is different: it tells the user when an email is already taken.
Hiding that requires an email round-trip, which this template does not have.

### Sessions

- Server-side, in PostgreSQL. The cookie holds only a random token.
- The cookie is `HttpOnly` and `SameSite=Lax`. In production it is also
  `Secure` and named with the `__Host-` prefix, which makes browsers reject
  any copy set over plain HTTP or by a subdomain.
- Every request re-reads the user row. Deactivating or deleting a user, or
  changing their password, ends their sessions on their next request.

### Cross-site request forgery

`http.CrossOriginProtection` from the standard library rejects any
non-`GET` request that the browser marks as cross-origin, using the
`Sec-Fetch-Site` header (and `Origin` for older browsers). There are no
tokens to embed or rotate.

Two things follow from that design:

- `GET` handlers must never change state. All mutations here are `POST`.
- If you serve the app from several origins that post to each other, list
  them in `TRUSTED_ORIGINS`.

### Output and injection

- templ escapes everything it interpolates, and sanitizes URLs.
- Every query is parameterized by sqlc. Search input is matched with
  `strpos`, so `%` and `_` in a search term are literal text.
- The Content Security Policy is `default-src 'self'` with no inline scripts
  or styles allowed. htmx is configured to work without `eval`. The site
  loads nothing from other origins: no CDN, fonts or analytics.

### Headers

Set in [`secureHeaders`](../internal/web/middleware.go): Content Security
Policy, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`,
`Permissions-Policy`, `Cross-Origin-Opener-Policy`,
`Cross-Origin-Resource-Policy`, and `Strict-Transport-Security` in
production. Pages are sent with `Cache-Control: no-store`.

### Authorization

- Route-level: `requireUser` and `requireAdmin` in
  [`internal/web/session.go`](../internal/web/session.go). The route table
  shows at a glance what each route requires.
- An admin cannot demote, deactivate or delete their own account, so the
  last admin cannot be removed.

### Operational

- Request bodies are capped at 1 MB, and the HTTP server has read, write and
  idle timeouts.
- Error pages show a request ID, never internal detail. The detail goes to
  the log under the same ID.
- `X-Forwarded-For` is ignored unless the connection comes from an address in
  `TRUSTED_PROXIES`.
- The container image is distroless and runs as a non-root user. The systemd
  unit sandboxes the process.
- CI runs `govulncheck` on every push.

## What you still need to do

- **Serve over HTTPS.** Put a TLS-terminating proxy in front (the
  [`deploy/Caddyfile`](../deploy/Caddyfile) example does this) and set
  `APP_ENV=production`.
- **Set `TRUSTED_PROXIES`** to your proxy's address. Without it, rate limits
  and logs see the proxy as the only client. With a wrong value, clients can
  spoof their address.
- **Use a least-privilege database role** for the running app, and a
  separate role for migrations. See
  [deployment.md](deployment.md#migrations).
- **Add per-record authorization** when you add data that belongs to a
  user. The pattern is to put the owner's ID in the `WHERE` clause of every
  query, not to check ownership after loading.

## Not included

- Email verification, password reset and account recovery
- Two-factor authentication
- An audit log in the database (admin actions are written to the
  application log)
- Rate limiting shared across instances; the built-in limiter is per process
- Account lockout after repeated failures (the rate limit is per address,
  not per account)

## Reporting a vulnerability

Please do not open a public issue for a security problem. Use **Report a
vulnerability** on the repository's
[Security tab](https://github.com/dunamismax/go-web-server/security) so the
details stay private until a fix is released.
