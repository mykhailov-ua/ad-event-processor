package governance

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const quotaFaultChunkMicro int64 = 1_000_000

func seedQuotaFaultCampaign(t *testing.T, pool *pgxpool.Pool, campaignID, customerID uuid.UUID, budgetLimit int64) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'quota-fault', $2, 0, 'ACTIVE', $3, 'ASAP', 'UTC', 86400)
		ON CONFLICT (id) DO NOTHING`,
		domain.ToUUID(campaignID), budgetLimit, domain.ToUUID(customerID))
	require.NoError(t, err)
}

func TestFault_QuotaRefillRace(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'quota-fault', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	seedQuotaFaultCampaign(t, pool, campaignID, customerID, 10_000_000)

	cfg := &config.Config{
		QuotaMode:               "live",
		QuotaChunkSize:          quotaFaultChunkMicro,
		QuotaRefillThresholdPct: 20,
	}
	host := newTestHost(t, pool, []redis.UniversalClient{redisClient}, cfg)
	qm := NewQuotaManager(host)

	lockKey := "budget:refill_lock:" + campaignID.String()
	require.NoError(t, redisClient.Set(ctx, lockKey, "1", 10*time.Second).Err())

	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			_ = qm.refillCampaign(ctx, campaignID, 0, redisClient)
		}()
	}
	wg.Wait()

	quotaRepo := domain.NewQuotaRepo(pool)
	pgQuota, err := quotaRepo.GetQuota(ctx, host.sharder, campaignID)
	require.NoError(t, err)
	require.Equal(t, quotaFaultChunkMicro, pgQuota.ReservedAmount, "exactly one PG chunk must be reserved")

	redisQuota, err := redisClient.Get(ctx, "budget:quota:"+campaignID.String()).Int64()
	require.NoError(t, err)
	require.Equal(t, quotaFaultChunkMicro, redisQuota, "exactly one Redis chunk must be credited")

	exists, err := redisClient.Exists(ctx, lockKey).Result()
	require.NoError(t, err)
	require.Equal(t, int64(0), exists, "refill lock must be consumed")

	faultproof.Log(t, "quota_refill_race", map[string]string{
		"subsystem":         "management_quota",
		"workers":           strconv.Itoa(workers),
		"pg_reserved":       strconv.FormatInt(pgQuota.ReservedAmount, 10),
		"redis_quota":       strconv.FormatInt(redisQuota, 10),
		"baseline_ok":       "true",
		"budget_consistent": "true",
	})
}

func TestFault_QuotaDeadShardRelease(t *testing.T) {
	t.Skip("integration: requires controlplane fault infra bridge")
}
