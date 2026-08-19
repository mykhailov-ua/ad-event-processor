#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

failed=0

scan() {
  local desc="$1"
  shift
  if rg -n "$@" internal/ pkg/ --glob '*.go' --glob '!*_test.go' \
    --glob '!**/db/*.sql.go' \
    --glob '!**/sqlc/**' \
    --glob '!internal/ingestion/clickhouse_store.go' \
    --glob '!internal/ingestion/campaign_flow_sync.go' \
    --glob '!internal/costsync/**' \
    2> /dev/null; then
    echo "sql-safety: FAIL $desc"
    failed=1
  fi
}

scan 'fmt.Sprintf with SQL keyword' \
  'fmt\.Sprintf.*(SELECT|INSERT|UPDATE|DELETE|FROM|WHERE)'

scan 'interpolated SQL string literal' \
  '"(SELECT|INSERT|UPDATE|DELETE) [^"]*%[sdv]"'

scan 'concatenated SQL string' \
  '"(SELECT|INSERT|UPDATE|DELETE) .*"\s*\+'

if [[ "$failed" -ne 0 ]]; then
  echo "sql-safety: use sqlc queries under internal/*/queries/ — see docs/CI.md#sql-safety"
  exit 1
fi

echo "sql-safety: OK"
