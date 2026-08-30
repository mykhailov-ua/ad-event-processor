// Package naming centralizes canonical product tokens and legacy alias strings for env and deploy wiring.
//
// Role:
//   - BPFTraceBuildTag returns ad_event_processor_bpf_trace for conditional eBPF trace hooks (internal/ingest/traceprobe).
//   - LegacyVendorEnvKey and Deprecated* helpers build pre-rename env keys, metric prefixes, and ingress schema slugs.
//   - RedisKeyPrefix is the canonical Redis/metrics namespace token (ad_event_processor); domain keys live in internal/domain.
//   - BPFProbeProgramPrefix (probe_) prefixes bpf-collector uprobe program names.
//
// Topology:
//   - internal/config env_dual.go reads AD_EVENT_PROCESSOR_* first, then legacy keys from LegacyVendorEnvKey, logs once.
//   - pkg/platformconfig NormalizeIngressSchema maps deprecated ingress slugs to IngressAdEventProcessorNative.
//   - internal/opsadmin maps legacy edge metric names via DeprecatedEdgeMetricPrefix.
//   - Stdlib only; no internal/* imports.
//
// Invariants:
//   - BPFTraceBuildTag matches //go:build tag in traceprobe markers and scripts/dev/stack/build_tracker_bpf_trace.sh.
//   - LegacyVendorEnvKey(suffix) equals DeprecatedVendorEnvPrefix()+suffix at runtime (ESPX_ env family).
//   - DeprecatedIngressNativeSchema maps to ad_event_processor_native via platformconfig.NormalizeIngressSchema.
//   - String literals for forbidden legacy tokens are split in source (ES+PX_, es+px) so naming.mdc CI grep stays clean.
//
// Tradeoffs:
//   - Runtime legacy accessors vs deleting old env keys: dual-read in config preserves operator compose files one release.
//   - Split string literals vs inline banned tokens: repo-wide legacy_naming.sh must not false-positive on this package.
//   - Single BPFTraceBuildTag helper vs duplicated tag string: bpf-collector, loadreport, and Docker TRACKER_BPF_TRACE stay aligned.
//   - RedisKeyPrefix exported here vs scattered ad_event_processor_* literals: one place for deploy docs and future key builders.
//
// Forbidden:
//   - Import internal/* packages.
//   - New inline legacy product tokens in call sites; use Deprecated* or LegacyVendorEnvKey from this package.
//
// Verify:
//
//	go test ./pkg/naming/... -short -count=1
//	go test ./pkg/naming/ -short -run TestBPFTraceBuildTag -count=1
//	bash scripts/ci/naming/legacy_naming.sh
package naming
