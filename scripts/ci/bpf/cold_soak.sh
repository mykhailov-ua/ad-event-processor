#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/bpf_collector.sh"
cd "$ROOT"

log() { printf 'bpf-cold-soak: %s\n' "$*"; }

STRICT="${BPF_GATE_STRICT:-false}"
if [[ "${PERF_RUNNER_LABEL:-}" != "" ]]; then
  STRICT="true"
fi

if [[ "$STRICT" != "true" ]]; then
  log "SKIPPED (github-hosted): set repo variable PERF_RUNNER_LABEL for cold soak"
  exit 0
fi

export BPF_GATE_STRICT=true
export BPF_COLD_GATE=1
export AD_EVENT_PROCESSOR_BPF_PROBE=1
export AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE="${AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE:-10}"
export AD_EVENT_PROCESSOR_BPF_SLOW_US="${AD_EVENT_PROCESSOR_BPF_SLOW_US:-10000}"
export AD_EVENT_PROCESSOR_BPF_DUMP_INTERVAL="${AD_EVENT_PROCESSOR_BPF_DUMP_INTERVAL:-60}"
export AD_EVENT_PROCESSOR_BPF_REFRESH_TARGETS="${AD_EVENT_PROCESSOR_BPF_REFRESH_TARGETS:-30}"
export LOAD_BPF_GATE=1
export PROMETHEUS_URL="${PROMETHEUS_URL:-http://127.0.0.1:9190}"

MODE="${BPF_GATE_MODE:-business}"
DURATION="${BPF_COLD_SOAK_DURATION:-${BPF_GATE_DURATION:-30m}}"
export DURATION
BASELINE_DIR="${BPF_BASELINE_DIR:-$ROOT/.ci-baselines/bpf/cold}"
GATE_LOG="${GATE_LOG:-$CI_ARTIFACT_DIR/bpf_cold_soak.log}"

log "cold soak (mode=$MODE duration=$DURATION baseline=$BASELINE_DIR)"

if ! command -v docker > /dev/null 2>&1; then
  log "ERROR: docker required"
  exit 1
fi

if ! bash "$SCRIPTS/test/bpf/requirements.sh" >> "$GATE_LOG" 2>&1; then
  log "ERROR: bpf preflight failed (see $GATE_LOG)"
  exit 1
fi

if ! bpf_require_privileged_collector "bpf-cold-soak"; then
  log "ERROR: bpf-collector needs root"
  exit 1
fi

if [[ -x "$SCRIPTS/perf/stabilize_cpu.sh" ]]; then
  bash "$SCRIPTS/perf/stabilize_cpu.sh" >> "$GATE_LOG" 2>&1 || log "WARN: stabilize_cpu failed"
fi

make bpf-dev >> "$GATE_LOG" 2>&1

OUT="${BPF_GATE_OUT:-$ROOT/var/bpf-cold-soak/$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT"
log "session output: $OUT"

log "running cold soak load test with BPF probe"
CONSTRAINED=1 AD_EVENT_PROCESSOR_BPF_PROBE=1 DURATION="$DURATION" \
  bash "$SCRIPTS/test/load/malformed.sh" "$MODE" 2>&1 | tee "$OUT/gate_run.log"

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

go run ./cmd/load-report bpf "$SESSION_DIR" >> "$GATE_LOG" 2>&1 || log "WARN: bpf report generation failed"

if ! go run ./cmd/load-report bpf-gate-compare "$BASELINE_DIR" "$SESSION_DIR" --prom "$PROMETHEUS_URL" 2>&1 | tee "$OUT/bpf_gate_compare.log"; then
  log "FAIL: bpf cold soak gate - see $SESSION_DIR/bpf-gate-compare.md"
  exit 1
fi

mkdir -p "$BASELINE_DIR"
cp -a "$SESSION_DIR/bpf-gate-compare.md" "$OUT/" 2> /dev/null || true
cp -a "$SESSION_DIR/bpf-gate.md" "$OUT/" 2> /dev/null || true
if [[ -f "$SESSION_DIR/bpf/maps/summary.json" ]]; then
  mkdir -p "$OUT/bpf/maps"
  cp -a "$SESSION_DIR/bpf/maps/summary.json" "$OUT/bpf/maps/"
fi

log "PASS: bpf cold soak ($SESSION_DIR)"
echo "$SESSION_DIR" > "$OUT/session_path.txt"
