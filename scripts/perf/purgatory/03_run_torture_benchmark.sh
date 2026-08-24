#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

require_root
require_cmd awk sed grep date tee
assert_cgroup_v2
ensure_state_dir

[[ -d "$CGROUP_PATH" ]] || die "cgroup ${CGROUP_PATH} missing — run 01_setup_purgatory_cgroup.sh"

LOADGEN=""
if have_cmd wrk; then
  LOADGEN=wrk
elif have_cmd wrk2; then
  LOADGEN=wrk2
elif have_cmd k6; then
  LOADGEN=k6
else
  die "need wrk, wrk2, or k6. Install: apt-get install -y wrk  |  go install github.com/tsenart/vegeta@latest is NOT a substitute"
fi
log "load generator: ${LOADGEN}"

if have_cmd cgexec; then
  log "cgexec available (will prefer cgroup.procs attach on cgroup v2)"
else
  warn "cgexec not installed (cgroup-tools); using direct cgroup.procs writes"
fi

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${RUN_DIR}/${TS}"
mkdir -p "$OUT"
log "artifacts → ${OUT}"

CPU_STAT_BEFORE="${OUT}/cpu.stat.before"
read_cpu_stat > "$CPU_STAT_BEFORE"
MEM_EVENTS_BEFORE="${OUT}/memory.events.before"
read_memory_events > "$MEM_EVENTS_BEFORE"
OOM_BEFORE="$(oom_kill_count)"

SUT_PID=""
STARTED_BY_US=0

cleanup_sut_on_exit() {
  local ec=$?
  if [[ "$STARTED_BY_US" -eq 1 ]] && [[ -n "$SUT_PID" ]] && kill -0 "$SUT_PID" 2> /dev/null; then
    log "stopping SUT pid=${SUT_PID}"
    kill -TERM "$SUT_PID" 2> /dev/null || true

    for _ in 1 2 3 4 5; do
      kill -0 "$SUT_PID" 2> /dev/null || break
      sleep 0.2
    done
    kill -KILL "$SUT_PID" 2> /dev/null || true
  fi
  return "$ec"
}
trap cleanup_sut_on_exit EXIT

if [[ "${SKIP_START:-0}" == "1" ]] || [[ -n "${TARGET_PID:-}" ]]; then
  SUT_PID="${TARGET_PID:-}"
  [[ -n "$SUT_PID" ]] || die "SKIP_START=1 requires TARGET_PID"
  log "attaching existing PID ${SUT_PID} into ${CGROUP_PATH}"
  cgroup_attach_pid "$SUT_PID"
else
  [[ -x "$TARGET_BIN" ]] || die "TARGET_BIN not executable: ${TARGET_BIN} (build tracker or set TARGET_BIN / TARGET_PID)"
  log "starting ${TARGET_BIN} inside ${CGROUP_PATH}"

  export GOMAXPROCS="${GOMAXPROCS:-1}"
  export GOMEMLIMIT="${GOMEMLIMIT:-60MiB}"
  export GOGC="${GOGC:-50}"

  FIFO="${STATE_DIR}/sut.ready.$$"
  mkfifo "$FIFO"

  (
    echo $$ > "${CGROUP_PATH}/cgroup.procs"
    echo ready > "$FIFO"
    exec "$TARGET_BIN" ${TARGET_ARGS:-}
  ) > "${OUT}/sut.stdout" 2> "${OUT}/sut.stderr" &
  SUT_PID=$!
  STARTED_BY_US=1

  if ! timeout 5 cat "$FIFO" > /dev/null; then
    rm -f "$FIFO"
    die "SUT failed to signal readiness within 5s; see ${OUT}/sut.stderr"
  fi
  rm -f "$FIFO"
  echo "$SUT_PID" > "$SUT_PID_FILE"
  cg_path="$(awk -F: '$1=="0"{print $3}' "/proc/${SUT_PID}/cgroup" 2> /dev/null || echo '?')"
  log "SUT pid=${SUT_PID} cgroup=${cg_path}"

  log "waiting for ${TARGET_HOST}:${TARGET_PORT}"
  ready=0
  for _ in $(seq 1 100); do
    if ! kill -0 "$SUT_PID" 2> /dev/null; then
      die "SUT died during startup (likely OOM under 64 MiB). Check dmesg and ${OUT}/sut.stderr"
    fi
    if have_cmd ss; then
      if ss -lnt "sport = :${TARGET_PORT}" 2> /dev/null | grep -q ":${TARGET_PORT}"; then
        ready=1
        break
      fi
    elif bash -c "echo >/dev/tcp/${TARGET_HOST}/${TARGET_PORT}" 2> /dev/null; then
      ready=1
      break
    fi
    sleep 0.1
  done
  [[ "$ready" -eq 1 ]] || warn "listen probe timed out; continuing anyway"
fi

if ! grep -qx "$SUT_PID" "${CGROUP_PATH}/cgroup.procs" 2> /dev/null; then

  warn "PID ${SUT_PID} not listed in cgroup.procs (threads may differ); forcing attach"
  cgroup_attach_pid "$SUT_PID" || true
