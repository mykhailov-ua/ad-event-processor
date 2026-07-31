package ingestion

import (
	"context"

	"espx/internal/domain"
	"espx/internal/licensing"
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
	if !licensing.IngestAllowed(state) {
		return ErrLicenseExpired
	}
	return nil
}
