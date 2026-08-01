#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

export CH_ENABLED=0
export CONTROL_ENABLE_PAYMENT=0
export CONTROL_ENABLE_BILLING=0
export CONTROL_ENABLE_NOTIFIER=0
export CONTROL_ENABLE_MARGIN_GUARD=0
export CONTROL_ENABLE_COST_SYNC=0
export DB_DSN="${DB_DSN:-postgres://user:pass@127.0.0.1:5432/espx?sslmode=disable}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-smoke}"

docker compose --profile ingest_only config >/dev/null
bash "$SCRIPTS/ci/compose_profile_check.sh"
go run ./cmd/espx doctor --profile ingest_only

echo "smoke_ingest_only: ok"
