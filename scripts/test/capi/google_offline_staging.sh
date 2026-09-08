#!/usr/bin/env bash
# Role: End-to-end Google Ads offline conversion staging postback with live track click and metrics assertion.
# Execution context: Running stack with TRACK_URL, ADMIN_API_KEY, Google OAuth + developer token configured.
# Env knobs: GOOGLE_OFFLINE_STAGING_DRY_RUN (1 prints plan only); TRACK_URL; CONTROL_URL (8188); CAMPAIGN_ID.
# Verify: GOOGLE_OFFLINE_STAGING_DRY_RUN=1 bash scripts/test/capi/google_offline_staging.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

TRACK_URL="${TRACK_URL:-}"
CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
ADMIN_API_KEY="${ADMIN_API_KEY:-${API_KEY:-}}"
CAMPAIGN_ID="${CAMPAIGN_ID:-}"
GOOGLE_DEVELOPER_TOKEN="${GOOGLE_DEVELOPER_TOKEN:-}"
POSTBACK_METRICS_URL="${POSTBACK_METRICS_URL:-http://127.0.0.1:9119/metrics}"
CLICK_ID="google-offline-staging-$(date +%s)"
GCLID="gclid.staging.${CLICK_ID}"
METRIC_GREP='ad_postback_dispatch_total{provider="google",status="success"}'

log() { printf 'google-offline-staging: %s\n' "$*"; }
die() {
  printf 'google-offline-staging: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "${GOOGLE_OFFLINE_STAGING_DRY_RUN:-}" == "1" ]]; then
  log "dry-run - would:"
  log "  1. GET ${TRACK_URL:-<TRACK_URL>}/click?campaign_id=${CAMPAIGN_ID:-<id>}&gclid=${GCLID}"
  log "  2. POST ${TRACK_URL:-<TRACK_URL>}/track conversion JSON with gclid"
  log "  3. Optional: GET ${CONTROL_URL}/api/v1/postbacks/config - verify google provider + customer_id|conversion_action_id url_template"
  log "  4. Poll ${POSTBACK_METRICS_URL} for ${METRIC_GREP}"
  log "Set TRACK_URL, CAMPAIGN_ID before live run; configure Google OAuth token + developer token in Integrations -> Postbacks."
  exit 0
fi

[[ -n "$TRACK_URL" ]] || die "TRACK_URL is required"
[[ -n "$CAMPAIGN_ID" ]] || die "CAMPAIGN_ID is required"

log "step 1: click redirect with gclid"
CLICK_CODE="$(curl -sS --max-time 15 -o /dev/null -w '%{http_code}' \
  "${TRACK_URL}/click?campaign_id=${CAMPAIGN_ID}&type=click&click_id=${CLICK_ID}&user_id=u-staging&gclid=${GCLID}" || true)"
log "GET /click -> HTTP ${CLICK_CODE}"
[[ "$CLICK_CODE" == "200" || "$CLICK_CODE" == "302" || "$CLICK_CODE" == "301" || "$CLICK_CODE" == "307" ]] \
  || die "click failed with HTTP ${CLICK_CODE}"

log "step 2: conversion POST /track"
CONV_BODY="$(
  cat << EOF
{"campaign_id":"${CAMPAIGN_ID}","type":"conversion","click_id":"${CLICK_ID}","user_id":"u-staging","gclid":"${GCLID}"}
EOF
)"
TRACK_CODE="$(curl -sS --max-time 15 -o /dev/null -w '%{http_code}' \
  -X POST "${TRACK_URL}/track" \
  -H 'Content-Type: application/json' \
  -H "Content-Length: ${#CONV_BODY}" \
  -d "$CONV_BODY" || true)"
log "POST /track -> HTTP ${TRACK_CODE}"
[[ "$TRACK_CODE" == "202" || "$TRACK_CODE" == "200" ]] || die "track failed with HTTP ${TRACK_CODE}"

if [[ -n "$ADMIN_API_KEY" ]]; then
  log "step 3: verify google postback config"
  cfg="$(curl -sS "${CONTROL_URL}/api/v1/postbacks/config" \
    -H "X-Admin-API-Key: ${ADMIN_API_KEY}" || true)"
  if echo "$cfg" | grep -q "\"campaign_id\":\"${CAMPAIGN_ID}\"" \
    && echo "$cfg" | grep -q '"provider":"google"'; then
    log "postback config provider=google for campaign"
  else
    log "WARN: configure provider=google with customer_id|conversion_action_id in Integrations -> Postbacks"
  fi
  if [[ -n "$GOOGLE_DEVELOPER_TOKEN" ]]; then
    log "developer token present in env (store in test_event_code on postback config)"
  fi
fi

log "step 4: wait for postback metrics (up to 90s) on ${POSTBACK_METRICS_URL}"
for _ in $(seq 1 18); do
  if curl -sf "$POSTBACK_METRICS_URL" 2> /dev/null | grep -qF "$METRIC_GREP"; then
    log "success: ${METRIC_GREP} observed"
    log "next: confirm conversion in Google Ads UI (Tools -> Conversions) within 24h"
    REPORT_DIR="${GOOGLE_OFFLINE_REPORT_DIR:-$ROOT/var/capi-lab/google-$(date -u +%Y%m%dT%H%M%SZ)}"
    mkdir -p "$REPORT_DIR"
    {
      echo "harness=google_offline_staging"
      echo "campaign_id=${CAMPAIGN_ID}"
      echo "click_id=${CLICK_ID}"
      echo "gclid=${GCLID}"
      echo "click_http=${CLICK_CODE}"
      echo "track_http=${TRACK_CODE}"
      echo "metric=${METRIC_GREP}"
      echo "fault_proof fault=google_offline_staging harness=google_offline_staging provider=google status=success"
    } > "$REPORT_DIR/summary.txt"
    log "report: $REPORT_DIR/summary.txt"
    exit 0
  fi
  sleep 5
done

die "timeout waiting for Google offline dispatch metric - check postback-sender, DLQ, and Google Ads conversion action"
