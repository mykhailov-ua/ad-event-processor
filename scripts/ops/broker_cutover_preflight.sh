#!/usr/bin/env bash
# Role: Pre/post broker cutover checks (shadow vs live) using processor metrics.
# Execution context: Operator during CH_INGEST_SOURCE migration; stack and processor must be up.
# Env knobs: ENV_FILE; PROCESSOR_METRICS_URL (default http://127.0.0.1:8186/metrics).
# Usage: broker_cutover_preflight.sh shadow | live
# Verify: bash scripts/ops/broker_cutover_preflight.sh live
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

MODE="${1:-live}"
ENV_FILE="${ENV_FILE:-$ROOT/.env}"
METRICS_URL="${PROCESSOR_METRICS_URL:-http://127.0.0.1:8186/metrics}"

log() { printf 'broker-cutover-preflight: %s\n' "$*"; }
die() {
  printf 'broker-cutover-preflight: ERROR: %s\n' "$*" >&2
  exit 1
}

case "$MODE" in
  shadow|live) ;;
  *)
    die "usage: $0 shadow|live"
    ;;
esac

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
fi

CH_INGEST_SOURCE="${CH_INGEST_SOURCE:-}"
BROKER_URL="${BROKER_URL:-}"
BROKER_SHADOW_MODE="${BROKER_SHADOW_MODE:-0}"

if [[ "$CH_INGEST_SOURCE" != "broker" ]]; then
  die "CH_INGEST_SOURCE must be broker (got ${CH_INGEST_SOURCE:-unset})"
fi
if [[ -z "$BROKER_URL" ]]; then
  die "BROKER_URL unset"
fi

if [[ "$MODE" == "shadow" ]]; then
  if [[ "$BROKER_SHADOW_MODE" != "1" ]]; then
    die "shadow mode requires BROKER_SHADOW_MODE=1 before cutover"
  fi
  log "shadow phase OK (BROKER_SHADOW_MODE=1); drain Redis _ch PEL before live flip"
else
  if [[ "$BROKER_SHADOW_MODE" != "0" ]]; then
    die "live mode requires BROKER_SHADOW_MODE=0 after cutover"
  fi
fi

if ! curl -sf --max-time 5 "$METRICS_URL" > /tmp/aed-broker-cutover-metrics.txt 2> /dev/null; then
  die "processor metrics unreachable at $METRICS_URL (start processor)"
fi

if grep -q 'ad_broker_ingest_divergence_high 1' /tmp/aed-broker-cutover-metrics.txt; then
  die "ad_broker_ingest_divergence_high=1 - rollback cutover and reconcile broker vs Redis stream"
fi

lag_lines="$(grep -c '^ad_broker_consumer_lag_messages' /tmp/aed-broker-cutover-metrics.txt || true)"
log "divergence_high=0 broker_consumer_lag_series=${lag_lines} mode=${MODE}"
log "OK: broker cutover preflight (${MODE})"
