#!/usr/bin/env bash

# Role: Library: Redis topology verify helpers.
# Execution context: Sourced by CI, fault, and dev scripts; not a standalone gate.
# Invariants/contracts enforced: Helpers must not exit 0 on error paths when used as gate prerequisites.
# Verify: source scripts/lib/redis_topology.sh
redis_topology_max_shards() {
  echo 6
}

redis_topology_count() {
  local n="${REDIS_SHARD_COUNT:-}"
  if [[ -z "$n" ]] && [[ -n "${ROOT:-}" ]] && [[ -f "${ROOT}/.env" ]]; then
    n="$(grep -m1 '^REDIS_SHARD_COUNT=' "${ROOT}/.env" 2> /dev/null | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')"
  fi
  if [[ -z "$n" ]]; then
    n=2
  fi
  if [[ "$n" -lt 1 ]]; then
    n=1
  fi
  local max
  max="$(redis_topology_max_shards)"
  if [[ "$n" -gt "$max" ]]; then
    n="$max"
  fi
  echo "$n"
}

redis_topology_services() {
  local count="${1:-$(redis_topology_count)}"
  local i=0
  local out=()
  while [[ "$i" -lt "$count" ]]; do
    out+=("redis-$i")
    i=$((i + 1))
  done
  echo "${out[@]}"
}

redis_topology_addrs_uds() {
  local count="${1:-$(redis_topology_count)}"
  local i=0
  local out=""
  while [[ "$i" -lt "$count" ]]; do
    if [[ -n "$out" ]]; then
      out+=","
    fi
    out+="/run/ad-event-processor/redis/redis-${i}.sock"
    i=$((i + 1))
  done
  echo "$out"
}
