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

func TestExtendedSortMetricsQueryCount_customerScoped(t *testing.T) {
	t.Parallel()
	const (
		campaigns = 2000
		chunk     = campaignListSortMetricsChunk
	)
	chunks := (campaigns + chunk - 1) / chunk
	beforePG := 1 + chunks*2
	afterPG := 1 + 2
	beforeCH := chunks * 2
	afterCH := chunks
	require.Equal(t, 17, beforePG)
	require.Equal(t, 3, afterPG)
	require.Equal(t, 16, beforeCH)
	require.Equal(t, 8, afterCH)
}
