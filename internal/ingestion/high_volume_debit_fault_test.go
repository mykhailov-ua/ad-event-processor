package ingestion

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func seedFaultCampaignHighVolume(t *testing.T, infra *adsFaultInfra, registry *Registry) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	pm := database.NewPartitionManager(infra.Pool, 7, 1)
	require.NoError(t, pm.Run(ctx))

	customerID := uuid.New()
	_, err := infra.Pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		customerID, "HV Customer", 1_000_000_000)
	require.NoError(t, err)

	campaignID := uuid.New()
	_, err = infra.Pool.Exec(ctx,
		`INSERT INTO campaigns (id, name, status, customer_id, budget_limit, behavior_flags)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		campaignID, "HV Campaign", "ACTIVE", customerID, 50_000_000,
		int(domain.BehaviorHighVolumeDebit),
	)
	require.NoError(t, err)

	_, _ = registry.Sync(ctx)
	return campaignID
}

func TestFault_HighVolumeDebit_subShardBudgetInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	const iterations = 2_000
	ctx := context.Background()
	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	registry := newFaultRegistry(t, infra.Queries)
	campaignID := seedFaultCampaignHighVolume(t, infra, registry)
	camp, ok := registry.GetCampaign(campaignID)
	require.True(t, ok)
	require.Equal(t, domain.BehaviorHighVolumeDebit, camp.BehaviorFlags&domain.BehaviorHighVolumeDebit)

	rdbs := []redis.UniversalClient{infra.Redis, infra.Redis, infra.Redis, infra.Redis}
	sharder := NewStaticSlotSharder(4)

	f := NewUnifiedFilter(
		rdbs,
		sharder,
		registry,
		NewCampaignRepo(infra.Queries),
		0,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"ad:events:stream",
		100_000,
	)
	f.SetLuaFastPathEnabled(true)
	f.SetTTCMin(0)
	require.NoError(t, f.PreloadScripts(ctx))
	require.NoError(t, infra.Redis.Set(ctx, camp.BudgetCampaignKey, 50_000_000, 0).Err())

	seenSub := make(map[int]struct{})
	for i := range iterations {
		userID := fmt.Sprintf("hv-user-%d", i)
		sub := debitSubSlot(camp, userID, "")
		seenSub[sub] = struct{}{}

		evt := &domain.Event{
			Type:       "impression",
			CampaignID: campaignID,
			ClickID:    fmt.Sprintf("hv-%d", i),
			IP:         "203.0.113.210",
			UserID:     userID,
		}
		checkCtx := attachFilterDeadline(ctx, 2*time.Second)
		require.NoError(t, f.Check(checkCtx, evt))
	}
	require.GreaterOrEqual(t, len(seenSub), 2, "high-volume debits must spread across sub-slots")

	campaignRepo := NewCampaignRepoWithDB(infra.Pool, infra.Queries)
	customerRepo := NewCustomerRepoWithDB(infra.Pool, infra.Queries)
	worker := NewSyncWorker(infra.Redis, campaignRepo, customerRepo, time.Hour, 0, nil, 0)
	for range 3 {
		worker.SyncAll(ctx)
	}

	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, campaignID)

	faultproof.Log(t, "high_volume_debit_budget_invariant", map[string]string{
		"iterations": fmt.Sprintf("%d", iterations),
		"sub_slots":  fmt.Sprintf("%d", len(seenSub)),
	})
}
