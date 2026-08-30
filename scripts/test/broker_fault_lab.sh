#!/usr/bin/env bash
set -euo pipefail

# Role: Broker durability fault lab (WAL, page cache, CPU throttle); BROKER_FAULT_LAB=1 enables extended tests.
# Execution context: Operator machine; stress-ng/cpulimit optional (tests skip when missing).
# Invariants/contracts enforced: broker fault_proof lines in BROKER_FAULT_LAB_LOG; no event loss on cutover paths.
# Verify: bash scripts/test/broker_fault_lab.sh
# Env: BROKER_FAULT_LAB_LOG, BROKER_FAULT_LAB=1
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

LOG="${BROKER_FAULT_LAB_LOG:-/tmp/ad-event-processor-broker-fault-lab.log}"
export BROKER_FAULT_LAB=1

echo "Broker fault lab (durability and coordination)"
if command -v stress-ng > /dev/null 2>&1; then
  echo "stress-ng: $(stress-ng --version 2>&1 | head -1)"
else
  echo "stress-ng not installed; page cache test will skip"
fi
if command -v cpulimit > /dev/null 2>&1; then
  echo "cpulimit: available"
else
  echo "cpulimit not installed; CPU throttle test will skip"
fi
go test -count=1 -v -run 'TestFault_(SlowFsync|PageCache|CPUThrottle|RedisOutage|RedisSentinel)' -timeout 25m ./internal/broker/... 2>&1 | tee "$LOG"

PROOFS="$(grep -c 'fault_proof fault=' "$LOG" || true)"
echo "fault_proof lines: $PROOFS"
test "$PROOFS" -ge 2

if command -v docker > /dev/null 2>&1 && [ "${BROKER_FAULT_SKIP_SENTINEL:-0}" != "1" ]; then
  echo "Sentinel coordination lab (optional)"
  COMPOSE_BASE="deploy/broker/docker-compose.yaml"
  COMPOSE_SENTINEL="deploy/broker/docker-compose.sentinel.yaml"
  docker compose -f "$COMPOSE_BASE" -f "$COMPOSE_SENTINEL" up -d redis redis-replica redis-sentinel
  trap 'docker compose -f "$COMPOSE_BASE" -f "$COMPOSE_SENTINEL" down' EXIT

  echo "waiting for sentinel to monitor master..."
  sleep 8

  export BROKER_FAULT_SENTINEL=1
  export BROKER_REDIS_SENTINEL_MASTER=broker-coord
  export BROKER_REDIS_SENTINEL_ADDRS=127.0.0.1:26379
  export BROKER_REDIS_URL=redis://127.0.0.1:6379/0
  export BROKER_FAULT_SENTINEL_STOP_CONTAINER=ad-event-processor-broker-redis

  go test -count=1 -v -run 'TestFault_RedisSentinelFailover' -timeout 10m ./internal/broker/... 2>&1 | tee -a "$LOG"

  SENTINEL_PROOFS="$(grep -c 'fault_proof fault=redis_sentinel_failover' "$LOG" || true)"
  test "$SENTINEL_PROOFS" -ge 1
fi

echo "Broker fault lab complete. Log: $LOG"
