#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
ADMIN_API_KEY="${ADMIN_API_KEY:-dev-admin-api-key-change-me}"
CAMPAIGN_ID="${CAMPAIGN_ID:-00000000-0000-0000-0000-000000000005}"

log() { printf 'integration-schema-smoke: %s\n' "$*"; }
die() {
  printf 'integration-schema-smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

if ! command -v curl > /dev/null 2>&1; then
  log "skip (curl missing)"
  exit 0
fi

if ! curl -fsS -m 3 "${CONTROL_URL}/health" > /dev/null 2>&1; then
  log "skip (control unreachable at ${CONTROL_URL})"
  exit 0
fi

schema_json="$(
  cat << 'JSON'
{
  "name": "m6-smoke-outbound",
  "version": 1,
  "schema": {
    "version": 1,
    "url_template": "https://aff.example.com/pb?click_id={click_id}&sub1={sub1}",
    "placeholders": ["click_id", "sub1"]
  }
}
JSON
)"

create_code="$(curl -sS -o /tmp/m6-schema-create.json -w '%{http_code}' \
  -X POST "${CONTROL_URL}/api/v1/integration/schemas" \
  -H "X-Admin-API-Key: ${ADMIN_API_KEY}" \
  -H 'Content-Type: application/json' \
  -d "$schema_json")"
log "POST /integration/schemas -> HTTP ${create_code}"
if [[ "$create_code" != "201" && "$create_code" != "409" ]]; then
  die "create failed: $(cat /tmp/m6-schema-create.json 2> /dev/null || true)"
fi

schema_id="$(python3 -c 'import json,sys; print(json.load(open("/tmp/m6-schema-create.json"))["id"])' 2> /dev/null || true)"
if [[ -z "$schema_id" ]]; then
  list_json="$(curl -sS "${CONTROL_URL}/api/v1/integration/schemas" -H "X-Admin-API-Key: ${ADMIN_API_KEY}")"
  schema_id="$(printf '%s' "$list_json" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(next((x["id"] for x in d if x.get("name")=="m6-smoke-outbound"), ""))')"
fi
[[ -n "$schema_id" ]] || die "could not resolve schema id"

apply_code="$(curl -sS -o /tmp/m6-schema-apply.json -w '%{http_code}' \
  -X POST "${CONTROL_URL}/api/v1/integration/schemas/${schema_id}/apply" \
  -H "X-Admin-API-Key: ${ADMIN_API_KEY}" \
  -H 'Content-Type: application/json' \
  -d "{\"campaign_id\":\"${CAMPAIGN_ID}\"}")"
log "POST /integration/schemas/${schema_id}/apply -> HTTP ${apply_code}"
[[ "$apply_code" == "200" ]] || die "apply failed: $(cat /tmp/m6-schema-apply.json 2> /dev/null || true)"

log "ok (outbound schema applied to campaign ${CAMPAIGN_ID})"
