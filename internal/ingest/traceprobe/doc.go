// Package traceprobe exposes BPF load-test markers around processTrack on the hot path.
//
// Role:
//   - ProcessTrackEnter and ProcessTrackExit are //go:noinline hooks when built with ad_event_processor_bpf_trace.
//   - Stub implementations are no-ops in default builds (!ad_event_processor_bpf_trace).
//   - cmd/bpf-collector attaches uprobes to these symbols for load-test syscall correlation.
//
// Topology:
//   - Called from trackwire.go immediately before and after processTrack on Tier B pinned workers.
//   - Slot argument is the PinnedWorkerPool worker index (routes probe attribution per worker).
//
// Thread model (hot-path.mdc Tracker thread model):
//
//	Tier B only: markers bracket synchronous processTrack -> FilterEngine.Check on LockOSThread workers.
//	Tier A gnet epoll must not call these markers (offload enqueues before track path runs).
//
// Invariants:
//   - Marker functions must stay allocation-free and must not perform I/O on the hot path.
//   - Default stub build: single dead-code branch only; no metrics or logging inside markers.
//
// Forbidden:
//   - Non-trivial work, heap allocations, or blocking inside marker functions.
//   - Calling ProcessTrackEnter/Exit from background goroutines unrelated to offload workers.
//
// Verify (subpackage has no *_test.go; markers exercised via parent processTrack path):
//
//	go test ./internal/ingest/ -short -run TestPinnedWorkerPool -count=1
//	go test ./internal/ingest/ -short -run TestFault_PinnedWorkerPoolSaturationSpike -count=1
//
// Tradeoffs:
//   - Build tag ad_event_processor_bpf_trace vs always-on uprobes: default stubs are single-branch no-ops
//     (0 allocs); BPF load-test builds opt in via tag so production tracker binary pays no probe tax.
//   - Exported //go:noinline symbol pair vs inline USDT/tracepoints: stable uprobe targets for
//     cmd/bpf-collector without kernel BTF coupling on every dev build; rejected always-linked eBPF
//     object in tracker.
//   - Tier B bracket only (before/after processTrack on pinned worker) vs Tier A gnet markers:
//     syscall attribution must cover FilterEngine.Check and sync EVALSHA; epoll thread returns before
//     filter runs (hot-path.mdc); rejected probes on SubmitOffload enqueue only.
//   - Worker-index slot argument vs global counter: attributes probe samples to per-worker queues for
//     load-test correlation; rejected unbounded shared atomic on every event.
//   - Rejected inside markers: Prometheus, logging, or heap work (violates allocation-free invariant;
//     use bpf-collector perf_hw_linux aggregation off-process).
package traceprobe
