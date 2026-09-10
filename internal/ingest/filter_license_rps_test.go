package ingest

import (
	"context"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensing/entitlements"

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
	resetGlobalDeploymentRPSForTests()

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
	resetGlobalDeploymentRPSForTests()
	setGlobalDeploymentRPSBurstForTests(1, 1)

	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 2})
	ctx := context.Background()
	evt := &domain.Event{}

	require.NoError(t, f.Check(ctx, evt))
	require.NoError(t, f.Check(ctx, evt))
	require.NoError(t, f.Check(ctx, evt))
	err := f.Check(ctx, evt)
	require.ErrorIs(t, err, ErrRateLimitExceeded)
	assert.Equal(t, uint64(0), globalDeploymentRPSBurstRemainForTests())
}

func TestLicenseRPSFilter_pilotUnlimitedFromSKU(t *testing.T) {
	resetGlobalDeploymentRPSForTests()

	doc, err := entitlements.LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU(entitlements.SKUCodePilot)
	require.NoError(t, err)
	require.Zero(t, sku.Limits.MaxRPS, "pilot SKU must not cap ingest RPS")

	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: sku.Limits.MaxRPS})
	ctx := context.Background()
	evt := &domain.Event{}

	for range 100 {
		require.NoError(t, f.Check(ctx, evt), "max_rps=0 must not rate-limit ingest")
	}
}

func TestLicenseRPSFilter_zeroUnlimited(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)

	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 0})
	for range 5 {
		assert.NoError(t, f.Check(context.Background(), &domain.Event{}))
	}
}

func TestLicenseRPSFilter_zeroUnlimited_seedCouplingBlocks(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0, false)

	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 0})
	err := f.Check(context.Background(), &domain.Event{})
	require.ErrorIs(t, err, ErrRateLimitExceeded)
}

func TestLicenseRPSFilter_seedCouplingBlocksWithoutValidSeed(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0, false)

	resetGlobalDeploymentRPSForTests()
	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 1000})
	err := f.Check(context.Background(), &domain.Event{})
	require.ErrorIs(t, err, ErrRateLimitExceeded)
}
