package controlplane

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_OutboxBudgetFreezePriority(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{CampaignUpdateChannel: "campaigns:update-freeze"}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	ctx := context.Background()

	const pacingBacklog = 200
	pacingPayload, err := json.Marshal(campaignPacingPayload{
		CampaignID: uuid.New().String(),
		PacingMode: "even",
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload)
		SELECT 'UPDATE_CAMPAIGN_PACING', $1::jsonb
		FROM generate_series(1, $2)`, pacingPayload, pacingBacklog)
	require.NoError(t, err)

	campID := uuid.New()
	freezePayload, err := json.Marshal(CampaignPayload{CampaignID: campID.String()})
	require.NoError(t, err)
	_, err = rdb.Set(ctx, "budget:campaign:"+campID.String(), 5_000_000, 0).Result()
	require.NoError(t, err)

	var freezeID int64
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload) VALUES ('BUDGET_FREEZE', $1) RETURNING id`,
		freezePayload).Scan(&freezeID))

	worker := NewOutboxWorker(svc)
	processed, err := worker.ProcessOutboxWithCount(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, processed)

	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM outbox_events WHERE id = $1`, freezeID).Scan(&status))
	require.Equal(t, "PROCESSED", status)

	exists, err := rdb.Exists(ctx, domain.BudgetFrozenRedisKey(campID)).Result()
	require.NoError(t, err)
	require.Equal(t, int64(1), exists)

	faultproof.Log(t, "outbox_budget_freeze_priority", map[string]string{
		"subsystem":      "management_outbox",
		"pacing_backlog": strconv.Itoa(pacingBacklog),
		"freeze_first":   "true",
	})
}

func TestFault_SlotMigrationFence(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	campID := uuid.New()
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'mig-fence', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'mig-fence', 10000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campID), domain.ToUUID(customerID))
	require.NoError(t, err)

	require.NoError(t, domain.BumpMigrationFences(ctx, pool, rdb, []uuid.UUID{campID}))
	require.NoError(t, rdb.Set(ctx, domain.BudgetCampaignKey(campID), 10_000_000, 0).Err())

	f := testutil.NewLuaUnifiedFilter(rdb, nil)
	require.NoError(t, f.PreloadScripts(ctx))

	const workers = 32
	var wg sync.WaitGroup
	var fenced, debited int64
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			evt := &domain.Event{
				Type:       "click",
				CampaignID: campID,
				ClickID:    uuid.NewString(),
				IP:         "203.0.113.60",
				UserID:     "mig-fence",
			}
			checkCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			err := f.Check(checkCtx, evt)
			if err != nil {
				if err == domain.ErrMigrationFenced {
					fenced++
				}
				return
			}
			debited++
		}()
	}
	wg.Wait()

	require.Equal(t, int64(workers), fenced)
	require.Equal(t, int64(0), debited)

	domain.AssertBudgetInvariant(t, ctx, pool, rdb, campID)

	faultproof.Log(t, "slot_migration_fence", map[string]string{
		"subsystem":         "slot_migration",
		"workers":           strconv.Itoa(workers),
		"fenced":            strconv.FormatInt(fenced, 10),
		"debited":           "0",
		"budget_consistent": "true",
	})
}
