package controlplane

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type faultRedisCmdable struct {
	redis.UniversalClient
	failDump    bool
	failRestore bool
	delay       time.Duration
}

func (c *faultRedisCmdable) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.Exists(ctx, keys...)
}

func (c *faultRedisCmdable) Dump(ctx context.Context, key string) *redis.StringCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	if c.failDump {
		cmd := redis.NewStringCmd(ctx, "dump", key)
		cmd.SetErr(errors.New("fault: network partition on DUMP"))
		return cmd
	}
	return c.UniversalClient.Dump(ctx, key)
}

func (c *faultRedisCmdable) TTL(ctx context.Context, key string) *redis.DurationCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.TTL(ctx, key)
}

func (c *faultRedisCmdable) RestoreReplace(ctx context.Context, key string, ttl time.Duration, value string) *redis.StatusCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	if c.failRestore {
		cmd := redis.NewStatusCmd(ctx, "restore", key)
		cmd.SetErr(errors.New("fault: network partition on RESTORE"))
		return cmd
	}
	return c.UniversalClient.RestoreReplace(ctx, key, ttl, value)
}

func (c *faultRedisCmdable) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.Scan(ctx, cursor, match, count)
}

func (c *faultRedisCmdable) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.Del(ctx, keys...)
}

func setupSlotMigrationFault(t *testing.T, rdbs []redis.UniversalClient) (*Service, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool, cleanupDB := database.SetupTestDB(t)
	t.Cleanup(cleanupDB)
	cfg := &config.Config{SlotMigrationEnabled: false}
	svc := newBareService(t, pool, rdbs, cfg)
	return svc, pool, context.Background()
}

func buildFourRedisShards(base redis.UniversalClient, customize func(rdbs []redis.UniversalClient)) []redis.UniversalClient {
	rdbs := []redis.UniversalClient{base, base, base, base}
	if customize != nil {
		customize(rdbs)
	}
	return rdbs
}

func campaignIDForSlot(t *testing.T, slot int16) uuid.UUID {
	t.Helper()
	for range 50_000 {
		id := uuid.New()
		if domain.CampaignSlotIndex(id) == slot {
			return id
		}
	}
	t.Fatalf("no campaign id for slot %d", slot)
	return uuid.Nil
}

func seedCampaignForSlot(t *testing.T, svc *Service, pool *pgxpool.Pool, ctx context.Context, slot int16, rdb redis.UniversalClient) (uuid.UUID, int16) {
	t.Helper()
	campID := campaignIDForSlot(t, slot)
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Fault", 5_000_000, "USD"))
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'fault-slot', 1000000, 222223, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campID), domain.ToUUID(customerID))
	require.NoError(t, err)
	key := domain.BudgetCampaignKey(campID)
	require.NoError(t, rdb.Set(ctx, key, "777777", 0).Err())
	return campID, slot
}

func prepareMigratingVersion(t *testing.T, ctx context.Context, mapRepo *domain.SlotMapRepo, slot int16, targetShard int16) int32 {
	t.Helper()
	active, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	v, err := mapRepo.CreateNextVersion(ctx, active, nil)
	require.NoError(t, err)
	require.NoError(t, mapRepo.MarkSlotsMigrating(ctx, v, []int16{slot}, targetShard))
	return v
}

func TestFault_SlotMigrationActivateBeforeCopyRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := []redis.UniversalClient{rdb, rdb, rdb, rdb}
	svc, _, ctx := setupSlotMigrationFault(t, rdbs)

	mapRepo := domain.NewSlotMapRepo(svc.GetPool())
	v := prepareMigratingVersion(t, ctx, mapRepo, 7, 2)

	err := svc.ActivateSlotMapVersion(ctx, uuid.Nil, v)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSlotMigrationNotReady)

	active, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int32(1), active, "active map must stay on old version when copy incomplete")

	faultproof.Log(t, "slot_migration_activate_before_copy", map[string]string{
		"subsystem":   "slot_migration",
		"fault_type":  "premature_cutover",
		"rejected":    "true",
		"active_safe": "true",
		"baseline_ok": "true",
	})
}

func TestFault_SlotMigrationCopyRedisPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := buildFourRedisShards(rdb, func(rdbs []redis.UniversalClient) {
		rdbs[3] = &faultRedisCmdable{UniversalClient: rdb, failDump: true}
	})
	svc, _, ctx := setupSlotMigrationFault(t, rdbs)

	const slot int16 = 3
	_, _ = seedCampaignForSlot(t, svc, svc.GetPool(), ctx, slot, rdbs[3])
	mapRepo := domain.NewSlotMapRepo(svc.GetPool())
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 1)

	err := svc.CopyAllMigratingSlots(ctx, v)
	require.Error(t, err)

	migrations, err := svc.GetSlotMigrations(ctx, v)
	require.NoError(t, err)
	require.Len(t, migrations, 1)
	assert.Equal(t, "failed", migrations[0].State)
	assert.NotEmpty(t, migrations[0].LastError)

	active, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int32(1), active)

	faultproof.Log(t, "slot_migration_redis_partition", map[string]string{
		"subsystem":    "slot_migration",
		"fault_type":   "network_partition",
		"dump_failed":  "true",
		"state_failed": "true",
		"active_safe":  "true",
		"baseline_ok":  "true",
	})
}

func TestFault_SlotMigrationCopySlowEventuallySucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	slow := &faultRedisCmdable{UniversalClient: rdb, delay: 15 * time.Millisecond}
	rdbs := buildFourRedisShards(rdb, func(rdbs []redis.UniversalClient) {
		rdbs[0] = slow
		rdbs[1] = slow
	})
	svc, _, ctx := setupSlotMigrationFault(t, rdbs)

	const slot int16 = 0
	campID, _ := seedCampaignForSlot(t, svc, svc.GetPool(), ctx, slot, rdbs[0])
	mapRepo := domain.NewSlotMapRepo(svc.GetPool())
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 1)

	start := time.Now()
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))
	elapsed := time.Since(start)

	migrations, err := svc.GetSlotMigrations(ctx, v)
	require.NoError(t, err)
	require.Len(t, migrations, 1)
	assert.Equal(t, "copied", migrations[0].State)

	key := domain.BudgetCampaignKey(campID)
	val, err := rdbs[1].Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "777777", val)

	faultproof.Log(t, "slot_migration_redis_slow_copy", map[string]string{
		"subsystem":   "slot_migration",
		"fault_type":  "latency_injection",
		"delay_ms":    "15",
		"state":       "copied",
		"elapsed_ms":  fmt.Sprintf("%d", elapsed.Milliseconds()),
		"baseline_ok": "true",
	})
}

func TestFault_SlotMigrationConcurrentCopySameSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := []redis.UniversalClient{rdb, rdb, rdb, rdb}
	svc, _, ctx := setupSlotMigrationFault(t, rdbs)

	const slot int16 = 17
	campID, _ := seedCampaignForSlot(t, svc, svc.GetPool(), ctx, slot, rdb)
	mapRepo := domain.NewSlotMapRepo(svc.GetPool())
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 3)

	const workers = 6
	var wg sync.WaitGroup
	var okCount atomic.Int32
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			if err := svc.CopyAllMigratingSlots(ctx, v); err == nil {
				okCount.Add(1)
			}
		}()
	}
	wg.Wait()

	migrations, err := svc.GetSlotMigrations(ctx, v)
	require.NoError(t, err)
	require.Len(t, migrations, 1)
	assert.Equal(t, "copied", migrations[0].State)

	key := domain.BudgetCampaignKey(campID)
	val, err := rdbs[3].Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "777777", val)

	faultproof.Log(t, "slot_migration_concurrent_copy", map[string]string{
		"subsystem":   "slot_migration",
		"fault_type":  "concurrency_stress",
		"workers":     "6",
		"ok_runs":     fmt.Sprintf("%d", okCount.Load()),
		"state":       "copied",
		"baseline_ok": "true",
	})
}

