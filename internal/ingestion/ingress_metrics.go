package ingestion

import "github.com/bidshard/ad-event-processor/internal/metrics"

func incIngressLegacyJSON() {
	metrics.IngressLegacyJSONTotal.Inc()
}
