#!/usr/bin/env bash
# Role: BPF-instrumented wrk load inside purgatory cgroup with collector session.
# Execution context: Root for BPF attach; requires bpf-collector and loadtest_probe.o.
# Env knobs: BENCH_DURATION (60s); BENCH_CONNECTIONS (10000); TRACKER_CTR; TARGET_URL.
# Verify: sudo AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/perf/purgatory/run_with_bpf.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

export PATH="${ROOT}/bin:${PATH}"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${RUN_DIR}/${TS}"
mkdir -p "$OUT" "$STATE_DIR"
log "run artifacts -> ${OUT}"

TRACKER_CTR="${TRACKER_CTR:-ad-event-processor-tracker-0-1}"
TARGET_URL="${TARGET_URL:-http://127.0.0.1:8181/track}"
BENCH_DURATION="${BENCH_DURATION:-60s}"
BENCH_CONNECTIONS="${BENCH_CONNECTIONS:-10000}"
BENCH_THREADS="${BENCH_THREADS:-4}"

PHASE="${PHASE:-survival}"
if [[ "$PHASE" == "strict" ]]; then
  MEM_BYTES="${MEM_BYTES:-67108864}"
  MEM_LABEL="64MiB"
  CPUS="${CPUS:-0.5}"
  GOMAXPROCS_PIN="${GOMAXPROCS_PIN:-1}"
else
  MEM_BYTES="${MEM_BYTES:-536870912}"
  MEM_LABEL="512MiB"
  CPUS="${CPUS:-1.0}"
  GOMAXPROCS_PIN="${GOMAXPROCS_PIN:-1}"
fi

CPU_MAX_QUOTA="$(awk -v c="$CPUS" 'BEGIN { printf "%d", c * 100000 + 0.5 }')"
CPU_MAX_SPEC="${CPU_MAX_QUOTA} 100000"

require_cmd docker taskset
[[ -x "${ROOT}/bin/wrk" ]] || die "missing ${ROOT}/bin/wrk"
[[ -x "${ROOT}/bin/stress-ng" ]] || die "missing ${ROOT}/bin/stress-ng"
[[ -x "${ROOT}/bin/bpf-collector" ]] || die "missing ${ROOT}/bin/bpf-collector"
[[ -f "${ROOT}/deploy/dev/bpf/loadtest_probe.o" ]] || die "missing BPF object (make bpf-dev)"

PRIV_CTR="${PRIV_CTR:-ad-event-processor-purgatory-priv}"

ensure_priv() {
  if docker inspect -f '{{.State.Running}}' "$PRIV_CTR" 2> /dev/null | grep -q true; then
    return 0
  fi
  docker rm -f "$PRIV_CTR" > /dev/null 2>&1 || true
  log "starting privileged helper ${PRIV_CTR}"
  docker run -d --name "$PRIV_CTR" --privileged --pid=host --network=host \
    -v /sys/fs/cgroup:/sys/fs/cgroup \
    -v /sys/kernel/tracing:/sys/kernel/tracing \
    -v /sys/kernel/debug:/sys/kernel/debug \
    -v "${ROOT}:${ROOT}" \
    -v /lib/modules:/lib/modules:ro \
    -v /var/run/docker.sock:/var/run/docker.sock \
    ubuntu:24.04 \
    bash -lc 'export DEBIAN_FRONTEND=noninteractive
mount -t tracefs tracefs /sys/kernel/tracing 2>/dev/null || true
mount -t debugfs debugfs /sys/kernel/debug 2>/dev/null || true
apt-get update -qq && apt-get install -y -qq iproute2 procps >/dev/null && exec sleep infinity' \
    > /dev/null

  for _ in $(seq 1 120); do
    if docker exec "$PRIV_CTR" bash -lc 'command -v tc >/dev/null' 2> /dev/null; then
      return 0
    fi
    sleep 1
  done
  die "privileged helper failed to become ready"
}

priv() {
  ensure_priv
  docker exec "$PRIV_CTR" bash -lc "$*"
}

SYSCTL_KEYS=(
  net.core.rmem_max net.core.wmem_max
  net.ipv4.tcp_rmem net.ipv4.tcp_wmem
  net.ipv4.tcp_max_syn_backlog net.core.somaxconn
)
: > "${OUT}/sysctl.backup"
for k in "${SYSCTL_KEYS[@]}"; do
  printf '%s=%s\n' "$k" "$(sysctl -n "$k")" >> "${OUT}/sysctl.backup"
