#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Applying app and River migrations"
go run ./cmd/demo migrate

echo "==> Seeding demo merchant, customer, price, and subscription"
go run ./cmd/demo seed

echo "==> Running dunning scanner once"
go run ./cmd/demo scan

worker_log="$(mktemp)"
cleanup() {
  rm -f "$worker_log"
}
trap cleanup EXIT

echo "==> Running worker briefly to process the queued reminder"
set +e
timeout 10s go run ./cmd/worker >"$worker_log" 2>&1
worker_status=$?
set -e
if [[ "$worker_status" -ne 0 && "$worker_status" -ne 124 ]]; then
  cat "$worker_log"
  echo "worker exited unexpectedly with status $worker_status" >&2
  exit "$worker_status"
fi

cat "$worker_log"

token="$(
  grep -oE '/dunning/[A-Za-z0-9_-]+' "$worker_log" \
    | tail -n 1 \
    | sed 's#^/dunning/##'
)"

if [[ -z "$token" ]]; then
  echo "failed to extract dunning token from worker output" >&2
  exit 1
fi

echo "==> Completing checkout using extracted dunning token"
go run ./cmd/demo complete -token "$token"

echo "==> Verifying demo lifecycle state"
go run ./cmd/demo verify
