#!/usr/bin/env bash
# Role: curl proof that lander edge serves first-party /_aed/track.js proxy to tracker /static/track.js.
# Execution context: Operator or CI with LANDER_PUBLIC_BASE_URL or FIRST_PARTY_PIXEL_URL set.
# Env: LANDER_PUBLIC_BASE_URL, FIRST_PARTY_PIXEL_URL (override full script URL), FIRST_PARTY_PIXEL_ARTIFACT_DIR.
# Verify: bash scripts/test/edge/first_party_pixel_drill.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT_DIR="${FIRST_PARTY_PIXEL_ARTIFACT_DIR:-$ROOT/var/ci/first_party_pixel}"
LANDER_BASE="${LANDER_PUBLIC_BASE_URL:-}"
SCRIPT_URL="${FIRST_PARTY_PIXEL_URL:-}"

if [[ -z "$SCRIPT_URL" ]]; then
  if [[ -z "$LANDER_BASE" ]]; then
    echo "first_party_pixel_drill: ERROR: set LANDER_PUBLIC_BASE_URL or FIRST_PARTY_PIXEL_URL" >&2
    exit 1
  fi
  LANDER_BASE="${LANDER_BASE%/}"
  SCRIPT_URL="${LANDER_BASE}/_aed/track.js"
fi

mkdir -p "$OUT_DIR"
BODY_FILE="$OUT_DIR/track.js"
META_FILE="$OUT_DIR/meta.txt"

log() { printf 'first_party_pixel_drill: %s\n' "$*" >&2; }

log "GET $SCRIPT_URL"
HTTP_CODE="$(curl -sS -L --max-time 15 -o "$BODY_FILE" -w '%{http_code}' "$SCRIPT_URL")"
printf '%s\n' "$HTTP_CODE" > "$META_FILE"

if [[ "$HTTP_CODE" != "200" ]]; then
  log "ERROR: expected HTTP 200, got $HTTP_CODE"
  exit 1
fi
if ! grep -q trackEvent "$BODY_FILE"; then
  log "ERROR: body missing trackEvent export"
  exit 1
fi

SHA="$(sha256sum "$BODY_FILE" | awk '{print $1}')"
printf 'fault_proof fault=first_party_pixel url=%s status=%s sha256=%s\n' "$SCRIPT_URL" "$HTTP_CODE" "$SHA" > "$OUT_DIR/fault_proof.txt"
log "OK status=200 sha256=$SHA artifact=$OUT_DIR"
