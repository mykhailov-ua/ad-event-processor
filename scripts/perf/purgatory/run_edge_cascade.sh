#!/usr/bin/env bash
# Role: Edge cascade failure drill under purgatory cgroup with multi-hop load and SLA checks.
# Execution context: Compose stack with trackers; writes under var/purgatory/edge-*.
# Env knobs: RUN_DIR; TRACKER_BASES; cascade step timeouts from script env.
# Verify: sudo bash scripts/perf/purgatory/run_edge_cascade.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

export PATH="${ROOT}/bin:${PATH}"

if [[ -f "${ROOT}/.env" ]]; then
  set -a

  source "${ROOT}/.env"
  set +a
fi

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${RUN_DIR}/edge-${TS}"
mkdir -p "$OUT" "$STATE_DIR"
log "edge-cascade artifacts -> ${OUT}"

TRACKER_CTR="${TRACKER_CTR:-ad-event-processor-tracker-0-1}"
TARGET_URL="${TARGET_URL:-http://127.0.0.1:8181/track}"
BENCH_DURATION="${BENCH_DURATION:-60s}"
BENCH_CONNECTIONS="${BENCH_CONNECTIONS:-10000}"
BENCH_THREADS="${BENCH_THREADS:-4}"
BURST_CONNECTIONS="${BURST_CONNECTIONS:-4000}"
BURST_DURATION="${BURST_DURATION:-12s}"
BURST_AT_SEC="${BURST_AT_SEC:-15}"
FLUSH_AT_SEC="${FLUSH_AT_SEC:-18}"
REDIS_SHARD_CTR="${REDIS_SHARD_CTR:-ad-event-processor-redis-1-1}"
REDIS_PASS="${REDIS_PASSWORD:-your_redis_password_here}"

NETEM_DELAY="${EDGE_NETEM_DELAY:-5ms}"
NETEM_LOSS="${EDGE_NETEM_LOSS:-0.3%}"
NETEM_DUPLICATE="${EDGE_NETEM_DUPLICATE:-0%}"
CPUS="${CPUS:-1.0}"
MEM_BYTES="${MEM_BYTES:-536870912}"
GOMAXPROCS_PIN="${GOMAXPROCS_PIN:-1}"
FILTER_TIMEOUT_MS="${EDGE_FILTER_TIMEOUT_MS:-100}"
CPU_MAX_QUOTA="$(awk -v c="$CPUS" 'BEGIN { printf "%d", c * 100000 + 0.5 }')"

require_cmd docker taskset curl python3
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

snapshot_listen() {
  local dest=$1
  {
    echo "# at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

    nstat -az 2> /dev/null | rg 'ListenOverflows|ListenDrops|TCPBacklogDrop|SyncookiesFailed|TCPFastOpenListenOverflow' || true
    ss -ltn 2> /dev/null | rg ':8181|:8180' || true
  } > "$dest"
}