fi

TRACK_BODY="{\"campaign_id\":\"${TRACK_CAMPAIGN_ID}\",\"type\":\"click\"}"
BODY_FILE="${OUT}/track_body.json"
printf '%s' "$TRACK_BODY" > "$BODY_FILE"

LOAD_LOG="${OUT}/loadgen.log"
log "driving ${BENCH_CONNECTIONS} connections for ${BENCH_DURATION} → ${TARGET_URL}"

run_wrk() {
  local bin=$1
  local lua="${OUT}/track.lua"

  cat > "$lua" << EOF
wrk.method = "POST"
wrk.body   = [=[${TRACK_BODY}]=]
wrk.headers["Content-Type"] = "application/json"
wrk.headers["Connection"]   = "keep-alive"
wrk.headers["Accept"]       = "application/json"
EOF

  if [[ "$bin" == "wrk2" ]]; then
    wrk2 -t "$BENCH_THREADS" -c "$BENCH_CONNECTIONS" -d "$BENCH_DURATION" \
      --latency -R "${BENCH_RATE:-50000}" -s "$lua" "$TARGET_URL" \
      | tee "$LOAD_LOG"
  else
    wrk -t "$BENCH_THREADS" -c "$BENCH_CONNECTIONS" -d "$BENCH_DURATION" \
      --latency -s "$lua" "$TARGET_URL" \
      | tee "$LOAD_LOG"
  fi
}

run_k6() {
  local js="${OUT}/track_k6.js"

  local secs
  secs="$(echo "$BENCH_DURATION" | sed -E 's/^([0-9]+).*/\1/')"
  cat > "$js" << EOF
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    torture: {
      executor: 'constant-vus',
      vus: ${BENCH_CONNECTIONS},
      duration: '${BENCH_DURATION}',
    },
  },
  thresholds: {
    http_req_duration: ['p(99)<10000'],
  },
};

const url = '${TARGET_URL}';
const body = '${TRACK_BODY}';
const params = {
  headers: {
    'Content-Type': 'application/json',
    'Connection': 'keep-alive',
    'Accept': 'application/json',
  },
};

export default function () {
  const res = http.post(url, body, params);
  check(res, { 'status is 2xx/4xx': (r) => r.status >= 200 && r.status < 500 });
}
EOF
  k6 run --summary-export "${OUT}/k6_summary.json" "$js" | tee "$LOAD_LOG"
}

case "$LOADGEN" in
  wrk | wrk2) run_wrk "$LOADGEN" || warn "loadgen exited non-zero (SUT may have been OOM-killed)" ;;
  k6) run_k6 || warn "k6 exited non-zero" ;;
esac

CPU_STAT_AFTER="${OUT}/cpu.stat.after"
read_cpu_stat > "$CPU_STAT_AFTER"
MEM_EVENTS_AFTER="${OUT}/memory.events.after"
read_memory_events > "$MEM_EVENTS_AFTER"
OOM_AFTER="$(oom_kill_count)"

stat_field() {
  local file=$1 key=$2
  awk -v k="$key" '$1 == k { print $2; found=1 } END { if (!found) print 0 }' "$file"
}

NR_THROTTLED_BEFORE="$(stat_field "$CPU_STAT_BEFORE" nr_throttled)"
NR_THROTTLED_AFTER="$(stat_field "$CPU_STAT_AFTER" nr_throttled)"
THROTTLED_USEC_BEFORE="$(stat_field "$CPU_STAT_BEFORE" throttled_usec)"
THROTTLED_USEC_AFTER="$(stat_field "$CPU_STAT_AFTER" throttled_usec)"
NR_THROTTLED_DELTA=$((NR_THROTTLED_AFTER - NR_THROTTLED_BEFORE))
THROTTLED_USEC_DELTA=$((THROTTLED_USEC_AFTER - THROTTLED_USEC_BEFORE))

OOM_DELTA=$((OOM_AFTER - OOM_BEFORE))

OOM_DMESG="${OUT}/oom_dmesg.txt"
{ dmesg -T 2> /dev/null || journalctl -k -n 200 --no-pager 2> /dev/null || true; } \
  | grep -iE "oom|killed process|${SUT_PID}" \
  | tail -n 50 > "$OOM_DMESG" || true

SUT_ALIVE=0
if [[ -n "$SUT_PID" ]] && kill -0 "$SUT_PID" 2> /dev/null; then
  SUT_ALIVE=1
fi

