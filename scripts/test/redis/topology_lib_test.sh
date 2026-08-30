#!/usr/bin/env bash
# Role: Unit test for redis_topology lib helpers (count, UDS addrs, service names).
# Execution context: Sources scripts/lib/redis_topology.sh directly (not go test).
# Env knobs: REDIS_SHARD_COUNT (2 default, 4 in second case).
# Verify: bash scripts/test/redis/topology_lib_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
# shellcheck source=scripts/lib/redis_topology.sh
source "$ROOT/scripts/lib/redis_topology.sh"

unset REDIS_SHARD_COUNT
if [[ "$(redis_topology_count)" != "2" ]]; then
  echo "redis_topology_lib_test: default count want 2, got $(redis_topology_count)" >&2
  exit 1
fi

REDIS_SHARD_COUNT=4
if [[ "$(redis_topology_count)" != "4" ]]; then
  echo "redis_topology_lib_test: REDIS_SHARD_COUNT=4 want 4, got $(redis_topology_count)" >&2
  exit 1
fi

addrs="$(redis_topology_addrs_uds 2)"
if [[ "$addrs" != "/run/ad-event-processor/redis/redis-0.sock,/run/ad-event-processor/redis/redis-1.sock" ]]; then
  echo "redis_topology_lib_test: addrs_uds(2)=$addrs" >&2
  exit 1
fi

services="$(redis_topology_services 2)"
if [[ "$services" != "redis-0 redis-1" ]]; then
  echo "redis_topology_lib_test: services(2)=$services" >&2
  exit 1
fi

echo "redis_topology_lib_test: OK"
