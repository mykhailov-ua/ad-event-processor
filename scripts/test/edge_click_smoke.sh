#!/usr/bin/env bash

set -euo pipefail

EDGE_URL="${EDGE_URL:-http://127.0.0.1:8180}"
CAMPAIGN_ID="${CAMPAIGN_ID:-550e8400-e29b-41d4-a716-446655440000}"
PATH_Q="/click?campaign_id=${CAMPAIGN_ID}&type=click"
HDR_FILE="$(mktemp)"
trap 'rm -f "$HDR_FILE"' EXIT

log() { printf 'edge-click-smoke: %s\n' "$*"; }
die() {
  printf 'edge-click-smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

if ! curl -sS --connect-timeout 2 -o /dev/null "${EDGE_URL}/healthz" 2> /dev/null; then
  log "skip (edge unreachable at ${EDGE_URL})"
  exit 0
fi

code="$(curl -sS -o /dev/null -w '%{http_code}' -D "$HDR_FILE" "${EDGE_URL}${PATH_Q}" || true)"
log "GET ${EDGE_URL}${PATH_Q} -> HTTP ${code}"

case "$code" in
  302 | 301)
    loc="$(awk 'BEGIN{IGNORECASE=1} /^Location:/{print $2}' "$HDR_FILE" | tr -d '\r' | head -1)"
    log "Location: ${loc:-<empty>}"
    ;;
  404)
    die "404 — edge_expose_click is off (Platform settings / EDGE_EXPOSE_CLICK) or path missing"
    ;;
  400 | 403 | 429 | 503)
    log "gate/filter response ${code} (edge reachable; campaign may be unknown or filtered)"
    ;;
  *)
    die "unexpected status ${code}"
    ;;
esac

post_code="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${EDGE_URL}/click?campaign_id=${CAMPAIGN_ID}" || true)"
log "POST /click -> HTTP ${post_code}"
if [[ "$post_code" != "405" && "$post_code" != "404" ]]; then
  die "expected 405 (or 404 if gated) for POST /click, got ${post_code}"
fi

log "ok"
