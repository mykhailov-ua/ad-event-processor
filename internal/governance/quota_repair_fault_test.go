package governance

import (
	"context"
	"strconv"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_QuotaDriftRepair(t *testing.T) {
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
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'quota-drift', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	seedQuotaFaultCampaign(t, pool, campaignID, customerID, 10_000_000)

	const reserved = quotaFaultChunkMicro
	_, err = pool.Exec(ctx, `
		INSERT INTO campaign_quotas (shard_id, campaign_id, reserved_amount, chunk_size, updated_at)
		VALUES (0, $1, $2, $3, NOW() - INTERVAL '35 seconds')`,
		domain.ToUUID(campaignID), reserved, quotaFaultChunkMicro)
	require.NoError(t, err)

	cfg := &config.Config{
		QuotaMode:       "live",
		QuotaChunkSize:  quotaFaultChunkMicro,
		QuotaAutoRepair: true,
	}
	host := newTestHost(t, pool, []redis.UniversalClient{redisClient}, cfg)
	runner := NewQuotaRepairRunner(host, nil)

	start := time.Now()
	runner.RepairQuotaDrift(ctx)

	var outboxID int64
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT id FROM outbox_events WHERE event_type = 'QUOTA_REPAIR' ORDER BY id DESC LIMIT 1`,
	).Scan(&outboxID))

	ob := NewOutboxWorker(host)
	require.NoError(t, ob.ApplyQuotaRepair(ctx, outboxID, payloadFromOutbox(t, ctx, pool, outboxID)))

	redisQuota, err := redisClient.Get(ctx, "budget:quota:"+campaignID.String()).Int64()
	require.NoError(t, err)
	require.Equal(t, reserved, redisQuota)

	var spend, limit int64
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT current_spend, budget_limit FROM campaigns WHERE id = $1`, domain.ToUUID(campaignID),
	).Scan(&spend, &limit))
	require.LessOrEqual(t, spend, limit)

	var auditCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM admin_audit_log WHERE action = 'QUOTA_REPAIR_TOPUP' AND target_id = $1`,
		domain.ToUUID(campaignID)).Scan(&auditCount))
	require.GreaterOrEqual(t, auditCount, 1)

	elapsed := time.Since(start)
	faultproof.Log(t, "quota_drift_repair", map[string]string{
		"subsystem":        "management_quota_recon",
		"repair_micro":     strconv.FormatInt(reserved, 10),
		"redis_quota":      strconv.FormatInt(redisQuota, 10),
		"elapsed_ms":       strconv.FormatInt(elapsed.Milliseconds(), 10),
		"budget_invariant": "true",
		"baseline_ok":      "true",
	})
}

func TestFault_QuotaDeadShardTransientBlip(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	campaignID := uuid.New()
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'quota-blip', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	seedQuotaFaultCampaign(t, pool, campaignID, customerID, 5_000_000)

	const stuckReserved int64 = 2_000_000
	_, err = pool.Exec(ctx, `
		INSERT INTO campaign_quotas (shard_id, campaign_id, reserved_amount, chunk_size, updated_at)
		VALUES (0, $1, $2, $3, NOW() - INTERVAL '90 seconds')`,
		domain.ToUUID(campaignID), stuckReserved, quotaFaultChunkMicro)
	require.NoError(t, err)

	cfg := &config.Config{QuotaMode: "live", QuotaChunkSize: quotaFaultChunkMicro, QuotaAutoRepair: true}
	host := newTestHost(t, pool, []redis.UniversalClient{redisClient}, cfg)
	runner := NewQuotaRepairRunner(host, stubQuorum{})
	runner.ReconcileQuotas(ctx)

	var reservedAfter int64
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT reserved_amount FROM campaign_quotas WHERE shard_id = 0 AND campaign_id = $1`,
		domain.ToUUID(campaignID)).Scan(&reservedAfter))
	require.Equal(t, stuckReserved, reservedAfter, "transient blip must not release reservations")

	faultproof.Log(t, "quota_dead_shard_transient_blip", map[string]string{
		"subsystem":      "management_quota_recon",
		"reserved_after": strconv.FormatInt(reservedAfter, 10),
		"released":       "false",
	})
}
