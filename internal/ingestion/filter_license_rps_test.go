package ingestion

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/licensing"

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

func TestLicenseRPSFilter_exceedsCap(t *testing.T) {
	globalDeploymentRPS.epoch.Store(0)
	globalDeploymentRPS.count.Store(0)

	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 2})
	ctx := context.Background()
	evt := &domain.Event{}

	require.NoError(t, f.Check(ctx, evt))
	require.NoError(t, f.Check(ctx, evt))
	err := f.Check(ctx, evt)
	require.ErrorIs(t, err, ErrRateLimitExceeded)
}

func TestLicenseRPSFilter_zeroUnlimited(t *testing.T) {
	f := NewLicenseRPSFilter(&stubLicenseRPSRegistry{maxRPS: 0})
	for i := 0; i < 5; i++ {
		assert.NoError(t, f.Check(context.Background(), &domain.Event{}))
	}
}
