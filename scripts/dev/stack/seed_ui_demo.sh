#!/usr/bin/env bash
# Role: Seed admin UI demo campaigns with varied names, spend, and delivery stats for charts.
# Execution context: Requires existing seed campaigns (deterministic ids seq 1..N); uses host PG on DB_PORT.
# Env knobs: SEED_UI_DEMO_COUNT (default 50).
# Verify: bash scripts/dev/stack/seed_ui_demo.sh && curl -sf http://127.0.0.1:8188/health
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

log() { printf 'seed-ui-demo: %s\n' "$*"; }
die() {
  printf 'seed-ui-demo: ERROR: %s\n' "$*" >&2
  exit 1
}

host_db_dsn() {
  printf 'postgres://%s:%s@127.0.0.1:%s/%s?sslmode=%s' "$DB_USER" "$DB_PASSWORD" "$DB_PORT" "$DB_NAME" "$DB_SSLMODE"
}

if ! docker compose --project-directory "$ROOT" -f docker-compose.yaml ps --status running db 2> /dev/null | grep -q db; then
  log "db container not running; start stack first (bash scripts/dev/stack/stack.sh ingest-only)"
  die "postgres unavailable"
fi

SEED_UI_DEMO_COUNT="${SEED_UI_DEMO_COUNT:-50}"

if [[ -x "$ROOT/scripts/test/load/seed_ingest_only_campaigns.sh" ]]; then
  campaign_count="$(docker exec ad-event-processor-db-1 psql -h localhost -p 5430 -U ad_event_processor_user -d ad_event_processor -tAc 'SELECT count(*) FROM campaigns;' 2>/dev/null || echo 0)"
  if [[ "${campaign_count:-0}" -lt "${SEED_UI_DEMO_COUNT}" ]]; then
    log "ensuring base campaigns exist (found ${campaign_count:-0}, need ${SEED_UI_DEMO_COUNT})"
    SEED_INGEST_CAMPAIGN_COUNT="${SEED_UI_DEMO_COUNT}" bash "$ROOT/scripts/test/load/seed_ingest_only_campaigns.sh"
  else
    log "skipping ingest seed (${campaign_count} campaigns already present)"
  fi
fi

log "seeding UI demo stats and campaign fields (count=${SEED_UI_DEMO_COUNT})"
DB_DSN="$(host_db_dsn)" go run ./cmd/admin --env-path .env db seed-ui --count "${SEED_UI_DEMO_COUNT}"

log "done — reload Campaigns in admin UI"