func TestFault_SlotMigrationConcurrentActivate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := []redis.UniversalClient{rdb, rdb, rdb, rdb}
	svc, _, ctx := setupSlotMigrationFault(t, rdbs)

	const slot int16 = 19
	_, _ = seedCampaignForSlot(t, svc, svc.GetPool(), ctx, slot, rdb)
	mapRepo := domain.NewSlotMapRepo(svc.GetPool())
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 2)
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))

	const workers = 4
	var wg sync.WaitGroup
	var success atomic.Int32
	var alreadyActive atomic.Int32
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			err := svc.ActivateSlotMapVersion(ctx, uuid.Nil, v)
			switch {
			case err == nil:
				success.Add(1)
			case errors.Is(err, domain.ErrSlotMapAlreadyActive):
				alreadyActive.Add(1)
			default:
				t.Errorf("unexpected activate error: %v", err)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), success.Load(), "exactly one concurrent activate must commit")
	assert.Equal(t, int32(workers-1), alreadyActive.Load(), "losers must get ErrSlotMapAlreadyActive")

	active, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, v, active)

	faultproof.Log(t, "slot_migration_concurrent_activate", map[string]string{
		"subsystem":      "slot_migration",
		"fault_type":     "concurrency_stress",
		"workers":        "4",
		"success":        "1",
		"already_active": fmt.Sprintf("%d", alreadyActive.Load()),
		"active":         fmt.Sprintf("%d", active),
		"baseline_ok":    "true",
	})
}

func TestFault_SlotMapMetaLockContention(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	svc, _, ctx := setupSlotMigrationFault(t, []redis.UniversalClient{rdb})
	repo := domain.NewSlotMapRepo(svc.GetPool())
	active, err := repo.GetActiveVersion(ctx)
	require.NoError(t, err)

	const workers = 5
	var wg sync.WaitGroup
	var mu sync.Mutex
	created := make([]int32, 0, workers)
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			v, err := repo.CreateNextVersion(ctx, active, []domain.SlotOverride{
				{Slot: 50, ShardID: 1, State: db.RedisSlotStateACTIVE},
			})
			if err == nil {
				mu.Lock()
				created = append(created, v)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	require.NotEmpty(t, created)
	versions := make(map[int32]struct{}, len(created))
	for _, v := range created {
		versions[v] = struct{}{}
	}
	assert.Equal(t, len(created), len(versions), "each successful create must yield unique version")

	faultproof.Log(t, "slot_map_meta_lock_contention", map[string]string{
		"subsystem":   "slot_map_control_plane",
		"fault_type":  "concurrency_stress",
		"workers":     "5",
		"created":     fmt.Sprintf("%d", len(created)),
		"unique":      fmt.Sprintf("%d", len(versions)),
		"baseline_ok": "true",
	})
}

func TestFault_SlotMapPGDeadlockRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	ctx := context.Background()
	repo := domain.NewSlotMapRepo(pool)

	active, err := repo.GetActiveVersion(ctx)
	require.NoError(t, err)
	v, err := repo.CreateNextVersion(ctx, active, nil)
	require.NoError(t, err)

	start := make(chan struct{})
	var wg sync.WaitGroup
	var errA, errB error
	wg.Add(2)

	run := func(slotA, slotB int16) {
		defer wg.Done()
		<-start
		e := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			q := db.New(tx)
			if _, err := q.LockSlotMapEntry(ctx, db.LockSlotMapEntryParams{Version: v, Slot: slotA}); err != nil {
				return err
			}
			time.Sleep(120 * time.Millisecond)
			_, err := q.LockSlotMapEntry(ctx, db.LockSlotMapEntryParams{Version: v, Slot: slotB})
			return err
		})
		if slotA == 60 {
			errA = e
		} else {
			errB = e
		}
	}

	go run(60, 61)
	go run(61, 60)
	close(start)
	wg.Wait()

	assert.True(t, isDeadlock(errA) || isDeadlock(errB), "expected deadlock, got: %v / %v", errA, errB)

	rows, err := repo.ListVersion(ctx, v)
	require.NoError(t, err)
	assert.Len(t, rows, domain.SlotCount)

	faultproof.Log(t, "slot_map_pg_deadlock_recovery", map[string]string{
		"subsystem":   "slot_map_control_plane",
		"fault_type":  "concurrency_stress",
		"deadlock":    "true",
		"rows_intact": fmt.Sprintf("%d", len(rows)),
		"baseline_ok": "true",
	})
}

