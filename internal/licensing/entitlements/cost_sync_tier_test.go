package entitlements

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCostSyncTier_disabledManualOnly_holdout(t *testing.T) {
	limits := Limits{MaxCostSyncNetworks: 0}
	require.True(t, CostSyncNetworksDisabled(limits))
	require.False(t, CostSyncNetworksUnlimited(limits))
	require.Equal(t, uint64(0), CostSyncNetworksCap(limits))
}

func TestCostSyncTier_starterCap_holdout(t *testing.T) {
	limits := Limits{MaxCostSyncNetworks: 3}
	require.False(t, CostSyncNetworksDisabled(limits))
	require.False(t, CostSyncNetworksUnlimited(limits))
	require.Equal(t, uint64(3), CostSyncNetworksCap(limits))
}

func TestCostSyncTier_scaleUnlimited_holdout(t *testing.T) {
	limits := Limits{MaxCostSyncNetworks: 999999}
	require.False(t, CostSyncNetworksDisabled(limits))
	require.True(t, CostSyncNetworksUnlimited(limits))
	require.Equal(t, uint64(0), CostSyncNetworksCap(limits))
}
