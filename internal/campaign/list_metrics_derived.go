package campaign

import (
	"ad-event-processor/pkg/money"
)

// Derived display fields for GET /api/v1/campaigns/metrics batch rows.
// Call after BatchCampaignListMetrics normalizes leads_raw and lp_views.
// CPA uses leads_raw; ECPA uses conversions (list row VM, not extended sort "cpa"/"ecpa").
func enrichCampaignListMetricsRowDerived(entry *CampaignListMetricsRowDTO) {
	if entry == nil {
		return
	}

	entry.RevenueMicro = entry.AdvertiserSpendMicro + entry.OperatorMarginMicro
	entry.CostMicro = entry.RtbCostMicro
	entry.ProfitMicro = entry.OperatorMarginMicro

	clicks := entry.Clicks
	impressions := entry.Impressions
	conversions := entry.Conversions
	leadsRaw := entry.LeadsRaw
	costMicro := entry.RtbCostMicro
	revenueMicro := entry.RevenueMicro
	profitMicro := entry.ProfitMicro

	if clicks > 0 {
		entry.EpcMicro = revenueMicro / clicks
		entry.CpcMicro = costMicro / clicks
	}
	if leadsRaw > 0 {
		entry.CpaMicro = costMicro / leadsRaw
	}
	if conversions > 0 {
		entry.EcpaMicro = costMicro / conversions
	}

	if pct, ok := campaignListRatePercent(clicks, impressions); ok {
		entry.CtrPct = pct
	}
	if pct, ok := campaignListRatePercent(entry.LPClicks, clicks); ok {
		entry.LpCtrPct = pct
	}
	entry.CrPct = campaignListRatePercentOrZero(conversions, clicks)
	if pct, ok := campaignListRatePercent(conversions, leadsRaw); ok {
		entry.ApproveRatePct = pct
	}
	if pct, ok := campaignListRatePercent(entry.Blocks, clicks); ok {
		entry.BlockPct = pct
	}
	if pct, ok := campaignListRatePercent(entry.Bots, clicks); ok {
		entry.BotPct = pct
	}
	if costMicro > 0 {
		entry.RoiPct = float64(profitMicro) / float64(costMicro) * 100
	}
	if impressions > 0 && costMicro > 0 {
		// CPM micro-units: (cost_micro * 1000 / impressions); FormatFixed2 yields USD per mille.
		cpmMicro := costMicro * 1000 / impressions
		entry.CpmUsd = money.FormatFixed2(cpmMicro)
	}
}

func campaignListRatePercent(numerator, denominator int64) (float64, bool) {
	if denominator <= 0 || numerator <= 0 {
		return 0, false
	}
	return float64(numerator) / float64(denominator) * 100, true
}

func campaignListRatePercentOrZero(numerator, denominator int64) float64 {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator) * 100
}