func TestFault_SlotMigrationCopyIdempotentRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	flaky := &faultRedisCmdable{UniversalClient: rdb}
	rdbs := buildFourRedisShards(rdb, func(rdbs []redis.UniversalClient) {
		rdbs[3] = flaky
	})
	svc, _, ctx := setupSlotMigrationFault(t, rdbs)

	const slot int16 = 3
	campID, _ := seedCampaignForSlot(t, svc, svc.GetPool(), ctx, slot, rdbs[3])
	mapRepo := domain.NewSlotMapRepo(svc.GetPool())
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 1)

	flaky.failDump = true
	err := svc.CopyAllMigratingSlots(ctx, v)
	require.Error(t, err)

	flaky.failDump = false
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))

	migrations, err := svc.GetSlotMigrations(ctx, v)
	require.NoError(t, err)
	require.Len(t, migrations, 1)
	assert.Equal(t, "copied", migrations[0].State)

	key := domain.BudgetCampaignKey(campID)
	val, err := rdbs[1].Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "777777", val)

	domain.AssertBudgetInvariant(t, ctx, svc.GetPool(), rdbs[1], campID)

	faultproof.Log(t, "slot_migration_copy_retry_recovery", map[string]string{
		"subsystem":   "slot_migration",
		"fault_type":  "transient_network",
		"first_fail":  "true",
		"retry_ok":    "true",
		"state":       "copied",
		"baseline_ok": "true",
	})
}

func TestFault_SlotMigrationRollbackAfterActivate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := []redis.UniversalClient{rdb, rdb, rdb, rdb}
	svc, _, ctx := setupSlotMigrationFault(t, rdbs)

	const slot int16 = 29
	_, _ = seedCampaignForSlot(t, svc, svc.GetPool(), ctx, slot, rdb)
	mapRepo := domain.NewSlotMapRepo(svc.GetPool())
	prevActive, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 2)
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))
	require.NoError(t, svc.ActivateSlotMapVersion(ctx, uuid.Nil, v))

	active, err := mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, v, active)

	require.NoError(t, svc.RollbackSlotMapVersion(ctx, uuid.Nil, prevActive))
	active, err = mapRepo.GetActiveVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, prevActive, active)

	faultproof.Log(t, "slot_migration_rollback", map[string]string{
		"subsystem":    "slot_migration",
		"fault_type":   "operator_recovery",
		"from_version": fmt.Sprintf("%d", v),
		"to_version":   fmt.Sprintf("%d", prevActive),
		"baseline_ok":  "true",
	})
}

func TestFault_DebitFencedDuringSlotCopy(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{MigrationFenceEnabled: true}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb, rdb}, cfg)

	const slot int16 = 5
	campID, _ := seedCampaignForSlot(t, svc, pool, ctx, slot, rdb)
	require.NoError(t, domain.BumpMigrationFences(ctx, pool, rdb, []uuid.UUID{campID}))

	f := testutil.NewLuaUnifiedFilter(rdb, nil)
	require.NoError(t, f.PreloadScripts(ctx))

	evt := &domain.Event{
		Type:       "click",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
		IP:         "203.0.113.70",
		UserID:     "lua10-fence",
	}
	checkCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	err := f.Check(checkCtx, evt)
	require.ErrorIs(t, err, domain.ErrMigrationFenced)

	key := domain.BudgetCampaignKey(campID)
	remaining, err := rdb.Get(ctx, key).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(777777), remaining)

	faultproof.Log(t, "slot_migration_fence_debit_copy", map[string]string{
		"subsystem":        "ads_lua",
		"fault":            "debit_during_copy",
		"fenced":           "true",
		"budget_unchanged": "true",
		"code":             "11",
	})
}

