package commandpalette

var reportNavEntries = []navEntry{
	{ID: "report:fraud-breakdown", Kind: "report", Label: "Fraud breakdown", Href: "/reports/fraud-breakdown", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "fraud-breakdown"},
	{ID: "report:customer-fraud-by-type", Kind: "report", Label: "Fraud by type", Href: "/reports/customer-fraud-by-type", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "customer-fraud-by-type"},
	{ID: "report:customer-fraud-by-dimension", Kind: "report", Label: "Fraud by dimension", Href: "/reports/customer-fraud-by-dimension", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "customer-fraud-by-dimension"},
	{ID: "report:customer-fraud-evidence", Kind: "report", Label: "Dispute evidence", Href: "/reports/customer-fraud-evidence", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "customer-fraud-evidence"},
	{ID: "report:signal-effectiveness", Kind: "report", Label: "Signal effectiveness", Href: "/reports/signal-effectiveness", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "signal-effectiveness"},
	{ID: "report:rtt-split-tunnel", Kind: "report", Label: "RTT split tunnel", Href: "/reports/rtt-split-tunnel", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read", "fraud:read"}, ReportKey: "rtt-split-tunnel"},
	{ID: "report:campaign-toggle-cohort", Kind: "report", Label: "Campaign toggle cohort", Href: "/reports/campaign-toggle-cohort", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read", "campaigns:read"}, ReportKey: "campaign-toggle-cohort"},
	{ID: "report:layer-desync-drilldown", Kind: "report", Label: "Layer desync drilldown", Href: "/reports/layer-desync-drilldown", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read", "fraud:read"}, ReportKey: "layer-desync-drilldown"},
	{ID: "report:layer-desync-summary", Kind: "report", Label: "Layer desync summary", Href: "/reports/layer-desync-summary", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read", "fraud:read"}, ReportKey: "layer-desync-summary"},
	{ID: "report:wire-signal-breakdown", Kind: "report", Label: "Wire signal breakdown", Href: "/reports/wire-signal-breakdown", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "wire-signal-breakdown"},
	{ID: "report:ml-feature-spikes", Kind: "report", Label: "ML feature spikes", Href: "/reports/ml%2Ffeature-spikes", Meta: "fraud", Group: "reports", Permissions: []string{"shards:read"}, ReportKey: "ml/feature-spikes"},
	{ID: "report:ml-score-distribution", Kind: "report", Label: "ML score distribution", Href: "/reports/ml%2Fscore-distribution", Meta: "fraud", Group: "reports", Permissions: []string{"shards:read"}, ReportKey: "ml/score-distribution"},
	{ID: "report:ml-shadow-delta", Kind: "report", Label: "ML shadow delta", Href: "/reports/ml%2Fshadow-delta", Meta: "fraud", Group: "reports", Permissions: []string{"shards:read"}, ReportKey: "ml/shadow-delta"},
	{ID: "report:silent-reject-impression-funnel", Kind: "report", Label: "Non-blocking fraud response funnel", Href: "/reports/silent-reject-impression-funnel", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "silent-reject-impression-funnel"},
	{ID: "report:ivt-by-source", Kind: "report", Label: "IVT by source", Href: "/reports/ivt-by-source", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "ivt-by-source"},
	{ID: "report:filter-rejects", Kind: "report", Label: "Filter rejects", Href: "/reports/filter-rejects", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read"}, ReportKey: "filter-rejects"},
	{ID: "report:fraud-evidence-pack", Kind: "report", Label: "Fraud evidence pack", Href: "/reports/fraud-evidence-pack", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read", "fraud:read"}, ReportKey: "fraud-evidence-pack"},
	{ID: "report:fraud-evidence-pack-bulk", Kind: "report", Label: "Fraud evidence pack bulk", Href: "/reports/fraud-evidence-pack-bulk", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read"}, ReportKey: "fraud-evidence-pack-bulk"},
	{ID: "report:placements", Kind: "report", Label: "Placements", Href: "/reports/placements", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "placements"},
	{ID: "report:keywords", Kind: "report", Label: "Keywords", Href: "/reports/keywords", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "keywords"},
	{ID: "report:campaign-overview", Kind: "report", Label: "Campaign overview", Href: "/reports/campaign-overview", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "campaign-overview"},
	{ID: "report:campaign-stats", Kind: "report", Label: "Campaign stats", Href: "/reports/campaign-stats", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "campaign-stats"},
	{ID: "report:campaign-geo-device", Kind: "report", Label: "Campaign geo and device", Href: "/reports/campaign-geo-device", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "campaign-geo-device"},
	{ID: "report:source-quality", Kind: "report", Label: "Source quality", Href: "/reports/source-quality", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "source-quality"},
	{ID: "report:spend-velocity", Kind: "report", Label: "Spend velocity", Href: "/reports/spend-velocity", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "spend-velocity"},
	{ID: "report:daypart-heatmap", Kind: "report", Label: "Daypart heatmap", Href: "/reports/daypart-heatmap", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "daypart-heatmap"},
	{ID: "report:traffic-sources", Kind: "report", Label: "Traffic sources", Href: "/reports/traffic-sources", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read"}, ReportKey: "traffic-sources"},
	{ID: "report:geo-roi", Kind: "report", Label: "Geo ROI", Href: "/reports/geo-roi", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read"}, ReportKey: "geo-roi"},
	{ID: "report:true-roi", Kind: "report", Label: "True ROI", Href: "/reports/true-roi", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "true-roi"},
	{ID: "report:pacing-drift", Kind: "report", Label: "Pacing drift", Href: "/reports/pacing-drift", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "pacing-drift"},
	{ID: "report:click-log", Kind: "report", Label: "Click log", Href: "/reports/click-log", Meta: "traffic", Group: "reports", Permissions: []string{"audit:read", "campaigns:read"}, ReportKey: "click-log"},
	{ID: "report:cost-sync-coverage", Kind: "report", Label: "Cost sync coverage", Href: "/reports/cost-sync-coverage", Meta: "billing", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "cost-sync-coverage"},
	{ID: "report:conversion-type-payout", Kind: "report", Label: "Conversion type payout", Href: "/reports/conversion-type-payout", Meta: "billing", Group: "reports", Permissions: []string{"audit:read", "campaigns:read"}, ReportKey: "conversion-type-payout"},
	{ID: "report:customer-portfolio", Kind: "report", Label: "Customer portfolio", Href: "/reports/customer-portfolio", Meta: "billing", Group: "reports", Permissions: []string{"customers:read"}, ReportKey: "customer-portfolio"},
	{ID: "report:discrepancy-buy-sell", Kind: "report", Label: "Buy vs sell discrepancy", Href: "/reports/discrepancy-buy-sell", Meta: "billing", Group: "reports", Permissions: []string{"customers:read"}, ReportKey: "discrepancy-buy-sell"},
	{ID: "report:postback-reconciliation", Kind: "report", Label: "Postback reconciliation", Href: "/reports/postback-reconciliation", Meta: "billing", Group: "reports", Permissions: []string{"audit:read", "campaigns:read"}, ReportKey: "postback-reconciliation"},
	{ID: "report:data-quality", Kind: "report", Label: "Data quality", Href: "/reports/data-quality", Meta: "ops", Group: "reports", Permissions: []string{"audit:read", "customers:read"}, ReportKey: "data-quality"},
	{ID: "report:edge-parity", Kind: "report", Label: "Edge parity", Href: "/reports/edge-parity", Meta: "ops", Group: "reports", Permissions: []string{"shards:read"}, ReportKey: "edge-parity"},
	{ID: "report:rtb-overview", Kind: "report", Label: "RTB overview", Href: "/reports/rtb-overview", Meta: "rtb", Group: "reports", Permissions: []string{"rtb:read"}, LicenseGated: true, FeatureKey: "openrtb", ReportKey: "rtb-overview"},
	{ID: "report:rtb-no-bid-reasons", Kind: "report", Label: "RTB no-bid reasons", Href: "/reports/rtb-no-bid-reasons", Meta: "rtb", Group: "reports", Permissions: []string{"rtb:read"}, LicenseGated: true, FeatureKey: "openrtb", ReportKey: "rtb-no-bid-reasons"},
	{ID: "report:rtb-geo-device", Kind: "report", Label: "RTB geo and device", Href: "/reports/rtb-geo-device", Meta: "rtb", Group: "reports", Permissions: []string{"rtb:read"}, LicenseGated: true, FeatureKey: "openrtb", ReportKey: "rtb-geo-device"},
	{ID: "report:telegram", Kind: "report", Label: "Telegram Mini App rollup", Href: "/reports/telegram", Meta: "telegram", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "telegram"},
	{ID: "report:telegram-summary", Kind: "report", Label: "Telegram summary KPIs", Href: "/reports/telegram/summary", Meta: "telegram", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "telegram/summary"},
	{ID: "report:telegram-funnel", Kind: "report", Label: "Telegram conversion funnel", Href: "/reports/telegram/funnel", Meta: "telegram", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "telegram/funnel"},
	{ID: "report:telegram-bots", Kind: "report", Label: "Telegram bot performance", Href: "/reports/telegram/bots", Meta: "telegram", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "telegram/bots"},
	{ID: "report:telegram-premium", Kind: "report", Label: "Telegram premium users", Href: "/reports/telegram/premium", Meta: "telegram", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "telegram/premium"},
	{ID: "report:telegram-fraud", Kind: "report", Label: "Telegram fraud signals", Href: "/reports/telegram/fraud", Meta: "telegram", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "telegram/fraud"},
}

func ReportNavKeys() []string {
	keys := make([]string, len(reportNavEntries))
	for i, entry := range reportNavEntries {
		keys[i] = entry.ReportKey
	}
	return keys
}
