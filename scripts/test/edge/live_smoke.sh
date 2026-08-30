#!/usr/bin/env bash
# Role: Live pixel/track smoke via Unix socket or local tracker against nginx edge.
# Execution context: Compose tracker with UDS volume or USE_LOCAL_TRACKER=1 host binary.
# Env knobs: TRACKER_SOCK_VOL; REBUILD_TRACKER (0); DEPLOY_COMPOSE_TRACKER (1); USE_LOCAL_TRACKER (0).
# Verify: bash scripts/test/edge/live_smoke.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

TRACKER_SOCK_VOL="${TRACKER_SOCK_VOL:-ad-event-processor_ad_event_processor_run}"
TRACKER_SOCK="${TRACKER_SOCK:-/run/ad-event-processor/tracker/tracker-0.sock}"
TRACK_URL="${TRACK_URL:-https://127.0.0.1}"
CORS_ORIGIN="${CORS_ORIGIN:-http://localhost:5173}"
REBUILD_TRACKER="${REBUILD_TRACKER:-0}"
DEPLOY_COMPOSE_TRACKER="${DEPLOY_COMPOSE_TRACKER:-1}"
LOCAL_TRACKER_PORT="${LOCAL_TRACKER_PORT:-8189}"
USE_LOCAL_TRACKER="${USE_LOCAL_TRACKER:-0}"

log() { printf 'pixel-live: %s\n' "$*"; }
die() {
  printf 'pixel-live: FAIL: %s\n' "$*" >&2
  exit 1
}
pass() { printf 'pixel-live: PASS: %s\n' "$*"; }

curl_unix() {
  docker run --rm -v "${TRACKER_SOCK_VOL}:/run/ad-event-processor:ro" curlimages/curl:8.5.0 \
    -sS --unix-socket "$TRACKER_SOCK" "$@"
}

start_local_tracker() {
  if [[ "$USE_LOCAL_TRACKER" != "1" ]]; then
    return 0
  fi
  if [[ ! -x "$ROOT/bin/tracker-live" ]]; then
    log "building static tracker binary"
    (cd "$ROOT" && CGO_ENABLED=0 go build -o bin/tracker-live ./cmd/tracker)
  fi
  if pgrep -f "bin/tracker-live" > /dev/null 2>&1; then
    pkill -f "bin/tracker-live" || true
    sleep 1
  fi
  log "starting local tracker on :${LOCAL_TRACKER_PORT} (TCP redis/pg)"
  (
    cd "$ROOT"
    redis_pass=""
    if [[ -f "$ROOT/.env" ]]; then
      redis_pass="$(grep -E '^REDIS_PASSWORD=' "$ROOT/.env" | head -1 | cut -d= -f2- | tr -d '"')"
    fi
    export SERVER_PORT="$LOCAL_TRACKER_PORT"
    export TRACKER_UNIX_SOCKET=
    export ENV=development
    export DB_DSN="postgres://ad_event_processor_user:secure_pass_123@127.0.0.1:5430/ad_event_processor?sslmode=disable"
    export REDIS_ADDRS="127.0.0.1:6479,127.0.0.1:6480,127.0.0.1:6481,127.0.0.1:6482"
    export REDIS_PASSWORD="$redis_pass"
    export REDIS_PASS="$redis_pass"
    export CH_USE_UDS=0
    export CH_DSN="clickhouse://default:secure_ch_pass@127.0.0.1:9000/ad_event_processor"
    export CH_INGEST_SOURCE=redis
    export BROKER_URL=
    export TRACK_CORS_ORIGINS="${TRACK_CORS_ORIGINS:-${CORS_ORIGIN}}"
    export AD_EVENT_PROCESSOR_LICENSE_PATH="$ROOT/var/license.jwt"
    export AD_EVENT_PROCESSOR_LICENSE_MODE=dev
    export FILTER_TIMEOUT_MS=5000
    export TOKEN_SYMMETRIC_KEY="01234567890123456789012345678901"
    export TRACKER_INGRESS_SCHEMA=ad_event_processor_native
    nohup ./bin/tracker-live > /tmp/pixel-tracker-live.log 2>&1 &
  )
  for _ in $(seq 1 30); do
    code="$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${LOCAL_TRACKER_PORT}/static/track.js" 2> /dev/null || true)"
    [[ "$code" == "200" ]] && return 0
    sleep 1
  done
  die "local tracker did not become ready (see /tmp/pixel-tracker-live.log)"
}

