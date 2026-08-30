#!/usr/bin/env bash
# Role: Shell harness asserting REDIS_ADDRS count matches REDIS_SHARD_COUNT via ops verifier.
# Execution context: Standalone test; writes temp env files (no running Redis).
# Env knobs: none (cases use REDIS_SHARD_COUNT 2, 4, and reject 6 addrs with count 4).
# Verify: bash scripts/test/redis/verify_topology_test.sh
set -eu

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

cat > "$TMP" << 'EOF'
REDIS_SHARD_COUNT=2
REDIS_ADDRS=/run/ad-event-processor/redis/redis-0.sock,/run/ad-event-processor/redis/redis-1.sock
EOF

bash "$ROOT/scripts/ops/verify_redis_topology.sh" "$TMP"

cat > "$TMP" << 'EOF'
REDIS_SHARD_COUNT=4
REDIS_ADDRS=/run/ad-event-processor/redis/redis-0.sock,/run/ad-event-processor/redis/redis-1.sock,/run/ad-event-processor/redis/redis-2.sock,/run/ad-event-processor/redis/redis-3.sock
EOF

bash "$ROOT/scripts/ops/verify_redis_topology.sh" "$TMP"

# Negative: six REDIS_ADDRS with REDIS_SHARD_COUNT=4 must fail (production expects addr count match).
cat > "$TMP" << 'EOF'
REDIS_ADDRS=/run/ad-event-processor/redis/redis-0.sock,/run/ad-event-processor/redis/redis-1.sock,/run/ad-event-processor/redis/redis-2.sock,/run/ad-event-processor/redis/redis-3.sock,/run/ad-event-processor/redis/redis-4.sock,/run/ad-event-processor/redis/redis-5.sock
EOF

if REDIS_SHARD_COUNT=4 bash "$ROOT/scripts/ops/verify_redis_topology.sh" "$TMP" 2> /dev/null; then
  echo "expected verify_redis_topology to reject 6-shard REDIS_ADDRS with REDIS_SHARD_COUNT=4" >&2
  exit 1
fi

echo "verify_redis_topology_test: OK"
