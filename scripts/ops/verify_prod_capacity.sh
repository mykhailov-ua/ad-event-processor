#!/usr/bin/env bash
# Role: Fail when production ENV misses P1 capacity knobs (UDS transport, CPU isolation, processor lag ceiling).
# Execution context: Post-deploy or release preflight after P0 verify_prod_tuning.sh passes.
# Env knobs: ENV=production; TRANSPORT_USE_UDS=1; CPU_ISOLATION_ENABLED=1; REDIS_ADDRS unix paths;
#   PROCESSOR_STREAM_LAG_MAX_SEC (>0, default 120); CH_SPOOL_DIR set.
# Verify: bash scripts/ops/verify_prod_capacity.sh .env.prod.example
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"
ENV_FILE="${1:-$ROOT/.env}"

log() { printf 'verify-prod-capacity: %s\n' "$*"; }
die() {
  printf 'verify-prod-capacity: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ ! -f "$ENV_FILE" ]]; then
  die "missing env file: $ENV_FILE"
fi

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

ENV="${ENV:-development}"
TRANSPORT_USE_UDS="${TRANSPORT_USE_UDS:-0}"
CPU_ISOLATION_ENABLED="${CPU_ISOLATION_ENABLED:-0}"
REDIS_ADDRS="${REDIS_ADDRS:-}"
PROCESSOR_STREAM_LAG_MAX_SEC="${PROCESSOR_STREAM_LAG_MAX_SEC:-0}"
CH_SPOOL_DIR="${CH_SPOOL_DIR:-}"

if [[ "$ENV" != "production" ]]; then
  log "ENV=$ENV (not production) - capacity checks skipped"
  exit 0
fi

if [[ "$TRANSPORT_USE_UDS" != "1" ]]; then
  die "production P1 requires TRANSPORT_USE_UDS=1 (co-located unix transport)"
fi

if [[ "$REDIS_ADDRS" != *"/run/"* && "$REDIS_ADDRS" != unix://* ]]; then
  die "REDIS_ADDRS must use unix socket paths when TRANSPORT_USE_UDS=1 (got: ${REDIS_ADDRS:-unset})"
fi

if [[ "$CPU_ISOLATION_ENABLED" != "1" ]]; then
  die "production P1 requires CPU_ISOLATION_ENABLED=1 (compose --profile cpu-isolation)"
fi

if [[ "$PROCESSOR_STREAM_LAG_MAX_SEC" -le 0 ]]; then
  die "production requires PROCESSOR_STREAM_LAG_MAX_SEC > 0 (default 120)"
fi

if [[ -z "$CH_SPOOL_DIR" ]]; then
  die "production requires CH_SPOOL_DIR for ClickHouse spool WAL"
fi

log "OK: TRANSPORT_USE_UDS=1 CPU_ISOLATION_ENABLED=1 lag_max=${PROCESSOR_STREAM_LAG_MAX_SEC}s CH_SPOOL_DIR=${CH_SPOOL_DIR}"
