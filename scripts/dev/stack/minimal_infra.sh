#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"

cd "$ROOT"

COMPOSE=(docker compose --project-directory "$ROOT" -f "$ROOT/docker-compose.yaml" --profile infra)

echo "dev_minimal_infra: stopping all infra-profile services"
"${COMPOSE[@]}" stop

echo "dev_minimal_infra: starting db redis-0"
"${COMPOSE[@]}" up -d db redis-0

"${COMPOSE[@]}" ps db redis-0
