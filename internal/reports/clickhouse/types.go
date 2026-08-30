package clickhouse

import "time"

const (
	// maxStatsRange: admin API guard against unbounded CH scans (90d ceiling).
	maxStatsRange = 90 * 24 * time.Hour
)

type ReportMetricsCHRow struct {
	Dimension    string
	CampaignID   string
	Impressions  int64
	Clicks       int64
	Conversions  int64
	SpendMicro   int64
	RevenueMicro int64
}

func ReportMetricsKey(dimension, campaignID string) string {
	return dimension + "\x00" + campaignID
}

func calcIVTRate(ivtEvents, clicks int64) float64 {
	if clicks <= 0 {
		return 0
	}
	rate := float64(ivtEvents) / float64(clicks)
	if rate < 0 {
		return 0
	}
	if rate > 1 {
		return 1
	}
	return rate
}
