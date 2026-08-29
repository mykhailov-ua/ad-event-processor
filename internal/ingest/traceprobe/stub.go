//go:build !ad_event_processor_bpf_trace

package traceprobe

func ProcessTrackEnter(slot uint32) { _ = slot }

func ProcessTrackExit(slot uint32) { _ = slot }
