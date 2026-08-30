#!/usr/bin/env bash
# Role: Validate network_operator compose profile (multi-shard Redis, db-payment, ClickHouse).
# Execution context: Pre-flight before network-operator stack profile.
# Env knobs: CH_ENABLED (default 1); CH_DSN, DB_DSN, REDIS_PASSWORD.
# Verify: bash scripts/dev/stack/smoke_network_operator.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

export DB_DSN="${DB_DSN:-postgres://user:pass@127.0.0.1:5432/ad-event-processor?sslmode=disable}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-smoke}"
export CH_ENABLED="${CH_ENABLED:-1}"
export CH_DSN="${CH_DSN:-clickhouse://default:@127.0.0.1:9000/default}"

docker compose --profile network_operator config > /dev/null
services=$(docker compose --profile network_operator config --services 2> /dev/null)
echo "$services" | grep -qx clickhouse
echo "$services" | grep -qx control
echo "$services" | grep -qx db-payment

go run ./cmd/operator doctor --profile network_operator

echo "smoke_network_operator: ok"