cleanup() {
  local ec=$?
  log "cleanup (exit=${ec})"
  if [[ -f "${OUT}/stress-ng.pid" ]]; then
    kill "$(cat "${OUT}/stress-ng.pid")" 2> /dev/null || true
  fi
  pkill -f 'stress-ng --cache' 2> /dev/null || true
  if [[ -f "${OUT}/bpf/collector.pid" ]]; then
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

OVERRIDES="${OUT}/compose.edge.yaml"
cat > "$OVERRIDES" << EOF
services:
  tracker-0:
    environment:
      GOMAXPROCS: "${GOMAXPROCS_PIN}"
      GOMEMLIMIT: "400MiB"
      GOGC: "300"
      LOCAL_QUOTA_MODE: "off"
      FILTER_TIMEOUT_MS: "${FILTER_TIMEOUT_MS}"
      METRICS_HISTOGRAM_SAMPLE_MASK: "-1"
EOF

COMPOSE=(docker compose -f "${ROOT}/docker-compose.yaml" -f "${ROOT}/docker-compose.load-test.yaml" -f "$OVERRIDES")

log "netem delay=${NETEM_DELAY} loss=${NETEM_LOSS} dup=${NETEM_DUPLICATE}; somaxconn=128"
priv "tc qdisc del dev lo root 2>/dev/null || true
tc qdisc add dev lo root netem delay ${NETEM_DELAY} loss ${NETEM_LOSS} duplicate ${NETEM_DUPLICATE}
sysctl -w net.core.rmem_max=65536
sysctl -w net.core.wmem_max=65536
sysctl -w net.ipv4.tcp_rmem='4096 87380 65536'
sysctl -w net.ipv4.tcp_wmem='4096 16384 65536'
sysctl -w net.ipv4.tcp_max_syn_backlog=128
sysctl -w net.core.somaxconn=128
tc qdisc show dev lo"

log "recreating ${TRACKER_CTR} after somaxconn (listen backlog picks up 128)"
"${COMPOSE[@]}" up -d --no-build --force-recreate --no-deps tracker-0
for _ in $(seq 1 90); do
  if docker inspect -f '{{.State.Health.Status}}' "$TRACKER_CTR" 2> /dev/null | grep -q healthy; then
    break
  fi
  sleep 1
done
bash "${SCRIPTS}/test/sync_tracker_registry.sh" > /dev/null 2>&1 || warn "registry sync skipped"

log "limits cpus=${CPUS} mem=${MEM_BYTES}"
docker update --cpus="$CPUS" --memory="${MEM_BYTES}" --memory-swap="${MEM_BYTES}" "$TRACKER_CTR"
sleep 1
docker inspect -f '{{.State.Running}}' "$TRACKER_CTR" | grep -q true || die "tracker dead after limits"

TRACKER_PID="$(docker inspect -f '{{.State.Pid}}' "$TRACKER_CTR")"
echo "${TRACKER_PID}" > "${OUT}/tracker.pid"
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
echo "${CG_PATH:-}" > "${OUT}/tracker.cgroup.path"
[[ -n "$CG_PATH" ]] && cp "${CG_PATH}/cpu.stat" "${OUT}/cpu.stat.before" || true
ss -ltn | rg ':8181' | tee "${OUT}/listen.bind.txt" || true

log "stress-ng L3 on CPU 1"
taskset -c 1 "${ROOT}/bin/stress-ng" --cache 1 --cache-level 3 --aggressive --timeout 0 \
  > "${OUT}/stress-ng.log" 2>&1 &
echo $! > "${OUT}/stress-ng.pid"
sleep 0.5
kill -0 "$(cat "${OUT}/stress-ng.pid")" 2> /dev/null || {
  taskset -c 1 "${ROOT}/bin/stress-ng" --cache 1 --timeout 0 > "${OUT}/stress-ng.log" 2>&1 &
  echo $! > "${OUT}/stress-ng.pid"
}

BPF_DIR="${OUT}/bpf"
mkdir -p "$BPF_DIR"
AD_EVENT_PROCESSOR_BPF_NATIVE=1 AD_EVENT_PROCESSOR_BPF_TRACK_LOADGEN=1 AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM=wrk \
  bash "${SCRIPTS}/test/bpf_resolve_targets.sh" "${BPF_DIR}/targets.json" tracker,nginx,redis,processor

log "bpf-collector start"
priv "ulimit -l unlimited
cd ${ROOT}
nohup ./bin/bpf-collector \
  -session-dir ${BPF_DIR} \
  -bpf-object ${ROOT}/deploy/dev/bpf/loadtest_probe.o \
  -sample-rate ${AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE:-10} \
  -slow-us ${AD_EVENT_PROCESSOR_BPF_SLOW_US:-10000} \
  -discover-loadgen=true \
  -loadgen-comms wrk \
  -dump-interval 10s \
  >${BPF_DIR}/collector.log 2>&1 &
echo \$! > ${BPF_DIR}/collector.pid
for i in \$(seq 1 50); do
  if grep -q 'bpf-collector running\\|maps ready\\|attached' ${BPF_DIR}/collector.log 2>/dev/null; then
    break
  fi
  if ! kill -0 \$(cat ${BPF_DIR}/collector.pid) 2>/dev/null; then
    echo 'collector died'; tail -80 ${BPF_DIR}/collector.log; exit 1
  fi
  sleep 0.2
done
date -u +%Y-%m-%dT%H:%M:%SZ > ${BPF_DIR}/collector.ready
"

BODY='{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}'
LUA_KA="${OUT}/track_ka.lua"
LUA_CLOSE="${OUT}/track_close.lua"
cat > "$LUA_KA" << EOF
wrk.method = "POST"
wrk.body   = [=[${BODY}]=]
wrk.headers["Content-Type"] = "application/json"
wrk.headers["Connection"]   = "keep-alive"
wrk.headers["Accept"]       = "application/json"
EOF
cat > "$LUA_CLOSE" << EOF
wrk.method = "POST"
wrk.body   = [=[${BODY}]=]
wrk.headers["Content-Type"] = "application/json"
wrk.headers["Connection"]   = "close"
wrk.headers["Accept"]       = "application/json"
EOF

ulimit -n 1048576 2> /dev/null || ulimit -n 65536 || true
METRICS_PORT="${METRICS_PORT:-9101}"
PROM_URL="${PROMETHEUS_URL:-http://127.0.0.1:9190}"

snapshot_tracker_metrics() {
  local dest=$1
  {
    echo "# scraped_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    curl -fsS --max-time 3 "http://127.0.0.1:${METRICS_PORT}/metrics" \
      | rg 'ad_redis_lua_|ad_http_request_duration|ad_events_processed|ad_filter_decisions|ad_redis_lua_noscript|ad_redis_breaker|ad_tracker_health|ad_filter_timeout|ad_filter_reject|ad_worker_pool' \
      || true
  } > "$dest"
}

snapshot_listen "${OUT}/listen.before.txt"
snapshot_tracker_metrics "${OUT}/metrics.before.txt"

log "primary wrk KA -t${BENCH_THREADS} -c${BENCH_CONNECTIONS} -d${BENCH_DURATION}"
set +e
"${ROOT}/bin/wrk" -t"$BENCH_THREADS" -c"$BENCH_CONNECTIONS" -d"$BENCH_DURATION" \
  --latency -s "$LUA_KA" "$TARGET_URL" > "${OUT}/loadgen.log" 2>&1 &
WRK_PID=$!
set -e

(
  sleep "$BURST_AT_SEC"
  date -u +%Y-%m-%dT%H:%M:%SZ > "${OUT}/burst.at"
  log "BURST Connection:close -c${BURST_CONNECTIONS} -d${BURST_DURATION}"
  set +e
  "${ROOT}/bin/wrk" -t2 -c"$BURST_CONNECTIONS" -d"$BURST_DURATION" \
    --latency -s "$LUA_CLOSE" "$TARGET_URL" | tee "${OUT}/burst.log"
  set -e
) &
BURST_BG=$!

(
  sleep "$FLUSH_AT_SEC"
  date -u +%Y-%m-%dT%H:%M:%SZ > "${OUT}/flush.at"
  log "SCRIPT FLUSH on ${REDIS_SHARD_CTR}"
  {
    echo "# flush_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "# redis_ctr=${REDIS_SHARD_CTR}"
    if out="$(docker exec "$REDIS_SHARD_CTR" redis-cli -a "$REDIS_PASS" --no-auth-warning SCRIPT FLUSH SYNC 2>&1)"; then
      echo "flush_out=${out}"
      echo "flush_ec=0"
    else
      echo "flush_out=${out}"
      echo "flush_ec=$?"
    fi

    if dout="$(docker exec "$REDIS_SHARD_CTR" redis-cli -a "$REDIS_PASS" --no-auth-warning DEBUG SLEEP 0.05 2>&1)" \
      && ! grep -qiE 'ERR|error' <<< "$dout"; then
      echo "debug_sleep=ok out=${dout}"
    else
      echo "debug_sleep=skipped out=${dout}"
    fi
  } | tee "${OUT}/redis.flush.log"
  snapshot_listen "${OUT}/listen.after_flush.txt"
  snapshot_tracker_metrics "${OUT}/metrics.after_flush.txt"
) &
FLUSH_BG=$!

wait "$WRK_PID"
WRK_EC=$?
log "primary wrk exit=${WRK_EC}"
wait "$BURST_BG" 2> /dev/null || true
wait "$FLUSH_BG" 2> /dev/null || true

snapshot_listen "${OUT}/listen.after.txt"
snapshot_tracker_metrics "${OUT}/metrics.after.txt"

python3 - "${OUT}/metrics.after.txt" "${OUT}/lua_quantiles.json" << 'PY' || true
import re, json, sys
from pathlib import Path
text = Path(sys.argv[1]).read_text()
buckets, counts = {}, {}
for line in text.splitlines():
    if line.startswith("#"): continue
    m = re.match(r'ad_redis_lua_duration_seconds_bucket\{shard="(\d+)",le="([^"]+)"\}\s+(\S+)', line)
    if m:
        buckets.setdefault(m.group(1), []).append((m.group(2), float(m.group(3)))); continue
    m = re.match(r'ad_redis_lua_duration_seconds_count\{shard="(\d+)"\}\s+(\S+)', line)
    if m:
        counts[m.group(1)] = float(m.group(2))
def quantile(cum, q):
    if not cum: return None
    total = cum[-1][1]
    if total <= 0: return None
    target = total * q
    for le, c in cum:
        if le == "+Inf": return float("inf")
        if c >= target: return float(le)
    return None
out = {}
for shard, rows in sorted(buckets.items()):
    nums = sorted((float(le), v) for le, v in rows if le != "+Inf")
    cum = [(str(le), v) for le, v in nums] + [(le, v) for le, v in rows if le == "+Inf"]
    out[shard] = {"count": counts.get(shard, 0), "p50_s": quantile(cum, 0.5), "p95_s": quantile(cum, 0.95), "p99_s": quantile(cum, 0.99)}
Path(sys.argv[2]).write_text(json.dumps(out, indent=2))
print(json.dumps(out, indent=2))
PY

python3 - "${OUT}/metrics.before.txt" "${OUT}/metrics.after.txt" "${OUT}/noscript.delta.txt" << 'PY' || true
import re, sys
from pathlib import Path
def noscript(path):
    total = 0.0
    for line in Path(path).read_text().splitlines():
        if line.startswith("ad_redis_lua_noscript_total"):
            total += float(line.split()[-1])
    return total
b, a = noscript(sys.argv[1]), noscript(sys.argv[2])
Path(sys.argv[3]).write_text(f"before={b}\nafter={a}\ndelta={a-b}\n")
print(f"noscript delta={a-b}")
PY

[[ -n "$CG_PATH" ]] && cp "${CG_PATH}/cpu.stat" "${OUT}/cpu.stat.after" || true
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
    || warn "load-report bpf failed"
else
  warn "no BPF summary - see ${BPF_DIR}/collector.log"
  tail -80 "${BPF_DIR}/collector.log" | tee "${OUT}/bpf-collector.tail.txt" || true
fi
(cd "$ROOT" && go run ./cmd/load-report prom "$OUT" --prom "$PROM_URL") \
  > "${OUT}/load-report-prom.log" 2>&1 || warn "load-report prom failed"
(cd "$ROOT" && LOAD_SLA_GATE=0 go run ./cmd/load-report all "$OUT" --prom "$PROM_URL") \
  > "${OUT}/load-report-all.log" 2>&1 || true

{
  echo "Edge Cascade + eBPF (${TS})"
  echo "netem: delay=${NETEM_DELAY} loss=${NETEM_LOSS} dup=${NETEM_DUPLICATE}"
  echo "FILTER_TIMEOUT_MS=${FILTER_TIMEOUT_MS} LOCAL_QUOTA=off GOMAXPROCS=${GOMAXPROCS_PIN} cpus=${CPUS}"
  echo "inject: burst@${BURST_AT_SEC}s close-c=${BURST_CONNECTIONS}; SCRIPT FLUSH@${FLUSH_AT_SEC}s on ${REDIS_SHARD_CTR}"
  echo
  echo "tracker"
  cat "${OUT}/tracker.state.txt" 2> /dev/null || true
  echo
  echo "noscript"
  cat "${OUT}/noscript.delta.txt" 2> /dev/null || true
  echo
  echo "listen counters"
  echo "BEFORE:"
  cat "${OUT}/listen.before.txt" 2> /dev/null || true
  echo "AFTER_FLUSH:"
  cat "${OUT}/listen.after_flush.txt" 2> /dev/null || true
  echo "AFTER:"
  cat "${OUT}/listen.after.txt" 2> /dev/null || true
  echo
  echo "redis flush"
  cat "${OUT}/redis.flush.log" 2> /dev/null || true
  echo
  echo "Lua quantiles"
  cat "${OUT}/lua_quantiles.json" 2> /dev/null || true
  echo
  echo "cpu.stat"
  echo BEFORE
  cat "${OUT}/cpu.stat.before" 2> /dev/null || true
  echo AFTER
  cat "${OUT}/cpu.stat.after" 2> /dev/null || true
  echo
  echo "bottleneck excerpt"
  head -n 80 "${OUT}/bottleneck-report.md" 2> /dev/null || echo missing
  echo
  echo "primary wrk"
  cat "${OUT}/loadgen.log" 2> /dev/null || true
  echo
  echo "burst wrk"
  cat "${OUT}/burst.log" 2> /dev/null || true
} | tee "${OUT}/REPORT.txt"

log "done -> ${OUT}/REPORT.txt"
echo "$OUT"
