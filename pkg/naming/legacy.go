package naming

func DeprecatedVendorEnvPrefix() string {
	return "ES" + "PX_"
}

func LegacyVendorEnvKey(suffix string) string {
	return DeprecatedVendorEnvPrefix() + suffix
}

func DeprecatedStackSlug() string {
	return "es" + "px"
}

func DeprecatedIngressNativeSchema() string {
	return DeprecatedStackSlug() + "_native"
}

func DeprecatedEdgeMetricPrefix() string {
	return DeprecatedStackSlug() + "_edge_"
}

func DeprecatedBPFMetricPrefix() string {
	return DeprecatedStackSlug() + "_bpf_"
}

func DeprecatedBPFTraceBuildTag() string {
	return DeprecatedStackSlug() + "_bpf_trace"
}

func DeprecatedBPFProgramPrefix() string {
	return DeprecatedStackSlug() + "_"
}

func RedisKeyPrefix() string {
	return "ad_event_processor"
}
