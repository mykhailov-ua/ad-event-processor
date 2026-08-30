// Package traceprobe exposes BPF load-test markers around processTrack on the hot path.
//
// Role:
//   - ProcessTrackEnter and ProcessTrackExit noinline hooks when built with ad_event_processor_bpf_trace.
//   - Stub implementations are no-ops in default builds (!ad_event_processor_bpf_trace).
//
// Topology:
//   - Called from Tier B pinned worker around processTrack; markers correlate with deploy/dev/bpf probes.
//   - Slot argument routes probe attribution to worker index.
//
// Forbidden:
//   - Non-trivial work, allocations, or I/O inside marker functions on hot path.
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
package traceprobe
