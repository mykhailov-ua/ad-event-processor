#!/usr/bin/env bash
# Role: Enable BehaviorHighVolumeDebit (bit 256) on hot campaigns via PATCH /api/v1/campaigns/{id}/fraud.
# Execution context: Operator host with control plane up; pairs with LOCAL_QUOTA_MODE=live on trackers.
# Env knobs: CONTROL_URL (default unix socket); ADMIN_API_KEY or accessToken cookie file.
# Verify: go test ./internal/ingestion/ -short -run TestHighVolumeDebit_subShardQuotaKeysDistinct -count=1
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

HIGH_VOLUME_DEBIT_FLAG=256

usage() {
  cat << 'EOF'
Usage: enable_high_volume_debit.sh <campaign-uuid> [campaign-uuid...]

Sets behavior_flags |= 256 (BehaviorHighVolumeDebit) on each campaign.
Requires campaigns:write (admin session cookie or ADMIN_API_KEY).

Env: CONTROL_URL (default from MANAGEMENT_URL), ADMIN_API_KEY, ACCESS_TOKEN_COOKIE.
EOF
}

if [[ $# -lt 1 ]]; then
  usage >&2
  exit 1
fi

ENV_FILE="${ENV_FILE:-$ROOT/.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
fi

CONTROL_URL="${CONTROL_URL:-${MANAGEMENT_URL:-unix:///run/ad-event-processor/control/http.sock}}"
AUTH_HEADER=()
if [[ -n "${ADMIN_API_KEY:-}" ]]; then
  AUTH_HEADER=(-H "X-Admin-API-Key: ${ADMIN_API_KEY}")
elif [[ -n "${ACCESS_TOKEN_COOKIE:-}" ]]; then
  AUTH_HEADER=(-H "Cookie: accessToken=${ACCESS_TOKEN_COOKIE}")
else
  printf 'enable-high-volume-debit: ERROR: set ADMIN_API_KEY or ACCESS_TOKEN_COOKIE in env\n' >&2
  exit 1
fi

curl_api() {
  local method=$1 path=$2
  shift 2
  if [[ "$CONTROL_URL" == unix://* ]]; then
    curl -sf --unix-socket "${CONTROL_URL#unix://}" -X "$method" \
      "http://localhost${path}" "${AUTH_HEADER[@]}" "$@"
  else
    curl -sf -X "$method" "${CONTROL_URL%/}${path}" "${AUTH_HEADER[@]}" "$@"
  fi
}

log() { printf 'enable-high-volume-debit: %s\n' "$*"; }

for camp_id in "$@"; do
  current="$(curl_api GET "/api/v1/campaigns/${camp_id}/fraud")"
  flags="$(printf '%s' "$current" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("behavior_flags",0))')"
  new_flags=$((flags | HIGH_VOLUME_DEBIT_FLAG))
  if [[ "$new_flags" -eq "$flags" ]]; then
    log "skip ${camp_id} (BehaviorHighVolumeDebit already set)"
    continue
  fi
  curl_api PATCH "/api/v1/campaigns/${camp_id}/fraud" \
    -H 'Content-Type: application/json' \
    -d "{\"behavior_flags\":${new_flags}}" > /dev/null
  log "enabled BehaviorHighVolumeDebit on ${camp_id} (behavior_flags ${flags} -> ${new_flags})"
done

log "done; confirm outbox drained and tracker registry reload before load test"
