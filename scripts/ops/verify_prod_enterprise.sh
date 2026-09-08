#!/usr/bin/env bash
# Role: Fail when production ENV misses P2 enterprise knobs (broker-primary, multi-shard, optional XDP).
# Execution context: Post-deploy after verify_prod_tuning.sh and verify_prod_capacity.sh pass.
# Env knobs: CH_INGEST_SOURCE=broker; BROKER_URL; BROKER_SHADOW_MODE=0 (live cutover);
#   REDIS_SHARD_COUNT=4; INGEST_TRACKER_COUNT=4; EDGE_XDP_ENABLED=1 enables XDP checks.
# Verify: bash scripts/ops/verify_prod_enterprise.sh .env.prod.example
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"
ENV_FILE="${1:-$ROOT/.env}"

log() { printf 'verify-prod-enterprise: %s\n' "$*"; }
die() {
  printf 'verify-prod-enterprise: ERROR: %s\n' "$*" >&2
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
CH_INGEST_SOURCE="${CH_INGEST_SOURCE:-}"
BROKER_URL="${BROKER_URL:-}"
BROKER_SHADOW_MODE="${BROKER_SHADOW_MODE:-0}"
REDIS_SHARD_COUNT="${REDIS_SHARD_COUNT:-0}"
INGEST_TRACKER_COUNT="${INGEST_TRACKER_COUNT:-1}"
EDGE_XDP_ENABLED="${EDGE_XDP_ENABLED:-0}"
EDGE_XDP_INGRESS_INTERFACE="${EDGE_XDP_INGRESS_INTERFACE:-}"
EDGE_BPF_PIN_DIR="${EDGE_BPF_PIN_DIR:-}"

if [[ "$ENV" != "production" ]]; then
  log "ENV=$ENV (not production) - enterprise checks skipped"
  exit 0
fi

bash "$SCRIPTS/ops/verify_redis_topology.sh" "$ENV_FILE"

if [[ "$CH_INGEST_SOURCE" != "broker" ]]; then
  die "production P2 requires CH_INGEST_SOURCE=broker (got ${CH_INGEST_SOURCE:-unset})"
fi

if [[ -z "$BROKER_URL" ]]; then
  die "CH_INGEST_SOURCE=broker requires BROKER_URL"
fi

if [[ "$BROKER_SHADOW_MODE" != "0" ]]; then
  die "production live cutover requires BROKER_SHADOW_MODE=0 (use broker_cutover_preflight.sh shadow before flip)"
fi

if [[ "$REDIS_SHARD_COUNT" -lt 4 ]]; then
  die "production P2 multi-shard scale requires REDIS_SHARD_COUNT=4 (got $REDIS_SHARD_COUNT)"
fi

if [[ "$INGEST_TRACKER_COUNT" -lt 4 ]]; then
  die "production P2 horizontal ingest requires INGEST_TRACKER_COUNT=4 (nginx peers tracker-0..3)"
fi

if [[ "$EDGE_XDP_ENABLED" == "1" ]]; then
  if [[ -z "$EDGE_XDP_INGRESS_INTERFACE" || "$EDGE_XDP_INGRESS_INTERFACE" == "lo" ]]; then
    die "EDGE_XDP_ENABLED=1 requires EDGE_XDP_INGRESS_INTERFACE=eth0 (or production NIC, not lo)"
  fi
  if [[ -z "$EDGE_BPF_PIN_DIR" ]]; then
    die "EDGE_XDP_ENABLED=1 requires EDGE_BPF_PIN_DIR"
  fi
  if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
    die "EDGE_XDP_ENABLED=1 requires BTF (/sys/kernel/btf/vmlinux)"
  fi
fi

log "OK: broker-primary REDIS_SHARD_COUNT=${REDIS_SHARD_COUNT} INGEST_TRACKER_COUNT=${INGEST_TRACKER_COUNT} EDGE_XDP_ENABLED=${EDGE_XDP_ENABLED}"
