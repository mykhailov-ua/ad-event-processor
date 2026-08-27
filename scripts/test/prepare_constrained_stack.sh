#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/load_test_env.sh"
cd "$ROOT"

load_test_compose COMPOSE "$ROOT"
load_test_export_derived
export SKIP_CODEGEN="${SKIP_CODEGEN:-1}"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  if ! source "$ROOT/.env" 2> /dev/null; then
    printf 'prepare-constrained: WARN: .env present but not sourced (parse error); using compose defaults\n'
  fi
  set +a
fi
DB_PORT="${DB_PORT:-5430}"
DB_DSN="${DB_DSN:-postgres://${DB_USER:-ad_event_processor_user}:${DB_PASSWORD:-secure_pass_123}@127.0.0.1:${DB_PORT}/${DB_NAME:-ad_event_processor}?sslmode=${DB_SSLMODE:-disable}}"

DATA_SERVICES=(
  run-dir-init db redis-0 redis-1 redis-2 redis-3 redis-4 redis-5 clickhouse processor prometheus grafana
)
TRACKER_SERVICES=(tracker-0 tracker-1 nginx)

log() { printf 'prepare-constrained: %s\n' "$*"; }
die() {
  printf 'prepare-constrained: ERROR: %s\n' "$*" >&2
  exit 1
}

wait_control_health() {
  local url="${LOAD_TEST_CONTROL_URL:-http://127.0.0.1:${LOAD_TEST_CONTROL_PORT:-8800}}"
  local i=0
  while [[ $i -lt 120 ]]; do
    if curl -sf "${url}/health" > /dev/null 2>&1; then
      log "control ready at ${url}/health"
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  die "control ${url}/health not ready within 240s"
}

log "bringing up data plane"
"${COMPOSE[@]}" up --force-recreate --no-deps run-dir-init
"${COMPOSE[@]}" up -d --remove-orphans "${DATA_SERVICES[@]}"

"${COMPOSE[@]}" stop tracker-2 tracker-3 2> /dev/null || true

log "waiting for postgres"
until "${COMPOSE[@]}" exec -T db pg_isready -h 127.0.0.1 -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor > /dev/null 2>&1; do
  sleep 1
done

log "reconciling partial migration state (load-test DB drift)"
bash "$SCRIPTS/test/reconcile_ingestion_migrations.sh"

log "applying postgres migrations (ads, auth, billing)"
export DB_DSN
if ! go run ./cmd/migrate-cold-path --only=ads,auth,billing; then
  die "postgres migrations failed - registry Sync will not load campaigns"
fi

log "repairing schema drift after migrations"
bash "$SCRIPTS/test/reconcile_ingestion_migrations.sh"
bash "$SCRIPTS/test/verify_load_test_schema.sh"

log "resetting clickhouse analytics (deterministic cold-path baseline)"
"${COMPOSE[@]}" exec -T clickhouse clickhouse-client --multiquery -u default --password "${CLICKHOUSE_PASSWORD:-secure_ch_pass}" -d ad_event_processor -q "
TRUNCATE TABLE impressions;
TRUNCATE TABLE clicks;
TRUNCATE TABLE conversions;
TRUNCATE TABLE fraud_events;
" 2> /dev/null || log "WARN: clickhouse truncate skipped"

log "repairing clickhouse hash columns (legacy volume drift before control migrations)"
"${COMPOSE[@]}" exec -T clickhouse clickhouse-client --multiquery -u default --password "${CLICKHOUSE_PASSWORD:-secure_ch_pass}" -d ad_event_processor -q "
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS ua_hash FixedString(16) DEFAULT '';
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS ua_hash FixedString(16) DEFAULT '';
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS ua_hash FixedString(16) DEFAULT '';
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS ua_hash FixedString(16) DEFAULT '';
" 2> /dev/null || log "WARN: clickhouse column repair skipped"

REDIS_PASS="${REDIS_PASSWORD:-redis_secure_pass_456}"
log "flushing redis shards"
for i in 0 1 2 3 4 5; do
  "${COMPOSE[@]}" exec -T "redis-${i}" redis-cli -p 6379 -a "$REDIS_PASS" FLUSHALL > /dev/null 2>&1