METRICS_JSON="${OUT}/metrics.json"
parse_wrk_metrics() {

  local rps p50 p95 p99 p999
  rps="$(awk '/Requests\/sec:/ { print $2 }' "$LOAD_LOG" | tail -1)"
  p50="$(awk '/^[[:space:]]*50%/ { print $2$3 }' "$LOAD_LOG" | tail -1)"

  p95="$(awk '/^[[:space:]]*90%/ { print $2$3 }' "$LOAD_LOG" | tail -1)"
  p99="$(awk '/^[[:space:]]*99%/ { print $2$3 }' "$LOAD_LOG" | tail -1)"

  p999="$(awk '/^[[:space:]]*99\.9%/ { print $2$3 }' "$LOAD_LOG" | tail -1)"
  [[ -n "$p999" ]] || p999="n/a (wrk has no p99.9; use wrk2/k6)"
  cat > "$METRICS_JSON" << EOF
{
  "loadgen": "${LOADGEN}",
  "rps": "${rps:-n/a}",
  "p50": "${p50:-n/a}",
  "p95_approx_p90": "${p95:-n/a}",
  "p99": "${p99:-n/a}",
  "p99_9": "${p999}",
  "connections": ${BENCH_CONNECTIONS},
  "duration": "${BENCH_DURATION}",
  "url": "${TARGET_URL}"
}
EOF
}

parse_k6_metrics() {
  local summary="${OUT}/k6_summary.json"
  if [[ ! -f "$summary" ]]; then
    warn "k6 summary missing"
    echo '{"loadgen":"k6","error":"no summary"}' > "$METRICS_JSON"
    return
  fi

  if have_cmd jq; then
    jq '{
		  loadgen: "k6",
		  rps: (.metrics.http_reqs.values.rate // .metrics.http_reqs.rate // "n/a"),
		  p50: (.metrics.http_req_duration.values["p(50)"] // "n/a"),
		  p95: (.metrics.http_req_duration.values["p(95)"] // "n/a"),
		  p99: (.metrics.http_req_duration.values["p(99)"] // "n/a"),
		  p99_9: (.metrics.http_req_duration.values["p(99.9)"] // "n/a"),
		  connections: '"${BENCH_CONNECTIONS}"',
		  duration: "'"${BENCH_DURATION}"'",
		  url: "'"${TARGET_URL}"'"
		}' "$summary" > "$METRICS_JSON"
  elif have_cmd python3; then
    python3 - "$summary" << 'PY' > "$METRICS_JSON"
import json, sys
s = json.load(open(sys.argv[1]))
m = s.get("metrics", {})
dur = m.get("http_req_duration", {}).get("values", m.get("http_req_duration", {}))
reqs = m.get("http_reqs", {}).get("values", m.get("http_reqs", {}))
def g(d, *keys):
    for k in keys:
        if isinstance(d, dict) and k in d: return d[k]
    return "n/a"
out = {
  "loadgen": "k6",
  "rps": g(reqs, "rate"),
  "p50": g(dur, "p(50)"),
  "p95": g(dur, "p(95)"),
  "p99": g(dur, "p(99)"),
  "p99_9": g(dur, "p(99.9)"),
}
print(json.dumps(out, indent=2))
PY
  else
    warn "install jq or python3 to parse k6 summary"
    echo '{"loadgen":"k6","note":"raw summary at k6_summary.json"}' > "$METRICS_JSON"
  fi
}

case "$LOADGEN" in
  wrk | wrk2) parse_wrk_metrics ;;
  k6) parse_k6_metrics ;;
esac

REPORT="${OUT}/REPORT.txt"
{
  echo "=== Purgatory Torture Report (${TS}) ==="
  echo
  echo "-- SUT --"
  echo "pid:            ${SUT_PID}"
  echo "alive:          ${SUT_ALIVE}"
  echo "binary:         ${TARGET_BIN}"
  echo "cgroup:         ${CGROUP_PATH}"
  echo "url:            ${TARGET_URL}"
  echo
  echo "-- CPU throttling (cgroup cpu.stat delta) --"
  echo "nr_throttled:   ${NR_THROTTLED_DELTA}  (before=${NR_THROTTLED_BEFORE} after=${NR_THROTTLED_AFTER})"
  echo "throttled_usec: ${THROTTLED_USEC_DELTA}  (before=${THROTTLED_USEC_BEFORE} after=${THROTTLED_USEC_AFTER})"
  echo
  echo "-- OOM --"
  echo "oom_kill delta: ${OOM_DELTA}  (before=${OOM_BEFORE} after=${OOM_AFTER})"
  if [[ "$OOM_DELTA" -gt 0 ]] || [[ "$SUT_ALIVE" -eq 0 ]]; then
    echo "OOM_TRIGGERED:  YES (or process dead)"
  else
    echo "OOM_TRIGGERED:  NO"
  fi
  echo "dmesg snippet:  ${OOM_DMESG}"
  echo
  echo "-- Load metrics --"
  cat "$METRICS_JSON"
  echo
  echo "-- cpu.stat (after) --"
  cat "$CPU_STAT_AFTER"
  echo
  echo "-- memory.events (after) --"
  cat "$MEM_EVENTS_AFTER"
  echo
  echo "artifacts: ${OUT}"
} | tee "$REPORT"

log "done. Report: ${REPORT}"
log "cleanup when finished: sudo bash scripts/perf/purgatory/04_cleanup.sh"
