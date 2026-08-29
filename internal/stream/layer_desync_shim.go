package stream

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

func isFraudStreamLayerDesyncTelemetry(evt *domain.Event) bool {
	if evt == nil {
		return false
	}
	return evt.FraudReason != "" || evt.FraudScore > 0
}

func observeFraudStreamLayerDesync(count uint8) {
	if count > 5 {
		count = 5
	}
	labels := [6]string{"0", "1", "2", "3", "4", "5"}
	metrics.FraudStreamLayerDesyncTotal.WithLabelValues(labels[count]).Inc()
}