deploy_compose_trackers() {
  if [[ "$DEPLOY_COMPOSE_TRACKER" != "1" ]]; then
    return 0
  fi
  if pgrep -f "bin/tracker-live" > /dev/null 2>&1; then
    pkill -f "bin/tracker-live" || true
    sleep 1
  fi
  log "building static tracker for compose (embed /static/track.js)"
  (cd "$ROOT" && CGO_ENABLED=0 go build -o bin/tracker ./cmd/tracker)
  log "restarting tracker shards + nginx with local binary"
  docker compose -f docker-compose.yaml -f deploy/compose/docker-compose.tracker-local.yaml \
    up -d tracker-0 tracker-1 tracker-2 tracker-3 nginx
  docker compose -f docker-compose.yaml -f deploy/compose/docker-compose.tracker-local.yaml \
    restart tracker-0 tracker-1 tracker-2 tracker-3
  for _ in $(seq 1 45); do
    if docker compose ps tracker-0 nginx 2> /dev/null | grep -q 'healthy'; then
      if docker compose ps nginx 2> /dev/null | grep -q 'Up'; then
        break
      fi
    fi
    sleep 1
  done
  sleep 3
}

if [[ "$REBUILD_TRACKER" == "1" ]]; then
  log "rebuilding tracker-0 (embed /static/track.js)"
  docker compose --project-directory "$ROOT" build tracker-0
  docker compose --project-directory "$ROOT" up -d tracker-0 nginx
  sleep 3
fi

deploy_compose_trackers

start_local_tracker

if [[ "$USE_LOCAL_TRACKER" == "1" ]]; then
  log "step 1: GET /static/track.js on local tracker :${LOCAL_TRACKER_PORT}"
  TRACK_HEAD="$(curl -sS -D - -o /tmp/pixel_track_live.js "http://127.0.0.1:${LOCAL_TRACKER_PORT}/static/track.js" 2>&1 | head -1)"
  printf '%s\n' "$TRACK_HEAD"
  [[ "$TRACK_HEAD" == *"200"* ]] || die "local track.js GET: $TRACK_HEAD"
  grep -q trackEvent /tmp/pixel_track_live.js || die "track.js body missing trackEvent"
  pass "local track.js 200 + trackEvent"
else
  log "step 1: GET /static/track.js via edge ${TRACK_URL}"
  TRACK_HEAD="$(curl -sk -D - -o /tmp/pixel_track_live.js "${TRACK_URL}/static/track.js" 2>&1 | head -1)"
  printf '%s\n' "$TRACK_HEAD"
  [[ "$TRACK_HEAD" == *"200"* ]] || die "edge track.js GET: $TRACK_HEAD"
  grep -q trackEvent /tmp/pixel_track_live.js || die "track.js body missing trackEvent"
  pass "edge track.js 200 + trackEvent"
fi

CAMPAIGN_ID="$(
  docker exec ad-event-processor-db-1 psql -h /run/ad-event-processor/postgresql -p 5430 \
    -U ad_event_processor_user -d ad_event_processor -t -A \
    -c "SELECT id::text FROM campaigns WHERE status = 'ACTIVE' ORDER BY created_at DESC LIMIT 1;" 2> /dev/null \
    | tr -d '[:space:]'
)"
[[ -n "$CAMPAIGN_ID" ]] || die "no active campaign in Postgres"
log "campaign_id=${CAMPAIGN_ID}"

