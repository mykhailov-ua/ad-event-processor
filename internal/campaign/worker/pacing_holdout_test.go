package worker

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pacingHostStub struct {
	tolerance float64
}

func (p pacingHostStub) PacingHourWeights(context.Context) [24]float64 {
	return campaign.UniformHourWeights()
}

func (p pacingHostStub) CampaignLocation(string) *time.Location {
	return time.UTC
}

func (p pacingHostStub) PacingToleranceMargin() float64 {
	if p.tolerance == 0 {
		return 0.05
	}
	return p.tolerance
}

func (p pacingHostStub) AuditPacingLoopAdjustment(context.Context, db.Querier, uuid.UUID, string, string, string, string) {
}

func TestPacingAdjustmentFromStatsRow_holdoutBalancedEvenSkipsLock(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	weights := campaign.UniformHourWeights()
	tolerancePPM := int64(50_000)
	row := db.GetAllActiveCampaignsWithStatsRow{
		Status:       db.CampaignStatusTypeACTIVE,
		PacingMode:   db.PacingModeTypeEVEN,
		BudgetLimit:  1_000_000,
		CurrentSpend: 500_000,
		Timezone:     "UTC",
	}
	_, shouldUpdate := pacingAdjustmentFromStatsRow(row, weights, now, tolerancePPM, pacingHostStub{tolerance: 0.05})
	assert.False(t, shouldUpdate, "balanced EVEN campaign must not trigger GetCampaignForUpdate path")
}

func TestPacingAdjustmentFromStatsRow_holdoutAheadAsapNeedsUpdate(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	weights := campaign.UniformHourWeights()
	tolerancePPM := int64(50_000)
	row := db.GetAllActiveCampaignsWithStatsRow{
		Status:       db.CampaignStatusTypeACTIVE,
		PacingMode:   db.PacingModeTypeASAP,
		BudgetLimit:  1_000_000,
		CurrentSpend: 900_000,
		Timezone:     "UTC",
	}
	target, shouldUpdate := pacingAdjustmentFromStatsRow(row, weights, now, tolerancePPM, pacingHostStub{tolerance: 0.05})
	require.True(t, shouldUpdate)
	assert.Equal(t, db.PacingModeTypeEVEN, target)
}

func TestPacingAdjustmentFromLockedCampaign_holdoutBehindEvenNeedsAsap(t *testing.T) {
	t.Parallel()
	tolerancePPM := int64(50_000)
	expectedSpendMicro := int64(500_000)
	target, shouldUpdate := pacingAdjustmentFromLockedCampaign(
		db.Campaign{PacingMode: db.PacingModeTypeEVEN},
		expectedSpendMicro,
		100_000,
		tolerancePPM,
	)
	require.True(t, shouldUpdate)
	assert.Equal(t, db.PacingModeTypeASAP, target)
}

func TestClosedLoopPacingControllerTx_holdoutSkipsGetCampaignForUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: pacing worker query budget needs Postgres testcontainers")
	}

	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	custID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO customers (id, name, balance, currency, allowed_overdraft)
VALUES ($1, 'Pacing Holdout', 1000000000, 'USD', 0)`, custID)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		campID := uuid.New()
		_, err := pool.Exec(ctx, `
INSERT INTO campaigns (
  id, customer_id, name, status, budget_limit, current_spend, pacing_mode, timezone, daypart_hours
) VALUES ($1, $2, $3, 'ACTIVE', 1000000, 500000, 'EVEN', 'UTC', '{}')`,
			campID, custID, "Balanced "+campID.String()[:8])
		require.NoError(t, err)
	}

	counter.Reset()
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return closedLoopPacingControllerTx(ctx, tx, nil, pacingHostStub{tolerance: 0.05})
	})
	require.NoError(t, err)
	queries := counter.Snapshot()
	t.Logf("pacing balanced campaigns queries=%d", queries)
	assert.LessOrEqual(t, queries, int64(3), "balanced active campaigns must not call GetCampaignForUpdate per row (N+1 would be >=4)")
}
