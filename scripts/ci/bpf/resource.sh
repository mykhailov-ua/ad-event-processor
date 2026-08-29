#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/bpf_collector.sh"
source "$SCRIPTS/lib/load_test_env.sh"
cd "$ROOT"

load_test_source_env "$ROOT" 2> /dev/null || true
load_test_export_derived 2> /dev/null || true

log() { printf 'bpf-resource-gate: %s\n' "$*"; }

STRICT="${BPF_GATE_STRICT:-false}"
if [[ "${PERF_RUNNER_LABEL:-}" != "" ]]; then
  STRICT="true"
fi

if [[ "$STRICT" != "true" ]]; then
  log "SKIPPED (github-hosted): set repo variable PERF_RUNNER_LABEL for strict eBPF gate"
  log "hint: BPF_GATE_STRICT=true bash scripts/ci/bpf/resource.sh"
  exit 0
fi

export BPF_GATE_STRICT=true
export AD_EVENT_PROCESSOR_BPF_PROBE=1
export AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE="${AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE:-10}"
export AD_EVENT_PROCESSOR_BPF_SLOW_US="${AD_EVENT_PROCESSOR_BPF_SLOW_US:-10000}"
export AD_EVENT_PROCESSOR_BPF_DUMP_INTERVAL="${AD_EVENT_PROCESSOR_BPF_DUMP_INTERVAL:-30}"
export AD_EVENT_PROCESSOR_BPF_REFRESH_TARGETS="${AD_EVENT_PROCESSOR_BPF_REFRESH_TARGETS:-30}"
export LOAD_BPF_GATE=1
export PROMETHEUS_URL="${PROMETHEUS_URL:-${LOAD_TEST_PROMETHEUS_URL:-http://127.0.0.1:9190}}"

MODE="${BPF_GATE_MODE:-smoke}"
GATE_LOG="${GATE_LOG:-$CI_ARTIFACT_DIR/bpf_resource_gate.log}"

if [[ -n "${BPF_GATE_DURATION:-}" ]]; then
  export DURATION="$BPF_GATE_DURATION"
fi

log "strict mode: eBPF resource gate (mode=$MODE)"

if ! command -v docker > /dev/null 2>&1; then
  log "ERROR: docker required for compose load test"
  exit 1
fi

if ! bash "$SCRIPTS/test/bpf/requirements.sh" >> "$GATE_LOG" 2>&1; then
  log "ERROR: bpf preflight failed (see $GATE_LOG)"
  exit 1
fi

if ! bpf_require_privileged_collector "bpf-resource-gate"; then
  log "ERROR: bpf-collector needs root (passwordless sudo or run as root)"
  exit 1
fi

if [[ -x "$SCRIPTS/perf/stabilize_cpu.sh" ]]; then
  bash "$SCRIPTS/perf/stabilize_cpu.sh" >> "$GATE_LOG" 2>&1 || log "WARN: stabilize_cpu failed"
fi

make bpf-dev >> "$GATE_LOG" 2>&1

OUT="${BPF_GATE_OUT:-$ROOT/var/bpf-gate/$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT"
log "session output: $OUT"

LG_ENV=()
[[ -n "${DURATION:-}" ]] && LG_ENV+=(DURATION="$DURATION")

if [[ "$MODE" == "smoke" ]]; then
  log "running constrained load smoke with BPF probe"
  CONSTRAINED=1 AD_EVENT_PROCESSOR_BPF_PROBE=1 \
    bash "$SCRIPTS/test/load/malformed.sh" smoke 2>&1 | tee "$OUT/gate_run.log"
elif [[ "$MODE" == "report-export-soak" ]]; then
  export BPF_COLD_GATE=1
  export BPF_GATE_PROFILE=lab
  export LOAD_SLA_GATE=0
  log "running report export soak (business load + parallel CSV jobs)"
  CONSTRAINED=1 AD_EVENT_PROCESSOR_BPF_PROBE=1 PREPARE="${PREPARE:-0}" \
    bash "$SCRIPTS/test/load/malformed.sh" report-export-soak 2>&1 | tee "$OUT/gate_run.log"
else
  log "running load test mode=$MODE with BPF probe"
  CONSTRAINED=1 AD_EVENT_PROCESSOR_BPF_PROBE=1 \
    bash "$SCRIPTS/test/load/malformed.sh" "$MODE" 2>&1 | tee "$OUT/gate_run.log"
fi

SESSION_DIR=""
SESSION_DIR="$(grep -E '^load-malformed: done - ' "$OUT/gate_run.log" | tail -1 | sed 's/^load-malformed: done - //' || true)"
if [[ -z "$SESSION_DIR" || ! -d "$SESSION_DIR" ]]; then
  SESSION_DIR="$(ls -td "$ROOT/var/load-test"/*/ 2> /dev/null | head -1 || true)"
fi
if [[ -z "$SESSION_DIR" || ! -d "$SESSION_DIR" ]]; then
  log "ERROR: no load-test session under var/load-test/"
  exit 1
fi
log "evaluating session: $SESSION_DIR"

if ! go run ./cmd/load-report bpf "$SESSION_DIR" >> "$GATE_LOG" 2>&1; then
  log "WARN: bpf report generation failed (continuing to gate)"
fi

if ! go run ./cmd/load-report bpf-gate "$SESSION_DIR" --prom "$PROMETHEUS_URL" 2>&1 | tee "$OUT/bpf_gate_eval.log"; then
  log "FAIL: bpf resource gate - see $SESSION_DIR/bpf-gate.md"
  exit 1
fi

BASELINE_DIR="${BPF_BASELINE_DIR:-$ROOT/.ci-baselines/bpf/hot}"
if [[ "${BPF_GATE_COMPARE:-0}" == "1" ]]; then
  mkdir -p "$BASELINE_DIR"
  if ! go run ./cmd/load-report bpf-gate-compare "$BASELINE_DIR" "$SESSION_DIR" --prom "$PROMETHEUS_URL" 2>&1 | tee "$OUT/bpf_gate_compare.log"; then
    log "FAIL: bpf baseline compare - see $SESSION_DIR/bpf-gate-compare.md"
    exit 1
  fi
  cp -a "$SESSION_DIR/bpf-gate-compare.md" "$OUT/" 2> /dev/null || true
fi

cp -a "$SESSION_DIR/bpf-gate.md" "$OUT/" 2> /dev/null || true
if [[ -f "$SESSION_DIR/bpf/maps/summary.json" ]]; then
  mkdir -p "$OUT/bpf/maps"
  cp -a "$SESSION_DIR/bpf/maps/summary.json" "$OUT/bpf/maps/"
fi
if [[ -f "$SESSION_DIR/bpf/events.ndjson" ]]; then
  mkdir -p "$OUT/bpf"
  cp -a "$SESSION_DIR/bpf/events.ndjson" "$OUT/bpf/"
fi

log "PASS: bpf resource gate ($SESSION_DIR)"
echo "$SESSION_DIR" > "$OUT/session_path.txt"
