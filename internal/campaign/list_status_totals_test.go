package campaign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyCampaignStatusCount(t *testing.T) {
	t.Parallel()
	var totals CampaignStatusTotalsDTO
	ApplyCampaignStatusCount(&totals, "ACTIVE", 3)
	ApplyCampaignStatusCount(&totals, "PAUSED", 2)
	ApplyCampaignStatusCount(&totals, "ARCHIVED", 1)
	ApplyCampaignStatusCount(&totals, "UNKNOWN", 4)
	require.Equal(t, int64(10), totals.Total)
	require.Equal(t, int64(3), totals.Active)
	require.Equal(t, int64(2), totals.Paused)
	require.Equal(t, int64(1), totals.Archived)
}
