//go:build !ad_event_processor_bpf_trace

// Package traceprobe provides optional BPF trace hooks for ingestion.
package traceprobe

func ProcessTrackEnter(slot uint32) { _ = slot }

func ProcessTrackExit(slot uint32) { _ = slot }

func FilterCheckEnter(slot uint32) { _ = slot }

func FilterCheckExit(slot uint32) { _ = slot }
