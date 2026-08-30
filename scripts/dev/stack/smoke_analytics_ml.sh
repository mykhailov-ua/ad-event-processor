#!/usr/bin/env bash
# Role: Validate analytics_ml and fraud-scorer compose profile wiring.
# Execution context: Pre-flight before analytics-ml stack; requires CH_ENABLED=1.
# Env knobs: CH_DSN, DB_DSN, REDIS_PASSWORD, ADMIN_API_KEY (smoke defaults).
# Verify: bash scripts/dev/stack/smoke_analytics_ml.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

export CH_ENABLED=1
export CH_DSN="${CH_DSN:-clickhouse://default:@127.0.0.1:9000/default}"
export DB_DSN="${DB_DSN:-postgres://user:pass@127.0.0.1:5432/ad-event-processor?sslmode=disable}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-smoke}"
export ADMIN_API_KEY="${ADMIN_API_KEY:-smoke-admin-key}"

# analytics_ml profile: clickhouse, ivt-detector, fraud-scorer must appear in compose services list.
docker compose --profile analytics_ml --profile fraud-scorer config > /dev/null
services=$(docker compose --profile analytics_ml --profile fraud-scorer config --services 2> /dev/null)
for svc in clickhouse ivt-detector fraud-scorer; do
  echo "$services" | grep -qx "$svc"
done

go run ./cmd/operator doctor --profile analytics_ml

echo "smoke_analytics_ml: ok"
