#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

DURATION="${CHAOS_LOAD_DURATION:-300s}"
RPS="${CHAOS_LOAD_RPS:-5000}"
CHAOS_PCT="${CHAOS_LOAD_CHAOS_PCT:-10}"
P99_MS="${CHAOS_LOAD_P99_MS:-80}"
WORKERS="${CHAOS_LOAD_WORKERS:-8}"
LOG="${CHAOS_LOAD_LOG:-/tmp/parser-chaos-load.log}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --duration=*)
      DURATION="${1#*=}"
      shift
      ;;
    --rps=*)
      RPS="${1#*=}"
      shift
      ;;
    --chaos-pct=*)
      CHAOS_PCT="${1#*=}"
      shift
      ;;
    --p99-ms=*)
      P99_MS="${1#*=}"
      shift
      ;;
    --workers=*)
      WORKERS="${1#*=}"
      shift
      ;;
    --log=*)
      LOG="${1#*=}"
      shift
      ;;
    *)
      echo "usage: $0 [--duration=300s] [--rps=5000] [--chaos-pct=10] [--p99-ms=80] [--workers=8] [--log=/tmp/parser-chaos-load.log]" >&2
      exit 2
      ;;
  esac
done

export CHAOS_LOAD_DURATION="$DURATION"
export CHAOS_LOAD_RPS="$RPS"
export CHAOS_LOAD_CHAOS_PCT="$CHAOS_PCT"
export CHAOS_LOAD_P99_MS="$P99_MS"
export CHAOS_LOAD_WORKERS="$WORKERS"

echo "parser-chaos-load: duration=${DURATION} rps=${RPS} chaos_pct=${CHAOS_PCT} p99_ms=${P99_MS} workers=${WORKERS}"
go test ./internal/ingestion/ -run='TestChaos_ParserLoad_CX02' -count=1 -timeout=15m -v 2>&1 | tee "$LOG"

if ! grep -q 'fault_proof fault=parser_chaos_load' "$LOG"; then
  echo "parser-chaos-load: missing fault_proof line" >&2
  exit 1
fi
if grep -q 'fault_proof fault=parser_chaos_load.*gap=open' "$LOG"; then
  echo "parser-chaos-load: gap still open" >&2
  exit 1
fi

echo "parser-chaos-load: PASS (see ${LOG})"
