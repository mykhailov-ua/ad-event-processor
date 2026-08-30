#!/usr/bin/env bash
# Role: Run userspace XDP benchmark harness (BenchmarkXDP_* via xdp_prog_test, not kernel RX).
# Execution context: Linux host with BTF vmlinux; skips when BTF missing.
# Env knobs: AD_EVENT_PROCESSOR_GO_BIN (via aed_go_bin).
# Verify: bash scripts/test/edge/xdp_bench.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"

source "$ROOT/scripts/lib/go.sh"

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
  echo "edge_xdp_bench_gate: skip (BTF vmlinux missing)"
  exit 0
fi

GO_BIN="$(aed_go_bin)"
echo "edge_xdp_bench_gate: BenchmarkXDP_* (harness xdp_prog_test - userspace prog.Run, not kernel RX)"
"$GO_BIN" test -run='^$' -bench='BenchmarkXDP_' -benchmem ./internal/edge/ -count=1

echo "edge_xdp_bench_gate: ok"
