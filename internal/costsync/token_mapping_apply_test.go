package costsync

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestApplyNetworkObjectToken(t *testing.T) {
	t.Parallel()
	campaignID := uuid.New()
	lines := []CostLine{{
		CampaignID:  campaignID,
		PlacementID: "legacy",
		AdsetID:     "set-1",
		AdID:        "ad-9",
	}}
	ApplyNetworkObjectToken(lines, TokenMapping{NetworkObject: "ad_id"})
	require.Equal(t, "ad-9", lines[0].PlacementID)

	lines[0].PlacementID = "legacy"
	ApplyNetworkObjectToken(lines, TokenMapping{NetworkObject: "adset_id"})
	require.Equal(t, "set-1", lines[0].PlacementID)

	lines[0].PlacementID = "legacy"
	ApplyNetworkObjectToken(lines, TokenMapping{NetworkObject: "placement_id"})
	require.Equal(t, "legacy", lines[0].PlacementID)
}
