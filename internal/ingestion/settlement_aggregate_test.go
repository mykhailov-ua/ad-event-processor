package ingestion

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCompactSettlementBatch_dedupesClickType(t *testing.T) {
	id := uuid.New()
	evt := &domain.Event{ClickID: "clk-1", Type: "click", CampaignID: id}
	batch := []*domain.Event{evt, evt, {ClickID: "clk-2", Type: "click", CampaignID: id}}

	out, dropped := compactSettlementBatch(batch)
	require.Equal(t, 2, len(out))
	require.Equal(t, 1, dropped)
}

func TestRollupCampaignStats_countsByType(t *testing.T) {
	camp := uuid.New()
	events := []*domain.Event{
		{Type: "impression", CampaignID: camp},
		{Type: "impression", CampaignID: camp},
		{Type: "click", CampaignID: camp},
		{Type: "conversion", CampaignID: camp},
		{Type: fraudAggregateEventType, CampaignID: camp},
		{Type: "video_start", CampaignID: camp},
	}

	rollup := rollupCampaignStats(events)
	require.Len(t, rollup, 1)
	row := rollup[camp]
	require.Equal(t, int64(2), row.impressions)
	require.Equal(t, int64(1), row.clicks)
	require.Equal(t, int64(1), row.conversions)
}

func TestAdaptiveRefillThresholdPct_hotCampaign(t *testing.T) {
	require.Equal(t, 10, adaptiveRefillThresholdPct(1000, 20))
	require.Equal(t, 20, adaptiveRefillThresholdPct(100, 20))
	require.Equal(t, 40, adaptiveRefillThresholdPct(1, 20))
	require.Equal(t, 20, adaptiveRefillThresholdPct(0, 20))
}

func TestCampaignStatRollupArrays_sorted(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	if a.String() > b.String() {
		a, b = b, a
	}
	rollup := map[uuid.UUID]campaignStatRollup{
		b: {impressions: 3},
		a: {clicks: 2},
	}
	ids, imps, clicks, convs := campaignStatRollupArrays(rollup)
	require.Len(t, ids, 2)
	require.Equal(t, a, uuid.UUID(ids[0].Bytes))
	require.Equal(t, int64(0), imps[0])
	require.Equal(t, int64(2), clicks[0])
	require.Equal(t, int64(3), imps[1])
	require.Equal(t, int64(0), convs[0])
	require.Equal(t, int64(0), convs[1])
}
