#!/usr/bin/env bash
set -euo pipefail

# Role: Fault/resilience: Slow-body DoS drill.
# Execution context: CI main-resilience or operator fault tier; needs Docker for compose drills.
# Invariants/contracts enforced: Success logs fault_proof fault=<name>; resilience_fault_gates.sh greps required proofs.
# Verify: bash scripts/fault/parser_slow_body_drill.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

CONNECTIONS=64
RATE=1
DURATION=8s

while [[ $# -gt 0 ]]; do
  case "$1" in
    --connections=*)
      CONNECTIONS="${1#*=}"
      shift
      ;;
    --rate=*)
      RATE="${1#*=}"
      shift
      ;;
    --duration=*)
      DURATION="${1#*=}"
      shift
      ;;
    *)
      echo "usage: $0 [--connections=N] [--rate=bytes_per_sec] [--duration=8s]" >&2
      exit 2
      ;;
  esac
done

echo "parser-slow-body: unit proof (spin close)"
go test ./internal/ingest/ -run='TestChaos_ParserSecurity_SlowBodyStall|TestHTTP1Incomplete' -count=1 -timeout=5m -v

echo "parser-slow-body: in-process p99 isolation (control cohort under ${CONNECTIONS} slow drips)"
export SLOW_BODY_DRILL_CONNECTIONS="${CONNECTIONS}"
export SLOW_BODY_DRILL_DURATION="${DURATION}"
export SLOW_BODY_DRILL_P99_MS=80
go test ./internal/ingest/ -run='TestParserSlowBodyDrill_P99Isolation' -count=1 -timeout=15m -v

echo "fault_proof fault=parser_slow_body_drill proof=closed connections=${CONNECTIONS} rate=${RATE} duration=${DURATION}"
