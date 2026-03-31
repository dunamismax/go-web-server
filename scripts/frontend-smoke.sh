#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_HOST="127.0.0.1"
APP_PORT="${APP_PORT:-$((19080 + RANDOM % 1000))}"
FRONTEND_PORT="${FRONTEND_PORT:-$((14321 + RANDOM % 1000))}"
DB_PORT="${SMOKE_DB_PORT:-$((16432 + RANDOM % 1000))}"
BASE_URL="http://${APP_HOST}:${APP_PORT}"
FRONTEND_URL="http://${APP_HOST}:${FRONTEND_PORT}"
WORK_DIR="$(mktemp -d)"
APP_LOG="${WORK_DIR}/app.log"
APP_PID=""
CONTAINER_NAME=""
DATABASE_URL="${DATABASE_URL:-}"

cleanup() {
  local exit_code=$?

  if [[ -n "${APP_PID}" ]] && kill -0 "${APP_PID}" >/dev/null 2>&1; then
    kill "${APP_PID}" >/dev/null 2>&1 || true
    wait "${APP_PID}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${CONTAINER_NAME}" ]]; then
    docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  fi

  rm -rf "${WORK_DIR}"
  exit "${exit_code}"
}
trap cleanup EXIT

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

wait_for_backend() {
  local attempt
  for attempt in $(seq 1 60); do
    if curl -fsS "${BASE_URL}/health" -o "${WORK_DIR}/health.json" >/dev/null 2>&1; then
      if grep -q '"status":"ok"' "${WORK_DIR}/health.json" && grep -q '"database":"ok"' "${WORK_DIR}/health.json"; then
        return 0
      fi
    fi

    if [[ -n "${APP_PID}" ]] && ! kill -0 "${APP_PID}" >/dev/null 2>&1; then
      echo "server exited before frontend smoke completed" >&2
      cat "${APP_LOG}" >&2 || true
      exit 1
    fi

    sleep 1
  done

  echo "timed out waiting for ${BASE_URL}/health" >&2
  cat "${APP_LOG}" >&2 || true
  exit 1
}

require_command bun
require_command curl
require_command go

if [[ -z "${DATABASE_URL}" ]]; then
  require_command docker

  CONTAINER_NAME="go-web-server-frontend-smoke-$RANDOM-$RANDOM"
  docker run -d --rm \
    --name "${CONTAINER_NAME}" \
    -e POSTGRES_DB=gowebserver \
    -e POSTGRES_USER=gowebserver \
    -e POSTGRES_PASSWORD=gowebserver \
    -p "${DB_PORT}:5432" \
    postgres:15-alpine >/dev/null

  for attempt in $(seq 1 60); do
    if docker exec "${CONTAINER_NAME}" pg_isready -U gowebserver -d gowebserver >/dev/null 2>&1; then
      DATABASE_URL="postgres://gowebserver:gowebserver@127.0.0.1:${DB_PORT}/gowebserver?sslmode=disable"
      break
    fi
    sleep 1
  done

  if [[ -z "${DATABASE_URL}" ]]; then
    echo "timed out waiting for PostgreSQL frontend smoke container" >&2
    docker logs "${CONTAINER_NAME}" >&2 || true
    exit 1
  fi
fi

(
  cd "${REPO_ROOT}"
  APP_ENVIRONMENT=development \
  APP_DEBUG=false \
  APP_LOG_LEVEL=warn \
  APP_LOG_FORMAT=text \
  SERVER_HOST="${APP_HOST}" \
  SERVER_PORT="${APP_PORT}" \
  AUTH_COOKIE_SECURE=false \
  DATABASE_URL="${DATABASE_URL}" \
  DATABASE_RUN_MIGRATIONS=true \
  DATABASE_MAX_CONNECTIONS=5 \
  DATABASE_MIN_CONNECTIONS=1 \
  go run ./cmd/web >"${APP_LOG}" 2>&1
) &
APP_PID=$!

wait_for_backend

echo "backend ready at ${BASE_URL}/health"

echo "building Astro frontend for smoke validation..."
(
  cd "${REPO_ROOT}/web"
  bun run build
)

echo "running Astro browser smoke flow against ${FRONTEND_URL} with backend ${BASE_URL}"
(
  cd "${REPO_ROOT}/web"
  FRONTEND_BACKEND_ORIGIN="${BASE_URL}" \
  FRONTEND_PORT="${FRONTEND_PORT}" \
  PLAYWRIGHT_BASE_URL="${FRONTEND_URL}" \
  bun run test:smoke
)

echo "frontend smoke passed: Astro home, registration, profile, users create/edit, and logout worked against the real Go backend"
