package campaign

import (
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// CampaignListPGStatsDates maps an RFC3339 window to inclusive UTC calendar days for campaign_stats.
func CampaignListPGStatsDates(from, to time.Time) (pgtype.Date, pgtype.Date) {
	from = from.UTC()
	to = to.UTC()
	return pgtype.Date{
			Time:  time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC),
			Valid: true,
		}, pgtype.Date{
			Time:  time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC),
			Valid: true,
		}
}

// CampaignListMarginWindow aligns ledger margin sums to the same UTC calendar-day span as campaign_stats.
func CampaignListMarginWindow(from, to time.Time) (start, end time.Time) {
	statsFrom, statsTo := CampaignListPGStatsDates(from, to)
	start = statsFrom.Time
	end = statsTo.Time.Add(24 * time.Hour)
	return start, end
}

func CampaignListSortNeedsCHMetrics(field string) bool {
	switch strings.TrimSpace(field) {
	case "unique_clicks", "blocks", "block_pct",
		"hold_leads", "rejected_leads", "leads", "approve_rate",
		"lp_clicks", "lp_views", "lp_ctr", "bots", "bot_pct":
		return true
	default:
		return false
	}
}
