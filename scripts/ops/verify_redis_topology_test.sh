#!/bin/sh
# shellcheck disable=SC2034
set -eu

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

cat > "$TMP" << 'EOF'
REDIS_ADDRS=/run/ad-event-processor/redis/redis-0.sock,/run/ad-event-processor/redis/redis-1.sock,/run/ad-event-processor/redis/redis-2.sock,/run/ad-event-processor/redis/redis-3.sock
EOF

bash "$ROOT/scripts/ops/verify_redis_topology.sh" "$TMP"

cat > "$TMP" << 'EOF'
REDIS_ADDRS=/run/ad-event-processor/redis/redis-0.sock,/run/ad-event-processor/redis/redis-1.sock,/run/ad-event-processor/redis/redis-2.sock,/run/ad-event-processor/redis/redis-3.sock,/run/ad-event-processor/redis/redis-4.sock,/run/ad-event-processor/redis/redis-5.sock
EOF

if REDIS_SHARD_COUNT=4 bash "$ROOT/scripts/ops/verify_redis_topology.sh" "$TMP" 2> /dev/null; then
  echo "expected verify_redis_topology to reject 6-shard REDIS_ADDRS with REDIS_SHARD_COUNT=4" >&2
  exit 1
fi

echo "verify_redis_topology_test: OK"
