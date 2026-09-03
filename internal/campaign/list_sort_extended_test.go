package campaign

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCampaignListSortMetrics_compareValue_holdout(t *testing.T) {
	t.Parallel()
	metrics := campaignListSortMetrics{
		impressions:   1000,
		clicks:        100,
		conversions:   10,
		uniqueClicks:  80,
		blocks:        5,
		rtbCostMicro:  50_000_000,
		revenueMicro:  80_000_000,
		profitMicro:   30_000_000,
		holdLeads:     3,
		rejectedLeads: 2,
		lpClicks:      40,
		bots:          10,
	}
	require.InDelta(t, 0.1, metrics.compareValue("ctr"), 1e-9)
	require.InDelta(t, 0.1, metrics.compareValue("cr"), 1e-9)
	require.InDelta(t, 0.05, metrics.compareValue("block_pct"), 1e-9)
	require.Equal(t, float64(80), metrics.compareValue("unique_clicks"))
	require.InDelta(t, 0.6, metrics.compareValue("roi"), 1e-9)
	require.Equal(t, float64(15), metrics.compareValue("leads"))
	require.InDelta(t, 10.0/15.0, metrics.compareValue("approve_rate"), 1e-9)
	require.InDelta(t, 0.4, metrics.compareValue("lp_ctr"), 1e-9)
	require.InDelta(t, 0.1, metrics.compareValue("bot_pct"), 1e-9)
}

func TestOrderCampaignDTOsByIDs_preservesPageOrder(t *testing.T) {
	t.Parallel()
	first := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	second := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	third := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	items := []CampaignDTO{
		{ID: first.String()},
		{ID: second.String()},
		{ID: third.String()},
	}
	ordered := OrderCampaignDTOsByIDs(items, []uuid.UUID{third, first})
	require.Len(t, ordered, 2)
	require.Equal(t, third.String(), ordered[0].ID)
	require.Equal(t, first.String(), ordered[1].ID)
}

func TestCampaignListExtendedSortMaxKeys_positive(t *testing.T) {
	t.Parallel()
	require.Equal(t, 5000, campaignListExtendedSortMaxKeys)
}
