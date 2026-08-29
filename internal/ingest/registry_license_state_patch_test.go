package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/require"
)

type activeLicenseStaleSeedRegistry struct{}

func (r activeLicenseStaleSeedRegistry) GetLicenseState() (licensing.LicenseState, licensing.Entitlements) {
	return licensing.StateActive, licensing.Entitlements{
		Limits: licensing.Limits{MaxRPS: 1000},
	}
}

func TestLicenseStatePatchInsufficient(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0, false)

	globalDeploymentRPS.resetForTests()
	filter := NewLicenseRPSFilter(activeLicenseStaleSeedRegistry{})
	err := filter.Check(context.Background(), &domain.Event{})
	require.ErrorIs(t, err, ErrRateLimitExceeded)

	licenseFilter := NewLicenseFilter(activeLicenseStaleSeedRegistry{})
	require.NoError(t, licenseFilter.Check(context.Background(), &domain.Event{}))
}
