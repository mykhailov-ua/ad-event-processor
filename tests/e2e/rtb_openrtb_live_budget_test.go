// Role: OpenRTB 2.6 bid request through live exchange handler; RTB budget authority and PG budget invariant.
// Tier: e2e.
// Infra: testcontainers Postgres (ads schema), single Redis.
// Invariants proved: bid response contains golden id and x-openrtb-version 2.6; RTB store debited; AssertBudgetInvariant holds.
// Verify: make test-integration
package e2e_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"ad-event-processor/internal/config"
	db "ad-event-processor/internal/domain/db"
	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/rtb"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Holdout: live OpenRTB auction must debit RTB store and keep PG budget invariant.
func TestE2E_OpenRTB26LiveBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	defer cleanupDB()

	rdb, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	queries := db.New(pool)
	cfg := &config.Config{
		RtbMode:            "live",
		RtbBudgetAuthority: "rtb",
		MaxRequestBodySize: 1 << 20,
		ClickAmount:        2_000_000,
	}

	customerID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)", customerID, "RTB Exchange", 1_000_000_000)
	require.NoError(t, err)

	campaignID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO campaigns (id, name, status, customer_id, budget_limit, target_countries)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		campaignID, "RTB Exchange Campaign", "ACTIVE", customerID, 100_000_000, []string{"US"},
	)
	require.NoError(t, err)

	registry := testutil.NewAdsRegistry(t, queries)
	_, err = registry.Sync(ctx)
	require.NoError(t, err)

	const redisBudgetMicro = int64(100_000_000)
	camp, ok := registry.GetCampaign(campaignID)
	require.True(t, ok)
	require.NoError(t, rdb.Set(ctx, camp.BudgetCampaignKey, redisBudgetMicro, 0).Err())

	rtbStore := rtb.NewBudgetStore()
	catalog := ingestion.NewRtbCatalog(rtbStore, ingestion.BudgetAuthorityRTB)
	sharder := ingestion.NewJumpHashSharder(1)
	budgetSync := ingestion.RtbBudgetSync{
		Authority: ingestion.BudgetAuthorityRTB,
		Redis:     []redis.UniversalClient{rdb},
		Sharder:   sharder,
	}
	ingestion.SyncRtbCatalog(ctx, registry, catalog, cfg, nil, budgetSync, nil)

	rtbCampID := ingestion.CampaignIDFromUUID(campaignID)
	rtbBudgetBefore := rtbStore.GetBudget(rtbCampID)
	require.Equal(t, redisBudgetMicro, rtbBudgetBefore)

	handler := ingestion.NewAdsPacketHandler(cfg, registry, nil, pool, []redis.UniversalClient{rdb}, sharder, cfg.FraudStreamName, nil)
	handler.ConfigureIngestGeo(staticGeoCountry{country: "US"})
	handler.ConfigureRtb(catalog, staticGeoCountry{country: "US"}, nil, nil)
	defer handler.Stop(ctx)

	body, err := os.ReadFile("../../internal/openrtb/testdata/bid_request_min.json")
	require.NoError(t, err)

	status, resp := ingestion.PostOpenRTBBidGnet(handler, body)
	require.Equal(t, http.StatusOK, status, string(resp))
	assert.Contains(t, string(resp), `"id":"req-golden-001"`)
	assert.Contains(t, string(resp), "x-openrtb-version: 2.6")

	rtbBudgetAfter := rtbStore.GetBudget(rtbCampID)
	assert.Less(t, rtbBudgetAfter, rtbBudgetBefore, "live exchange auction must debit RTB budget store")

	ingestion.AssertBudgetInvariant(t, ctx, pool, rdb, campaignID)
}
