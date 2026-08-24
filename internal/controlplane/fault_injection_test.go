package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_DualOutboxWorkerRace(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{CampaignUpdateChannel: "campaigns:update-fault"}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	ctx := context.Background()

	require.NoError(t, svc.UpdateSettings(ctx, map[string]string{"rate_limit_per_min": "100"}))

	var eventID int64
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT id FROM outbox_events WHERE event_type = 'UPDATE_SETTINGS' ORDER BY id DESC LIMIT 1`,
	).Scan(&eventID))

	worker := NewOutboxWorker(svc)
	const workers = 4
	var wg sync.WaitGroup
	var totalProcessed atomic.Int32
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			n, err := worker.ProcessOutboxWithCount(ctx, 10)
			require.NoError(t, err)
			totalProcessed.Add(int32(n))
		}()
	}
	wg.Wait()

	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM outbox_events WHERE id = $1`, eventID).Scan(&status))
	assert.Equal(t, "PROCESSED", status)

	version, err := rdb.Get(ctx, "config:version").Int64()
	require.NoError(t, err)
	assert.Equal(t, eventID, version, "config:version must equal event id exactly once")

	val, err := rdb.HGet(ctx, "config:values", "rate_limit_per_min").Result()
	require.NoError(t, err)
	assert.Equal(t, "100", val)

	var processedCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE event_type = 'UPDATE_SETTINGS' AND status = 'PROCESSED'`,
	).Scan(&processedCount))
	assert.Equal(t, 1, processedCount)
	assert.Equal(t, int32(1), totalProcessed.Load(), "exactly one worker batch should mark the event processed")

	faultproof.Log(t, "outbox_worker_race", map[string]string{
		"subsystem":   "management_outbox",
		"workers":     "4",
		"processed":   "1",
		"baseline_ok": "true",
		"fault_type":  "concurrency_stress",
	})
}

func TestFault_PGDeadlockRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	ctx := context.Background()
	id1, id2 := uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES
		($1, 'Deadlock A', 100000000, 'USD'),
		($2, 'Deadlock B', 100000000, 'USD')`, domain.ToUUID(id1), domain.ToUUID(id2))
	require.NoError(t, err)

	start := make(chan struct{})
	var wg sync.WaitGroup
	var err1, err2 error
	wg.Add(2)

	run := func(first, second uuid.UUID) {
		defer wg.Done()
		<-start
		e := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			if _, e := tx.Exec(ctx, `SELECT 1 FROM customers WHERE id = $1 FOR UPDATE`, domain.ToUUID(first)); e != nil {
				return e
			}
			time.Sleep(150 * time.Millisecond)
			_, e := tx.Exec(ctx, `SELECT 1 FROM customers WHERE id = $1 FOR UPDATE`, domain.ToUUID(second))
			return e
		})
		if first == id1 {
			err1 = e
		} else {
			err2 = e
		}
	}

	go run(id1, id2)
	go run(id2, id1)
	close(start)
	wg.Wait()

	assert.True(t, isDeadlock(err1) || isDeadlock(err2), "expected deadlock on one session, got: %v / %v", err1, err2)

	var bal1, bal2 int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT balance FROM customers WHERE id = $1`, domain.ToUUID(id1)).Scan(&bal1))
	require.NoError(t, pool.QueryRow(ctx, `SELECT balance FROM customers WHERE id = $1`, domain.ToUUID(id2)).Scan(&bal2))
	assert.Equal(t, int64(100_000_000), bal1)
	assert.Equal(t, int64(100_000_000), bal2)

	faultproof.Log(t, "postgres_deadlock_recovery", map[string]string{
		"subsystem":   "management_ledger",
		"deadlock":    "true",
		"balances_ok": "true",
		"fault_type":  "concurrency_stress",
	})
}

func TestFault_RedisSlowEventuallySucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	slowRDB := &slowRedisClient{UniversalClient: rdb, delay: 50 * time.Millisecond}
	cfg := &config.Config{CampaignUpdateChannel: "campaigns:update-slow"}
	svc := newBareService(t, pool, []redis.UniversalClient{slowRDB}, cfg)
	ctx := context.Background()

	require.NoError(t, svc.UpdateSettings(ctx, map[string]string{"emergency_breaker": "false"}))

	start := time.Now()
	worker := NewOutboxWorker(svc)
	processed, err := worker.ProcessOutboxWithCount(ctx, 10)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)

	val, err := slowRDB.HGet(ctx, "config:values", "emergency_breaker").Result()
	require.NoError(t, err)
	assert.Equal(t, "false", val)

	faultproof.Log(t, "redis_slow_degradation", map[string]string{
		"subsystem":   "management_outbox",
		"delay_ms":    "50",
		"processed":   "1",
		"baseline_ok": "true",
		"fault_type":  "latency_injection",
	})
}

func TestFault_ScheduleTickRace(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, nil)
	ctx := context.Background()

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Schedule Race", 500_000_000, "USD"))

	start := time.Now().Add(2 * time.Hour)
	end := time.Now().Add(48 * time.Hour)
	spec := testCampaignSpec(custID, "Future Race", 50_000_000, "sched-race-idem")
	spec.StartAt = &start
	spec.EndAt = &end
	spec.DaypartHours = []int16{}
	campID, err := svc.CreateCampaign(ctx, spec)
	require.NoError(t, err)

	camp, err := svc.GetCampaignRow(ctx, campID)
	require.NoError(t, err)
	assert.Equal(t, db.CampaignStatusTypePAUSED, camp.Status)

	const workers = 8
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			_ = svc.ProcessScheduleTick(ctx)
		}()
	}
	wg.Wait()

	camp, err = svc.GetCampaignRow(ctx, campID)
	require.NoError(t, err)
	assert.Equal(t, db.CampaignStatusTypePAUSED, camp.Status)

	var historyCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM campaign_status_history WHERE campaign_id = $1`, domain.ToUUID(campID),
	).Scan(&historyCount))
	assert.Equal(t, 1, historyCount, "concurrent ticks must not duplicate status transitions")

	faultproof.Log(t, "schedule_tick_race", map[string]string{
		"subsystem":    "management_schedule",
		"workers":      "8",
		"history_rows": "1",
		"baseline_ok":  "true",
		"fault_type":   "concurrency_stress",
	})
}

