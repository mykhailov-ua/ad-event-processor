package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestQuotaManager_refillCampaign_modes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mode           string
		wantRedisQuota int64
		wantPGReserved int64
	}{
		{
			name:           "live credits Redis and Postgres",
			mode:           "live",
			wantRedisQuota: quotaFaultChunkMicro,
			wantPGReserved: quotaFaultChunkMicro,
		},
		{
			name:           "shadow reserves Postgres only",
			mode:           "shadow",
			wantRedisQuota: 0,
			wantPGReserved: quotaFaultChunkMicro,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if testing.Short() {
				t.Skip("integration: run make test-integration (Docker testcontainers)")
			}

			ctx := context.Background()
			pool, cleanupDB := database.SetupTestDB(t)
			defer cleanupDB()
			redisClient, cleanupRedis := database.SetupTestRedis(t)
			defer cleanupRedis()

			customerID := uuid.New()
			campaignID := uuid.New()
			_, err := pool.Exec(ctx, `
				INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'quota-mode', 0, 'USD')`,
				domain.ToUUID(customerID))
			require.NoError(t, err)
			seedQuotaFaultCampaign(t, pool, campaignID, customerID, 10_000_000)

			cfg := &config.Config{
				QuotaMode:               tc.mode,
				QuotaChunkSize:          quotaFaultChunkMicro,
				QuotaRefillThresholdPct: 20,
			}
			svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
			qm := NewQuotaManager(svc)

			lockKey := "budget:refill_lock:" + campaignID.String()
			require.NoError(t, redisClient.Set(ctx, lockKey, "1", 10*time.Second).Err())
			require.NoError(t, qm.refillCampaign(ctx, campaignID, 0, redisClient))

			quotaRepo := domain.NewQuotaRepo(pool)
			pgQuota, err := quotaRepo.GetQuota(ctx, svc.sharder, campaignID)
			require.NoError(t, err)
			require.Equal(t, tc.wantPGReserved, pgQuota.ReservedAmount)

			redisQuota, err := redisClient.Get(ctx, "budget:quota:"+campaignID.String()).Int64()
			if tc.wantRedisQuota == 0 {
				require.Error(t, err)
				require.ErrorIs(t, err, redis.Nil)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantRedisQuota, redisQuota)
			}
		})
	}
}

func TestNewQuotaManager_defaultsChunkSize(t *testing.T) {
	t.Parallel()

	svc := &Service{cfg: &config.Config{}}
	qm := NewQuotaManager(svc)
	require.Equal(t, int64(5_000_000), qm.chunkSize)
	require.Equal(t, 20, qm.thresholdPct)
}
