package management

import (
	"espx/pkg/faultproof"

	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockShardMetricsProvider struct {
	metrics map[int16]ShardMetrics
}

func (p *mockShardMetricsProvider) GetMetrics(ctx context.Context, shardID int16, rdb redis.UniversalClient) (ShardMetrics, error) {
	m, ok := p.metrics[shardID]
	if !ok {
		return ShardMetrics{ShardID: shardID}, nil
	}
	return m, nil
}

func TestFault_ShardAutoscale_SuddenLoadSpike(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb0, cleanup0 := database.SetupTestRedis(t)
	defer cleanup0()
	rdb1, cleanup1 := database.SetupTestRedis(t)
	defer cleanup1()

	cfg := &config.Config{SlotMigrationEnabled: false}
	svc := NewService(pool, []redis.UniversalClient{rdb0, rdb1}, ingestion.NewStaticSlotSharder(2), cfg)
	defer svc.Close()

	mapRepo := ingestion.NewSlotMapRepo(pool)
	activeVer, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)

	var campID uuid.UUID
	var slot int16 = 0
	for {
		campID = uuid.New()
		if ingestion.CampaignSlotIndex(campID) == slot {
			break
		}
	}

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Autoscale Cust", 1_000_000, "USD"))

	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'autoscale-test', 1000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		ingestion.ToUUID(campID), ingestion.ToUUID(customerID))
	require.NoError(t, err)

	key := "budget:campaign:" + campID.String()
	require.NoError(t, rdb0.Set(ctx, key, "850000", 0).Err())

	provider := &mockShardMetricsProvider{
		metrics: map[int16]ShardMetrics{
			0: {ShardID: 0, CPUUsage: 95.0, MemoryPct: 90.0, OpsPerSec: 60000, LuaP99Ms: 25.0},
			1: {ShardID: 1, CPUUsage: 10.0, MemoryPct: 15.0, OpsPerSec: 1000, LuaP99Ms: 1.0},
		},
	}

	autoscaleCfg := ShardAutoscaleConfig{
		Enabled:        true,
		CPULimit:       80.0,
		MemoryPctLimit: 85.0,
		OpsLimit:       50000,
		LuaP99Limit:    15.0,
		SlotsToMigrate: 1,
	}

	newVer, err := svc.AutoscaleShards(ctx, provider, autoscaleCfg)
	require.NoError(t, err)
	assert.True(t, newVer > activeVer, "expected a new slot map version to be created and activated")

	activeVerAfter, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, newVer, activeVerAfter)

	rows, err := mapRepo.ListVersion(ctx, newVer)
	require.NoError(t, err)
	assert.Equal(t, int16(1), rows[slot].ShardID)

	val, err := rdb1.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "850000", val)

	existsOnSource := rdb0.Exists(ctx, key).Val()
	assert.Equal(t, int64(0), existsOnSource, "expected old keys to be drained from Shard 0")

	faultproof.Log(t, "shard_autoscale_sudden_load_spike", map[string]string{
		"new_version":      strconv.FormatInt(int64(newVer), 10),
		"slot_migrated":    strconv.FormatInt(int64(slot), 10),
		"budget_copied":    "true",
		"metrics_injected": "true",
		"source_shard":     "0",
		"target_shard":     "1",
	})
}

func TestFault_ShardAutoscale_ShuffledShards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb0, cleanup0 := database.SetupTestRedis(t)
	defer cleanup0()
	rdb1, cleanup1 := database.SetupTestRedis(t)
	defer cleanup1()

	cfg := &config.Config{SlotMigrationEnabled: false}
	svc := NewService(pool, []redis.UniversalClient{rdb1, rdb0}, ingestion.NewStaticSlotSharder(2), cfg)
	defer svc.Close()

	mapRepo := ingestion.NewSlotMapRepo(pool)
	activeVer, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)

	var campID uuid.UUID
	var slot int16 = 0
	for {
		campID = uuid.New()
		if ingestion.CampaignSlotIndex(campID) == slot {
			break
		}
	}

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Shuffled Cust", 1_000_000, "USD"))

	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'shuffled-test', 1000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		ingestion.ToUUID(campID), ingestion.ToUUID(customerID))
	require.NoError(t, err)

	key := "budget:campaign:" + campID.String()
	require.NoError(t, rdb1.Set(ctx, key, "700000", 0).Err())

	provider := &mockShardMetricsProvider{
		metrics: map[int16]ShardMetrics{
			0: {ShardID: 0, CPUUsage: 90.0, MemoryPct: 90.0, OpsPerSec: 60000, LuaP99Ms: 20.0},
			1: {ShardID: 1, CPUUsage: 10.0, MemoryPct: 10.0, OpsPerSec: 500, LuaP99Ms: 1.0},
		},
	}

	autoscaleCfg := ShardAutoscaleConfig{
		Enabled:        true,
		CPULimit:       80.0,
		MemoryPctLimit: 85.0,
		OpsLimit:       50000,
		LuaP99Limit:    15.0,
		SlotsToMigrate: 1,
	}

	newVer, err := svc.AutoscaleShards(ctx, provider, autoscaleCfg)
	require.NoError(t, err)
	assert.True(t, newVer > activeVer)

	rows, err := mapRepo.ListVersion(ctx, newVer)
	require.NoError(t, err)
	assert.Equal(t, int16(1), rows[slot].ShardID)

	val, err := rdb0.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "700000", val)

	existsOnSource := rdb1.Exists(ctx, key).Val()
	assert.Equal(t, int64(0), existsOnSource, "expected old keys to be drained from Shard 0 (rdb1)")

	faultproof.Log(t, "shard_autoscale_shuffled_shards", map[string]string{
		"new_version":      strconv.FormatInt(int64(newVer), 10),
		"slot_migrated":    strconv.FormatInt(int64(slot), 10),
		"budget_copied":    "true",
		"metrics_injected": "true",
		"shuffled_rdbs":    "true",
	})
}

