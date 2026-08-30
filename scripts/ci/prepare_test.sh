#!/bin/bash
set -e

# Role: Bring up compose stack for integration prep (operator/CI integration tier).
# Execution context: CI merge-integration prep and operator; starts Docker compose services.
# Invariants/contracts enforced: Waits for Postgres/Redis/ClickHouse health before tests.
# Verify: bash scripts/ci/prepare_test.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "Stopping and cleaning up containers (including orphans)"
docker compose down --remove-orphans

echo "Starting services"
docker compose up -d --build --remove-orphans

echo "Waiting for services to become healthy"
echo "Waiting for Postgres..."
until docker exec ad-event-processor-db-1 pg_isready -p 5440 -U ad_event_processor_user -d ad_event_processor > /dev/null 2>&1; do
  sleep 1
done

echo "Waiting for ClickHouse..."
until docker exec ad-event-processor-clickhouse-1 wget -qO- http://127.0.0.1:8123/ping > /dev/null 2>&1; do
  sleep 1
done

echo "Waiting for Redis shards..."
for i in 0 1 2 3 4 5; do
  until docker exec ad-event-processor-redis-$i-1 redis-cli -p 6379 -a redis_secure_pass_456 ping > /dev/null 2>&1; do
    sleep 1
  done
done

echo "Cleaning Redis shards"
for i in 0 1 2 3 4 5; do
  docker exec ad-event-processor-redis-$i-1 redis-cli -p 6379 -a redis_secure_pass_456 FLUSHALL > /dev/null 2>&1
done

echo "Resetting Postgres database"
docker exec -i ad-event-processor-db-1 psql -h localhost -p 5440 -U ad_event_processor_user -d ad_event_processor << 'EOF'
TRUNCATE TABLE events CASCADE;
TRUNCATE TABLE campaign_stats CASCADE;

-- Insert 100 customers with varied wallet balances (micro-units)
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
    ])[(i - 1) % 20 + 1]
    || ' - ' ||
    (ARRAY['US East', 'US West', 'EU North', 'APAC', 'LATAM'])[((i - 1) / 20) % 5 + 1],
    (2400000000 + ((i * 2817431) % 281600000000) + ((i % 7) * 97000000))::bigint,
    'USD',
    0
FROM generate_series(1, 100) s(i)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  balance = EXCLUDED.balance;

-- Insert 100 active campaigns with realistic budgets (micro-units)
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
      'Global Brand Lift',
      'DE Finance Leads',
      'BR Nutra Push',
      'JP Mobile Subs',
      'CA Insurance CPL',
      'AU Solar Quotes',
      'MX Remittance App',
      'IN UPI Onboarding',
      'PL Ecom Remarketing',
      'IT Travel Meta',
      'ES Telco Prepaid'
    ])[(i - 1) % 30 + 1]
    || ' - ' ||
    (ARRAY['US', 'GB', 'CA', 'UA', 'DE', 'FR', 'JP'])[((i - 1) / 30) % 7 + 1]
    || ' - ' ||
    (ARRAY['Alpha desk', 'Bravo desk', 'Cedar desk', 'Delta desk', 'Echo desk'])[((i - 1) / (30 * 7)) % 5 + 1],
    (4200000000 + ((i % 17) * 650000000) + ((i % 9) * 384729))::bigint,
    'ACTIVE',
    ('00000000-0000-0000-0000-' || LPAD(to_hex(i), 12, '0'))::uuid,
    'ASAP',
    (380000000 + ((i % 11) * 95000000) + ((i % 6) * 18473))::bigint,
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

echo "Resetting ClickHouse database"
docker exec -i ad-event-processor-clickhouse-1 clickhouse-client --multiquery -u default --password secure_ch_pass -d ad_event_processor -q "
TRUNCATE TABLE impressions;
TRUNCATE TABLE clicks;
TRUNCATE TABLE conversions;
TRUNCATE TABLE fraud_events;
"

echo "Applying postgres migrations (ads, auth, billing)"
if [[ -f .env ]]; then
  set -a
  source .env
  set +a
fi
DB_PORT=${DB_PORT:-5440}
export DB_DSN="${DB_DSN:-postgres://${DB_USER:-ad_event_processor_user}:${DB_PASSWORD:-secure_pass_123}@127.0.0.1:${DB_PORT}/${DB_NAME:-ad_event_processor}?sslmode=${DB_SSLMODE:-disable}}"
go run ./cmd/migrate-cold-path --only=ads,auth,billing

echo "Repairing schema drift after migrations"
DB_PORT="$DB_PORT" bash scripts/test/reconcile_ingestion_migrations.sh
DB_PORT="$DB_PORT" bash scripts/test/load/verify_schema.sh

echo "Restarting trackers and processor to recreate consumer groups"
docker compose up -d --build --force-recreate processor tracker-0 tracker-1 tracker-2 tracker-3

echo "Triggering full campaign registry snapshot via Redis Pub/Sub"
REDIS_PASS="${REDIS_PASSWORD:-redis_secure_pass_456}"
docker exec ad-event-processor-redis-0-1 redis-cli -p 6379 -a "$REDIS_PASS" \
  PUBLISH campaigns:update "*" > /dev/null 2>&1
sleep 3

echo "Verification"
echo "Active campaign count in Postgres:"
docker exec ad-event-processor-db-1 psql -h localhost -p 5440 -U ad_event_processor_user -d ad_event_processor -c "SELECT COUNT(*) FROM campaigns WHERE status = 'ACTIVE';"

echo "All systems ready for load test!"
