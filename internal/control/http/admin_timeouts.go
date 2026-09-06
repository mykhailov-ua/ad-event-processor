package http

import (
	"time"

	"ad-event-processor/internal/costsync"
	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/pkg/supportbundle"
)

// AdminLongRouteTimeouts maps "METHOD /path" to handler ctx ceilings longer than HTTP_WRITE_TIMEOUT_MS.
func AdminLongRouteTimeouts() map[string]time.Duration {
	return map[string]time.Duration{
		"POST /api/v1/campaigns/migrate/pull/preview": migrationsource.PullTimeout(),
		"POST /api/v1/ops/support/bundle":             supportbundle.DefaultTimeout,
		"POST /api/v1/cost-sync/run":                  costsync.CycleTimeout(),
	}
}

// ManagementWriteTimeout is the http.Server WriteTimeout ceiling for the admin gateway.
func ManagementWriteTimeout(defaultMs int, longRoutes map[string]time.Duration) time.Duration {
	max := time.Duration(defaultMs) * time.Millisecond
	if max <= 0 {
		max = 10 * time.Second
	}
	for _, d := range longRoutes {
		if d > max {
			max = d
		}
	}
	return max
}