done
tc qdisc show dev lo > "${OUT}/tc.backup" || true
docker inspect -f '{{.HostConfig.NanoCpus}} {{.HostConfig.Memory}} {{.HostConfig.MemorySwap}}' "$TRACKER_CTR" \
  > "${OUT}/docker.limits.backup" || true

cleanup() {
  local ec=$?
  log "cleanup (exit=${ec})"

  if [[ -f "${OUT}/stress-ng.pid" ]]; then
    kill "$(cat "${OUT}/stress-ng.pid")" 2> /dev/null || true
  fi
  pkill -f 'stress-ng --cache' 2> /dev/null || true

  if [[ -f "${OUT}/bpf/collector.pid" ]]; then
    kill "$(cat "${OUT}/bpf/collector.pid")" 2> /dev/null || true
    priv "kill -TERM $(cat "${OUT}/bpf/collector.pid") 2>/dev/null || true" || true
  fi

  priv 'tc qdisc del dev lo root 2>/dev/null || true; tc qdisc add dev lo root noqueue 2>/dev/null || true' || true

  if [[ -f "${OUT}/sysctl.backup" ]]; then
    while IFS= read -r line; do
      [[ -z "$line" ]] && continue
      priv "sysctl -w '${line}' >/dev/null" || true
    done < "${OUT}/sysctl.backup"
  fi

  docker update --cpus=2 --memory=536870912 --memory-swap=536870912 "$TRACKER_CTR" > /dev/null 2>&1 || true
  docker compose -f "${ROOT}/docker-compose.yaml" -f "${ROOT}/docker-compose.load-test.yaml" \
    up -d --no-build --force-recreate --no-deps tracker-0 > /dev/null 2>&1 || true

  priv "rmdir /sys/fs/cgroup/purgatory 2>/dev/null || true" || true
  docker rm -f "$PRIV_CTR" > /dev/null 2>&1 || true
  return "$ec"
}
trap cleanup EXIT

OVERRIDES="${OUT}/compose.gomax.yaml"
cat > "$OVERRIDES" << EOF
services:
  tracker-0:
    environment:
      GOMAXPROCS: "${GOMAXPROCS_PIN}"
      GOMEMLIMIT: "400MiB"
      GOGC: "300"
      LOCAL_QUOTA_MODE: "off"
      METRICS_HISTOGRAM_SAMPLE_MASK: "-1"
EOF
log "recreating ${TRACKER_CTR} with GOMAXPROCS=${GOMAXPROCS_PIN} LOCAL_QUOTA=off Lua histogram 100% sample"
COMPOSE=(docker compose -f "${ROOT}/docker-compose.yaml" -f "${ROOT}/docker-compose.load-test.yaml" -f "$OVERRIDES")
"${COMPOSE[@]}" up -d --no-build --force-recreate --no-deps tracker-0

for _ in $(seq 1 60); do
  if docker inspect -f '{{.State.Health.Status}}' "$TRACKER_CTR" 2> /dev/null | grep -q healthy; then
    break
  fi
  sleep 1
done
bash "${SCRIPTS}/test/sync_tracker_registry.sh" > /dev/null 2>&1 || warn "registry sync skipped"

log "PHASE=${PHASE} memory=${MEM_LABEL} cpus=${CPUS} GOMAXPROCS=${GOMAXPROCS_PIN} on ${TRACKER_CTR}"
docker update --cpus="$CPUS" --memory="${MEM_BYTES}" --memory-swap="${MEM_BYTES}" "$TRACKER_CTR"
sleep 1
if ! docker inspect -f '{{.State.Running}}' "$TRACKER_CTR" | grep -q true; then
  log "tracker not running after limit apply - likely OOM under ${MEM_LABEL}"
  docker events --since 30s --until 0s --filter container="$TRACKER_CTR" 2> /dev/null | tee "${OUT}/docker.events.txt" || true
  docker logs "$TRACKER_CTR" 2>&1 | tail -50 | tee "${OUT}/tracker.oom.log" || true
  if [[ "$PHASE" == "strict" ]]; then
    printf 'OOM_OR_DEAD under 64MiB\n' | tee "${OUT}/REPORT.txt"
    exit 0
  fi
  die "tracker dead under survival memory - abort"
fi

docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$TRACKER_CTR" \
  | awk -F= '$1=="GOMAXPROCS"{print $2; found=1} END{if(!found) print "unset"}' \
  | tee "${OUT}/gomaxprocs.txt"
