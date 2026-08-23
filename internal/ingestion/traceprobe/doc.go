// Package traceprobe provides BPF uprobe marker hooks on the tracker hot path.
//
// Default builds use stub no-ops (stub.go). Build with -tags ad_event_processor_bpf_trace
// for noinline markers attached by bpf-collector (see cmd/bpf-collector/uprobe.go).
package traceprobe
