#!/usr/bin/env bash
# Role: Full PG synthetic demo for appliance testers (campaigns, stats, billing, landers, Redis budgets).
# Execution context: systemd install with docker PG/Redis; uses bin/admin on INSTALL_ROOT.
# Env knobs: SEED_APPLIANCE_COUNT (default 100); SEED_GLORY_CUSTOMER_ID; SEED_STATS_DAYS (default 30).
# Verify: bash scripts/ops/seed_appliance_demo.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

DB_PORT="${DB_PORT:-5430}"
DB_USER="${DB_USER:-ad_event_processor_user}"
DB_NAME="${DB_NAME:-ad_event_processor}"
SEED_APPLIANCE_COUNT="${SEED_APPLIANCE_COUNT:-100}"
SEED_STATS_DAYS="${SEED_STATS_DAYS:-30}"
SEED_GLORY_CUSTOMER_ID="${SEED_GLORY_CUSTOMER_ID:-8c66adae-da43-41d0-94fc-6a1abb54fc55}"
GLORY_CAMPAIGNS="${SEED_GLORY_CAMPAIGNS:-35}"

ADMIN="${ROOT}/bin/admin"
if [[ ! -x "$ADMIN" ]]; then
  ADMIN="${ROOT}/bin/aed-admin"
fi
if [[ ! -x "$ADMIN" ]]; then
  echo "seed-appliance-demo: admin binary missing at ${ROOT}/bin/admin" >&2
  exit 1
fi

DB_CONTAINER="${SEED_DB_CONTAINER:-ad-event-processor-db-1}"
REDIS_CONTAINER_0="${SEED_REDIS_CONTAINER_0:-ad-event-processor-redis-0-1}"
REDIS_CONTAINER_1="${SEED_REDIS_CONTAINER_1:-ad-event-processor-redis-1-1}"

psql_exec() {
  docker exec -i "$DB_CONTAINER" psql -h localhost -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" "$@"
}

log() { printf 'seed-appliance-demo: %s\n' "$*"; }

log "step 1/7 ingest SQL (count=${SEED_APPLIANCE_COUNT})"
"$ADMIN" --env-path "$ROOT/.env" db seed-ingest-sql --count "$SEED_APPLIANCE_COUNT" | psql_exec

log "step 2/7 redis budgets and campaign publish"
eval "$("$ADMIN" --env-path "$ROOT/.env" db seed-uuids-shell --count "$SEED_APPLIANCE_COUNT")"

redis_cli() {
  local container="$1"
  shift
  docker exec "$container" redis-cli -p 6379 -a "$REDIS_PASSWORD" "$@" 2> /dev/null | tail -1
}

CHANNEL="${CAMPAIGN_UPDATE_CHANNEL:-campaigns:update}"
redis_cli "$REDIS_CONTAINER_0" PUBLISH "$CHANNEL" '*' > /dev/null || true

for i in $(seq 1 "$SEED_APPLIANCE_COUNT"); do
  camp_id_var="AED_CAMPAIGN_UUID_${i}"
  brand_id_var="AED_BRAND_UUID_${i}"
  creative_id_var="AED_CREATIVE_UUID_${i}"
  camp_id="${!camp_id_var}"
  brand_id="${!brand_id_var}"
  creative_id="${!creative_id_var}"
  payload='[{"id":"'"${creative_id}"'","url":"https://trk.horizon-media.io/landing?cid={click_id}","weight":'$((97 + i % 13))'}]'
  budget=$((4200000000 + (i % 17) * 650000000 + (i % 9) * 384729))
  for container in "$REDIS_CONTAINER_0" "$REDIS_CONTAINER_1"; do
    redis_cli "$container" SET "brand:creatives:${brand_id}" "$payload"
    redis_cli "$container" SET "{${camp_id}}budget:campaign:${camp_id}" "$budget"
    redis_cli "$container" SET "{${camp_id}}budget:quota:${camp_id}" "$budget"
  done
done

log "step 3/7 PG UI stats (${SEED_STATS_DAYS} days)"
"$ADMIN" --env-path "$ROOT/.env" db seed-ui --count "$SEED_APPLIANCE_COUNT" --stats-days "$SEED_STATS_DAYS"

log "step 4/7 campaign list UX"
"$ADMIN" --env-path "$ROOT/.env" db seed-campaign-list-ux --count "$SEED_APPLIANCE_COUNT"

log "step 5/7 appliance enrich (billing, landers, flows, ledger)"
"$ADMIN" --env-path "$ROOT/.env" db seed-appliance-demo \
  --count "$SEED_APPLIANCE_COUNT" \
  --glory-customer-id "$SEED_GLORY_CUSTOMER_ID" \
  --glory-campaigns "$GLORY_CAMPAIGNS"

log "step 6/7 ARCHIVED status enum"
psql_exec -c "ALTER TYPE campaign_status_type ADD VALUE IF NOT EXISTS 'ARCHIVED';" > /dev/null || true

log "step 7/7 counts"
psql_exec -tAc "
SELECT 'customers', count(*)::text FROM customers
UNION ALL SELECT 'campaigns', count(*)::text FROM campaigns
UNION ALL SELECT 'campaign_stats', count(*)::text FROM campaign_stats
UNION ALL SELECT 'landers', count(*)::text FROM landers
UNION ALL SELECT 'offers', count(*)::text FROM offers
UNION ALL SELECT 'flows', count(*)::text FROM flows
UNION ALL SELECT 'balance_ledger', count(*)::text FROM balance_ledger
UNION ALL SELECT 'billing.invoices', count(*)::text FROM billing.invoices
UNION ALL SELECT 'glory_campaigns', count(*)::text FROM campaigns WHERE customer_id='${SEED_GLORY_CUSTOMER_ID}';
"

log "done"
