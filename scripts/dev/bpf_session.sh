#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/bpf_collector.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env" 2> /dev/null || printf 'bpf-session: WARN: .env present but not sourced (parse error)\n'
  set +a
fi

CMD="${1:-status}"
SESSION_ROOT="${AD_EVENT_PROCESSOR_BPF_SESSION_ROOT:-$ROOT/var/bpf-session}"
CURRENT_LINK="$SESSION_ROOT/current"
CURRENT_PATH_FILE="$SESSION_ROOT/current_path"

log() { printf 'bpf-session: %s\n' "$*"; }

resolve_out_dir() {
  local explicit="${1:-}"
  if [[ -n "$explicit" ]]; then
    printf '%s' "$explicit"
    return
  fi
  if [[ -f "$CURRENT_PATH_FILE" ]]; then
    cat "$CURRENT_PATH_FILE"
    return
  fi
  printf '%s/%s' "$SESSION_ROOT" "$(date -u +%Y%m%dT%H%M%SZ)"
}

case "$CMD" in
  start)
    OUT="$(resolve_out_dir "${2:-}")"
    if [[ "$OUT" != */* ]]; then
      OUT="$SESSION_ROOT/$OUT"
    fi
    mkdir -p "$OUT"
    export AD_EVENT_PROCESSOR_BPF_NATIVE="${AD_EVENT_PROCESSOR_BPF_NATIVE:-0}"
    export AD_EVENT_PROCESSOR_BPF_TRACK_LOADGEN="${AD_EVENT_PROCESSOR_BPF_TRACK_LOADGEN:-1}"
    if [[ -f "$ROOT/.env.load-test" ]]; then
      # shellcheck source=scripts/lib/load_test_env.sh
      source "$SCRIPTS/lib/load_test_env.sh"
      load_test_bootstrap "$ROOT" 2> /dev/null || true
      export PROMETHEUS_URL="${PROMETHEUS_URL:-${LOAD_TEST_PROMETHEUS_URL:-}}"
    fi
    export AD_EVENT_PROCESSOR_BPF_DUMP_INTERVAL="${AD_EVENT_PROCESSOR_BPF_DUMP_INTERVAL:-30}"
    export AD_EVENT_PROCESSOR_BPF_REFRESH_TARGETS="${AD_EVENT_PROCESSOR_BPF_REFRESH_TARGETS:-30}"
    export AD_EVENT_PROCESSOR_BPF_METRICS_ADDR="${AD_EVENT_PROCESSOR_BPF_METRICS_ADDR:-:9464}"
    bash "$SCRIPTS/test/bpf_probe_session.sh" start "$OUT"
    if [[ ! -f "$OUT/bpf/collector.ready" ]]; then
      log "ERROR: bpf collector did not start - see $OUT/bpf/collector.log"
      exit 1
    fi
    mkdir -p "$SESSION_ROOT"
    ln -sfn "$(basename "$OUT")" "$CURRENT_LINK"
    printf '%s\n' "$OUT" > "$CURRENT_PATH_FILE"
    log "session started: $OUT"
    log "metrics: ${AD_EVENT_PROCESSOR_BPF_METRICS_ADDR} (/metrics)"
    log "load: PREPARE=1 AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/test/malformed.sh business"
    log "stop: bash scripts/dev/bpf_session.sh stop"
    ;;
  stop)
    OUT="$(resolve_out_dir "${2:-}")"
    PID=""
    if [[ -f "$OUT/bpf/collector.pid" ]]; then
      PID="$(cat "$OUT/bpf/collector.pid")"
    fi
    bash "$SCRIPTS/test/bpf_probe_session.sh" stop "$OUT" "$PID"
    log "session stopped: $OUT"
    ;;
  status)
    if [[ -f "$CURRENT_PATH_FILE" ]]; then
      OUT="$(cat "$CURRENT_PATH_FILE")"
      log "current session: $OUT"
      if [[ -f "$OUT/bpf/collector.pid" ]]; then
        PID="$(cat "$OUT/bpf/collector.pid")"
        if bpf_collector_pid_alive "$PID" && [[ -f "$OUT/bpf/collector.ready" ]]; then
          log "collector: running pid=$PID"
        elif bpf_collector_pid_alive "$PID"; then
          log "collector: pid=$PID (not ready - see $OUT/bpf/collector.log)"
        else
          log "collector: dead (stale pid=$PID - see $OUT/bpf/collector.log)"
        fi
      else
        log "collector: not running"
      fi
    else
      log "no active session (run: sudo bash scripts/dev/bpf_session.sh start)"
    fi
    ;;
  report)
    OUT="$(resolve_out_dir "${2:-}")"
    source "$SCRIPTS/lib/go.sh"
    if ! ad_event_processor_go_run ./cmd/load-report bpf "$OUT"; then
      log "ERROR: report failed (go missing or no bpf/maps/summary.json)"
      exit 1
    fi
    log "report: $OUT/bpf-report.md"
    ;;
  *)
    printf 'usage: %s start|stop|status|report [session_dir]\n' "$0" >&2
    exit 2
    ;;
esac
