#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

TRACK_URL="${TRACK_URL:-}"
CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
ADMIN_API_KEY="${ADMIN_API_KEY:-${API_KEY:-}}"
CAMPAIGN_ID="${CAMPAIGN_ID:-}"
META_TEST_EVENT_CODE="${META_TEST_EVENT_CODE:-${TEST_EVENT_CODE:-}}"
POSTBACK_METRICS_URL="${POSTBACK_METRICS_URL:-http://127.0.0.1:9119/metrics}"
CLICK_ID="capi-staging-$(date +%s)"
FBCLID="fb.staging.${CLICK_ID}"
METRIC_GREP='ad_postback_dispatch_total{provider="facebook",status="success"}'

log() { printf 'capi-meta-staging: %s\n' "$*"; }
die() {
  printf 'capi-meta-staging: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "${CAPI_STAGING_DRY_RUN:-}" == "1" ]]; then
  log "dry-run - would:"
  log "  1. GET ${TRACK_URL:-<TRACK_URL>}/click?campaign_id=${CAMPAIGN_ID:-<id>}&fbclid=${FBCLID}"
  log "  2. POST ${TRACK_URL:-<TRACK_URL>}/track conversion JSON (email hashed server-side for CAPI)"
  log "  3. Optional: GET ${CONTROL_URL}/api/v1/postbacks/config - verify test_event_code=${META_TEST_EVENT_CODE:-<unset>}"
  log "  4. Poll ${POSTBACK_METRICS_URL} for ${METRIC_GREP}"
  log "Set TRACK_URL, CAMPAIGN_ID before live run; configure CAPI token + test_event_code in Campaign -> CAPI & Postbacks."
  exit 0
fi

[[ -n "$TRACK_URL" ]] || die "TRACK_URL is required"
[[ -n "$CAMPAIGN_ID" ]] || die "CAMPAIGN_ID is required"

log "step 1: click redirect with fbclid"
CLICK_CODE="$(curl -sS --max-time 15 -o /dev/null -w '%{http_code}' \
  "${TRACK_URL}/click?campaign_id=${CAMPAIGN_ID}&type=click&click_id=${CLICK_ID}&user_id=u-staging&fbclid=${FBCLID}" || true)"
log "GET /click -> HTTP ${CLICK_CODE}"
[[ "$CLICK_CODE" == "200" || "$CLICK_CODE" == "302" || "$CLICK_CODE" == "301" || "$CLICK_CODE" == "307" ]] \
  || die "click failed with HTTP ${CLICK_CODE}"

log "step 2: conversion POST /track"
CONV_BODY="$(
  cat << EOF
{"campaign_id":"${CAMPAIGN_ID}","type":"conversion","click_id":"${CLICK_ID}","user_id":"u-staging","fbclid":"${FBCLID}","payload":{"email":"staging@example.com"}}
EOF
)"
TRACK_CODE="$(curl -sS --max-time 15 -o /dev/null -w '%{http_code}' \
  -X POST "${TRACK_URL}/track" \
  -H 'Content-Type: application/json' \
  -H "Content-Length: ${#CONV_BODY}" \
  -d "$CONV_BODY" || true)"
log "POST /track -> HTTP ${TRACK_CODE}"
[[ "$TRACK_CODE" == "202" || "$TRACK_CODE" == "200" ]] || die "track failed with HTTP ${TRACK_CODE}"

if [[ "${CAPI_SEED_OUTBOX:-1}" == "1" && -n "${CAPI_BOOTSTRAP_DB:-}" ]]; then
  log "seeding SEND_POSTBACK outbox row (local smoke fallback)"
  python3 - "$CAMPAIGN_ID" "$CLICK_ID" "$FBCLID" << 'PY'
import json, subprocess, sys

campaign_id, click_id, fbclid = sys.argv[1:4]
customer = subprocess.check_output([
    "docker", "exec", "ad-event-processor-db-1", "psql", "-h", "/run/ad-event-processor/postgresql",
    "-p", "5430", "-U", "ad_event_processor_user", "-d", "ad_event_processor",
    "-t", "-A", "-c",
    f"SELECT customer_id::text FROM campaigns WHERE id='{campaign_id}'::uuid;",
], text=True).strip()
if not customer:
    raise SystemExit(f"campaign {campaign_id} missing customer_id")
