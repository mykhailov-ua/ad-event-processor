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
EOF

go run ./cmd/admin db seed-prep-test-sql --count 100 | docker exec -i ad-event-processor-db-1 psql -h localhost -p 5440 -U ad_event_processor_user -d ad_event_processor

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
