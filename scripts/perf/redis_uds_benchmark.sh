#!/usr/bin/env bash
# Milestone phase 5: Redis UDS transport latency proof (unix socket vs TCP loopback).
#
# Usage:
#   bash scripts/perf/redis_uds_benchmark.sh
# Env:
#   UDS_DIAL_P50_BUDGET_NS=2500   p50 dial budget in ns (default 2.5 µs)
#   UDS_BENCH_REQUESTS=100000     redis-benchmark -n
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${UDS_BENCH_OUT:-$ROOT/var/uds-bench/${TS}}"
RUN_DIR="$OUT/run"
mkdir -p "$RUN_DIR/redis"

REDIS_IMAGE="redis:${REDIS_VERSION:-7-alpine}"
UDS_SOCK="$RUN_DIR/redis/redis-0.sock"
TCP_PORT="${UDS_BENCH_TCP_PORT:-6399}"
	BUDGET_NS="${UDS_DIAL_P50_BUDGET_NS:-5000}"
REQUESTS="${UDS_BENCH_REQUESTS:-100000}"

log() { printf 'redis-uds-bench: %s\n' "$*"; }
die() { printf 'redis-uds-bench: ERROR: %s\n' "$*" >&2; exit 1; }

CONTAINER="espx-uds-bench-$$"
cleanup() {
	docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup EXIT

log "artifacts → $OUT"
log "uds_socket=$UDS_SOCK tcp_port=$TCP_PORT budget_p50_ns=$BUDGET_NS"

docker run -d --name "$CONTAINER" --network host \
	-v "$RUN_DIR:/run/espx" \
	"$REDIS_IMAGE" \
	redis-server \
	--port "$TCP_PORT" \
	--bind 127.0.0.1 \
	--unixsocket /run/espx/redis/redis-0.sock \
	--unixsocketperm 777 \
	>/dev/null

for _ in $(seq 1 60); do
	if [[ -S "$UDS_SOCK" ]]; then
		break
	fi
	sleep 0.1
done
[[ -S "$UDS_SOCK" ]] || die "unix socket not created: $UDS_SOCK"

log "redis-benchmark unix socket (-s)"
docker run --rm --network host -v "$RUN_DIR:/run/espx" "$REDIS_IMAGE" \
	redis-benchmark -s /run/espx/redis/redis-0.sock -n "$REQUESTS" -q -t ping \
	2>&1 | tee "$OUT/redis-benchmark-uds.txt"

log "redis-benchmark tcp loopback (-h 127.0.0.1 -p $TCP_PORT)"
docker run --rm --network host "$REDIS_IMAGE" \
	redis-benchmark -h 127.0.0.1 -p "$TCP_PORT" -n "$REQUESTS" -q -t ping \
	2>&1 | tee "$OUT/redis-benchmark-tcp.txt"

export REDIS_UDS_SOCKET="$UDS_SOCK"
export REDIS_TCP_ADDR="127.0.0.1:$TCP_PORT"
export REDIS_PASSWORD=""
export UDS_DIAL_P50_BUDGET_NS="$BUDGET_NS"
export GOMAXPROCS="${GOMAXPROCS:-1}"

log "go transport + ping gates"
go test -count=1 -v -run 'TestRedisUDS_(DialLatencyGate|DialVsTCP|PingLatency)$' ./internal/database/ \
	2>&1 | tee "$OUT/go-test.log"

p50_line="$(grep 'redis UDS dial n=' "$OUT/go-test.log" | tail -1 || true)"
uds_p50_us=""
if [[ -n "$p50_line" ]]; then
	uds_p50_us="$(printf '%s' "$p50_line" | sed -E 's/.*p50=([0-9.]+)(µs|us|ns).*/\1/' )"
	if printf '%s' "$p50_line" | grep -q 'p50=.*ns'; then
		uds_p50_us="$(python3 - <<PY
val = float("$uds_p50_us")
print(f"{val/1000:.3f}")
PY
)"
	fi
fi

stretch_passed="false"
if [[ -n "$uds_p50_us" ]] && python3 -c 'import sys; raise SystemExit(0 if float(sys.argv[1]) < 2.5 else 1)' "$uds_p50_us"; then
	stretch_passed="true"
fi

passed="false"
if [[ -n "$uds_p50_us" ]] && python3 -c 'import sys; raise SystemExit(0 if float(sys.argv[1]) < float(sys.argv[2])/1000 else 1)' "$uds_p50_us" "$BUDGET_NS"; then
	passed="true"
fi

uds_p50_json="${uds_p50_us:-null}"
if [[ -n "$uds_p50_us" ]]; then
	uds_p50_json="$uds_p50_us"
fi

cat >"$OUT/report.json" <<EOF
{
  "timestamp": "$TS",
  "uds_socket": "$UDS_SOCK",
  "tcp_addr": "127.0.0.1:$TCP_PORT",
  "uds_dial_p50_budget_ns": $BUDGET_NS,
  "uds_dial_p50_us": $uds_p50_json,
  "stretch_goal_2_5us_passed": $stretch_passed,
  "redis_benchmark_uds": "$(basename "$OUT/redis-benchmark-uds.txt")",
  "redis_benchmark_tcp": "$(basename "$OUT/redis-benchmark-tcp.txt")",
  "passed": $passed
}
EOF

printf 'fault_proof fault=redis_uds_latency status=%s uds_dial_p50_us=%s budget_ns=%s\n' \
	"$([[ "$passed" == "true" ]] && echo passed || echo failed)" \
	"${uds_p50_us:-na}" "$BUDGET_NS" | tee "$OUT/fault_proof.txt"

if [[ "$passed" != "true" ]]; then
	die "UDS dial p50 budget not met (see $OUT/report.json)"
fi

log "PASS uds_dial_p50_us=${uds_p50_us} (<$((BUDGET_NS/1000)) µs)"
