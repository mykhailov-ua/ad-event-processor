#!/usr/bin/env bash
# Role: wrk-based load with optional BPF probes for ingest-only or constrained-ch profiles.
# Execution context: Dev host; requires bin/wrk and deploy/dev/bpf/loadtest_probe.o.
# Env knobs: WRK_DURATION (30s); WRK_THREADS (4); WRK_CONNECTIONS (200); AD_EVENT_PROCESSOR_BPF_PROBE (1).
# Verify: bash scripts/test/load/wrk_bpf.sh ingest-only
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/load_test_env.sh"
cd "$ROOT"

PROFILE="${1:-ingest-only}"
DURATION="${WRK_DURATION:-30s}"
THREADS="${WRK_THREADS:-4}"
CONNECTIONS="${WRK_CONNECTIONS:-200}"
TARGET_URL="${WRK_TARGET_URL:-}"

log() { printf 'wrk-load-bpf: %s\n' "$*"; }
die() {
  printf 'wrk-load-bpf: ERROR: %s\n' "$*" >&2
  exit 1
}

[[ "$(id -u)" -eq 0 ]] && die "do not run as root"

case "$PROFILE" in
  ingest-only | constrained-ch) ;;
  -h | --help)
    cat << 'EOF'
usage: wrk_load_bpf.sh [ingest-only|constrained-ch]

env: WRK_DURATION WRK_THREADS WRK_CONNECTIONS WRK_TARGET_URL
     WRK_PCT_OPENRTB_BID WRK_PCT_CLICK WRK_MIX_CLICK_PCT
     WRK_TRACK_BODY (openrtb3|native, default openrtb3; WRK_NATIVE is ignored)
     AD_EVENT_PROCESSOR_BPF_PROBE=1 AD_EVENT_PROCESSOR_BPF_SUDO_PASS=...
EOF
    exit 0
    ;;
  *) die "unknown profile: $PROFILE (use ingest-only|constrained-ch)" ;;
esac

[[ -x "$ROOT/bin/wrk" ]] || die "missing bin/wrk (apt install wrk && cp $(command -v wrk) bin/wrk)"
[[ -f "$ROOT/deploy/dev/bpf/loadtest_probe.o" ]] || die "missing BPF object (make bpf-dev)"

if [[ -z "$TARGET_URL" ]]; then
  case "$PROFILE" in
    ingest-only) TARGET_URL="http://127.0.0.1:8181" ;;
    constrained-ch)
      load_test_bootstrap "$ROOT" 2> /dev/null || true
      TARGET_URL="http://127.0.0.1:$(load_test_tracker_ingest_port 0)"
      ;;
  esac
fi

WRK_BASE="${TARGET_URL%/track}"
WRK_BASE="${WRK_BASE%/}"

unset WRK_NATIVE
export WRK_TRACK_BODY="${WRK_TRACK_BODY:-openrtb3}"
case "$WRK_TRACK_BODY" in
  openrtb3 | openrtb_3 | native | ad_event_processor_native) ;;
  *)
    die "WRK_TRACK_BODY must be openrtb3 or native (got: $WRK_TRACK_BODY)"
    ;;
esac

OUT="${WRK_OUT:-$ROOT/var/load-test/$(date -u +%Y%m%dT%H%M%SZ)-${PROFILE}-wrk}"
mkdir -p "$OUT"
printf 'profile=%s\nduration=%s\nthreads=%s\nconnections=%s\ntarget=%s\ntrack_body=%s\nopenrtb_bid_pct=%s\nclick_pct=%s\n' \
  "$PROFILE" "$DURATION" "$THREADS" "$CONNECTIONS" "$WRK_BASE" "$WRK_TRACK_BODY" \
  "${WRK_PCT_OPENRTB_BID:-5}" "${WRK_PCT_CLICK:-${WRK_MIX_CLICK_PCT:-20}}" > "$OUT/profile.txt"

curl -sf --max-time 3 "${WRK_BASE}/health" > /dev/null \
  || die "tracker not healthy at ${WRK_BASE}/health"

BPF_PID=""
if [[ "${AD_EVENT_PROCESSOR_BPF_PROBE:-1}" == "1" ]]; then
  export AD_EVENT_PROCESSOR_BPF_PROBE=1
  export AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE="${AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE:-10}"
  export AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM="${AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM:-wrk}"
  export AD_EVENT_PROCESSOR_BPF_TRACK_LOADGEN="${AD_EVENT_PROCESSOR_BPF_TRACK_LOADGEN:-1}"
  if ! bash "$SCRIPTS/test/bpf/probe_session.sh" start "$OUT"; then
    die "BPF probe start failed (sudo password? set AD_EVENT_PROCESSOR_BPF_SUDO_PASS)"
  fi
  [[ -f "$OUT/bpf/collector.pid" ]] && BPF_PID="$(cat "$OUT/bpf/collector.pid")"
fi

log "wrk -t${THREADS} -c${CONNECTIONS} -d${DURATION} ${WRK_BASE} track_body=${WRK_TRACK_BODY} openrtb_bid=${WRK_PCT_OPENRTB_BID:-5}% click=${WRK_PCT_CLICK:-${WRK_MIX_CLICK_PCT:-20}}%"
set +e
"$ROOT/bin/wrk" -t"$THREADS" -c"$CONNECTIONS" -d"$DURATION" \
  --latency -s "$SCRIPTS/test/load/wrk_track.lua" "$WRK_BASE" 2>&1 | tee "$OUT/wrk.log"
WRK_EC=$?
set -e

[[ -n "$BPF_PID" ]] && bash "$SCRIPTS/test/bpf/probe_session.sh" stop "$OUT" "$BPF_PID" || true

if [[ "$WRK_EC" -ne 0 ]]; then
  die "wrk exited ${WRK_EC} - see $OUT/wrk.log"
fi

read -r ATTEMPT_RPS SUCCESS_RPS OK_PCT TOTAL_REQ NON2XX < <(
  awk -v dur="${DURATION%s}" '
    /requests in/ { req = $1 }
    /Non-2xx/ { gsub(/[^0-9]/, "", $5); bad = $5 + 0 }
    /Requests\/sec:/ { attempt = $2 }
    END {
      ok = req - bad
      if (dur + 0 > 0) {
        success = ok / dur
      } else {
        success = 0
      }
      pct = (req > 0 ? ok * 100 / req : 0)
      printf "%s %.2f %.2f %d %d\n", attempt, success, pct, req + 0, bad + 0
    }
  ' "$OUT/wrk.log"
)
printf 'attempt_rps=%s\nsuccess_rps=%s\nok_pct=%s\ntotal_requests=%s\nnon2xx=%s\n' \
  "${ATTEMPT_RPS:-unknown}" "${SUCCESS_RPS:-unknown}" "${OK_PCT:-unknown}" \
  "${TOTAL_REQ:-0}" "${NON2XX:-0}" >> "$OUT/profile.txt"
log "attempt_RPS=${ATTEMPT_RPS:-unknown} success_RPS=${SUCCESS_RPS:-unknown} ok_pct=${OK_PCT:-unknown}% non2xx=${NON2XX:-0}/${TOTAL_REQ:-0}"
log "artifacts: $OUT"

if command -v go > /dev/null 2>&1; then
  export LOAD_SLA_GATE=0
  go run ./cmd/load-report all "$OUT" 2>&1 | tee "$OUT/load-report.log" || log "WARN: load-report failed"
fi
