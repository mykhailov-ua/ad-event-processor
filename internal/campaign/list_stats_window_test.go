package campaign

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCampaignListPGStatsDates_utcCalendarDays(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 3, 1, 15, 30, 0, 0, time.UTC)
	to := time.Date(2026, 3, 3, 4, 0, 0, 0, time.UTC)
	statsFrom, statsTo := CampaignListPGStatsDates(from, to)
	require.True(t, statsFrom.Valid)
	require.True(t, statsTo.Valid)
	require.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), statsFrom.Time)
	require.Equal(t, time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), statsTo.Time)
}

func TestCampaignListMarginWindow_matchesStatsDays(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 2, 1, 0, 0, 0, time.UTC)
	start, end := CampaignListMarginWindow(from, to)
	require.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), start)
	require.Equal(t, time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), end)
}

func TestCampaignListSortNeedsCHMetrics_holdout(t *testing.T) {
	t.Parallel()
	require.True(t, CampaignListSortNeedsCHMetrics("unique_clicks"))
	require.True(t, CampaignListSortNeedsCHMetrics("block_pct"))
	require.True(t, CampaignListSortNeedsCHMetrics("hold_leads"))
	require.True(t, CampaignListSortNeedsCHMetrics("lp_ctr"))
	require.False(t, CampaignListSortNeedsCHMetrics("cost"))
	require.False(t, CampaignListSortNeedsCHMetrics("approved"))
}
