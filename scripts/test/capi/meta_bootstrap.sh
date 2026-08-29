#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

CAMPAIGN_ID="${CAMPAIGN_ID:-00000000-0000-0000-0000-000000000005}"
CAPI_BRAND_ID="${CAPI_BRAND_ID:-00000000-0000-0000-0000-00000000000b}"
CAPI_CREATIVE_ID="${CAPI_CREATIVE_ID:-00000000-0000-0000-0000-00000000000c}"
META_MOCK_PORT="${META_MOCK_PORT:-9199}"
META_TEST_EVENT_CODE="${META_TEST_EVENT_CODE:-TEST12345}"
TRACK_URL="${TRACK_URL:-http://127.0.0.1:8181}"
CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
POSTBACK_METRICS_URL="${POSTBACK_METRICS_URL:-http://127.0.0.1:9119/metrics}"
MOCK_PID_FILE="${TMPDIR:-/tmp}/capi-meta-mock.pid"

log() { printf 'capi-bootstrap: %s\n' "$*"; }
die() {
  printf 'capi-bootstrap: ERROR: %s\n' "$*" >&2
  exit 1
}

load_env() {
  if [[ -f "$ROOT/.env" ]]; then
    set -a

    source "$ROOT/.env"
    set +a
  fi
  : "${DB_USER:=ad_event_processor_user}"
  : "${DB_PASSWORD:=secure_pass_123}"
  : "${DB_PORT:=5430}"
  : "${DB_NAME:=ad_event_processor}"
  : "${REDIS_PASSWORD:=your_redis_password_here}"
  : "${ADMIN_API_KEY:=dev-admin-api-key-change-me}"
  : "${TOKEN_SYMMETRIC_KEY:=01234567890123456789012345678901}"
  export POSTBACK_ENCRYPTION_KEY="${POSTBACK_ENCRYPTION_KEY:-$TOKEN_SYMMETRIC_KEY}"
  export DB_DSN="${DB_DSN:-postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:${DB_PORT}/${DB_NAME}?sslmode=disable}"

  export REDIS_ADDRS="127.0.0.1:6479,127.0.0.1:6480,127.0.0.1:6481,127.0.0.1:6482"
  export CH_ENABLED="${CH_ENABLED:-0}"
  export TRACKER_INGRESS_SCHEMA="${TRACKER_INGRESS_SCHEMA:-ad_event_processor_native}"
  export CH_INGEST_SOURCE="${CH_INGEST_SOURCE:-redis}"
}

psql_exec() {
  docker exec ad-event-processor-db-1 psql -h /run/ad-event-processor/postgresql -p "$DB_PORT" \
    -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "$1"
}

ensure_stack() {
  log "ensuring db/redis/tracker/processor/control (REDIS_ADDRS=${REDIS_ADDRS}, CH_INGEST_SOURCE=${CH_INGEST_SOURCE})"
  CH_ENABLED="${CH_ENABLED}" REDIS_ADDRS="$REDIS_ADDRS" TRACKER_INGRESS_SCHEMA="$TRACKER_INGRESS_SCHEMA" \
    CH_INGEST_SOURCE="$CH_INGEST_SOURCE" \
    docker compose up -d db redis-0 redis-1 redis-2 redis-3 > /dev/null

  CH_ENABLED="${CH_ENABLED}" CH_INGEST_SOURCE="$CH_INGEST_SOURCE" REDIS_ADDRS="$REDIS_ADDRS" \
    docker compose up -d --no-deps control > /dev/null || true
  CH_INGEST_SOURCE="$CH_INGEST_SOURCE" REDIS_ADDRS="$REDIS_ADDRS" TRACKER_INGRESS_SCHEMA="$TRACKER_INGRESS_SCHEMA" \
    docker compose up -d --no-deps processor tracker-0 > /dev/null
  for _ in $(seq 1 30); do
    curl -sf -m 2 "${TRACK_URL%/}/health" > /dev/null 2>&1 && break
    sleep 2
  done
  curl -sf -m 5 "${TRACK_URL%/}/health" > /dev/null || die "tracker not healthy at ${TRACK_URL}"
  for _ in $(seq 1 30); do
    curl -sf -m 2 "${CONTROL_URL%/}/health" > /dev/null 2>&1 && break
    sleep 2
  done
  curl -sf -m 5 "${CONTROL_URL%/}/health" > /dev/null || die "control not healthy at ${CONTROL_URL}"
}

