package filter

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/metrics"
)

type LicenseMonthlyEventsFilter struct {
	registry LicenseStateReader
}

func NewLicenseMonthlyEventsFilter(registry LicenseStateReader) *LicenseMonthlyEventsFilter {
	return &LicenseMonthlyEventsFilter{registry: registry}
}

func (f *LicenseMonthlyEventsFilter) Check(_ context.Context, _ *domain.Event) error {
	if f == nil || f.registry == nil {
		return nil
	}
	state, ent := f.registry.GetLicenseState()
	if !entitlements.MonthlyEventsIngestAllowed(ent.Limits, state, true, time.Now().UTC()) {
		if metrics.LicenseMonthlyEventsExceededTotal != nil {
			metrics.LicenseMonthlyEventsExceededTotal.Inc()
		}
		return ErrRateLimitExceeded
	}
	return nil
}