EVENT_ID="pixel-live-$(date +%s)"
CLICK_ID="clk-${EVENT_ID}"
FBCLID="fb.live.${CLICK_ID}"
CONV_BODY="$(
  cat << EOF
{"campaign_id":"${CAMPAIGN_ID}","type":"conversion","click_id":"${CLICK_ID}","user_id":"pixel-live","fbclid":"${FBCLID}","event_id":"${EVENT_ID}"}
EOF
)"

log "step 2: POST /track conversion"
if [[ "$USE_LOCAL_TRACKER" == "1" ]]; then
  TRACK_CODE="$(curl -sS -o /tmp/pixel_track_resp.txt -w '%{http_code}' \
    -X POST "http://127.0.0.1:${LOCAL_TRACKER_PORT}/track" \
    -H 'Content-Type: application/json' \
    -H "Content-Length: ${#CONV_BODY}" \
    -d "$CONV_BODY")"
else
  TRACK_CODE="$(curl -sk -o /tmp/pixel_track_resp.txt -w '%{http_code}' \
    -X POST "${TRACK_URL}/track" \
    -H 'Content-Type: application/json' \
    -H "Origin: ${CORS_ORIGIN}" \
    -H "Content-Length: ${#CONV_BODY}" \
    -d "$CONV_BODY")"
fi
log "POST /track -> HTTP ${TRACK_CODE}"
[[ "$TRACK_CODE" == "202" || "$TRACK_CODE" == "200" ]] || die "track conversion failed HTTP ${TRACK_CODE}"
pass "POST /track ${TRACK_CODE}"

log "step 3: OPTIONS /track CORS preflight"
if [[ "$USE_LOCAL_TRACKER" == "1" ]]; then
  CORS_HEADERS="$(curl -sS -D - -o /dev/null -X OPTIONS "http://127.0.0.1:${LOCAL_TRACKER_PORT}/track" \
    -H "Origin: ${CORS_ORIGIN}" \
    -H 'Access-Control-Request-Method: POST' 2>&1 || true)"
else
  CORS_HEADERS="$(curl -sk -D - -o /dev/null -X OPTIONS "${TRACK_URL}/track" \
    -H "Origin: ${CORS_ORIGIN}" \
    -H 'Access-Control-Request-Method: POST' 2>&1 || true)"
fi
if printf '%s' "$CORS_HEADERS" | grep -qi 'access-control-allow-origin'; then
  pass "CORS preflight includes Access-Control-Allow-Origin"
else
  die "CORS preflight missing Allow-Origin (TRACK_CORS_ORIGINS must include ${CORS_ORIGIN})"
fi

if [[ "$USE_LOCAL_TRACKER" == "1" ]]; then
  log "step 4: GET /static/track.js via edge ${TRACK_URL}"
  EDGE_CODE="$(curl -sk --max-time 10 -o /dev/null -w '%{http_code}' "${TRACK_URL}/static/track.js" || true)"
  log "edge GET /static/track.js -> HTTP ${EDGE_CODE}"
  [[ "$EDGE_CODE" == "200" ]] || die "edge static track.js HTTP ${EDGE_CODE}"

  log "step 5: POST /track via edge"
  EDGE_TRACK="$(curl -sk --max-time 15 -o /dev/null -w '%{http_code}' \
    -X POST "${TRACK_URL}/track" \
    -H 'Content-Type: application/json' \
    -H "Content-Length: ${#CONV_BODY}" \
    -d "$CONV_BODY" || true)"
  log "edge POST /track -> HTTP ${EDGE_TRACK}"
  [[ "$EDGE_TRACK" == "202" || "$EDGE_TRACK" == "200" ]] || die "edge POST /track HTTP ${EDGE_TRACK}"
  pass "edge POST /track ${EDGE_TRACK}"
fi

log "done: event_id=${EVENT_ID} click_id=${CLICK_ID}"
