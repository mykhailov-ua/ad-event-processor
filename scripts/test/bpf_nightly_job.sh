#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

KIND="${1:-}"
case "$KIND" in
  hot)
    BASELINE_DIR=".ci-baselines/bpf/hot"
    GATE_SCRIPT="$SCRIPTS/ci/bpf_resource_gate.sh"
    OUT_LOG="bpf_hot_nightly.log"
    ;;
  cold)
    BASELINE_DIR=".ci-baselines/bpf/cold"
    GATE_SCRIPT="$SCRIPTS/ci/bpf_cold_soak.sh"
    OUT_LOG="bpf_cold_nightly.log"
    ;;
  *)
    echo "usage: $0 hot|cold" >&2
    exit 2
    ;;
esac

mkdir -p "$BASELINE_DIR"

export BPF_GATE_STRICT=true
export BPF_BASELINE_DIR="$ROOT/$BASELINE_DIR"
export BPF_GATE_COMPARE=1

if [[ "$KIND" == "hot" ]]; then
  export BPF_GATE_MODE="${BPF_GATE_MODE:-smoke}"
  export BPF_GATE_DURATION="${BPF_GATE_DURATION:-90s}"
else
  export BPF_GATE_MODE="${BPF_GATE_MODE:-business}"
  export BPF_COLD_SOAK_DURATION="${BPF_COLD_SOAK_DURATION:-30m}"
fi

bash "$GATE_SCRIPT" 2>&1 | tee "$OUT_LOG"
