#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  source "$ROOT/.env"
  set +a
fi

DB_PORT="${DB_PORT:-5430}"
REDIS_PASS="${REDIS_PASSWORD:-redis_secure_pass_456}"
COMPOSE=(docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml)

CAMPAIGN_ID="${STRICT_CAMPAIGN_ID:-00000000-0000-0000-0000-000000000001}"
BUDGET_LIMIT_MICRO="${STRICT_BUDGET_LIMIT_MICRO:-12000000}"
REMAINING_MICRO="${STRICT_REMAINING_MICRO:-4000000}"
SYNC_INFLIGHT_MICRO="${STRICT_SYNC_INFLIGHT_MICRO:-2000000}"
REDIS_SHARD="${STRICT_REDIS_SHARD:-3}"

log() { printf 'seed-strict-flush: %s\n' "$*"; }
die() {
  printf 'seed-strict-flush: ERROR: %s\n' "$*" >&2
  exit 1
}

log "campaign=$CAMPAIGN_ID budget_limit=$BUDGET_LIMIT_MICRO remaining=$REMAINING_MICRO sync_inflight=$SYNC_INFLIGHT_MICRO shard=redis-${REDIS_SHARD}"

"${COMPOSE[@]}" exec -T db psql -h localhost -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor -v ON_ERROR_STOP=1 << EOF
UPDATE campaigns
SET budget_limit = ${BUDGET_LIMIT_MICRO},
    current_spend = 0,
    status = 'ACTIVE',
    updated_at = NOW()
WHERE id = '${CAMPAIGN_ID}'::uuid;
EOF

BUDGET_KEY="budget:campaign:${CAMPAIGN_ID}"
SYNC_KEY="budget:sync:campaign:${CAMPAIGN_ID}"

"${COMPOSE[@]}" exec -T "redis-${REDIS_SHARD}" redis-cli -p 6379 -a "$REDIS_PASS" --no-auth-warning \
  SET "$BUDGET_KEY" "$REMAINING_MICRO" EX 86400 > /dev/null
"${COMPOSE[@]}" exec -T "redis-${REDIS_SHARD}" redis-cli -p 6379 -a "$REDIS_PASS" --no-auth-warning \
  SET "$SYNC_KEY" "$SYNC_INFLIGHT_MICRO" EX 86400 > /dev/null
"${COMPOSE[@]}" exec -T "redis-${REDIS_SHARD}" redis-cli -p 6379 -a "$REDIS_PASS" --no-auth-warning \
  SADD budget:dirty_campaigns "$CAMPAIGN_ID" > /dev/null

VERIFY_TRACKERS="${VERIFY_TRACKERS:-0}"
export VERIFY_TRACKERS
bash "$SCRIPTS/test/sync_tracker_registry.sh"

log "seed complete (harness=loadgen constrained stack strict_flush)"
