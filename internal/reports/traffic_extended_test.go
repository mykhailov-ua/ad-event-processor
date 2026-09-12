package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactTrueROIRows_masksEconomics_holdout(t *testing.T) {
	rows := []TrueROIReportRowDTO{{
		CampaignID:      "camp-1",
		AdSpendMicro:    1000,
		RevenueMicro:    5000,
		TrueProfitMicro: 4000,
		TrueRoiPct:      4.0,
		TrueCpaMicro:    200,
		Conversions:     5,
	}}
	out := redactTrueROIRows(rows)
	require.Equal(t, int64(1000), out[0].AdSpendMicro)
	require.Equal(t, int64(0), out[0].RevenueMicro)
	require.Equal(t, int64(0), out[0].TrueProfitMicro)
	require.Equal(t, float64(0), out[0].TrueRoiPct)
	require.Equal(t, int64(0), out[0].TrueCpaMicro)
}

func TestAttachTrueROICompareDeltas_maskedPrevDoesNotLeakRevenue_holdout(t *testing.T) {
	rows := []TrueROIReportRowDTO{{
		CampaignID:   "camp-1",
		AdSpendMicro: 1000,
		RevenueMicro: 0,
		Conversions:  2,
	}}
	prev := []TrueROIReportRowDTO{{
		CampaignID:   "camp-1",
		AdSpendMicro: 800,
		RevenueMicro: 9000,
		Conversions:  1,
	}}
	prev = redactTrueROIRows(prev)
	attachTrueROICompareDeltas(rows, prev)
	require.NotNil(t, rows[0].Compare)
	require.Equal(t, int64(0), rows[0].Compare.RevenueMicroDelta)
}

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
