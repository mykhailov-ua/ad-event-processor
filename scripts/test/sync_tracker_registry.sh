#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  source "$ROOT/.env"
  set +a
fi

CHANNEL="${CAMPAIGN_UPDATE_CHANNEL:-campaigns:update}"
PAYLOAD="${REGISTRY_FULL_SYNC_PAYLOAD:-*}"
COMPOSE=(docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml)
REDIS_PASS="${REDIS_PASSWORD:-redis_secure_pass_456}"
VERIFY_TRACKERS="${VERIFY_TRACKERS:-1}"
VERIFY_RETRIES="${VERIFY_RETRIES:-30}"

log() { printf 'sync-tracker-registry: %s\n' "$*"; }
die() {
  printf 'sync-tracker-registry: ERROR: %s\n' "$*" >&2
  exit 1
}

redis_publish() {
  "${COMPOSE[@]}" exec -T redis-0 redis-cli -p 6379 -a "$REDIS_PASS" \
    PUBLISH "$CHANNEL" "$PAYLOAD" > /dev/null
}

verify_track() {
  local port="$1"
  local code
  local click_id="registry-verify-${port}-$(date +%s%N)"
  code="$(curl -sf -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${port}/track" \
    -H 'Content-Type: application/json' \
    -d '{"campaign_id":"00000000-0000-0000-0000-000000000001","user_id":"registry-verify","type":"impression","click_id":"'"$click_id"'","payload":{"slot":"top"}}' \
    2> /dev/null || echo 000)"
  [[ "$code" == "204" || "$code" == "200" || "$code" == "202" ]]
}

log "publishing full registry snapshot on ${CHANNEL} payload=${PAYLOAD}"
redis_publish
sleep 2

if [[ "$VERIFY_TRACKERS" != "1" ]]; then
  exit 0
fi

for port in 8181 8182; do
  accepted=0
  for _ in $(seq 1 "$VERIFY_RETRIES"); do
    if verify_track "$port"; then
      log "tracker :${port} accepted seeded campaign (202/204/200)"
      accepted=1
      break
    fi
    sleep 1
  done
  if [[ "$accepted" != "1" ]]; then
    die "tracker :${port} still rejects seeded campaign after full sync"
  fi
done

log "registry snapshot verified"
