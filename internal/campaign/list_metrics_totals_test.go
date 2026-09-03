package campaign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateCampaignListMetricsRows_holdout_recomputesWeightedRoi(t *testing.T) {
	rows := []CampaignListMetricsRowDTO{
		{
			Clicks:               100,
			Conversions:          10,
			LeadsRaw:             10,
			RtbCostMicro:         1_000_000,
			AdvertiserSpendMicro: 2_000_000,
			OperatorMarginMicro:  500_000,
		},
		{
			Clicks:               50,
			Conversions:          5,
			LeadsRaw:             5,
			RtbCostMicro:         500_000,
			AdvertiserSpendMicro: 1_000_000,
			OperatorMarginMicro:  250_000,
		},
	}

	total := aggregateCampaignListMetricsRows(rows)

	require.Equal(t, int64(150), total.Clicks)
	require.Equal(t, int64(15), total.Conversions)
	require.Equal(t, int64(1_500_000), total.RtbCostMicro)
	require.Equal(t, int64(750_000), total.ProfitMicro)
	require.InDelta(t, 50.0, total.RoiPct, 0.01)
}

func TestAggregateCampaignListMetricsRows_holdout_doesNotSumRowRates(t *testing.T) {
	rows := []CampaignListMetricsRowDTO{
		{Clicks: 100, Impressions: 1000, Conversions: 10, CrPct: 10, RtbCostMicro: 1_000_000, OperatorMarginMicro: 500_000},
		{Clicks: 50, Impressions: 500, Conversions: 10, CrPct: 20, RtbCostMicro: 500_000, OperatorMarginMicro: 250_000},
	}

	total := aggregateCampaignListMetricsRows(rows)

	require.InDelta(t, 13.33, total.CrPct, 0.1)
	require.NotEqual(t, 30.0, total.CrPct)
}

func TestMergeCampaignListMetricsRows_sumsCounters(t *testing.T) {
	var total CampaignListMetricsRowDTO
	mergeCampaignListMetricsRow(&total, CampaignListMetricsRowDTO{Clicks: 3, Blocks: 1})
	mergeCampaignListMetricsRow(&total, CampaignListMetricsRowDTO{Clicks: 7, Blocks: 2})

	require.Equal(t, int64(10), total.Clicks)
	require.Equal(t, int64(3), total.Blocks)
}

func TestCountCampaignListMetricsMarginBreaches_holdout_countsTrueRowsOnly(t *testing.T) {
	count := countCampaignListMetricsMarginBreaches(map[string]CampaignListMetricsRowDTO{
		"a": {MarginBreach: true},
		"b": {MarginBreach: false},
		"c": {MarginBreach: true},
	})
	require.Equal(t, int64(2), count)
}
