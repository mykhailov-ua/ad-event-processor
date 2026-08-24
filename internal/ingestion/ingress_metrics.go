package ingestion

import "ad-event-processor/internal/metrics"

func incIngressLegacyJSON() {
	metrics.IngressLegacyJSONTotal.Inc()
}
