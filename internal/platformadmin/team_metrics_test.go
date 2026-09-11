package platformadmin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinalizeTeamMetricsBlock_computesDerivedFields(t *testing.T) {
	block := TeamMetricsBlockDTO{
		SpendMicro:   1_000_000,
		RevenueMicro: 1_500_000,
		Conversions:  10,
	}
	finalizeTeamMetricsBlock(&block, 100)
	require.Equal(t, int64(500_000), block.ProfitMicro)
	require.Equal(t, int64(10_000), block.CPCMicro)
	require.Equal(t, int64(100_000), block.CPAMicro)
	require.InDelta(t, 50.0, block.ROIPct, 0.001)
	require.InDelta(t, 10.0, block.CRPct, 0.001)
}

func TestSortTeamOwnerMetricsByROI_holdout(t *testing.T) {
	rows := []TeamOwnerMetricsDTO{
		{UserID: "low", KPIs: TeamMetricsBlockDTO{ROIPct: 5, RevenueMicro: 100}},
		{UserID: "high", KPIs: TeamMetricsBlockDTO{ROIPct: 40, RevenueMicro: 50}},
		{UserID: "mid", KPIs: TeamMetricsBlockDTO{ROIPct: 20, RevenueMicro: 200}},
	}
	sortTeamOwnerMetricsByROI(rows)
	require.Equal(t, "high", rows[0].UserID)
	require.Equal(t, "mid", rows[1].UserID)
	require.Equal(t, "low", rows[2].UserID)
}

func TestTeamMetricsByOwnerCap_constant(t *testing.T) {
	require.Equal(t, 50, TeamMetricsByOwnerCap)
}
