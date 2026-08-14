#!/usr/bin/env bash
# XDP prog.Run microbench gate (harness xdp_prog_test — not kernel NIC RX).
# Precondition: BTF vmlinux readable — skips exit 0 when BTF missing.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=../lib/go.sh
source "$ROOT/scripts/lib/go.sh"

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
	echo "edge_xdp_bench_gate: skip (BTF vmlinux missing)"
	exit 0
fi

GO_BIN="$(ad_event_processor_go_bin)"
echo "edge_xdp_bench_gate: BenchmarkXDP_* (harness xdp_prog_test — userspace prog.Run, not kernel RX)"
"$GO_BIN" test -run='^$' -bench='BenchmarkXDP_' -benchmem ./internal/edge/bpf/ -count=1

echo "edge_xdp_bench_gate: ok"
