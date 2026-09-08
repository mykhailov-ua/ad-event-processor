package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttachTrueROICompareDeltas_holdout(t *testing.T) {
	t.Parallel()
	rows := []TrueROIReportRowDTO{{
		CampaignID:   "c1",
		AdSpendMicro: 200,
		RevenueMicro: 300,
		Conversions:  2,
	}}
	prev := []TrueROIReportRowDTO{{
		CampaignID:   "c1",
		AdSpendMicro: 100,
		RevenueMicro: 150,
		Conversions:  1,
	}}
	attachTrueROICompareDeltas(rows, prev)
	require.NotNil(t, rows[0].Compare)
	require.Equal(t, int64(100), rows[0].Compare.SpendMicroDelta)
	require.Equal(t, int64(150), rows[0].Compare.RevenueMicroDelta)
	require.Equal(t, int64(1), rows[0].Compare.ConversionsDelta)
}

func TestCampaignOverviewRowsFromPortfolio(t *testing.T) {
	t.Parallel()
	out := campaignOverviewRowsFromPortfolio([]BuyerCampaignPortfolioRowDTO{{
		ID:             "c1",
		Name:           "Test",
		Status:         "active",
		Impressions7d:  10,
		Clicks7d:       2,
		UtilizationPct: 0.5,
		PacingDriftPct: 0.1,
		OverspendRisk:  true,
	}})
	require.Len(t, out, 1)
	require.Equal(t, "c1", out[0].CampaignID)
	require.True(t, out[0].OverspendRisk)
}
