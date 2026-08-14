// Package bpf loads and tests the edge XDP filter program.
//
// BenchmarkXDP_* (bench_test.go) use userspace prog.Run — harness xdp_prog_test only;
// not kernel NIC RX or pinned-map drop proof. Kernel proof: cmd/edge-xdp-fault,
// scripts/test/xdp_resilience_drill.sh, pinned attach drills — docs/enterprise/EDGE_XDP.md.
package bpf

//go:generate sh -c "go run github.com/cilium/ebpf/cmd/bpf2go -no-strip -cc clang -cflags '-O2 -g -Wall -Werror' -target bpfel,bpfeb Edge ../../../deploy/edge/xdp/bpf/edge_filter.c -- -I/usr/include/$(uname -m)-linux-gnu"
