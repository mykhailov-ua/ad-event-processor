#!/usr/bin/env bash
# Role: Fail when production ENV misses hot-path SLA knobs or sets unsafe values.
# Execution context: Post-deploy or release preflight; sources env file argument.
# Env knobs: ENV (production triggers checks); FILTER_TIMEOUT_MS (max 100 ms);
#   STREAM_PRODUCER_ADMISSION_PCT (80-85); LOCAL_QUOTA_MODE=live; QUOTA_MODE=live;
#   TRACKER_PG_FALLBACK=0.
# Verify: bash scripts/ops/verify_prod_tuning.sh .env.prod.example
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"
ENV_FILE="${1:-$ROOT/.env}"
MAX_FILTER_TIMEOUT_MS=100
MIN_STREAM_ADMISSION_PCT=80
MAX_STREAM_ADMISSION_PCT=85

log() { printf 'verify-prod-tuning: %s\n' "$*"; }
die() {
  printf 'verify-prod-tuning: ERROR: %s\n' "$*" >&2
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
FILTER_TIMEOUT_MS="${FILTER_TIMEOUT_MS:-0}"
STREAM_PRODUCER_ADMISSION_PCT="${STREAM_PRODUCER_ADMISSION_PCT:-85}"
LOCAL_QUOTA_MODE="${LOCAL_QUOTA_MODE:-off}"
QUOTA_MODE="${QUOTA_MODE:-}"
TRACKER_PG_FALLBACK="${TRACKER_PG_FALLBACK:-0}"

if [[ "$ENV" != "production" ]]; then
  log "ENV=$ENV (not production) - production tuning checks skipped"
  exit 0
fi

# Production FILTER_TIMEOUT_MS ceiling 100 ms per core.mdc ingest SLA.
if [[ "$FILTER_TIMEOUT_MS" -le 0 ]]; then
  die "production requires explicit FILTER_TIMEOUT_MS (see .env.prod.example)"
fi

if [[ "$FILTER_TIMEOUT_MS" -gt "$MAX_FILTER_TIMEOUT_MS" ]]; then
  die "FILTER_TIMEOUT_MS=$FILTER_TIMEOUT_MS exceeds production ceiling ${MAX_FILTER_TIMEOUT_MS}ms"
fi

if [[ "$STREAM_PRODUCER_ADMISSION_PCT" -lt "$MIN_STREAM_ADMISSION_PCT" \
  || "$STREAM_PRODUCER_ADMISSION_PCT" -gt "$MAX_STREAM_ADMISSION_PCT" ]]; then
  die "STREAM_PRODUCER_ADMISSION_PCT=$STREAM_PRODUCER_ADMISSION_PCT must be ${MIN_STREAM_ADMISSION_PCT}-${MAX_STREAM_ADMISSION_PCT} in production (0 disables TryReserve)"
fi

if [[ "$LOCAL_QUOTA_MODE" != "live" ]]; then
  die "production requires LOCAL_QUOTA_MODE=live (got ${LOCAL_QUOTA_MODE})"
fi

if [[ "$QUOTA_MODE" != "live" ]]; then
  die "production requires QUOTA_MODE=live when LOCAL_QUOTA_MODE=live (got ${QUOTA_MODE:-unset})"
fi

if [[ "$TRACKER_PG_FALLBACK" != "0" ]]; then
  die "production requires TRACKER_PG_FALLBACK=0 (got $TRACKER_PG_FALLBACK)"
fi

log "OK: ENV=production FILTER_TIMEOUT_MS=${FILTER_TIMEOUT_MS}ms admission=${STREAM_PRODUCER_ADMISSION_PCT}% LOCAL_QUOTA_MODE=live QUOTA_MODE=live"
