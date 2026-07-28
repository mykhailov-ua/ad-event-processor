//go:build !espx_bpf_trace

package traceprobe

// Dev BPF uprobes attach to the espx_bpf_trace build (see .cursor/rules/load-test-bpf.mdc).
// Default stubs compile away on the hot path.

func ProcessTrackEnter(slot uint32) { _ = slot }

func ProcessTrackExit(slot uint32) { _ = slot }

func FilterCheckEnter(slot uint32) { _ = slot }

func FilterCheckExit(slot uint32) { _ = slot }
