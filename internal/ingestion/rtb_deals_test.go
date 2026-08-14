package ingestion

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/rtb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReloadRtbDeals_buildsDealIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name) VALUES ($1, 'deal-test')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO rtb_deals (deal_id, floor_micro, geo_mask, cat_mask, pacing, customer_id, seats)
		VALUES ('pmp-001', 500000, 15, 7, 1, $1, 3)`, customerID)
	require.NoError(t, err)

	catalog := NewRtbCatalog(rtb.NewBudgetStore(), BudgetAuthorityShadow)
	require.NoError(t, ReloadRtbDeals(ctx, db.New(pool), catalog))
	assert.Equal(t, 1, catalog.DealCount())

	deal, ok := catalog.LookupDeal("pmp-001")
	require.True(t, ok)
	assert.Equal(t, int64(500000), deal.FloorMicro)
	assert.Equal(t, uint64(15), deal.GeoMask)
	assert.Equal(t, rtb.PacingOpen, deal.PacingOpen)
	assert.Equal(t, int32(3), deal.Seats)
}

func TestRtbCatalogReloadChannel_default(t *testing.T) {
	assert.Equal(t, "rtb:catalog:reload", RtbCatalogReloadChannel(nil))
}

func TestReloadRtbCatalog_withinSLO(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name) VALUES ($1, 'slo-test')`, customerID)
	require.NoError(t, err)
	for i := range 8 {
		_, err = pool.Exec(ctx, `
			INSERT INTO rtb_deals (deal_id, floor_micro, geo_mask, cat_mask, pacing, customer_id, seats)
			VALUES ($1, 100000, 15, 7, 1, $2, 1)`, fmt.Sprintf("slo-deal-%d", i), customerID)
		require.NoError(t, err)
	}

	cfg := &config.Config{
		RtbMode:               "live",
		RtbCatalogReloadSLOMs: 5000,
	}
	registry := NewRegistry(db.New(pool))
	catalog := NewRtbCatalog(rtb.NewBudgetStore(), BudgetAuthorityShadow)

	start := time.Now()
	require.NoError(t, ReloadRtbCatalog(ctx, db.New(pool), registry, catalog, cfg, nil, RtbBudgetSync{}, nil))
	elapsed := time.Since(start)
	slo := time.Duration(cfg.RtbCatalogReloadSLOMs) * time.Millisecond
	require.Less(t, elapsed, slo, "reload took %s, slo=%s", elapsed, slo)
	assert.Equal(t, 8, catalog.DealCount())
}
