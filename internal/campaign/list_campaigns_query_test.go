package campaign

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestListCampaignsUsesStatsJoin_holdout(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	filter := ListCampaignsFilter{
		SortField:     "clicks",
		StatsRangeSet: true,
		StatsFrom:     pgtype.Date{Time: from, Valid: true},
		StatsTo:       pgtype.Date{Time: from.Add(7 * 24 * time.Hour), Valid: true},
	}
	require.True(t, ListCampaignsUsesStatsJoin(filter))
	filter.SortField = "updated_at"
	require.False(t, ListCampaignsUsesStatsJoin(filter))
}

func TestCampaignListSortMetricsRoundTrips_customerScoped(t *testing.T) {
	t.Parallel()
	const campaigns = 2000
	require.Equal(t, 17, campaignListExtendedSortPGRoundTrips(campaigns, false))
	require.Equal(t, 3, campaignListExtendedSortPGRoundTrips(campaigns, true))
	require.Equal(t, 16, campaignListSortMetricsCHRoundTrips(campaigns))
}
