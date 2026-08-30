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
	{ID: "report:wire-signal-breakdown", Kind: "report", Label: "Wire signal breakdown", Href: "/reports/wire-signal-breakdown", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "wire-signal-breakdown"},
	{ID: "report:silent-reject-impression-funnel", Kind: "report", Label: "Silent reject impression funnel", Href: "/reports/silent-reject-impression-funnel", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "silent-reject-impression-funnel"},
	{ID: "report:ivt-by-source", Kind: "report", Label: "IVT by source", Href: "/reports/ivt-by-source", Meta: "fraud", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked", "fraud:read"}, ReportKey: "ivt-by-source"},
	{ID: "report:filter-rejects", Kind: "report", Label: "Filter rejects", Href: "/reports/filter-rejects", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read"}, ReportKey: "filter-rejects"},
	{ID: "report:fraud-evidence-pack", Kind: "report", Label: "Fraud evidence pack", Href: "/reports/fraud-evidence-pack", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read", "fraud:read"}, ReportKey: "fraud-evidence-pack"},
	{ID: "report:fraud-evidence-pack-bulk", Kind: "report", Label: "Fraud evidence pack bulk", Href: "/reports/fraud-evidence-pack-bulk", Meta: "fraud", Group: "reports", Permissions: []string{"audit:read"}, ReportKey: "fraud-evidence-pack-bulk"},
	{ID: "report:placements", Kind: "report", Label: "Placements", Href: "/reports/placements", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "placements"},
	{ID: "report:campaign-overview", Kind: "report", Label: "Campaign overview", Href: "/reports/campaign-overview", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "campaign-overview"},
	{ID: "report:pacing-drift", Kind: "report", Label: "Pacing drift", Href: "/reports/pacing-drift", Meta: "traffic", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "pacing-drift"},
	{ID: "report:cost-sync-coverage", Kind: "report", Label: "Cost sync coverage", Href: "/reports/cost-sync-coverage", Meta: "billing", Group: "reports", Permissions: []string{"campaigns:read", "campaigns:read:masked"}, ReportKey: "cost-sync-coverage"},
	{ID: "report:rtb-overview", Kind: "report", Label: "RTB overview", Href: "/reports/rtb-overview", Meta: "rtb", Group: "reports", Permissions: []string{"rtb:read"}, LicenseGated: true, FeatureKey: "openrtb", ReportKey: "rtb-overview"},
}

func ReportNavKeys() []string {
	keys := make([]string, len(reportNavEntries))
	for i, entry := range reportNavEntries {
		keys[i] = entry.ReportKey
	}
	return keys
}
