#!/usr/bin/env bash
# Role: Start only Postgres and redis-0 under the infra compose profile.
# Execution context: Dev host when full stack is too heavy; stops other infra-profile services first.
# Env knobs: none (uses docker-compose.yaml --profile infra).
# Verify: bash scripts/dev/stack/minimal_infra.sh && docker compose --profile infra ps db redis-0
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"

cd "$ROOT"

# infra profile: PG and redis shards only; no tracker, processor, or ClickHouse.
COMPOSE=(docker compose --project-directory "$ROOT" -f "$ROOT/docker-compose.yaml" --profile infra)

echo "dev_minimal_infra: stopping all infra-profile services"
"${COMPOSE[@]}" stop

echo "dev_minimal_infra: starting db redis-0"
"${COMPOSE[@]}" up -d db redis-0

"${COMPOSE[@]}" ps db redis-0
