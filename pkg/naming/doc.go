// Package naming holds build-tag helpers and legacy token guards for env validation.
//
// Role:
//   - BPFTraceBuildTag returns ad_event_processor_bpf_trace for conditional eBPF compile.
//   - legacy.go rejects forbidden legacy product tokens in env values during config load.
//
// Topology:
//   - Imported by internal/config and scripts/ci naming gates; stdlib only.
//
// Invariants:
//   - Build tag string matches go build tag in deploy/dev/bpf (naming.mdc).
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/naming/... -short -count=1
package naming
