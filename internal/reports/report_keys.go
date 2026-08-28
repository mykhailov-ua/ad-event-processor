package reports

func LiveReportExportKeys() []string {
	return liveReportExportKeys()
}

func liveReportExportKeys() []string {
	return []string{
		"placements", "keywords", "pacing-drift", "filter-rejects", "fraud-breakdown", "customer-fraud-by-type", "customer-fraud-by-dimension", "wire-signal-breakdown", "signal-effectiveness", "rtt-split-tunnel", "layer-desync-drilldown", "campaign-toggle-cohort", "silent-reject-impression-funnel",
		"spend-velocity", "daypart-heatmap", "campaign-geo-device", "geo-roi", "source-quality",
		"ivt-by-source", "rtb-overview", "rtb-no-bid-reasons", "rtb-geo-device", "traffic-sources",
		"discrepancy-buy-sell", "true-roi", "customer-portfolio", "data-quality", "campaign-overview",
		"postback-reconciliation", "conversion-type-payout", "click-log", "telegram", "cost-sync-coverage",
	}
}

func LiveReportMetricKeys() []string {
	keys := append([]string(nil), liveReportExportKeys()...)
	keys = append(keys,
		"campaign-stats",
		"customer-fraud-by-type",
		"customer-fraud-by-dimension",
		"wire-signal-breakdown",
		"signal-effectiveness",
		"rtt-split-tunnel",
		"layer-desync-drilldown",
		"edge-parity",
		"ml/score-distribution",
		"ml/shadow-delta",
		"ml/feature-spikes",
	)
	return keys
}

func reportErrorReasons() []string {
	return []string{"ch_unavailable", "bad_request", "forbidden", "query_timeout", "internal"}
}