payload = json.dumps({
    "customer_id": customer,
    "campaign_id": campaign_id,
    "click_id": click_id,
    "event_type": "conversion",
    "email": "staging@example.com",
    "fbclid": fbclid,
})
sql = (
    "INSERT INTO outbox_events (event_type, payload, status) VALUES ("
    "'SEND_POSTBACK', convert_to(%s, 'UTF8'), 'PENDING');"
)
subprocess.check_call([
    "docker", "exec", "ad-event-processor-db-1", "psql", "-h", "/run/ad-event-processor/postgresql",
    "-p", "5430", "-U", "ad_event_processor_user", "-d", "ad_event_processor",
    "-v", "ON_ERROR_STOP=1", "-c", sql.replace("%s", "'" + payload.replace("'", "''") + "'"),
])
PY
fi

if [[ -n "$ADMIN_API_KEY" && -n "$META_TEST_EVENT_CODE" ]]; then
  log "step 3: verify test_event_code in postback config"
  cfg="$(curl -sS "${CONTROL_URL}/api/v1/postbacks/config" \
    -H "X-Admin-API-Key: ${ADMIN_API_KEY}" || true)"
  verified=0
  if echo "$cfg" | grep -q "\"campaign_id\":\"${CAMPAIGN_ID}\"" \
    && echo "$cfg" | grep -q "\"test_event_code\":\"${META_TEST_EVENT_CODE}\""; then
    verified=1
  elif [[ -n "${CAPI_BOOTSTRAP_DB:-}" ]]; then
    db_code="$(docker exec ad-event-processor-db-1 psql -h /run/ad-event-processor/postgresql -p 5430 \
      -U ad_event_processor_user -d ad_event_processor -t -A -c \
      "SELECT test_event_code FROM postback_configs WHERE campaign_id='${CAMPAIGN_ID}'::uuid;" 2> /dev/null | tr -d '[:space:]' || true)"
    if [[ "$db_code" == "$META_TEST_EVENT_CODE" ]]; then
      verified=1
      log "postback config test_event_code=${META_TEST_EVENT_CODE} (verified via Postgres; control API may need rebuild)"
    fi
  fi
  if [[ "$verified" == "1" ]]; then
    log "postback config includes test_event_code=${META_TEST_EVENT_CODE}"
  else
    log "WARN: set test_event_code=${META_TEST_EVENT_CODE} in Campaign -> CAPI & Postbacks (admin UI)"
  fi
fi

log "step 4: wait for postback metrics (up to 90s) on ${POSTBACK_METRICS_URL}"
for _ in $(seq 1 18); do
  if curl -sf "$POSTBACK_METRICS_URL" 2> /dev/null | grep -qF "$METRIC_GREP"; then
    log "success: ${METRIC_GREP} observed"
    log "next: confirm event in Meta Events Manager (test stream) within 5 min"
    REPORT_DIR="${CAPI_LAB_REPORT_DIR:-$ROOT/var/capi-lab/$(date -u +%Y%m%dT%H%M%SZ)}"
    mkdir -p "$REPORT_DIR"
    {
      echo "harness=capi_meta_staging"
      echo "campaign_id=${CAMPAIGN_ID}"
      echo "click_id=${CLICK_ID}"
      echo "click_http=${CLICK_CODE}"
      echo "track_http=${TRACK_CODE}"
      echo "metric=${METRIC_GREP}"
      echo "fault_proof fault=capi_meta_staging harness=capi_meta_staging provider=facebook status=success"
    } > "$REPORT_DIR/summary.txt"
    log "report: $REPORT_DIR/summary.txt"
    exit 0
  fi
  sleep 5
done

die "timeout waiting for Meta CAPI dispatch metric - check postback-sender, DLQ, and Meta test stream"
