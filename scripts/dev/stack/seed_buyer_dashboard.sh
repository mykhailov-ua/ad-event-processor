#!/usr/bin/env bash
# Role: Seed Postgres + ClickHouse with realistic buyer dashboard demo data.
# Execution context: ingest-only stack with optional ClickHouse; host-side admin CLI on DB_PORT.
# Env knobs: SEED_BUYER_COUNT (default 50); CH_PORT, CH_PASSWORD, CH_ENABLED.
# Verify: bash scripts/dev/stack/seed_buyer_dashboard.sh
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
DB_USER="${DB_USER:-ad_event_processor_user}"
DB_PASSWORD="${DB_PASSWORD:-secure_pass_123}"
DB_NAME="${DB_NAME:-ad_event_processor}"
DB_SSLMODE="${DB_SSLMODE:-disable}"
CH_PORT="${CH_PORT:-9000}"
CH_PASSWORD="${CH_PASSWORD:-secure_ch_pass}"
SEED_BUYER_COUNT="${SEED_BUYER_COUNT:-120}"

log() { printf 'seed-buyer-dashboard: %s\n' "$*"; }
die() {
  printf 'seed-buyer-dashboard: ERROR: %s\n' "$*" >&2
  exit 1
}

host_db_dsn() {
  printf 'postgres://%s:%s@127.0.0.1:%s/%s?sslmode=%s' "$DB_USER" "$DB_PASSWORD" "$DB_PORT" "$DB_NAME" "$DB_SSLMODE"
}

run_admin() {
  DB_DSN="$(host_db_dsn)" go run ./cmd/admin --env-path .env "$@"
}

if ! docker compose --project-directory "$ROOT" -f docker-compose.yaml ps --status running db 2> /dev/null | grep -q db; then
  die "postgres not running; start stack: bash scripts/dev/stack/stack.sh ingest-only"
fi

log "step 0: ensure ARCHIVED campaign status enum"
docker exec ad-event-processor-db-1 psql -h localhost -p "${DB_PORT:-5430}" -U ad_event_processor_user -d ad_event_processor \
  -c "ALTER TYPE campaign_status_type ADD VALUE IF NOT EXISTS 'ARCHIVED';" > /dev/null

log "step 1/6: ensure base campaigns (count=${SEED_BUYER_COUNT})"
SEED_INGEST_CAMPAIGN_COUNT="${SEED_BUYER_COUNT}" bash "$ROOT/scripts/test/load/seed_ingest_only_campaigns.sh"

log "step 2/6: PG UI stats (names, spend, campaign_stats)"
DB_DSN="$(host_db_dsn)" go run ./cmd/admin --env-path .env db seed-ui --count "${SEED_BUYER_COUNT}"

log "step 3/6: assign campaigns to demo customer (Horizon portfolio)"
run_admin db seed-buyer-pg --count "${SEED_BUYER_COUNT}" --customer-seq 1

log "step 4/6: campaign list UX (owners, countries, margin breach, customers)"
run_admin db seed-campaign-list-ux --count "${SEED_BUYER_COUNT}"

log "step 5/6: start ClickHouse if needed"
if ! docker compose --project-directory "$ROOT" -f docker-compose.yaml ps --status running clickhouse 2> /dev/null | grep -q clickhouse; then
  bash "$ROOT/scripts/dev/stack/stack.sh" clickhouse
fi

for attempt in $(seq 1 30); do
  if docker exec ad-event-processor-clickhouse-1 wget -qO- http://127.0.0.1:8123/ping > /dev/null 2>&1; then
    break
  fi
  sleep 2
  if [[ "$attempt" -eq 30 ]]; then
    die "clickhouse did not become healthy"
  fi
done

log "step 6/6: seed ClickHouse traffic and economics"
export CH_USE_UDS=0
export CH_ENABLED=1
export CH_DSN="clickhouse://${CH_USER:-default}:${CH_PASSWORD}@127.0.0.1:${CH_PORT}/${CH_NAME:-ad_event_processor}"
export CH_READONLY_DSN="$CH_DSN"
run_admin db seed-buyer-ch --count "${SEED_BUYER_COUNT}" --customer-seq 1 --replace

log "recreating control with ClickHouse enabled for dashboard queries"
COMPOSE=(docker compose --project-directory "$ROOT" -f docker-compose.yaml -f deploy/compose/docker-compose.control-dev.yaml)
if [[ -f "$ROOT/.env" ]]; then
  COMPOSE+=(--env-file "$ROOT/.env")
fi
CH_USE_UDS=0 CH_ENABLED=1 CH_DSN="$CH_DSN" CH_READONLY_DSN="$CH_DSN" \
  "${COMPOSE[@]}" --profile ingest_only up -d --force-recreate control

demo_customer="$(run_admin db seed-uuids-shell --count 1 2>/dev/null | sed -n "s/^AED_CUSTOMER_UUID_1='\\(.*\\)'/\\1/p")"
demo_customer_name="$(DB_DSN="$(host_db_dsn)" go run ./cmd/admin --env-path .env db customer list 2>/dev/null | awk -F'|' 'NR>1 && index($0,"'"${demo_customer}"'"){gsub(/^ +| +$/,"",$2); print $2; exit}')"

cat << EOF

seed-buyer-dashboard: done
  Campaigns:   ${SEED_BUYER_COUNT} under demo customer
  Customer ID: ${demo_customer}
  Customer:    ${demo_customer_name:-Horizon Media Group}
  ClickHouse:  enabled on 127.0.0.1:${CH_PORT}/${CH_NAME:-ad_event_processor}
  Next: open Buyer dashboard and select the demo customer in scope bar

EOF
