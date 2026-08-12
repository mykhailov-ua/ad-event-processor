#!/usr/bin/env bash
# Live wrapper for scripts/test/capi_meta_staging.sh (local or staging stack).
#
# Usage:
#   set -a; source .env; set +a
#   export META_TEST_EVENT_CODE=TEST12345   # from Meta Events Manager
#   export CAMPAIGN_ID=<uuid-with-facebook-capi-configured>
#   export TRACK_URL=http://127.0.0.1:8181  # or https://trk.staging.example
#   bash scripts/test/capi_meta_staging_live.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

TRACK_URL="${TRACK_URL:-http://127.0.0.1:8181}"
CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
POSTBACK_METRICS_URL="${POSTBACK_METRICS_URL:-http://127.0.0.1:9119/metrics}"
CAMPAIGN_ID="${CAMPAIGN_ID:-}"

log() { printf 'capi-meta-live: %s\n' "$*"; }
die() { printf 'capi-meta-live: ERROR: %s\n' "$*" >&2; exit 1; }

[[ -n "$CAMPAIGN_ID" ]] || die "CAMPAIGN_ID is required (campaign with Facebook CAPI token in admin)"

code="$(curl -sf -o /dev/null -w '%{http_code}' -m 5 "${TRACK_URL%/}/health" 2>/dev/null || echo 000)"
[[ "$code" == "200" ]] || die "tracker/edge health failed at ${TRACK_URL} (HTTP ${code})"

if ! curl -sf -m 3 "$POSTBACK_METRICS_URL" >/dev/null 2>&1; then
  if [[ -x "$ROOT/bin/postback-sender" ]]; then
    log "starting postback-sender (metrics ${POSTBACK_METRICS_URL})"
    if [[ -n "${DB_USER:-}" && -n "${DB_PASSWORD:-}" && -n "${DB_PORT:-}" && -n "${DB_NAME:-}" ]]; then
      export DB_DSN="postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:${DB_PORT}/${DB_NAME}?sslmode=disable"
    fi
    export POSTBACK_ENCRYPTION_KEY="${POSTBACK_ENCRYPTION_KEY:-${TOKEN_SYMMETRIC_KEY:-}}"
    nohup env DB_DSN="$DB_DSN" POSTBACK_ENCRYPTION_KEY="$POSTBACK_ENCRYPTION_KEY" \
      POSTBACK_METRICS_ADDR="127.0.0.1:9119" \
      "$ROOT/bin/postback-sender" > /tmp/postback-sender.log 2>&1 &
    sleep 2
  else
    die "postback-sender not reachable at ${POSTBACK_METRICS_URL} and bin/postback-sender missing"
  fi
fi

if [[ -n "${ADMIN_API_KEY:-}" ]]; then
  cfg="$(curl -sf -m 5 "${CONTROL_URL}/api/v1/postbacks/config" \
    -H "X-Admin-API-Key: ${ADMIN_API_KEY}" 2>/dev/null || true)"
  if ! echo "$cfg" | grep -q "\"campaign_id\":\"${CAMPAIGN_ID}\""; then
    log "WARN: no postback config for campaign ${CAMPAIGN_ID} — configure CAPI & Postbacks in admin"
  elif ! echo "$cfg" | grep -q '"provider":"facebook"'; then
    log "WARN: campaign has no facebook provider — Meta dispatch will not run"
  fi
fi

if [[ -z "${META_TEST_EVENT_CODE:-}" ]]; then
  log "WARN: META_TEST_EVENT_CODE unset — Meta Events Manager test stream will not show the event"
fi

export TRACK_URL CONTROL_URL POSTBACK_METRICS_URL CAMPAIGN_ID
exec bash "$SCRIPTS/test/capi_meta_staging.sh"
