#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/load_test_env.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env" 2> /dev/null || printf 'load-malformed: WARN: .env present but not sourced (parse error)\n'
  set +a
fi

MODE="${1:-full}"
CONSTRAINED="${CONSTRAINED:-1}"
RATE="${RATE:-}"
DURATION="${DURATION:-}"
PREPARE="${PREPARE:-0}"
PCT_BROKEN="${PCT_BROKEN:-}"
PCT_GRAY="${PCT_GRAY:-}"
PCT_CLICK_PROXY="${PCT_CLICK_PROXY:-}"
PCT_PROXY_VPN="${PCT_PROXY_VPN:-}"
PCT_FLOW_ROUTE="${PCT_FLOW_ROUTE:-}"
OUT="$ROOT/var/load-test/$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUT"

if [[ "$CONSTRAINED" == "1" ]]; then
  load_test_compose COMPOSE "$ROOT"
else
  COMPOSE=(docker compose)
fi

LOAD_SERVICES=(
  db redis-0 redis-1 redis-2 redis-3 redis-4 redis-5 clickhouse
  processor tracker-0 tracker-1 nginx prometheus grafana
)

log() { printf 'load-malformed: %s\n' "$*"; }
die() {
  printf 'load-malformed: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "$(id -u)" -eq 0 && "${AD_EVENT_PROCESSOR_LOAD_MALFORMED_ROOT_OK:-0}" != "1" ]]; then
  die "do not run as root/sudo — bpf_probe_session calls sudo internally; root breaks 'docker compose -f' (use: AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/test/malformed.sh smoke)"
fi

case "$MODE" in
  smoke | business | full) ;;
  *) die "unknown mode: $MODE (use smoke|business|full)" ;;
esac

if [[ "$CONSTRAINED" == "1" ]]; then
  TRACKER_BASES="${TRACKER_BASES:-$LOAD_TEST_CONSTRAINED_TRACKER_BASES_CSV}"
  OVERSIZE_BYTES="${OVERSIZE_BYTES:-65536}"
else
  TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
  OVERSIZE_BYTES="${OVERSIZE_BYTES:-2097152}"
fi

if [[ "$PREPARE" == "1" || "$MODE" == "business" ]]; then
  PREPARE=1
fi

if [[ "$PREPARE" == "1" ]]; then
  if [[ "$CONSTRAINED" == "1" ]]; then
    log "preparing constrained stack"
    bash "$SCRIPTS/test/prepare_constrained_stack.sh" 2>&1 | tee -a "$OUT/compose.log"
  else
    export RATE_LIMIT_PER_MIN=1000000
    bash "$SCRIPTS/ci/prepare_test.sh"
  fi
else
  if [[ "$CONSTRAINED" == "1" ]]; then
    export SKIP_CODEGEN="${SKIP_CODEGEN:-1}"
    log "bringing up constrained stack (rebuild trackers for TCP health probes)"
    "${COMPOSE[@]}" up -d --build --force-recreate --remove-orphans \
      db redis-0 redis-1 redis-2 redis-3 redis-4 redis-5 clickhouse \
      processor tracker-0 tracker-1 prometheus grafana 2>&1 | tee -a "$OUT/compose.log"
    while IFS= read -r port; do
      for _ in $(seq 1 90); do
        if curl -sf "http://127.0.0.1:${port}/health" > /dev/null 2>&1; then
          break
        fi
        sleep 1
      done
    done < <(load_test_constrained_ingest_ports)
    "${COMPOSE[@]}" stop tracker-2 tracker-3 2> /dev/null || true
    "${COMPOSE[@]}" rm -f tracker-2 tracker-3 2> /dev/null || true
    bash "$SCRIPTS/test/seed_load_test_limits.sh" || log "WARN: seed limits failed"
    "${COMPOSE[@]}" up -d --no-deps nginx 2>&1 | tee -a "$OUT/compose.log" || true
  else
    "${COMPOSE[@]}" up -d --remove-orphans "${LOAD_SERVICES[@]}" 2>&1 | tee -a "$OUT/compose.log"
  fi
fi

{
  echo "constrained=$CONSTRAINED mode=$MODE"
  echo "trackers=$TRACKER_BASES"
} > "$OUT/profile.txt"

SNAP_PID=""
[[ "${SNAPSHOT_RUNTIME:-1}" == "1" ]] && bash "$SCRIPTS/test/snapshot_runtime.sh" "$OUT/runtime-pre" 5 &
SNAP_PID=$!

BPF_PID=""
if [[ "${AD_EVENT_PROCESSOR_BPF_PROBE:-0}" == "1" ]]; then
  bash "$SCRIPTS/test/bpf_probe_session.sh" start "$OUT" || log "WARN: BPF start failed"
  [[ -f "$OUT/bpf/collector.pid" ]] && BPF_PID="$(cat "$OUT/bpf/collector.pid")"
fi

LG_ARGS=(-mode "$MODE" -out "$OUT" -trackers "$TRACKER_BASES" -edge "${EDGE_URL:-$LOAD_TEST_EDGE_URL}" -oversize-bytes "$OVERSIZE_BYTES")
[[ -n "$RATE" ]] && LG_ARGS+=(-rate "$RATE")
[[ -n "$DURATION" ]] && LG_ARGS+=(-duration "$DURATION")
[[ -n "$PCT_BROKEN" ]] && LG_ARGS+=(-pct-broken "$PCT_BROKEN")
[[ -n "$PCT_GRAY" ]] && LG_ARGS+=(-pct-gray "$PCT_GRAY")
[[ -n "$PCT_CLICK_PROXY" ]] && LG_ARGS+=(-pct-click-proxy "$PCT_CLICK_PROXY")
[[ -n "$PCT_PROXY_VPN" ]] && LG_ARGS+=(-pct-proxy-vpn "$PCT_PROXY_VPN")
[[ -n "$PCT_FLOW_ROUTE" ]] && LG_ARGS+=(-pct-flow-route "$PCT_FLOW_ROUTE")

log "starting loadgen (${MODE})"
go run ./cmd/loadgen "${LG_ARGS[@]}" 2>&1 | tee "$OUT/loadgen.log"

[[ -n "$SNAP_PID" ]] && wait "$SNAP_PID" 2> /dev/null || true
[[ -n "$BPF_PID" ]] && bash "$SCRIPTS/test/bpf_probe_session.sh" stop "$OUT" "$BPF_PID" || true
bash "$SCRIPTS/test/snapshot_runtime.sh" "$OUT/runtime-post" 10

export LOAD_SLA_GATE=1
go run ./cmd/load-report all "$OUT"
if [[ "${LOAD_TG_GATE:-0}" == "1" ]]; then
  go run ./cmd/load-report telegram "$OUT" --prom "${PROMETHEUS_URL:-$LOAD_TEST_PROMETHEUS_URL}"
fi
log "done — $OUT"
