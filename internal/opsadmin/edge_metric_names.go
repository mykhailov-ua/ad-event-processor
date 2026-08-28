package opsadmin

import "ad-event-processor/pkg/naming"

func edgeMetricNames(canonical string) []string {
	legacy := naming.DeprecatedEdgeMetricPrefix() + canonical[len("ad_event_processor_edge_"):]
	return []string{canonical, legacy}
}

func edgeMetricMatch(name, canonical string) bool {
	for _, n := range edgeMetricNames(canonical) {
		if name == n {
			return true
		}
	}
	return false
}
