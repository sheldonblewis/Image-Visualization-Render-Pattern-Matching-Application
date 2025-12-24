#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
GO_BIN="${GO_BIN:-$(command -v go || echo /usr/local/go/bin/go)}"

if [ ! -x "$GO_BIN" ]; then
  echo "Cannot find Go binary. Set GO_BIN env var or ensure 'go' is in PATH." >&2
  exit 1
fi

start_backend() {
  echo "[dev] Starting backend on :8080"
  (cd "$BACKEND_DIR" && "$GO_BIN" run cmd/server/main.go)
}

start_frontend() {
  echo "[dev] Starting frontend on :5173"
  (cd "$FRONTEND_DIR" && npm run dev -- --host)
}

cleanup() {
  echo "\n[dev] Shutting down..."
  [[ -n "${BACKEND_PID:-}" ]] && kill "$BACKEND_PID" 2>/dev/null || true
  [[ -n "${FRONTEND_PID:-}" ]] && kill "$FRONTEND_PID" 2>/dev/null || true
}

trap cleanup INT TERM EXIT

start_backend &
BACKEND_PID=$!

start_frontend &
FRONTEND_PID=$!

echo "[dev] Backend PID: $BACKEND_PID"
echo "[dev] Frontend PID: $FRONTEND_PID"

tail -f /dev/null --pid "$BACKEND_PID" --pid "$FRONTEND_PID" 2>/dev/null || wait -n "$BACKEND_PID" "$FRONTEND_PID"