done

log "recreating processor (volume perms, health probe, clean consumer groups after flush)"
"${COMPOSE[@]}" up -d --build --force-recreate --no-deps processor

log "seeding campaigns (100 active, matches loadgen campaign IDs)"
"${COMPOSE[@]}" exec -T db psql -h localhost -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor << 'EOF'
TRUNCATE TABLE events CASCADE;
TRUNCATE TABLE campaign_stats CASCADE;
TRUNCATE TABLE campaigns CASCADE;

INSERT INTO customers (id, name, balance, currency, allowed_overdraft)
SELECT
    ('00000000-0000-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    (ARRAY[
      'Horizon Media Group',
      'Pacific Ads Studio',
      'Nordic Performance Co',
      'Atlas Buying Desk',
      'Summit Traffiq',
      'Bluewave Partners',
      'Velocity Affiliates',
      'Prime Reach Agency',
      'Lumen Digital',
      'Crestline Media',
      'Meridian Performance',
      'Vantage Growth Labs',
      'Redwood Acquisition',
      'Kite & Compass Media',
      'Northgate Buying',
      'Silverline Performance',
      'Harborfront Ads',
      'Quartzlane Partners',
      'Everpeak Media',
      'Bridgeport Digital'
    ])[(i - 1) % 20 + 1],
    (2400000000 + ((i * 2817431) % 281600000000) + ((i % 7) * 97000000))::bigint,
    'USD',
    0
FROM generate_series(1, 100) s(i)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  balance = EXCLUDED.balance;

INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window)
SELECT
    ('00000000-0000-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    (ARRAY[
      'US Summer Surge',
      'EU Retargeting V2',
      'LATAM Mobile App',
      'Global Video Reach',
      'APAC Crypto Swap',
      'Nordic Ecom Promo',
      'DACH High Intent',
      'UK Search Ads Q3',
      'US Gaming Install',
      'SA Ecom Flash',
      'SaaS Leads Global',
      'Fintech Acquisition',
      'B2B Enterprise EU',
      'APAC Direct Sales',
      'US Display Retarget',
      'Crypto Exchange VIP',
      'Mobile Gaming Tier1',
      'EU Ecom Sales',
      'US Performance Push',
      'Global Brand Lift'
    ])[(i - 1) % 20 + 1],
    (4200000000 + ((i % 17) * 650000000))::bigint,
    'ACTIVE',
    ('00000000-0000-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    'ASAP',
    (380000000 + ((i % 11) * 95000000))::bigint,
    'UTC',
    100000000,
    3600
FROM generate_series(1, 100) s(i)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  current_spend = 0,
  status = 'ACTIVE',
  budget_limit = EXCLUDED.budget_limit,
  daily_budget = EXCLUDED.daily_budget,
  freq_limit = 100000000;
EOF

log "seeding active license_status (load-test ingest gate)"
"${COMPOSE[@]}" exec -T db psql -h localhost -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor << 'EOF'
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

log "starting control (after postgres/redis/clickhouse ready)"
"${COMPOSE[@]}" up -d --build --no-deps --force-recreate control
wait_control_health

log "starting trackers (registry Sync on boot + pub/sub watch)"
"${COMPOSE[@]}" up -d --build --force-recreate tracker-0 tracker-1

while IFS= read -r port; do
  for _ in $(seq 1 120); do
    if curl -sf "http://127.0.0.1:${port}/health" > /dev/null 2>&1; then
      break
    fi
    sleep 1
  done
done < <(load_test_constrained_ingest_ports)

"${COMPOSE[@]}" stop tracker-2 tracker-3 2> /dev/null || true
"${COMPOSE[@]}" rm -f tracker-2 tracker-3 2> /dev/null || true

"${COMPOSE[@]}" up -d --no-deps nginx

log "full registry snapshot (campaigns:update *)"
bash "$SCRIPTS/test/sync_tracker_registry.sh"

bash "$SCRIPTS/test/seed_load_test_limits.sh"

log "active campaigns:"
"${COMPOSE[@]}" exec -T db psql -h localhost -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor \
  -c "SELECT COUNT(*) FROM campaigns WHERE status = 'ACTIVE';"
