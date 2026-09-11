#!/usr/bin/env bash
# Role: Optional ClickHouse ingest lag proof under constant /track load.
# Execution context: compose stack with processor + prometheus; gated by LOAD_CH_PROOF=1.
# Env knobs: LOAD_CH_PROOF (1); CH_LAG_P99_MAX (120); CH_SINGLE_ROW_MAX_DELTA (50); BASE_RATE (500); DURATION (60s).
# Verify: LOAD_CH_PROOF=1 bash scripts/test/load/ch_ingest_spike.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/load_test_env.sh"
cd "$ROOT"

if [[ "${LOAD_CH_PROOF:-0}" != "1" ]]; then
  printf 'ch_ingest_spike: skip (set LOAD_CH_PROOF=1 to run)\n'
  exit 0
fi

PROMETHEUS_URL="${PROMETHEUS_URL:-http://127.0.0.1:9190}"
CH_LAG_P99_MAX="${CH_LAG_P99_MAX:-120}"
CH_SINGLE_ROW_MAX_DELTA="${CH_SINGLE_ROW_MAX_DELTA:-50}"
BASE_RATE="${BASE_RATE:-500}"
DURATION="${DURATION:-60s}"
OUT="$ROOT/var/load-test/ch-ingest-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUT"

CONSTRAINED="${CONSTRAINED:-1}"
if [[ "$CONSTRAINED" == "1" ]]; then
  load_test_compose COMPOSE "$ROOT"
else
  COMPOSE=(docker compose)
fi

log() { printf 'ch_ingest_spike: %s\n' "$*"; }
die() {
  printf 'ch_ingest_spike: ERROR: %s\n' "$*" >&2
  exit 1
}

prom_query_scalar() {
  local query=$1
  local url body
  url="${PROMETHEUS_URL%/}/api/v1/query"
  body="$(curl -sfG --max-time 15 --data-urlencode "query=${query}" "$url" 2>/dev/null)" || return 1
  python3 - "$body" << 'PY'
import json, sys
data = json.loads(sys.argv[1])
if data.get("status") != "success":
    sys.exit(1)
results = data.get("data", {}).get("result") or []
if not results:
    sys.exit(1)
val = results[0].get("value", [None, None])[1]
if val is None:
    sys.exit(1)
print(val)
PY
}

if ! curl -sf --max-time 3 "${PROMETHEUS_URL%/}/-/ready" >/dev/null 2>&1; then
  die "prometheus not ready at $PROMETHEUS_URL"
fi

log "ensuring stack (constrained=${CONSTRAINED})"
"${COMPOSE[@]}" up -d --remove-orphans db redis-0 redis-1 redis-2 redis-3 processor tracker-0 tracker-1 nginx prometheus 2>&1 | tee "$OUT/compose.log"

if [[ "$CONSTRAINED" == "1" ]]; then
  TRACKER_BASES="${TRACKER_BASES:-$LOAD_TEST_CONSTRAINED_TRACKER_BASES_CSV}"
else
  TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
fi

single_before="$(prom_query_scalar 'sum(ad_ch_single_row_inserts_total)' || echo 0)"
log "baseline ad_ch_single_row_inserts_total=${single_before}"

log "constant load rate=${BASE_RATE} duration=${DURATION}"
go run ./cmd/loadgen \
  -out "$OUT" \
  -profile constant \
  -rate "$BASE_RATE" \
  -duration "$DURATION" \
  -trackers "$TRACKER_BASES" \
  2>&1 | tee "$OUT/loadgen.log"

sleep 10

lag_p99="$(prom_query_scalar 'histogram_quantile(0.99, sum(rate(ad_ch_ingest_lag_seconds_bucket[2m])) by (le))' || true)"
single_after="$(prom_query_scalar 'sum(ad_ch_single_row_inserts_total)' || echo 0)"
single_delta="$(python3 - "$single_before" "$single_after" << 'PY'
import sys
print(float(sys.argv[2]) - float(sys.argv[1]))
PY
)"

log "ad_ch_ingest_lag_seconds p99=${lag_p99:-na} single_row_delta=${single_delta}"

{
  echo "ch_ingest_spike_outcome=1"
  echo "lag_p99_seconds=${lag_p99:-na}"
  echo "single_row_delta=${single_delta}"
} > "$OUT/summary.env"

if [[ -z "${lag_p99:-}" ]]; then
  die "ad_ch_ingest_lag_seconds unavailable (processor metrics or insufficient samples)"
fi

python3 - "$lag_p99" "$CH_LAG_P99_MAX" "$single_delta" "$CH_SINGLE_ROW_MAX_DELTA" << 'PY'
import sys
lag = float(sys.argv[1])
lag_max = float(sys.argv[2])
delta = float(sys.argv[3])
delta_max = float(sys.argv[4])
if lag > lag_max:
    raise SystemExit(f"lag p99 {lag:.2f}s exceeds max {lag_max:.2f}s")
if delta > delta_max:
    raise SystemExit(f"single-row insert delta {delta:.0f} exceeds max {delta_max:.0f}")
print(f"ok lag_p99={lag:.2f}s single_row_delta={delta:.0f}")
PY

log "done - $OUT"
