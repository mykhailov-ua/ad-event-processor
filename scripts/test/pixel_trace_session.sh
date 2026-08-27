#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/go.sh"
cd "$ROOT"

SESSION_ROOT="${AD_EVENT_PROCESSOR_BPF_SESSION_ROOT:-$ROOT/var/bpf-session/pixel}"
OUT_DIR="${1:-$SESSION_ROOT/$(date -u +%Y%m%dT%H%M%SZ)}"
TRACK_REPS="${PIXEL_TRACE_TRACK_REPS:-50}"
METRICS_PORT="${PIXEL_TRACE_METRICS_PORT:-9101}"
CORS_ORIGIN="${CORS_ORIGIN:-http://localhost:5173}"
TRACK_URL="${TRACK_URL:-https://127.0.0.1}"

log() { printf 'pixel-trace: %s\n' "$*"; }
die() {
  printf 'pixel-trace: FAIL: %s\n' "$*" >&2
  exit 1
}

CAMPAIGN_ID="$(
  docker exec ad-event-processor-db-1 psql -h /run/ad-event-processor/postgresql -p 5430 \
    -U ad_event_processor_user -d ad_event_processor -t -A \
    -c "SELECT id::text FROM campaigns WHERE status = 'ACTIVE' ORDER BY created_at DESC LIMIT 1;" 2> /dev/null \
    | tr -d '[:space:]'
)"
[[ -n "$CAMPAIGN_ID" ]] || die "no active campaign in Postgres"

log "building tracker with bpf trace markers (-tags ad_event_processor_bpf_trace)"
CGO_ENABLED=0 ad_event_processor_go_build -tags ad_event_processor_bpf_trace -o bin/tracker ./cmd/tracker

log "restarting tracker shards (tracker-local override)"
docker compose -f docker-compose.yaml -f deploy/compose/docker-compose.tracker-local.yaml \
  up -d tracker-0 tracker-1 tracker-2 tracker-3 nginx
docker compose -f docker-compose.yaml -f deploy/compose/docker-compose.tracker-local.yaml \
  restart tracker-0 tracker-1 tracker-2 tracker-3
for _ in $(seq 1 90); do
  code="$(curl -sk -o /dev/null -w '%{http_code}' "${TRACK_URL}/static/track.js" 2> /dev/null || true)"
  if [[ "$code" == "200" ]]; then
    break
  fi
  sleep 1
done
for _ in $(seq 1 30); do
  body="$(printf '{"campaign_id":"%s","type":"conversion","click_id":"warmup","event_id":"warmup"}' "$CAMPAIGN_ID")"
  code="$(curl -sk -o /dev/null -w '%{http_code}' \
    -X POST "${TRACK_URL}/track" \
    -H 'Content-Type: application/json' \
    -H "Origin: ${CORS_ORIGIN}" \
    -H "Content-Length: ${#body}" \
    -d "$body" 2> /dev/null || true)"
  if [[ "$code" == "202" ]]; then
    break
  fi
  sleep 1
done

if ! bash "$SCRIPTS/test/bpf_requirements.sh"; then
  die "BPF preflight failed (kernel headers, memlock, or tracing fs)"
fi

log "starting bpf session: $OUT_DIR"
export AD_EVENT_PROCESSOR_BPF_TRACKER_BINARY="$ROOT/bin/tracker"
export AD_EVENT_PROCESSOR_BPF_DUMP_INTERVAL="${AD_EVENT_PROCESSOR_BPF_DUMP_INTERVAL:-15}"
export AD_EVENT_PROCESSOR_BPF_SLOW_US="${AD_EVENT_PROCESSOR_BPF_SLOW_US:-1000}"
export AD_EVENT_PROCESSOR_BPF_TRACK_LOADGEN="${AD_EVENT_PROCESSOR_BPF_TRACK_LOADGEN:-0}"
bash "$SCRIPTS/dev/bpf_session.sh" start "$OUT_DIR"

log "load POST /track x${TRACK_REPS} via edge (campaign_id=${CAMPAIGN_ID})"
for i in $(seq 1 "$TRACK_REPS"); do
  event_id="pixel-trace-${i}-$(date +%s)"
  body="$(printf '{"campaign_id":"%s","type":"conversion","click_id":"clk-%s","event_id":"%s","user_id":"pixel-trace"}' \
    "$CAMPAIGN_ID" "$event_id" "$event_id")"
  curl -sk -o /dev/null -w '%{http_code}\n' \
    -X POST "${TRACK_URL}/track" \
    -H 'Content-Type: application/json' \
    -H "Origin: ${CORS_ORIGIN}" \
    -H "Content-Length: ${#body}" \
    -d "$body" > /dev/null || true
done

log "pixel live smoke (sanity, no tracker rebuild)"
DEPLOY_COMPOSE_TRACKER=0 bash "$SCRIPTS/test/pixel_live_smoke.sh" || die "pixel_live_smoke failed"

log "collecting metrics snapshot from :${METRICS_PORT}"
mkdir -p "$OUT_DIR"
curl -sf "http://127.0.0.1:${METRICS_PORT}/metrics" \
  | grep -E 'ad_http_request_duration_seconds_|ad_redis_lua_duration_seconds_|filter_lua_slow_total' \
    > "$OUT_DIR/tracker-metrics-snapshot.txt" || true

if curl -sf "http://127.0.0.1:${METRICS_PORT}/debug/pprof/" > /dev/null 2>&1; then
  log "capturing 5s execution trace (TRACKER_PPROF_ENABLED=1)"
  curl -sf "http://127.0.0.1:${METRICS_PORT}/debug/pprof/trace?seconds=5" -o "$OUT_DIR/track.trace" || true
fi

log "stopping bpf session"
bash "$SCRIPTS/dev/bpf_session.sh" stop "$OUT_DIR"

if command -v go > /dev/null 2>&1; then
  log "generating bpf report"
  bash "$SCRIPTS/dev/bpf_session.sh" report "$OUT_DIR" || true
fi

log "done"
log "  bpf report:  $OUT_DIR/bpf-report.md"
log "  bpf summary: $OUT_DIR/bpf/maps/summary.json"
log "  metrics:     $OUT_DIR/tracker-metrics-snapshot.txt"
if [[ -f "$OUT_DIR/track.trace" ]]; then
  log "  go trace:    go tool trace $OUT_DIR/track.trace"
fi
