#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"
ENV_FILE="${1:-$ROOT/.env}"
STRICT="${STRICT:-0}"

log() { printf 'edge-phase0: %s\n' "$*"; }
warn() { printf 'edge-phase0: WARN: %s\n' "$*" >&2; }
die() {
  printf 'edge-phase0: ERROR: %s\n' "$*" >&2
  exit 1
}

fail=0

run_check() {
  local name=$1
  shift
  log "check: $name"
  if "$@"; then
    log "  OK: $name"
  else
    warn "  FAIL: $name"
    fail=1
  fi
}

if [[ -f "$ENV_FILE" ]]; then
  set -a

  . "$ENV_FILE"
  set +a
fi

run_check "prod FILTER_TIMEOUT_MS" bash "$SCRIPTS/ops/verify_prod_tuning.sh" "$ENV_FILE" || true

if [[ -x "$SCRIPTS/ops/sysctl.sh" ]]; then
  run_check "sysctl" bash "$SCRIPTS/ops/sysctl.sh" verify || {
    [[ "$STRICT" == "1" ]] || warn "sysctl not applied (run: sudo bash scripts/ops/sysctl.sh apply)"
  }
else
  warn "sysctl.sh missing"
fi

if [[ -x "$SCRIPTS/ops/nic_tune.sh" ]]; then
  run_check "NIC RX/IRQ" bash "$SCRIPTS/ops/nic_tune.sh" verify || {
    [[ "$STRICT" == "1" ]] || warn "NIC tuning not applied (run: sudo bash scripts/ops/nic_tune.sh apply)"
  }
else
  warn "nic_tune.sh missing"
fi

run_check "nginx edge metrics" bash -c '
	curl -sf --max-time 3 http://127.0.0.1:8180/metrics/edge | grep -q ad_event_processor_edge_phase1_pass_total
' || warn "nginx :8180 /metrics/edge unreachable (start full stack)"

run_check "prometheus baseline snapshot" bash "$SCRIPTS/ops/baseline.sh" snapshot || true

if [[ "$fail" -ne 0 && "$STRICT" == "1" ]]; then
  die "one or more Phase 0 checks failed (STRICT=1)"
fi

log "Phase 0 preflight complete (see var/edge-baseline/latest.txt)"
