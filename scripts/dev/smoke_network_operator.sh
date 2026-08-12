#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

export DB_DSN="${DB_DSN:-postgres://user:pass@127.0.0.1:5432/espx?sslmode=disable}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-smoke}"
export CH_ENABLED="${CH_ENABLED:-1}"
export CH_DSN="${CH_DSN:-clickhouse://default:@127.0.0.1:9000/default}"

docker compose --profile network_operator config >/dev/null
services=$(docker compose --profile network_operator config --services 2>/dev/null)
echo "$services" | grep -qx clickhouse
echo "$services" | grep -qx control
echo "$services" | grep -qx db-payment

go run ./cmd/ad-event-processor doctor --profile network_operator

echo "smoke_network_operator: ok"