db_prep() {
  log "applying postback + safe_page migrations if missing"
  psql_exec "ALTER TABLE postback_configs ADD COLUMN IF NOT EXISTS test_event_code TEXT NOT NULL DEFAULT '';" > /dev/null
  psql_exec "ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS safe_page_url TEXT NOT NULL DEFAULT '', ADD COLUMN IF NOT EXISTS safe_page_enabled BOOLEAN NOT NULL DEFAULT false;" > /dev/null
  psql_exec "ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS cidr_block_enabled BOOLEAN NOT NULL DEFAULT true;" > /dev/null 2>&1 || true
  psql_exec "ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS click_delivery TEXT NOT NULL DEFAULT 'redirect';" > /dev/null 2>&1 || true
  psql_exec "ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS proxy_upstream_url TEXT NOT NULL DEFAULT '';" > /dev/null 2>&1 || true
  psql_exec "ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS proxy_rewrite_assets BOOLEAN NOT NULL DEFAULT false;" > /dev/null 2>&1 || true

  log "ensuring brand + landing creative for /click redirect"
  psql_exec "
    INSERT INTO advertiser_brands (id, customer_id, name)
    SELECT '${CAPI_BRAND_ID}'::uuid, c.customer_id, 'capi-staging-brand'
    FROM campaigns c WHERE c.id='${CAMPAIGN_ID}'::uuid
    ON CONFLICT (id) DO NOTHING;
    INSERT INTO brand_creatives (id, brand_id, name, landing_url, weight, status)
    VALUES ('${CAPI_CREATIVE_ID}'::uuid, '${CAPI_BRAND_ID}'::uuid, 'capi-default',
            'https://example.com/landing?cid={click_id}', 100, 'ACTIVE')
    ON CONFLICT (brand_id, name) DO UPDATE
      SET landing_url=EXCLUDED.landing_url, status='ACTIVE', updated_at=NOW();
    UPDATE campaigns SET brand_id='${CAPI_BRAND_ID}'::uuid
    WHERE id='${CAMPAIGN_ID}'::uuid AND brand_id IS NULL;
  " > /dev/null

  local creatives_json='[{"id":"'"${CAPI_CREATIVE_ID}"'","url":"https://example.com/landing?cid={click_id}","weight":100}]'
  for port in 6479 6480 6481 6482; do
    docker run --rm --network host redis:7-alpine redis-cli -h 127.0.0.1 -p "$port" \
      -a "$REDIS_PASSWORD" SET "brand:creatives:${CAPI_BRAND_ID}" "$creatives_json" > /dev/null
  done

  log "setting target_url on campaign ${CAMPAIGN_ID} (legacy fallback)"
  psql_exec "UPDATE campaigns SET target_url='https://example.com/landing?cid={click_id}' WHERE id='${CAMPAIGN_ID}'::uuid AND length(target_url)=0;" > /dev/null
}

start_mock_meta() {
  if [[ -n "${META_CAPI_URL:-}" ]]; then
    log "using real Meta CAPI URL (mock skipped)"
    CAPI_URL="$META_CAPI_URL"
    return
  fi
  CAPI_URL="http://127.0.0.1:${META_MOCK_PORT}/events"
  if curl -sf -m 1 -o /dev/null "http://127.0.0.1:${META_MOCK_PORT}/health" 2> /dev/null; then
    log "mock Meta receiver already listening on :${META_MOCK_PORT}"
    return
  fi
  log "starting mock Meta CAPI receiver on :${META_MOCK_PORT}"
  python3 - "$META_MOCK_PORT" "$MOCK_PID_FILE" << 'PY' &
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

port = int(sys.argv[1])
pid_file = sys.argv[2]

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"OK")
            return
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        _ = self.rfile.read(int(self.headers.get("Content-Length", 0) or 0))
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"events_received":1}')

    def log_message(self, fmt, *args):
        return

srv = HTTPServer(("127.0.0.1", port), Handler)
with open(pid_file, "w", encoding="utf-8") as f:
    f.write(str(__import__("os").getpid()))