func TestFault_SlotMigrationPGRewarmCutover(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := buildFourRedisShards(rdb, nil)
	cfg := &config.Config{SlotMigrationEnabled: false, MigrationFenceEnabled: true}
	svc, pool, ctx := setupSlotMigrationFault(t, rdbs)
	svc.cfg = cfg

	const slot int16 = 8
	campID, _ := seedCampaignForSlot(t, svc, pool, ctx, slot, rdbs[0])
	mapRepo := domain.NewSlotMapRepo(pool)
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 2)

	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))
	require.NoError(t, svc.ActivateSlotMapVersion(ctx, uuid.Nil, v))

	domain.AssertBudgetInvariant(t, ctx, pool, rdbs[2], campID)
	require.NoError(t, svc.VerifySlotMigrationR5(ctx))

	key := domain.BudgetCampaignKey(campID)
	val, err := rdbs[2].Get(ctx, key).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(777777), val)

	faultproof.Log(t, "slot_migration_pg_rewarm", map[string]string{
		"subsystem":   "slot_migration",
		"fault":       "hot_slot_cutover",
		"r5_ok":       "true",
		"pg_rewarm":   "true",
		"campaign_id": campID.String(),
	})
}

func TestFault_SlotMigrationDualWriteCutover(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := buildFourRedisShards(rdb, nil)
	svc, pool, ctx := setupSlotMigrationFault(t, rdbs)
	svc.cfg = &config.Config{
		SlotMigrationEnabled:          false,
		MigrationFenceEnabled:         true,
		SlotMigrationDualWriteEnabled: true,
		SlotMigrationLagEpsilon:       0,
		SlotMigrationLagThreshold:     1000,
	}

	const slot int16 = 8
	campID, _ := seedCampaignForSlot(t, svc, pool, ctx, slot, rdbs[0])
	mapRepo := domain.NewSlotMapRepo(pool)
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 2)

	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))
	require.NoError(t, domain.PublishSlotMigrationDeltaTestHelper(ctx, rdbs[0], domain.SlotMigrationDelta{
		CampaignID: campID,
		Amount:     250,
		SpendKey:   domain.BudgetCampaignKey(campID),
	}))
	require.NoError(t, svc.CatchUpDualWriteSlots(ctx, v))
	require.NoError(t, svc.ActivateSlotMapVersion(ctx, uuid.Nil, v))

	domain.AssertBudgetInvariant(t, ctx, pool, rdbs[2], campID)
	require.NoError(t, svc.VerifySlotMigrationR5(ctx))

	faultproof.Log(t, "slot_migration_dual_write", map[string]string{
		"subsystem":   "slot_migration",
		"fault":       "hot_slot_dual_write",
		"r5_ok":       "true",
		"dual_write":  "true",
		"campaign_id": campID.String(),
	})
}

func TestFault_SlotMigrationPGRewarmColdStart(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()
	rdbs := buildFourRedisShards(rdb, nil)
	svc, pool, ctx := setupSlotMigrationFault(t, rdbs)

	const slot int16 = 11
	campID := campaignIDForSlot(t, slot)
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "NoRedis", 1_000_000, "USD"))
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'no-redis', 500000, 100000, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campID), domain.ToUUID(customerID))
	require.NoError(t, err)

	mapRepo := domain.NewSlotMapRepo(pool)
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 1)
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))
	require.NoError(t, svc.ActivateSlotMapVersion(ctx, uuid.Nil, v))

	key := domain.BudgetCampaignKey(campID)
	val, err := rdbs[1].Get(ctx, key).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(400000), val)

	faultproof.Log(t, "slot_migration_pg_rewarm_cold_start", map[string]string{
		"subsystem":  "slot_migration",
		"pg_rewarm":  "true",
		"cold_start": "true",
	})
}