func TestFault_ConcurrentBalanceDepletion(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, nil)
	ctx := context.Background()

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Concurrent", 500_000_000, "USD"))

	const workers = 10
	campaignBudget := int64(100_000_000)
	var wg sync.WaitGroup
	results := make(chan error, workers)
	wg.Add(workers)
	for i := range workers {
		go func(idx int) {
			defer wg.Done()
			_, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
				CustomerID:       customerID,
				Name:             "Camp",
				BudgetLimitMicro: campaignBudget,
				PacingMode:       string(db.PacingModeTypeASAP),
				Timezone:         "UTC",
				FreqWindow:       86400,
				DaypartHours:     []int16{},
				IdempotencyKey:   fmt.Sprintf("idem-%d", idx),
			})
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)

	var successCount, failureCount int
	for err := range results {
		if err == nil {
			successCount++
		} else {
			failureCount++
		}
	}
	assert.Equal(t, 5, successCount)
	assert.Equal(t, 5, failureCount)

	var balance int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT balance FROM customers WHERE id = $1`, domain.ToUUID(customerID)).Scan(&balance))
	assert.Equal(t, int64(0), balance)

	faultproof.Log(t, "concurrent_balance_depletion", map[string]string{
		"subsystem":     "management_ledger",
		"workers":       "10",
		"success":       "5",
		"failure":       "5",
		"final_balance": "0",
		"fault_type":    "concurrency_stress",
	})
}

func TestFault_AutoscaleInsufficientTotalBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		CampaignUpdateChannel:       "test:autoscale-insufficient",
		AutoscaleHighCTRThreshold:   0.015,
		AutoscaleMinImpressions:     100,
		AutoscaleLowCTRThreshold:    0.005,
		AutoscaleMinRemainingBudget: 5_000_000,
		AutoscaleShiftAmount:        10_000_000,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	ctx := context.Background()

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Autoscale Edge", 1_000_000_000, "USD"))

	lowCTR, err := svc.CreateCampaign(ctx, testCampaignSpec(customerID, "Low CTR", 100_000_000, "low-insuf"))
	require.NoError(t, err)
	highCTR, err := svc.CreateCampaign(ctx, testCampaignSpec(customerID, "High CTR", 100_000_000, "high-insuf"))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO campaign_stats (campaign_id, date, impressions_count, clicks_count, conversions_count) VALUES
		($1, CURRENT_DATE, 1000, 2, 0),
		($2, CURRENT_DATE, 500, 15, 0)`,
		domain.ToUUID(lowCTR), domain.ToUUID(highCTR))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `UPDATE campaigns SET current_spend = $1 WHERE id = $2`, int64(94_000_000), domain.ToUUID(lowCTR))
	require.NoError(t, err)

	queries := db.New(pool)
	syncWorker := domain.NewSyncWorker(rdb, domain.NewCampaignRepo(queries), domain.NewCustomerRepo(queries), 100*time.Millisecond, 0, nil, 0)
	_, err = pool.Exec(ctx, `DELETE FROM outbox_events`)
	require.NoError(t, err)

	err = svc.AutoscaleBudgets(ctx, []*domain.SyncWorker{syncWorker})
	require.NoError(t, err)

	var limitLow, limitHigh, spend int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT budget_limit FROM campaigns WHERE id = $1`, domain.ToUUID(lowCTR)).Scan(&limitLow))
	require.NoError(t, pool.QueryRow(ctx, `SELECT budget_limit FROM campaigns WHERE id = $1`, domain.ToUUID(highCTR)).Scan(&limitHigh))
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`, domain.ToUUID(lowCTR)).Scan(&spend))

	assert.Equal(t, int64(100_000_000), limitLow, "budget must not decrease when spend would exceed new limit")
	assert.Equal(t, int64(100_000_000), limitHigh)
	assert.LessOrEqual(t, spend, limitLow)

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type = 'CREATE_CAMPAIGN'`).Scan(&outboxCount))
	assert.Equal(t, 0, outboxCount)

	faultproof.Log(t, "autoscale_insufficient_budget", map[string]string{
		"subsystem":   "management_autoscale",
		"limit_low":   "100000000",
		"limit_high":  "100000000",
		"baseline_ok": "true",
		"fault_type":  "edge_case",
	})
}

func TestFault_DualOutboxWorkerManyEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{CampaignUpdateChannel: "campaigns:update-fault-batch"}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	ctx := context.Background()

	const eventCount = 20
	for range eventCount {
		payload, err := json.Marshal(CampaignPayload{
			CampaignID:  uuid.New().String(),
			BudgetLimit: 10_000_000,
		})
		require.NoError(t, err)
		_, err = db.New(pool).CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "CREATE_CAMPAIGN",
			Payload:   payload,
		})
		require.NoError(t, err)
	}

	worker := NewOutboxWorker(svc)
	var wg sync.WaitGroup
	var total atomic.Int32
	wg.Add(3)
	for range 3 {
		go func() {
			defer wg.Done()
			for {
				n, err := worker.ProcessOutboxWithCount(ctx, 5)
				require.NoError(t, err)
				if n == 0 {
					return
				}
				total.Add(int32(n))
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(eventCount), total.Load())

	var pending int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE status = 'PENDING' AND event_type = 'CREATE_CAMPAIGN'`,
	).Scan(&pending))
	assert.Equal(t, 0, pending)

	var processed int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE status = 'PROCESSED' AND event_type = 'CREATE_CAMPAIGN'`,
	).Scan(&processed))
	assert.Equal(t, eventCount, processed)

	faultproof.Log(t, "outbox_worker_many_events", map[string]string{
		"subsystem":   "management_outbox",
		"events":      itoaFault(eventCount),
		"workers":     "3",
		"pending":     "0",
		"processed":   itoaFault(processed),
		"baseline_ok": "true",
		"fault_type":  "concurrency_stress",
	})
}
