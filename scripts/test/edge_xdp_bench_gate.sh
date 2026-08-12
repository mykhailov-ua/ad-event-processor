#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=../lib/go.sh
source "$ROOT/scripts/lib/go.sh"

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
	echo "edge_xdp_bench_gate: skip (BTF vmlinux missing)"
	exit 0
fi

GO_BIN="$(ad_event_processor_go_bin)"
echo "edge_xdp_bench_gate: BenchmarkXDP_* (generic program tests)"
"$GO_BIN" test -run='^$' -bench='BenchmarkXDP_' -benchmem ./internal/edge/bpf/ -count=1

echo "edge_xdp_bench_gate: ok"