func TestFault_ShardAutoscale_ConcurrentAutoscaleDeadlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb0, cleanup0 := database.SetupTestRedis(t)
	defer cleanup0()
	rdb1, cleanup1 := database.SetupTestRedis(t)
	defer cleanup1()

	cfg := &config.Config{SlotMigrationEnabled: false}
	svc := NewService(pool, []redis.UniversalClient{rdb0, rdb1}, ingestion.NewStaticSlotSharder(2), cfg)
	defer svc.Close()

	mapRepo := ingestion.NewSlotMapRepo(pool)
	activeVer, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Concurrent Cust", 10_000_000, "USD"))

	var slot0CampID uuid.UUID
	const migratedSlot int16 = 0
	for i := int16(0); i < 5; i++ {
		var campID uuid.UUID
		for {
			campID = uuid.New()
			if ingestion.CampaignSlotIndex(campID) == i {
				break
			}
		}
		if i == migratedSlot {
			slot0CampID = campID
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
			VALUES ($1, 'concurrent-test-%d', 1000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`, i),
			ingestion.ToUUID(campID), ingestion.ToUUID(customerID))
		require.NoError(t, err)

		key := "budget:campaign:" + campID.String()
		require.NoError(t, rdb0.Set(ctx, key, "500000", 0).Err())
	}

	provider := &mockShardMetricsProvider{
		metrics: map[int16]ShardMetrics{
			0: {ShardID: 0, CPUUsage: 95.0, MemoryPct: 90.0, OpsPerSec: 60000, LuaP99Ms: 25.0},
			1: {ShardID: 1, CPUUsage: 10.0, MemoryPct: 15.0, OpsPerSec: 1000, LuaP99Ms: 1.0},
		},
	}

	autoscaleCfg := ShardAutoscaleConfig{
		Enabled:        true,
		CPULimit:       80.0,
		MemoryPctLimit: 85.0,
		OpsLimit:       50000,
		LuaP99Limit:    15.0,
		SlotsToMigrate: 2,
	}

	const concurrency = 4
	var wg sync.WaitGroup
	errorsChan := make(chan error, concurrency)
	var maxNewVer int32

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Intn(15)) * time.Millisecond)
			newVer, err := svc.AutoscaleShards(ctx, provider, autoscaleCfg)
			if err != nil {
				errorsChan <- err
				return
			}
			if newVer > maxNewVer {
				maxNewVer = newVer
			}
		}(i)
	}

	wg.Wait()
	close(errorsChan)

	var failedCount int
	for err := range errorsChan {
		failedCount++
		t.Logf("Expected concurrency conflict: %v", err)
	}

	activeVerAfter, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	assert.Greater(t, activeVerAfter, activeVer, "at least one autoscale worker must publish a new slot map")

	rows, err := mapRepo.ListVersion(ctx, activeVerAfter)
	require.NoError(t, err)
	assert.Equal(t, int16(1), rows[migratedSlot].ShardID, "slot 0 must migrate off overloaded shard 0")

	slot0Key := "budget:campaign:" + slot0CampID.String()
	budgetCopied := rdb1.Exists(ctx, slot0Key).Val() == 1 && rdb0.Exists(ctx, slot0Key).Val() == 0
	assert.True(t, budgetCopied, "migrated slot campaign budget must be copied to target shard")

	t.Logf("Concurrent autoscaling completed: %d/%d workers finished with expected lock conflicts", failedCount, concurrency)

	faultproof.Log(t, "shard_autoscale_concurrent_deadlock", map[string]string{
		"new_version":      strconv.FormatInt(int64(activeVerAfter), 10),
		"slot_migrated":    strconv.FormatInt(int64(migratedSlot), 10),
		"budget_copied":    strconv.FormatBool(budgetCopied),
		"metrics_injected": "true",
		"workers_conflict": strconv.Itoa(failedCount) + "/" + strconv.Itoa(concurrency),
		"max_worker_ver":   strconv.FormatInt(int64(maxNewVer), 10),
	})
}
