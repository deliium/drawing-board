#!/usr/bin/env bash
# Starts Go + built SPA for Playwright learner journeys (Prompt 18).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${E2E_PORT:-18080}"
BASE="http://127.0.0.1:${PORT}"

echo "[e2e-webserver] start port=${PORT}"

if [ ! -d "${ROOT}/web/dist" ]; then
  echo "[e2e-webserver] building web/dist…"
  (cd "${ROOT}/web" && npm run build)
fi

DB_PATH="$(mktemp "${TMPDIR:-/tmp}/drawing-board-e2e-XXXXXX.db")"
trap 'rm -f "${DB_PATH}" "${DB_PATH}-wal" "${DB_PATH}-shm" 2>/dev/null || true' EXIT

export ADDR=":${PORT}"
export DB_PATH
export STATIC_DIR="${ROOT}/web/dist"
export ALLOWED_ORIGINS="${BASE},http://localhost:${PORT}"
export COOKIE_KEY="${COOKIE_KEY:-e2e-cookie-key-32-bytes-minimum!!}"
# Never enable coordinate debug dumps in CI/E2E.
unset RECOGNIZE_DEBUG || true
export APP_ENV="${APP_ENV:-development}"

echo "[e2e-webserver] DB_PATH=${DB_PATH} STATIC_DIR=${STATIC_DIR} ALLOWED_ORIGINS=${ALLOWED_ORIGINS}"
cd "${ROOT}"
exec go run ./cmd/server
