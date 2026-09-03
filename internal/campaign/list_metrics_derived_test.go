package campaign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnrichCampaignListMetricsRowDerived_ratesAndMoney(t *testing.T) {
	entry := CampaignListMetricsRowDTO{
		Impressions:          10_000,
		Clicks:               500,
		Conversions:          40,
		Blocks:               25,
		Bots:                 10,
		LeadsRaw:             50,
		HoldLeads:            5,
		RejectedLeads:        5,
		LPClicks:             200,
		LPViews:              8_000,
		AdvertiserSpendMicro: 900_000,
		OperatorMarginMicro:  100_000,
		RtbCostMicro:         600_000,
	}
	enrichCampaignListMetricsRowDerived(&entry)

	require.Equal(t, int64(1_000_000), entry.RevenueMicro)
	require.Equal(t, int64(600_000), entry.CostMicro)
	require.Equal(t, int64(100_000), entry.ProfitMicro)
	require.Equal(t, int64(2_000), entry.EpcMicro)
	require.Equal(t, int64(1_200), entry.CpcMicro)
	require.Equal(t, int64(12_000), entry.CpaMicro)
	require.Equal(t, int64(15_000), entry.EcpaMicro)
	require.InDelta(t, 5.0, entry.CtrPct, 1e-9)
	require.InDelta(t, 40.0, entry.LpCtrPct, 1e-9)
	require.InDelta(t, 8.0, entry.CrPct, 1e-9)
	require.InDelta(t, 80.0, entry.ApproveRatePct, 1e-9)
	require.InDelta(t, 5.0, entry.BlockPct, 1e-9)
	require.InDelta(t, 2.0, entry.BotPct, 1e-9)
	require.InDelta(t, 100.0/6.0, entry.RoiPct, 1e-9)
	require.Equal(t, "0.06", entry.CpmUsd)
}

func TestEnrichCampaignListMetricsRowDerived_omitsNonComputable_holdout(t *testing.T) {
	entry := CampaignListMetricsRowDTO{
		Impressions: 0,
		Clicks:      0,
	}
	enrichCampaignListMetricsRowDerived(&entry)

	require.Zero(t, entry.EpcMicro)
	require.Zero(t, entry.CpcMicro)
	require.Zero(t, entry.CpaMicro)
	require.Zero(t, entry.EcpaMicro)
	require.Zero(t, entry.CtrPct)
	require.Zero(t, entry.LpCtrPct)
	require.Zero(t, entry.CrPct)
	require.Zero(t, entry.ApproveRatePct)
	require.Zero(t, entry.BlockPct)
	require.Zero(t, entry.BotPct)
	require.Zero(t, entry.RoiPct)
	require.Empty(t, entry.CpmUsd)
}

func TestAttachCampaignBudgetUsedPct_clampsAndOmits(t *testing.T) {
	dto := &CampaignDTO{
		BudgetLimit:  "100.00",
		CurrentSpend: "25.50",
	}
	AttachCampaignBudgetUsedPct(dto)
	require.InDelta(t, 25.5, *dto.BudgetUsedPct, 1e-9)

	dto = &CampaignDTO{
		BudgetLimit:  "100.00",
		CurrentSpend: "150.00",
	}
	AttachCampaignBudgetUsedPct(dto)
	require.InDelta(t, 100.0, *dto.BudgetUsedPct, 1e-9)

	dto = &CampaignDTO{
		BudgetLimit:  "0.00",
		CurrentSpend: "10.00",
	}
	AttachCampaignBudgetUsedPct(dto)
	require.Nil(t, dto.BudgetUsedPct)
}
