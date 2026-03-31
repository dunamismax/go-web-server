#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_HOST="127.0.0.1"
APP_PORT="${APP_PORT:-$((18080 + RANDOM % 1000))}"
DB_PORT="${SMOKE_DB_PORT:-$((15432 + RANDOM % 1000))}"
BASE_URL="http://${APP_HOST}:${APP_PORT}"
WORK_DIR="$(mktemp -d)"
COOKIE_JAR="${WORK_DIR}/cookies.txt"
APP_LOG="${WORK_DIR}/app.log"
APP_PID=""
CONTAINER_NAME=""
DATABASE_URL="${DATABASE_URL:-}"
REGISTER_EMAIL="smoke-$(date +%s)@example.com"
REGISTER_PASSWORD="SmokePass123!"
REGISTER_NAME="Smoke Test"

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

wait_for_http() {
  local attempt
  for attempt in $(seq 1 60); do
    if curl -fsS "${BASE_URL}/health" -o "${WORK_DIR}/health.json" >/dev/null 2>&1; then
      if grep -q '"status":"ok"' "${WORK_DIR}/health.json" && grep -q '"database":"ok"' "${WORK_DIR}/health.json"; then
        return 0
      fi
    fi

    if [[ -n "${APP_PID}" ]] && ! kill -0 "${APP_PID}" >/dev/null 2>&1; then
      echo "server exited before smoke check completed" >&2
      cat "${APP_LOG}" >&2 || true
      exit 1
    fi

    sleep 1
  done

  echo "timed out waiting for ${BASE_URL}/health" >&2
  cat "${APP_LOG}" >&2 || true
  exit 1
}

extract_csrf_token() {
  grep -i '^X-CSRF-Token:' "$1" | awk '{gsub(/\r/, "", $2); print $2}' | tail -n 1
}

require_command curl
require_command go

if [[ -z "${DATABASE_URL}" ]]; then
  require_command docker

  CONTAINER_NAME="go-web-server-smoke-$RANDOM-$RANDOM"
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
    echo "timed out waiting for PostgreSQL smoke container" >&2
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

wait_for_http

echo "health check passed at ${BASE_URL}/health"

curl -fsS \
  -D "${WORK_DIR}/register.headers" \
  -o "${WORK_DIR}/register.html" \
  -c "${COOKIE_JAR}" \
  "${BASE_URL}/auth/register" >/dev/null

if ! grep -q '/_astro/' "${WORK_DIR}/register.html"; then
  echo "register page did not include embedded Astro assets" >&2
  cat "${WORK_DIR}/register.html" >&2 || true
  exit 1
fi

if grep -q 'htmx.min.js' "${WORK_DIR}/register.html"; then
  echo "register page still referenced legacy HTMX assets" >&2
  cat "${WORK_DIR}/register.html" >&2 || true
  exit 1
fi

CSRF_TOKEN="$(extract_csrf_token "${WORK_DIR}/register.headers")"
if [[ -z "${CSRF_TOKEN}" ]]; then
  echo "failed to capture CSRF token from register page" >&2
  cat "${WORK_DIR}/register.headers" >&2 || true
  exit 1
fi

register_status="$(curl -sS \
  -o "${WORK_DIR}/register-post.html" \
  -D "${WORK_DIR}/register-post.headers" \
  -w '%{http_code}' \
  -b "${COOKIE_JAR}" \
  -c "${COOKIE_JAR}" \
  -H "X-CSRF-Token: ${CSRF_TOKEN}" \
  --data-urlencode "email=${REGISTER_EMAIL}" \
  --data-urlencode "name=${REGISTER_NAME}" \
  --data-urlencode "password=${REGISTER_PASSWORD}" \
  --data-urlencode "confirm_password=${REGISTER_PASSWORD}" \
  --data-urlencode 'bio=Runtime smoke test user' \
  --data-urlencode 'avatar_url=https://example.com/avatar.png' \
  "${BASE_URL}/auth/register")"

if [[ "${register_status}" != "302" ]]; then
  echo "registration failed with status ${register_status}" >&2
  cat "${WORK_DIR}/register-post.html" >&2 || true
  cat "${APP_LOG}" >&2 || true
  exit 1
fi

profile_status="$(curl -sS -o "${WORK_DIR}/profile.html" -w '%{http_code}' -b "${COOKIE_JAR}" "${BASE_URL}/profile")"
if [[ "${profile_status}" != "200" ]]; then
  echo "profile request failed with status ${profile_status}" >&2
  cat "${WORK_DIR}/profile.html" >&2 || true
  cat "${APP_LOG}" >&2 || true
  exit 1
fi

if ! grep -q '/_astro/' "${WORK_DIR}/profile.html"; then
  echo "profile page did not include embedded Astro assets" >&2
  cat "${WORK_DIR}/profile.html" >&2 || true
  exit 1
fi

users_status="$(curl -sS -o "${WORK_DIR}/users.html" -w '%{http_code}' -b "${COOKIE_JAR}" "${BASE_URL}/users")"
if [[ "${users_status}" != "200" ]]; then
  echo "users page request failed with status ${users_status}" >&2
  cat "${WORK_DIR}/users.html" >&2 || true
  cat "${APP_LOG}" >&2 || true
  exit 1
fi

if ! grep -q '/_astro/' "${WORK_DIR}/users.html"; then
  echo "users page did not include embedded Astro assets" >&2
  cat "${WORK_DIR}/users.html" >&2 || true
  exit 1
fi

if grep -q 'htmx.min.js' "${WORK_DIR}/users.html"; then
  echo "users page still referenced legacy HTMX assets" >&2
  cat "${WORK_DIR}/users.html" >&2 || true
  exit 1
fi

auth_state_status="$(curl -sS -o "${WORK_DIR}/auth-state.json" -w '%{http_code}' -b "${COOKIE_JAR}" "${BASE_URL}/_backend/api/auth/state")"
if [[ "${auth_state_status}" != "200" ]]; then
  echo "auth state request through /_backend failed with status ${auth_state_status}" >&2
  cat "${WORK_DIR}/auth-state.json" >&2 || true
  cat "${APP_LOG}" >&2 || true
  exit 1
fi

if ! grep -q '"authenticated":true' "${WORK_DIR}/auth-state.json"; then
  echo "auth state response did not report an authenticated session" >&2
  cat "${WORK_DIR}/auth-state.json" >&2 || true
  exit 1
fi

if ! grep -q "${REGISTER_EMAIL}" "${WORK_DIR}/auth-state.json"; then
  echo "auth state response did not include registered email ${REGISTER_EMAIL}" >&2
  cat "${WORK_DIR}/auth-state.json" >&2 || true
  exit 1
fi

users_api_status="$(curl -sS -o "${WORK_DIR}/users.json" -w '%{http_code}' -b "${COOKIE_JAR}" "${BASE_URL}/_backend/api/users")"
if [[ "${users_api_status}" != "200" ]]; then
  echo "users API request through /_backend failed with status ${users_api_status}" >&2
  cat "${WORK_DIR}/users.json" >&2 || true
  cat "${APP_LOG}" >&2 || true
  exit 1
fi

if ! grep -q "${REGISTER_EMAIL}" "${WORK_DIR}/users.json"; then
  echo "users API response did not include registered email ${REGISTER_EMAIL}" >&2
  cat "${WORK_DIR}/users.json" >&2 || true
  exit 1
fi

echo "runtime smoke passed: registration, embedded Astro pages, same-origin /_backend API bridge, protected JSON contracts, and database-backed health all succeeded"