GOT_GOMAX="$(cat "${OUT}/gomaxprocs.txt")"
if [[ "$GOT_GOMAX" != "$GOMAXPROCS_PIN" ]]; then
  warn "GOMAXPROCS want=${GOMAXPROCS_PIN} got=${GOT_GOMAX} (compose merge may have lost the pin)"
fi

TRACKER_PID="$(docker inspect -f '{{.State.Pid}}' "$TRACKER_CTR")"
log "tracker host pid=${TRACKER_PID}"

priv "mkdir -p /sys/fs/cgroup/purgatory
echo '+cpu +cpuset +memory' > /sys/fs/cgroup/cgroup.subtree_control 2>/dev/null || true
echo 0 > /sys/fs/cgroup/purgatory/cpuset.mems 2>/dev/null || true
echo 0 > /sys/fs/cgroup/purgatory/cpuset.cpus
echo '${CPU_MAX_SPEC}' > /sys/fs/cgroup/purgatory/cpu.max
echo ${MEM_BYTES} > /sys/fs/cgroup/purgatory/memory.max
echo 0 > /sys/fs/cgroup/purgatory/memory.swap.max
echo ok" || warn "host purgatory cgroup setup partial"

CID="$(docker inspect -f '{{.Id}}' "$TRACKER_CTR")"
CG_PATH=""

for cand in \
  "/sys/fs/cgroup/system.slice/docker-${CID}.scope" \
  "/sys/fs/cgroup/docker/${CID}"; do
  if [[ -f "${cand}/cpu.stat" ]]; then
    CG_PATH="$cand"
    break
  fi
done
if [[ -z "$CG_PATH" ]]; then
  CG_REL="$(awk -F: '$1=="0"{
		line=$0; sub(/^0::?/,"",line); gsub(/^\/+/,"",line); print line
	}' "/proc/${TRACKER_PID}/cgroup" 2> /dev/null || true)"
  if [[ -n "$CG_REL" && -f "/sys/fs/cgroup/${CG_REL}/cpu.stat" ]]; then
    CG_PATH="/sys/fs/cgroup/${CG_REL}"
  fi
fi
[[ -n "$CG_PATH" ]] || warn "could not resolve tracker cgroup path (cpu.stat will be empty)"
log "tracker cgroup path=${CG_PATH:-unknown}"
echo "${CG_PATH:-}" > "${OUT}/tracker.cgroup.path"
if [[ -n "$CG_PATH" ]]; then
  cp "${CG_PATH}/cpu.stat" "${OUT}/cpu.stat.before" 2> /dev/null || true
  cp "${CG_PATH}/memory.events" "${OUT}/memory.events.before" 2> /dev/null || true
  cp "${CG_PATH}/memory.current" "${OUT}/memory.current.before" 2> /dev/null || true
fi

log "applying tc netem on lo + degraded sysctl"
priv 'tc qdisc del dev lo root 2>/dev/null || true
tc qdisc add dev lo root netem delay 5ms loss 1% duplicate 1%
sysctl -w net.core.rmem_max=65536
sysctl -w net.core.wmem_max=65536
sysctl -w net.ipv4.tcp_rmem="4096 87380 65536"
sysctl -w net.ipv4.tcp_wmem="4096 16384 65536"
sysctl -w net.ipv4.tcp_max_syn_backlog=128
sysctl -w net.core.somaxconn=128
tc qdisc show dev lo'

log "starting stress-ng L3 polluter on CPU 1"
taskset -c 1 "${ROOT}/bin/stress-ng" --cache 1 --cache-level 3 --aggressive --timeout 0 \
  > "${OUT}/stress-ng.log" 2>&1 &
echo $! > "${OUT}/stress-ng.pid"
sleep 0.5
kill -0 "$(cat "${OUT}/stress-ng.pid")" 2> /dev/null || {
  warn "cache-level failed; retrying minimal --cache"
  taskset -c 1 "${ROOT}/bin/stress-ng" --cache 1 --timeout 0 > "${OUT}/stress-ng.log" 2>&1 &
  echo $! > "${OUT}/stress-ng.pid"
}

BPF_DIR="${OUT}/bpf"
mkdir -p "$BPF_DIR"
log "resolving BPF targets"
AD_EVENT_PROCESSOR_BPF_NATIVE=1 AD_EVENT_PROCESSOR_BPF_TRACK_LOADGEN=1 AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM=wrk \
  bash "${SCRIPTS}/test/bpf_resolve_targets.sh" "${BPF_DIR}/targets.json" tracker,nginx,redis,processor

