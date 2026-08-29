package filter

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"
)

type LicenseFilter struct {
	registry LicenseStateReader
}

func NewLicenseFilter(registry LicenseStateReader) *LicenseFilter {
	return &LicenseFilter{registry: registry}
}

func (f *LicenseFilter) Check(_ context.Context, _ *domain.Event) error {
	if f == nil || f.registry == nil {
		return nil
	}
	state, _ := f.registry.GetLicenseState()
	if state == licensing.StateExpired || state == licensing.StateRevoked {
		return ErrLicenseExpired
	}
	return nil
}
