package ingestion

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLicenseRPSRegistry struct {
	maxRPS uint64
}

func (s *stubLicenseRPSRegistry) GetLicenseState() (licensing.LicenseState, licensing.Entitlements) {
	return licensing.StateActive, licensing.Entitlements{
		Limits: licensing.Limits{MaxRPS: s.maxRPS},
	}
}

func TestLicenseRPSSoftCeil(t *testing.T) {
	assert.Equal(t, uint64(3), licenseRPSSoftCeil(2))
	assert.Equal(t, uint64(11000), licenseRPSSoftCeil(10000))
}

func TestLicenseRPSBurstCap(t *testing.T) {
	assert.Equal(t, uint64(9), licenseRPSBurstCap(2))
	assert.Equal(t, uint64(45000), licenseRPSBurstCap(10000))
}

func TestLicenseRPSFilter_exceedsCap(t *testing.T) {
	globalDeploymentRPS.resetForTests()

	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 2})
	ctx := context.Background()
	evt := &domain.Event{}

	require.NoError(t, f.Check(ctx, evt))
	require.NoError(t, f.Check(ctx, evt))
	require.NoError(t, f.Check(ctx, evt))
	err := f.Check(ctx, evt)
	require.ErrorIs(t, err, ErrRateLimitExceeded)
}

func TestLicenseRPSFilter_burstConsumesCredits(t *testing.T) {
	globalDeploymentRPS.resetForTests()
	globalDeploymentRPS.burstInit.Store(1)
	globalDeploymentRPS.burstRemain.Store(1)

	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 2})
	ctx := context.Background()
	evt := &domain.Event{}

	require.NoError(t, f.Check(ctx, evt))
	require.NoError(t, f.Check(ctx, evt))
	require.NoError(t, f.Check(ctx, evt))
	err := f.Check(ctx, evt)
	require.ErrorIs(t, err, ErrRateLimitExceeded)
	assert.Equal(t, uint64(0), globalDeploymentRPS.burstRemain.Load())
}

func TestLicenseRPSFilter_pilotCap(t *testing.T) {
	globalDeploymentRPS.resetForTests()

	const pilotRPS = uint64(5000)
	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: pilotRPS})
	ctx := context.Background()
	evt := &domain.Event{}

	ceil := licenseRPSSoftCeil(pilotRPS)
	for i := uint64(0); i < ceil; i++ {
		require.NoError(t, f.Check(ctx, evt), "request %d", i+1)
	}
	require.ErrorIs(t, f.Check(ctx, evt), ErrRateLimitExceeded)
}

func TestLicenseRPSFilter_zeroUnlimited(t *testing.T) {
	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 0})
	for range 5 {
		assert.NoError(t, f.Check(context.Background(), &domain.Event{}))
	}
}

func TestLicenseRPSFilter_seedCouplingBlocksWithoutValidSeed(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0, false)

	globalDeploymentRPS.resetForTests()
	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 1000})
	err := f.Check(context.Background(), &domain.Event{})
	require.ErrorIs(t, err, ErrRateLimitExceeded)
}
