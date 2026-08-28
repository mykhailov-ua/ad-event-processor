#!/usr/bin/env bash
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
REDIS_PASS="${REDIS_PASSWORD:-}"
COMPOSE=(docker compose --project-directory "$ROOT" -f docker-compose.yaml -f deploy/compose/docker-compose.control-dev.yaml)
if [[ -f "$ROOT/.env" ]]; then
  COMPOSE+=(--env-file "$ROOT/.env")
fi

log() { printf 'seed-ingest-only: %s\n' "$*"; }

log "seeding 100 active campaigns"
"${COMPOSE[@]}" exec -T db psql -h localhost -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor << 'EOF'
INSERT INTO customers (id, name, balance, currency, allowed_overdraft)
SELECT
    ('00000000-0000-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    'loadgen-customer-' || i,
    500000000000::bigint,
    'USD',
    0
FROM generate_series(1, 100) s(i)
ON CONFLICT (id) DO UPDATE SET balance = EXCLUDED.balance;

INSERT INTO advertiser_brands (id, customer_id, name)
SELECT
    ('00000000-0001-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    ('00000000-0000-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    'loadgen-brand-' || i
FROM generate_series(1, 100) s(i)
ON CONFLICT (id) DO NOTHING;

INSERT INTO brand_creatives (id, brand_id, name, landing_url, weight, status)
SELECT
    ('00000000-0002-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    ('00000000-0001-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    'loadgen-creative-' || i,
    'https://example.com/landing?cid={click_id}',
    100,
    'ACTIVE'
FROM generate_series(1, 100) s(i)
ON CONFLICT (brand_id, name) DO UPDATE SET
  landing_url = EXCLUDED.landing_url,
  status = 'ACTIVE',
  updated_at = NOW();

INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window, brand_id, target_url)
SELECT
    ('00000000-0000-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    'loadgen-campaign-' || i,
    420000000000::bigint,
    'ACTIVE',
    ('00000000-0000-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    'ASAP',
    380000000::bigint,
    'UTC',
    100000000,
    3600,
    ('00000000-0001-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    'https://example.com/landing?cid={click_id}'
FROM generate_series(1, 100) s(i)
ON CONFLICT (id) DO UPDATE SET
  current_spend = 0,
  status = 'ACTIVE',
  budget_limit = EXCLUDED.budget_limit,
  brand_id = EXCLUDED.brand_id,
  target_url = EXCLUDED.target_url;

INSERT INTO billing.license_status (
    deployment_id, license_id, plan_code, valid_until, state, entitlements_json, last_verified_at
) VALUES (
    '00000000-0000-0000-0000-0000000000ab',
    '00000000-0000-0000-0000-0000000000cd',
    'pilot',
    NOW() + INTERVAL '365 days',
    'ACTIVE',
    '{"limits":{"max_active_campaigns":1000,"max_rps":200000,"max_requests_per_day":0,"max_events_per_month":0,"max_regions":4,"max_api_keys":10,"max_export_chunk_bytes":10485760,"quota_reset_timezone":"UTC"},"features":{"rtb_live":true,"ml_fraud_boost":true,"multi_region":true,"slot_migration":true}}'::jsonb,
    NOW()
)
ON CONFLICT (deployment_id) DO UPDATE SET
    state = EXCLUDED.state,
    valid_until = EXCLUDED.valid_until,
    entitlements_json = EXCLUDED.entitlements_json,
    last_verified_at = EXCLUDED.last_verified_at;
EOF

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
  for i in $(seq 1 100); do
    hex=$(printf '%012x' "$i")
    camp_id="00000000-0000-0000-0000-${hex}"
    brand_id="00000000-0001-0000-0000-${hex}"
    creative_id="00000000-0002-0000-0000-${hex}"
    payload='[{"id":"'"${creative_id}"'","url":"https://example.com/landing?cid={click_id}","weight":100}]'
    redis_set "$shard" "brand:creatives:${brand_id}" "$payload"
    redis_set "$shard" "{${camp_id}}budget:campaign:${camp_id}" "420000000000"
    redis_set "$shard" "{${camp_id}}budget:quota:${camp_id}" "420000000000"
  done
done

bash "$SCRIPTS/test/seed_load_test_limits.sh" || true
log "done"
