#!/usr/bin/env bash
# Role: Insert active campaigns into Postgres for ingest-only load and dev testing.
# Execution context: Compose ingest_only profile with control-dev overlay; reads .env for DB_PORT.
# Env knobs: DB_PORT (5430); REDIS_PASSWORD (for compose env-file); SEED_INGEST_CAMPAIGN_COUNT (default 100).
# Verify: bash scripts/test/load/seed_ingest_only_campaigns.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

DB_PORT="${DB_PORT:-5430}"
REDIS_PASS="${REDIS_PASSWORD:-}"
COMPOSE=(docker compose --project-directory "$ROOT" -f docker-compose.yaml -f deploy/compose/docker-compose.control-dev.yaml)
if [[ -f "$ROOT/.env" ]]; then
  COMPOSE+=(--env-file "$ROOT/.env")
fi

log() { printf 'seed-ingest-only: %s\n' "$*"; }

SEED_INGEST_CAMPAIGN_COUNT="${SEED_INGEST_CAMPAIGN_COUNT:-100}"

log "seeding ${SEED_INGEST_CAMPAIGN_COUNT} active campaigns"
go run ./cmd/admin db seed-ingest-sql --count "${SEED_INGEST_CAMPAIGN_COUNT}" | "${COMPOSE[@]}" exec -T db psql -h localhost -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor

eval "$(go run ./cmd/admin db seed-uuids-shell --count "${SEED_INGEST_CAMPAIGN_COUNT}")"

CHANNEL="${CAMPAIGN_UPDATE_CHANNEL:-campaigns:update}"
if [[ -n "$REDIS_PASS" ]]; then
  "${COMPOSE[@]}" exec -T redis-0 redis-cli -p 6379 -a "$REDIS_PASS" PUBLISH "$CHANNEL" '*' > /dev/null
else
  "${COMPOSE[@]}" exec -T redis-0 redis-cli -p 6379 PUBLISH "$CHANNEL" '*' > /dev/null
fi

redis_set() {
  local shard="$1"
  local key="$2"
  local value="$3"
  local reply
  if [[ -n "$REDIS_PASS" ]]; then
    reply="$("${COMPOSE[@]}" exec -T "$shard" redis-cli -p 6379 -a "$REDIS_PASS" SET "$key" "$value" 2> /dev/null | tail -1)"
  else
    reply="$("${COMPOSE[@]}" exec -T "$shard" redis-cli -p 6379 SET "$key" "$value" 2> /dev/null | tail -1)"
  fi
  if [[ "$reply" != "OK" ]]; then
    log "redis SET failed on ${shard} key=${key}: ${reply:-empty}"
    exit 1
  fi
}

log "seeding brand creatives in redis"
for shard in redis-0 redis-1; do
  for i in $(seq 1 "${SEED_INGEST_CAMPAIGN_COUNT}"); do
    camp_id_var="AED_CAMPAIGN_UUID_${i}"
    brand_id_var="AED_BRAND_UUID_${i}"
    creative_id_var="AED_CREATIVE_UUID_${i}"
    camp_id="${!camp_id_var}"
    brand_id="${!brand_id_var}"
    creative_id="${!creative_id_var}"
    payload='[{"id":"'"${creative_id}"'","url":"https://trk.horizon-media.io/landing?cid={click_id}","weight":'$((97 + i % 13))'}]'
    budget=$((4200000000 + (i % 17) * 650000000 + (i % 9) * 384729))
    redis_set "$shard" "brand:creatives:${brand_id}" "$payload"
    redis_set "$shard" "{${camp_id}}budget:campaign:${camp_id}" "$budget"
    redis_set "$shard" "{${camp_id}}budget:quota:${camp_id}" "$budget"
  done
done

bash "$SCRIPTS/test/load/seed_limits.sh" || true
log "done"
