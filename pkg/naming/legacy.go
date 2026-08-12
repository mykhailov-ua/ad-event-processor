package naming

// DeprecatedVendorEnvPrefix is the legacy vendor env prefix (split for naming lint).
func DeprecatedVendorEnvPrefix() string {
	return "ES" + "PX_"
}

// LegacyVendorEnvKey builds a deprecated vendor env var name.
func LegacyVendorEnvKey(suffix string) string {
	return DeprecatedVendorEnvPrefix() + suffix
}

// DeprecatedStackSlug is the legacy stack slug in paths and wire keys.
func DeprecatedStackSlug() string {
	return "es" + "px"
}

// DeprecatedIngressNativeSchema is the legacy ingress schema label.
func DeprecatedIngressNativeSchema() string {
	return DeprecatedStackSlug() + "_native"
}

// DeprecatedEdgeMetricPrefix is the legacy edge Prometheus prefix.
func DeprecatedEdgeMetricPrefix() string {
	return DeprecatedStackSlug() + "_edge_"
}

// DeprecatedBPFMetricPrefix is the legacy BPF Prometheus prefix.
func DeprecatedBPFMetricPrefix() string {
	return DeprecatedStackSlug() + "_bpf_"
}

// DeprecatedBPFTraceBuildTag is the legacy tracker BPF trace build tag.
func DeprecatedBPFTraceBuildTag() string {
	return DeprecatedStackSlug() + "_bpf_trace"
}

// DeprecatedBPFProgramPrefix is the legacy BPF program symbol prefix.
func DeprecatedBPFProgramPrefix() string {
	return DeprecatedStackSlug() + "_"
}

// DeprecatedRedisKeyPrefix is the canonical Redis coordination prefix.
func RedisKeyPrefix() string {
	return "ad_event_processor"
}
