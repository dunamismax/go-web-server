# Deployment

The build produces one static binary. Templates, the stylesheet, htmx and the
database migrations are embedded in it, so there is nothing else to copy.

Whichever path you choose, production needs:

- `APP_ENV=production`
- `DATABASE_URL` pointing at PostgreSQL
- a reverse proxy terminating TLS in front of the app
- `TRUSTED_PROXIES` set to that proxy's address

## Container

```bash
make docker
docker run --rm -p 8080:8080 \
  -e DATABASE_URL='postgres://user:pass@db-host:5432/dbname' \
  go-web-server
```

The image is about 25 MB, based on distroless, and runs as a non-root user.
It has no shell. Its health check runs `server healthcheck`, which asks the
running process for `/healthz`.

To try the image together with PostgreSQL locally:

```bash
docker compose --profile app up --build
```

Most container platforms set `PORT`; the server listens on it when
`HTTP_ADDR` is unset.

## Linux host with systemd and Caddy

The files are in [`deploy/`](../deploy/). On the server:

1. Create the database and a role for the app:

   ```bash
   sudo -u postgres createuser --pwprompt gowebserver
   sudo -u postgres createdb --owner gowebserver gowebserver
   ```

2. Create the service account and install the binary. Build it on any
   machine with `make build` (add `GOOS=linux GOARCH=amd64` when
   cross-compiling) and copy `bin/server` over:

   ```bash
   sudo useradd --system --no-create-home --shell /usr/sbin/nologin gowebserver
   sudo install -m 0755 server /usr/local/bin/gowebserver
   ```

3. Install the environment file and fill in the database password:

   ```bash
   sudo install -d -m 0750 -g gowebserver /etc/gowebserver
   sudo install -m 0640 -g gowebserver deploy/env.example /etc/gowebserver/env
   sudoedit /etc/gowebserver/env
   ```

4. Install and start the service:

   ```bash
   sudo install -m 0644 deploy/gowebserver.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable --now gowebserver
   ```

5. Add the site from [`deploy/Caddyfile`](../deploy/Caddyfile) to your Caddy
   configuration with your domain, then `sudo systemctl reload caddy`. Caddy
   obtains the certificate and compresses responses.

Check it:

```bash
systemctl status gowebserver
journalctl -u gowebserver -f
curl -s http://127.0.0.1:8080/readyz
```

To release a new version, replace the binary and run
`sudo systemctl restart gowebserver`. On `SIGTERM` the server stops
accepting connections and lets in-flight requests finish for up to
`SHUTDOWN_TIMEOUT`.

## Migrations

By default the server applies pending migrations when it starts. A
PostgreSQL advisory lock makes that safe when several instances start
together.

For tighter control, give the running app a role that cannot change the
schema:

1. Set `DATABASE_AUTO_MIGRATE=false` for the service.
2. During each deploy, before restarting, run the migrations as a role that
   owns the schema:

   ```bash
   DATABASE_URL='postgres://migrator:...@localhost:5432/gowebserver' \
     gowebserver migrate up
   ```

`gowebserver migrate status` lists what is applied, and
`gowebserver migrate down` rolls back the latest migration.

Write migrations so the previous version of the app keeps working against
the new schema (add columns before using them, remove them a release after
you stop). That is what makes a restart-based deploy safe.

## Health checks

| Endpoint | Meaning | Use it for |
| --- | --- | --- |
| `GET /healthz` | The process is up | Liveness probes, the container health check |
| `GET /readyz` | The database is reachable | Readiness probes, load balancer health checks |

`/healthz` deliberately ignores the database, so a database outage does not
cause the orchestrator to restart healthy app processes.

## Logs

Logs go to stdout: JSON in production, one object per line. Every request
log carries a `request_id`, which is also returned in the `X-Request-Id`
header and printed on 5xx error pages.

## Scaling notes

- Sessions are in PostgreSQL, so any number of instances can serve any
  request.
- The sign-in rate limiter is in memory, per instance. With more than one
  instance, also enforce a limit at the proxy.
- Size `DATABASE_MAX_CONNS` so that instances × pool size stays below
  PostgreSQL's `max_connections`.
