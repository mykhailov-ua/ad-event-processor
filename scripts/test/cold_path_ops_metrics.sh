#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a

  source "$ROOT/.env"
  set +a
fi

PROC_URL="${COLD_OPS_PROCESSOR_METRICS:-http://127.0.0.1:9091/metrics}"

log() { printf 'cold-path-ops-metrics: %s\n' "$*"; }

print_catalog() {
  cat << 'EOF'




ad_micro_batch_boosts_written_total







ad_ch_janitor_retention_drop_total
ad_ch_janitor_emergency_drop_total
ad_ch_janitor_recompress_total

EOF
}

fetch_metric() {
  local url="$1"
  local pattern="$2"
  if command -v curl > /dev/null 2>&1; then
    curl -fsS --max-time 3 "$url" 2> /dev/null | grep -E "$pattern" || true
  fi
}

log "metric catalog"
print_catalog
echo

if [[ "${COLD_OPS_SKIP_LIVE:-0}" == "1" ]]; then
  log "live scrape skipped"
  printf 'fault_proof fault=cold_path_ops_metrics status=partial proof=catalog_only harness=script\n'
  exit 0
fi

log "live scrape processor: $PROC_URL"
fetch_metric "$PROC_URL" '^(ad_micro_batch_boosts_written_total|ad_ch_janitor_)'
echo

printf 'fault_proof fault=cold_path_ops_metrics status=partial proof=catalog+live_scrape harness=script fraud_scoring_enabled=%s\n' \
  "${FRAUD_SCORING_ENABLED:-}"
log "done"
