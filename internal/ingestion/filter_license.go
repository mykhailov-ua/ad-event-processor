package ingestion

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/licensing"
)

type licenseStateReader interface {
	GetLicenseState() (licensing.LicenseState, licensing.Entitlements)
}

type LicenseFilter struct {
	registry licenseStateReader
}

func NewLicenseFilter(registry licenseStateReader) *LicenseFilter {
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
