#!/usr/bin/env bash

set -euo pipefail

redis_memory_used_bytes() {
  local shard="$1"
  shift
  local pass="${REDIS_PASSWORD:-redis_secure_pass_456}"
  "$@" exec -T "redis-${shard}" \
    redis-cli -p 6379 -a "$pass" --no-auth-warning INFO memory 2> /dev/null \
    | awk -F: '/^used_memory:/{gsub(/\r/,"",$2); print $2; exit}'
}

redis_memory_max_shard_bytes() {
  local shards="${REDIS_SHARD_COUNT:-6}"
  local max=0
  local i mem
  for i in $(seq 0 $((shards - 1))); do
    mem="$(redis_memory_used_bytes "$i" "$@" || echo 0)"
    if [[ "$mem" -gt "$max" ]]; then
      max="$mem"
    fi
  done
  echo "$max"
}

redis_memory_snapshot_all() {
  local shards="${REDIS_SHARD_COUNT:-6}"
  local ts
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  local i mem
  for i in $(seq 0 $((shards - 1))); do
    mem="$(redis_memory_used_bytes "$i" "$@" || echo 0)"
    echo "${ts} redis-${i} used_memory_bytes=${mem}"
  done
}