srv.serve_forever()
PY
  sleep 1
  curl -sf -m 3 "http://127.0.0.1:${META_MOCK_PORT}/health" > /dev/null || die "mock Meta receiver failed to start"
}

configure_postback() {
  local token="${META_ACCESS_TOKEN:-local-mock-capi-token}"
  log "upserting facebook postback config for ${CAMPAIGN_ID}"
  local body
  body="$(printf '{"provider":"facebook","url_template":"%s","api_token":"%s","target_event":"conversion","test_event_code":"%s"}' \
    "$CAPI_URL" "$token" "$META_TEST_EVENT_CODE")"
  code="$(curl -sS -o /dev/null -w '%{http_code}' -X PUT \
    "${CONTROL_URL}/api/v1/postbacks/config/${CAMPAIGN_ID}" \
    -H "X-Admin-API-Key: ${ADMIN_API_KEY}" \
    -H 'Content-Type: application/json' \
    -d "$body")"
  [[ "$code" == "200" ]] || die "postback config PUT failed (HTTP ${code})"
  psql_exec "UPDATE postback_configs SET test_event_code='${META_TEST_EVENT_CODE}' WHERE campaign_id='${CAMPAIGN_ID}'::uuid;" > /dev/null
}

trim_event_streams() {
  log "trimming poisoned ad:events:stream batches on redis shards"
  for port in 6479 6480 6481 6482; do
    docker run --rm --network host redis:7-alpine redis-cli -h 127.0.0.1 -p "$port" \
      -a "$REDIS_PASSWORD" XTRIM ad:events:stream MAXLEN 0 > /dev/null 2>&1 || true
  done
}

sync_registry() {
  log "publishing registry reload for ${CAMPAIGN_ID}"
  docker exec ad-event-processor-redis-0-1 redis-cli -p 6379 -a "$REDIS_PASSWORD" \
    PUBLISH campaigns:update "$CAMPAIGN_ID" > /dev/null 2>&1 || true
  docker exec ad-event-processor-redis-0-1 redis-cli -p 6379 -a "$REDIS_PASSWORD" \
    PUBLISH campaigns:update '*' > /dev/null 2>&1 || true
  sleep 3
}

build_tracker_if_needed() {
  if git diff --quiet HEAD -- internal/ingest/schedule_filter.go 2> /dev/null; then
    return
  fi
  log "rebuilding tracker (brand creative lazy-load)"
  SKIP_CODEGEN=1 docker compose build tracker-0 > /dev/null
}

start_postback_sender() {
  if curl -sf -m 2 "$POSTBACK_METRICS_URL" > /dev/null 2>&1; then
    log "postback-sender metrics already up"
    return
  fi
  pkill -f '/bin/postback-sender' 2> /dev/null || true
  [[ -x "$ROOT/bin/postback-sender" ]] || die "build postback-sender: go build -o bin/postback-sender ./cmd/postback-sender"
  log "starting postback-sender (${POSTBACK_METRICS_URL})"
  nohup env DB_DSN="$DB_DSN" POSTBACK_ENCRYPTION_KEY="$POSTBACK_ENCRYPTION_KEY" \
    POSTBACK_METRICS_ADDR="127.0.0.1:9119" \
    "$ROOT/bin/postback-sender" > "${TMPDIR:-/tmp}/postback-sender.log" 2>&1 &
  for _ in $(seq 1 15); do
    curl -sf -m 2 "$POSTBACK_METRICS_URL" > /dev/null 2>&1 && return
    sleep 1
  done
  die "postback-sender did not expose metrics"
}

run_smoke() {
  export TRACK_URL CAMPAIGN_ID CONTROL_URL POSTBACK_METRICS_URL META_TEST_EVENT_CODE ADMIN_API_KEY
  export CAPI_BOOTSTRAP_DB=1
  exec bash "$SCRIPTS/test/capi/meta_staging.sh"
}

main() {
  load_env
  build_tracker_if_needed
  ensure_stack
  db_prep
  start_mock_meta
  configure_postback
  sync_registry
  trim_event_streams
  start_postback_sender
  if [[ "${1:-}" == "run" ]]; then
    run_smoke
  else
    log "bootstrap complete - run: bash scripts/test/capi/meta_bootstrap.sh run"
  fi
}

main "$@"
