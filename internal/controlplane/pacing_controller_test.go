package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClosedLoopPacingController(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		CampaignUpdateChannel: "test:pacing-updates",
	}

	sharder := domain.NewJumpHashSharder(1)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, sharder, cfg)
	defer svc.Close()

	ctx := context.Background()
	customerID := uuid.New()

	err := svc.CreateCustomer(ctx, customerID, "Pacing Customer", 1_000_000_000, "USD")
	require.NoError(t, err)

	campaignID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID: customerID, Name: "Pacing Test", BudgetLimitMicro: 100_000_000,
		PacingMode: string(db.PacingModeTypeEVEN), DailyBudgetMicro: 100_000_000, Timezone: "UTC", FreqWindow: 86400, IdempotencyKey: "pacing-idem",
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "DELETE FROM outbox_events")
	require.NoError(t, err)

	queries := db.New(pool)
	campaignRepo := domain.NewCampaignRepo(queries)
	customerRepo := domain.NewCustomerRepo(queries)
	syncWorker := domain.NewSyncWorker(redisClient, campaignRepo, customerRepo, 100*time.Millisecond, 0, nil, 0)

	_, err = pool.Exec(ctx, "UPDATE campaigns SET current_spend = 50000, pacing_mode = 'EVEN' WHERE id = $1", domain.ToUUID(campaignID))
	require.NoError(t, err)

	err = svc.ClosedLoopPacingController(ctx, []*domain.SyncWorker{syncWorker})
	require.NoError(t, err)

	var pacing db.PacingModeType
	err = pool.QueryRow(ctx, "SELECT pacing_mode FROM campaigns WHERE id = $1", domain.ToUUID(campaignID)).Scan(&pacing)
	require.NoError(t, err)
	assert.Equal(t, db.PacingModeTypeASAP, pacing)

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'UPDATE_CAMPAIGN_PACING'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	_, err = pool.Exec(ctx, "DELETE FROM outbox_events")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE campaigns SET current_spend = 150000000, pacing_mode = 'ASAP' WHERE id = $1", domain.ToUUID(campaignID))
	require.NoError(t, err)

	err = svc.ClosedLoopPacingController(ctx, []*domain.SyncWorker{syncWorker})
	require.NoError(t, err)

	err = pool.QueryRow(ctx, "SELECT pacing_mode FROM campaigns WHERE id = $1", domain.ToUUID(campaignID)).Scan(&pacing)
	require.NoError(t, err)
	assert.Equal(t, db.PacingModeTypeEVEN, pacing)

	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'UPDATE_CAMPAIGN_PACING'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestClosedLoopPacingController_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		CampaignUpdateChannel: "test:pacing-updates",
	}

	sharder := domain.NewJumpHashSharder(1)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, sharder, cfg)
	defer svc.Close()

	ctx := context.Background()
	customerID := uuid.New()

	err := svc.CreateCustomer(ctx, customerID, "Pacing Customer Edge", 1_000_000_000, "USD")
	require.NoError(t, err)

	campaignID1, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID: customerID, Name: "Pacing Timezone Edge", BudgetLimitMicro: 100_000_000,
		PacingMode: string(db.PacingModeTypeEVEN), DailyBudgetMicro: 100_000_000, Timezone: "Invalid/Zone", FreqWindow: 86400, IdempotencyKey: "pacing-idem-1",
	})
	require.NoError(t, err)

	campaignID2, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID: customerID, Name: "Pacing Zero Budget Edge", BudgetLimitMicro: 0,
		PacingMode: string(db.PacingModeTypeEVEN), DailyBudgetMicro: 0, Timezone: "UTC", FreqWindow: 86400, IdempotencyKey: "pacing-idem-2",
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "DELETE FROM outbox_events")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "UPDATE campaigns SET current_spend = 50000, pacing_mode = 'EVEN' WHERE id = $1", domain.ToUUID(campaignID1))
	require.NoError(t, err)

	queries := db.New(pool)
	campaignRepo := domain.NewCampaignRepo(queries)
	customerRepo := domain.NewCustomerRepo(queries)
	syncWorker := domain.NewSyncWorker(redisClient, campaignRepo, customerRepo, 100*time.Millisecond, 0, nil, 0)

	err = svc.ClosedLoopPacingController(ctx, []*domain.SyncWorker{syncWorker})
	require.NoError(t, err)

	var pacing1 db.PacingModeType
	err = pool.QueryRow(ctx, "SELECT pacing_mode FROM campaigns WHERE id = $1", domain.ToUUID(campaignID1)).Scan(&pacing1)
	require.NoError(t, err)
	assert.Equal(t, db.PacingModeTypeASAP, pacing1)

	var pacing2 db.PacingModeType
	err = pool.QueryRow(ctx, "SELECT pacing_mode FROM campaigns WHERE id = $1", domain.ToUUID(campaignID2)).Scan(&pacing2)
	require.NoError(t, err)
	assert.Equal(t, db.PacingModeTypeEVEN, pacing2)
}

func BenchmarkClosedLoopPacingController(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping integration benchmark")
	}

	pool, cleanupDB := database.SetupTestDB(b)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(b)
	defer cleanupRedis()

	cfg := &config.Config{
		CampaignUpdateChannel: "test:pacing-updates",
	}

	sharder := domain.NewJumpHashSharder(1)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, sharder, cfg)
	defer svc.Close()

	ctx := context.Background()
	customerID := uuid.New()

	err := svc.CreateCustomer(ctx, customerID, "Bench Pacing Customer", 1_000_000_000, "USD")
	if err != nil {
		b.Fatal(err)
	}

	for range 10 {
		_, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
			CustomerID: customerID, Name: uuid.New().String(), BudgetLimitMicro: 100_000_000,
			PacingMode: string(db.PacingModeTypeEVEN), DailyBudgetMicro: 100_000_000, Timezone: "UTC", FreqWindow: 86400, IdempotencyKey: uuid.New().String(),
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	queries := db.New(pool)
	campaignRepo := domain.NewCampaignRepo(queries)
	customerRepo := domain.NewCustomerRepo(queries)
	syncWorker := domain.NewSyncWorker(redisClient, campaignRepo, customerRepo, 100*time.Millisecond, 0, nil, 0)
	b.ReportAllocs()

	for b.Loop() {
		err = svc.ClosedLoopPacingController(ctx, []*domain.SyncWorker{syncWorker})
		if err != nil {
			b.Fatal(err)
		}
	}
}
