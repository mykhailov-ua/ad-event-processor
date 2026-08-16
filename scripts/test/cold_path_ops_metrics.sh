#!/usr/bin/env bash
# Cold-path ops metrics — fraud ML, billing exports, ClickHouse housekeeping.
#
# Save output before/after tuning; does not claim SLA wins without your diff.
#
# Usage:
#   bash scripts/test/cold_path_ops_metrics.sh > var/cold-ops-before.txt
#
# Env:
#   COLD_OPS_PROCESSOR_METRICS  default http://127.0.0.1:9091/metrics
#   COLD_OPS_CONTROL_METRICS    default http://127.0.0.1:8188/metrics (if exposed)
#   COLD_OPS_SKIP_LIVE=1        catalog only
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

PROC_URL="${COLD_OPS_PROCESSOR_METRICS:-http://127.0.0.1:9091/metrics}"

log() { printf 'cold-path-ops-metrics: %s\n' "$*"; }

print_catalog() {
	cat <<'EOF'
# --- Fraud ML (cold; tracker must NOT import internal/fraud) ---
# Enable: FRAUD_SCORING_ENABLED=1 + ml_fraud_boost license + model at FRAUD_SCORING_MODEL_PATH
# fraud-scorer: FRAUD_SCORING_SCAN_INTERVAL_MS (default 300000), FRAUD_SCORING_BATCH_SIZE (default 1000)
# Processor micro-batcher boost writes:
ad_micro_batch_boosts_written_total
# ONNX: only if build uses -tags fraudscoring_onnx and CPU profiling shows LGBM bottleneck

# --- Billing exports (admin cold path) ---
# BILLING_EXPORT_FETCH_ROWS, BILLING_EXPORT_JOB_TIMEOUT_MIN
# License cap: max_export_chunk_bytes (download LimitReader)

# --- ClickHouse housekeeping (processor janitor) ---
ad_ch_janitor_retention_drop_total
ad_ch_janitor_emergency_drop_total
ad_ch_janitor_recompress_total
# Tune when disk > ~85% or merge storm: CH_JANITOR_*, CH_RECOMPRESS_PARTS_THRESHOLD, off-peak UTC hours
EOF
}

fetch_metric() {
	local url="$1"
	local pattern="$2"
	if command -v curl >/dev/null 2>&1; then
		curl -fsS --max-time 3 "$url" 2>/dev/null | grep -E "$pattern" || true
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
