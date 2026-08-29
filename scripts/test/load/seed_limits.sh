#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"

REDIS_PORT="${REDIS_PORT:-6479}"
REDIS_PASS="${REDIS_PASSWORD:-}"
LIMIT_PER_MIN="${LOAD_TEST_RATE_LIMIT_PER_MIN:-500000}"
WINDOW_MS="${LOAD_TEST_RATE_LIMIT_WINDOW_MS:-60000}"

redis_cmd() {
  if command -v redis-cli > /dev/null 2>&1; then
    if [[ -n "$REDIS_PASS" ]]; then
      redis-cli -p "$REDIS_PORT" -a "$REDIS_PASS" --no-auth-warning "$@"
    else
      redis-cli -p "$REDIS_PORT" "$@"
    fi
    return
  fi
  docker exec ad-event-processor-redis-0-1 redis-cli "$@"
}

log() { printf 'seed-load-test-limits: %s\n' "$*"; }

if ! redis_cmd PING > /dev/null 2>&1; then
  log "WARN: redis not reachable on port $REDIS_PORT - skip"
  exit 0
fi

redis_cmd HSET config:values \
  rate_limit_per_min "$LIMIT_PER_MIN" \
  rate_limit_window_ms "$WINDOW_MS" > /dev/null

now="$(date +%s)"
redis_cmd HSET edge:blacklist:meta version "loadtest" sync_ts "$now" > /dev/null 2>&1 || true

log "config:values rate_limit_per_min=$LIMIT_PER_MIN window_ms=$WINDOW_MS"
