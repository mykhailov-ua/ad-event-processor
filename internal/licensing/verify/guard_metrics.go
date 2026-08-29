//go:build linux && license_guard

package verify

import "ad-event-processor/internal/metrics"

func init() {
	guardTripRecorder = func(reason string) {
		metrics.LicenseGuardTripTotal.WithLabelValues(reason).Inc()
	}
}
