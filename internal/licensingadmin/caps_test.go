package licensingadmin

import (
	"context"
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestEnforceDeploymentExportAllowed_pilotDisabled_holdout(t *testing.T) {
	host := stubCapHost{
		limits: licensing.Limits{MaxExportChunkBytes: 0},
		state:  licensing.StateActive,
		ok:     true,
	}
	err := EnforceDeploymentExportAllowed(host)
	require.ErrorIs(t, err, ErrDeploymentExportDisabled)
}

func TestEnforceDeploymentExportAllowed_activeTierAllowed(t *testing.T) {
	host := stubCapHost{
		limits: licensing.Limits{MaxExportChunkBytes: 5 << 20},
		state:  licensing.StateActive,
		ok:     true,
	}
	require.NoError(t, EnforceDeploymentExportAllowed(host))
}

func TestEnforceCostSyncNetworkCap_disabledTier_holdout(t *testing.T) {
	host := stubCapHost{
		limits: licensing.Limits{MaxCostSyncNetworks: 0},
		state:  licensing.StateActive,
		ok:     true,
	}
	err := EnforceCostSyncNetworkCap(context.Background(), host, uuid.New(), true)
	require.ErrorIs(t, err, ErrDeploymentCostSyncNetworkLimit)
}

func TestEnforceCostSyncNetworkCap_existingNetworkAllowedAtCap(t *testing.T) {
	host := stubCapHost{
		limits: licensing.Limits{MaxCostSyncNetworks: 2},
		state:  licensing.StateActive,
		ok:     true,
	}
	require.NoError(t, EnforceCostSyncNetworkCap(context.Background(), host, uuid.New(), false))
}

func TestLimitUnlimited(t *testing.T) {
	require.True(t, limitUnlimited(0))
	require.True(t, limitUnlimited(999999))
	require.False(t, limitUnlimited(3))
}

type stubCapHost struct {
	limits licensing.Limits
	state  licensing.LicenseState
	ok     bool
}

func (s stubCapHost) Pool() *pgxpool.Pool { return nil }

func (s stubCapHost) DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool) {
	return s.limits, s.state, s.ok
}

func (s stubCapHost) ErrValidation(msg string) error {
	return ErrDeploymentExportDisabled
}
