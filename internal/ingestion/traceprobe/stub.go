//go:build !ad_event_processor_bpf_trace

// Package traceprobe provides no-op BPF trace hooks when tracing is disabled.
package traceprobe

func ProcessTrackEnter(slot uint32) { _ = slot }

func ProcessTrackExit(slot uint32) { _ = slot }

func FilterCheckEnter(slot uint32) { _ = slot }

func FilterCheckExit(slot uint32) { _ = slot }
