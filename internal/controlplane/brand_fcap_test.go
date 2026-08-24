package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrandFrequencyCapping(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey: "test-secret",
	}

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	err := svc.CreateCustomer(ctx, custID, "Brand Owner", 1_000_000_000, "USD")
	require.NoError(t, err)

	brandID, err := svc.CreateBrand(ctx, custID, "Nike Group")
	require.NoError(t, err)

	brands, err := svc.ListBrandsByCustomer(ctx, custID)
	require.NoError(t, err)
	require.Len(t, brands, 1)
	assert.Equal(t, brandID.String(), brands[0].ID)
	assert.Equal(t, custID.String(), brands[0].CustomerID)
	assert.Equal(t, "Nike Group", brands[0].Name)

	require.NoError(t, svc.ConfigureBrandFcap(ctx, brandID, 3, 3600))

	var dbLimit, dbWindow int32
	err = pool.QueryRow(ctx, "SELECT freq_limit, freq_window FROM advertiser_brands WHERE id = $1", brandID).Scan(&dbLimit, &dbWindow)
	require.NoError(t, err)
	assert.Equal(t, int32(3), dbLimit)
	assert.Equal(t, int32(3600), dbWindow)

	var auditCount int64
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM admin_audit_log WHERE action = 'CONFIGURE_BRAND_FCAP' AND target_id = $1", brandID).Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, int64(1), auditCount)

	var outboxCount int64
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'CONFIGURE_BRAND_FCAP'").Scan(&outboxCount)
	require.NoError(t, err)
	assert.Equal(t, int64(1), outboxCount)

	campAID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       custID,
		BrandID:          &brandID,
		Name:             "Air Max Run",
		BudgetLimitMicro: 100_000_000,
		DailyBudgetMicro: 10_000_000,
		PacingMode:       string(db.PacingModeTypeASAP),
		Timezone:         "UTC",
		FreqLimit:        2,
		FreqWindow:       3600,
		TargetCountries:  []string{"US"},
		IdempotencyKey:   "brand-fcap-camp-a",
	})
	require.NoError(t, err)

	campBID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       custID,
		BrandID:          &brandID,
		Name:             "Air Max Walk",
		BudgetLimitMicro: 150_000_000,
		DailyBudgetMicro: 15_000_000,
		PacingMode:       string(db.PacingModeTypeASAP),
		Timezone:         "UTC",
		FreqLimit:        2,
		FreqWindow:       3600,
		TargetCountries:  []string{"US"},
		IdempotencyKey:   "brand-fcap-camp-b",
	})
	require.NoError(t, err)

	var brandFcapKeyA, brandFcapKeyB string
	var brandIDDbA, brandIDDbB uuid.UUID
	err = pool.QueryRow(ctx, "SELECT brand_id, brand_fcap_key FROM campaigns WHERE id = $1", campAID).Scan(&brandIDDbA, &brandFcapKeyA)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, "SELECT brand_id, brand_fcap_key FROM campaigns WHERE id = $1", campBID).Scan(&brandIDDbB, &brandFcapKeyB)
	require.NoError(t, err)

	expectedFcapKey := "fcap:b:" + brandID.String()
	assert.Equal(t, brandID, brandIDDbA)
	assert.Equal(t, brandID, brandIDDbB)
	assert.Equal(t, expectedFcapKey, brandFcapKeyA)
	assert.Equal(t, expectedFcapKey, brandFcapKeyB)

	queries := db.New(pool)
	registry := testutil.NewAdsRegistry(t, queries)
	_, err = registry.Sync(ctx)
	require.NoError(t, err)

	filter := testutil.NewLuaUnifiedFilter(rdb, registry)

	rdb.Set(ctx, "budget:campaign:"+campAID.String(), 1000000000, 0)
	rdb.Set(ctx, "budget:campaign:"+campBID.String(), 1000000000, 0)

	evtUser1A := &domain.Event{
		CampaignID: campAID,
		Type:       "click",
		ClickID:    "click_u1_a1",
		UserID:     "user_1",
		IP:         "1.1.1.1",
	}

	evtUser1B := &domain.Event{
		CampaignID: campBID,
		Type:       "click",
		ClickID:    "click_u1_b1",
		UserID:     "user_1",
		IP:         "1.1.1.1",
	}

	evtUser1ASecond := &domain.Event{
		CampaignID: campAID,
		Type:       "click",
		ClickID:    "click_u1_a2",
		UserID:     "user_1",
		IP:         "1.1.1.1",
	}

	err = filter.Check(ctx, evtUser1A)
	assert.NoError(t, err)

	err = filter.Check(ctx, evtUser1B)
	assert.NoError(t, err)

	err = filter.Check(ctx, evtUser1ASecond)
	assert.ErrorIs(t, err, domain.ErrFreqLimitExceeded)
}
