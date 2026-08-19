#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

source "$ROOT/scripts/lib/go.sh"

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
  echo "edge_xdp_bench_gate: skip (BTF vmlinux missing)"
  exit 0
fi

GO_BIN="$(ad_event_processor_go_bin)"
echo "edge_xdp_bench_gate: BenchmarkXDP_* (harness xdp_prog_test — userspace prog.Run, not kernel RX)"
"$GO_BIN" test -run='^$' -bench='BenchmarkXDP_' -benchmem ./internal/edge/ -count=1

echo "edge_xdp_bench_gate: ok"
