package naming

// BPFTraceBuildTag is the Go build tag that enables traceprobe uprobe markers.
func BPFTraceBuildTag() string {
	return "ad_event_processor_bpf_trace"
}
