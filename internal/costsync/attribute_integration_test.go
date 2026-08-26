package costsync

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAttribute_tokenMatch_integration(t *testing.T) {
	pool := setupCostSyncDB(t)
	conn := setupClickHouseCostSyncTest(t)
	ctx := context.Background()

	customerID, campaignID := seedCustomerCampaign(t, pool)
	day := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	runID := insertCostSyncRun(t, pool, customerID, "facebook", day)

	insertCostSyncClick(t, conn, "clk-match", campaignID, day, "ad-100", "")
	insertCostSyncClick(t, conn, "clk-other", campaignID, day, "ad-999", "")

	attr := NewClickCostAttributor(pool, conn)
	lines := []CostLine{{
		CustomerID:  customerID,
		CampaignID:  campaignID,
		Date:        day,
		Network:     "facebook",
		PlacementID: "ad-100",
		LineType:    LineTypeSpend,
		AmountMicro: 5_000_000,
		Currency:    "USD",
	}}
	require.NoError(t, attr.AttributeLines(ctx, runID, uuid.New(), TokenMapping{
		PlacementField:  "placement_id",
		AttributionMode: AttributionModeToken,
	}, lines, []int64{5_000_000}, day))

	cost, source := readClickAttribution(t, conn, "clk-match")
	require.Equal(t, int64(5_000_000), cost)
	require.Equal(t, costSourceAPIToken, source)

	otherCost, otherSource := readClickAttribution(t, conn, "clk-other")
	require.Equal(t, int64(0), otherCost)
	require.Equal(t, "", otherSource)
}

func TestAttribute_tokenMatch_sub1_integration(t *testing.T) {
	pool := setupCostSyncDB(t)
	conn := setupClickHouseCostSyncTest(t)
	ctx := context.Background()

	customerID, campaignID := seedCustomerCampaign(t, pool)
	day := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	runID := insertCostSyncRun(t, pool, customerID, "facebook", day)

	insertCostSyncClick(t, conn, "clk-sub1", campaignID, day, "", "fb-ad-7")
	insertCostSyncClick(t, conn, "clk-sub1-miss", campaignID, day, "", "other")

	attr := NewClickCostAttributor(pool, conn)
	lines := []CostLine{{
		CustomerID:  customerID,
		CampaignID:  campaignID,
		Date:        day,
		Network:     "facebook",
		PlacementID: "fb-ad-7",
		LineType:    LineTypeSpend,
		AmountMicro: 2_000_000,
		Currency:    "USD",
	}}
	require.NoError(t, attr.AttributeLines(ctx, runID, uuid.New(), TokenMapping{
		PlacementField:  "sub1",
		AttributionMode: AttributionModeToken,
	}, lines, []int64{2_000_000}, day))

	cost, source := readClickAttribution(t, conn, "clk-sub1")
	require.Equal(t, int64(2_000_000), cost)
	require.Equal(t, costSourceAPIToken, source)

	missCost, _ := readClickAttribution(t, conn, "clk-sub1-miss")
	require.Equal(t, int64(0), missCost)
}

func TestAttribute_spread_integration(t *testing.T) {
	pool := setupCostSyncDB(t)
	conn := setupClickHouseCostSyncTest(t)
	ctx := context.Background()

	customerID, campaignID := seedCustomerCampaign(t, pool)
	day := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	runID := insertCostSyncRun(t, pool, customerID, "facebook", day)

	insertCostSyncClick(t, conn, "clk-spread-1", campaignID, day, "p1", "")
	insertCostSyncClick(t, conn, "clk-spread-2", campaignID, day, "p2", "")
	insertCostSyncClick(t, conn, "clk-spread-3", campaignID, day, "p3", "")

	attr := NewClickCostAttributor(pool, conn)
	lines := []CostLine{
		{
			CustomerID:  customerID,
			CampaignID:  campaignID,
			Date:        day,
			Network:     "facebook",
			PlacementID: "p1",
			LineType:    LineTypeSpend,
			AmountMicro: 3_000_000,
			Currency:    "USD",
		},
		{
			CustomerID:  customerID,
			CampaignID:  campaignID,
			Date:        day,
			Network:     "facebook",
			PlacementID: "p2",
			LineType:    LineTypeSpend,
			AmountMicro: 6_000_000,
			Currency:    "USD",
		},
	}
	require.NoError(t, attr.AttributeLines(ctx, runID, uuid.New(), TokenMapping{
		AttributionMode: AttributionModeSpread,
	}, lines, []int64{3_000_000, 6_000_000}, day))

	for _, clickID := range []string{"clk-spread-1", "clk-spread-2", "clk-spread-3"} {
		cost, source := readClickAttribution(t, conn, clickID)
		require.Equal(t, int64(3_000_000), cost, clickID)
		require.Equal(t, costSourceAPISpread, source, clickID)
	}
}

func TestAttribute_idempotentRerun_integration(t *testing.T) {
	pool := setupCostSyncDB(t)
	conn := setupClickHouseCostSyncTest(t)
	ctx := context.Background()

	customerID, campaignID := seedCustomerCampaign(t, pool)
	day := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	runID := insertCostSyncRun(t, pool, customerID, "facebook", day)

	insertCostSyncClick(t, conn, "clk-idem", campaignID, day, "ad-idem", "")

	attr := NewClickCostAttributor(pool, conn)
	lines := []CostLine{{
		CustomerID:  customerID,
		CampaignID:  campaignID,
		Date:        day,
		Network:     "facebook",
		PlacementID: "ad-idem",
		LineType:    LineTypeSpend,
		AmountMicro: 4_000_000,
		Currency:    "USD",
	}}
	mapping := TokenMapping{PlacementField: "placement_id", AttributionMode: AttributionModeToken}

	require.NoError(t, attr.AttributeLines(ctx, runID, uuid.New(), mapping, lines, []int64{4_000_000}, day))
	require.NoError(t, attr.AttributeLines(ctx, runID, uuid.New(), mapping, lines, []int64{9_000_000}, day))

	cost, source := readClickAttribution(t, conn, "clk-idem")
	require.Equal(t, int64(4_000_000), cost)
	require.Equal(t, costSourceAPIToken, source)
}