log "starting bpf-collector as root (privileged sidecar)"
priv "ulimit -l unlimited
cd ${ROOT}
nohup ./bin/bpf-collector \
  -session-dir ${BPF_DIR} \
  -bpf-object ${ROOT}/deploy/dev/bpf/loadtest_probe.o \
  -sample-rate ${AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE:-10} \
  -slow-us ${AD_EVENT_PROCESSOR_BPF_SLOW_US:-10000} \
  -discover-loadgen=true \
  -loadgen-comms wrk \
  -dump-interval 15s \
  >${BPF_DIR}/collector.log 2>&1 &
echo \$! > ${BPF_DIR}/collector.pid
for i in \$(seq 1 50); do
  if grep -q 'bpf-collector running\\|maps ready\\|attached' ${BPF_DIR}/collector.log 2>/dev/null; then
    break
  fi
  if ! kill -0 \$(cat ${BPF_DIR}/collector.pid) 2>/dev/null; then
    echo 'collector died'; tail -50 ${BPF_DIR}/collector.log; exit 1
  fi
  sleep 0.2
done
date -u +%Y-%m-%dT%H:%M:%SZ > ${BPF_DIR}/collector.ready
echo collector_pid=\$(cat ${BPF_DIR}/collector.pid)
"

BODY='{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}'
LUA="${OUT}/track.lua"
cat > "$LUA" << EOF
wrk.method = "POST"
wrk.body   = [=[${BODY}]=]
wrk.headers["Content-Type"] = "application/json"
wrk.headers["Connection"]   = "keep-alive"
wrk.headers["Accept"]       = "application/json"
EOF

log "raising nofile for 10k connections"
ulimit -n 1048576 2> /dev/null || ulimit -n 65536 || true

METRICS_PORT="${METRICS_PORT:-9101}"
PROM_URL="${PROMETHEUS_URL:-http://127.0.0.1:9190}"
snapshot_tracker_metrics() {
  local dest=$1
  {
    echo "# scraped_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    curl -fsS --max-time 3 "http://127.0.0.1:${METRICS_PORT}/metrics" \
      | rg 'ad_redis_lua_|ad_local_quota_|ad_http_request_duration|ad_events_processed|ad_filter_decisions|ad_redis_ops|ad_filter_lua_slow|ad_redis_lua_noscript|ad_redis_lua_fast|ad_redis_lua_full|ad_redis_lua_skipped' \
      || true
  } > "$dest"
}
log "metrics snapshot before -> ${OUT}/metrics.before.txt"
snapshot_tracker_metrics "${OUT}/metrics.before.txt"

log "wrk -t${BENCH_THREADS} -c${BENCH_CONNECTIONS} -d${BENCH_DURATION} ${TARGET_URL}"
set +e
"${ROOT}/bin/wrk" -t"$BENCH_THREADS" -c"$BENCH_CONNECTIONS" -d"$BENCH_DURATION" \
  --latency -s "$LUA" "$TARGET_URL" | tee "${OUT}/loadgen.log"
WRK_EC=$?
set -e
log "wrk exit=${WRK_EC}"

log "metrics snapshot after -> ${OUT}/metrics.after.txt"
snapshot_tracker_metrics "${OUT}/metrics.after.txt"

python3 - "${OUT}/metrics.after.txt" "${OUT}/lua_quantiles.json" << 'PY' || true
import re, json, sys
from pathlib import Path
text = Path(sys.argv[1]).read_text()

buckets = {}
counts = {}
for line in text.splitlines():
    if line.startswith("#"): continue
    m = re.match(r'ad_redis_lua_duration_seconds_bucket\{shard="(\d+)",le="([^"]+)"\}\s+(\S+)', line)
    if m:
        shard, le, val = m.group(1), m.group(2), float(m.group(3))
        buckets.setdefault(shard, []).append((le, val))
        continue
    m = re.match(r'ad_redis_lua_duration_seconds_count\{shard="(\d+)"\}\s+(\S+)', line)
    if m:
        counts[m.group(1)] = float(m.group(2))
def quantile(cum, q):
    if not cum: return None
    total = cum[-1][1]
    if total <= 0: return None
    target = total * q
    for le, c in cum:
        if le == "+Inf":
            return float("inf")
        if c >= target:
            return float(le)
    return None
out = {}
for shard, rows in sorted(buckets.items()):

    rows_sorted = []
    for le, v in rows:
        if le == "+Inf":
            rows_sorted.append((le, v))
        else:
            rows_sorted.append((le, v))

    nums = [(float(le), v) for le, v in rows if le != "+Inf"]
    nums.sort()
    cum = [(str(le), v) for le, v in nums] + [(le, v) for le, v in rows if le == "+Inf"]
    out[shard] = {
        "count": counts.get(shard, 0),
        "p50_s": quantile(cum, 0.50),
        "p95_s": quantile(cum, 0.95),
        "p99_s": quantile(cum, 0.99),
    }
Path(sys.argv[2]).write_text(json.dumps(out, indent=2))
print(json.dumps(out, indent=2))
PY

sleep 2
cp "${CG_PATH}/cpu.stat" "${OUT}/cpu.stat.after" 2> /dev/null || true
cp "${CG_PATH}/memory.events" "${OUT}/memory.events.after" 2> /dev/null || true
docker inspect -f '{{.State.Status}} OOM={{.State.OOMKilled}} Exit={{.State.ExitCode}}' "$TRACKER_CTR" \
  | tee "${OUT}/tracker.state.txt"

if [[ -f "${BPF_DIR}/collector.pid" ]]; then
  BPID="$(cat "${BPF_DIR}/collector.pid")"
  priv "kill -TERM ${BPID} 2>/dev/null || true
	for i in \$(seq 1 30); do kill -0 ${BPID} 2>/dev/null || break; sleep 0.2; done
	kill -KILL ${BPID} 2>/dev/null || true" || true
fi
sleep 1
if [[ -f "${BPF_DIR}/maps/summary.json" ]]; then
  (cd "$ROOT" && go run ./cmd/load-report bpf "$OUT") > "${OUT}/bpf-report.md" 2> "${OUT}/bpf-report.err" \
    || warn "load-report bpf failed - see ${OUT}/bpf-report.err"
else
  warn "no ${BPF_DIR}/maps/summary.json - check ${BPF_DIR}/collector.log"
  tail -80 "${BPF_DIR}/collector.log" 2> /dev/null | tee "${OUT}/bpf-collector.tail.txt" || true
fi

log "load-report prom -> ${OUT}/bottleneck-report.md (PROM=${PROM_URL})"
(cd "$ROOT" && go run ./cmd/load-report prom "$OUT" --prom "$PROM_URL") \
  > "${OUT}/load-report-prom.log" 2>&1 \
  || warn "load-report prom failed - see ${OUT}/load-report-prom.log"

(cd "$ROOT" && LOAD_SLA_GATE=0 go run ./cmd/load-report all "$OUT" --prom "$PROM_URL") \
  > "${OUT}/load-report-all.log" 2>&1 \
  || warn "load-report all soft-failed - see ${OUT}/load-report-all.log"

{
  echo "Purgatory + eBPF Report (${TS})"
  echo "phase: ${PHASE} memory=${MEM_LABEL} cpus=${CPUS} GOMAXPROCS=${GOMAXPROCS_PIN}"
  echo "url: ${TARGET_URL}"
  echo "tracker: ${TRACKER_CTR} pid=${TRACKER_PID}"
  echo
  echo "tracker state"
  cat "${OUT}/tracker.state.txt" 2> /dev/null || true
  echo "GOMAXPROCS=$(cat "${OUT}/gomaxprocs.txt" 2> /dev/null || echo '?')"
  echo
  echo "cpu.stat before"
  cat "${OUT}/cpu.stat.before" 2> /dev/null || true
  echo "cpu.stat after"
  cat "${OUT}/cpu.stat.after" 2> /dev/null || true
  if [[ -n "${CG_PATH:-}" && -f "${CG_PATH}/cpu.stat" ]]; then
    cp "${CG_PATH}/cpu.stat" "${OUT}/cpu.stat.after.docker" 2> /dev/null || true
    cp "${CG_PATH}/memory.events" "${OUT}/memory.events.after.docker" 2> /dev/null || true
    cp "${CG_PATH}/memory.current" "${OUT}/memory.current.after.docker" 2> /dev/null || true
  fi
  echo
  echo "memory.events after"
  cat "${OUT}/memory.events.after" 2> /dev/null || true
  echo
  echo "Lua quantiles (tracker :${METRICS_PORT} histogram)"
  cat "${OUT}/lua_quantiles.json" 2> /dev/null || echo "n/a"
  echo
  echo "Prometheus bottleneck (excerpt)"
  if [[ -f "${OUT}/bottleneck-report.md" ]]; then
    head -n 80 "${OUT}/bottleneck-report.md"
  else
    echo "missing bottleneck-report.md"
  fi
  echo
  echo "wrk"
  cat "${OUT}/loadgen.log" 2> /dev/null || true
} | tee "${OUT}/REPORT.txt"

log "done -> ${OUT}/REPORT.txt"
