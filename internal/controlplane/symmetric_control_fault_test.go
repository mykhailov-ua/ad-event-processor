package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/testutil"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_Shard0PubsubDown(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	const numShards = 4

	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	defer cleanupDB()

	shardInfra := testutil.SetupRedisShardsFault(t, numShards)
	rdbs := shardInfra.UniversalClients()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queries := db.New(pool)
	sharder := domain.NewStaticSlotSharder(numShards)
	campaignIDs := make([]uuid.UUID, numShards)
	for i := range campaignIDs {
		campaignIDs[i] = testutil.CampaignIDForShard(t, sharder, i)
	}

	customerID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)", customerID, "Symmetric Control Customer", 1_000_000_000)
	require.NoError(t, err)

	for _, campaignID := range campaignIDs {
		_, err = pool.Exec(ctx,
			"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
			campaignID, "Symmetric Control Campaign", "ACTIVE", customerID, 100_000_000,
		)
		require.NoError(t, err)
	}

	registry := testutil.NewAdsRegistry(t, queries)
	registry.ConfigureStaleMode(30 * time.Second)
	testutil.AttachBudgetCacheWarmer(registry, rdbs, sharder)
	_, err = registry.Sync(ctx)
	require.NoError(t, err)

	channel := "campaigns:symmetric-control"
	registry.StartWatchShards(ctx, rdbs, channel)
	registry.StartEpochPoll(ctx, rdbs, 200*time.Millisecond)

	cfg := &config.Config{
		CampaignUpdateChannel: channel,
		RegistryStaleTTLSec:   30,
		RegistryPollMs:        200,
	}

	svc := NewService(context.Background(), pool, rdbs, sharder, cfg)
	defer svc.Close()

	testutil.StopRedisShardContainer(t, shardInfra.Containers[0])
	require.Eventually(t, func() bool {
		return shardInfra.Clients[0].Ping(ctx).Err() != nil
	}, 15*time.Second, 100*time.Millisecond)

	for shard := 1; shard < numShards; shard++ {
		campaignID := campaignIDs[shard]
		require.NoError(t, svc.publishCampaignUpdate(ctx, campaignID.String()))
	}

	require.Eventually(t, func() bool {
		for shard := 1; shard < numShards; shard++ {
			epoch, err := rdbs[shard].Get(ctx, domain.CampaignEpochKey).Int64()
			if err != nil || epoch < 1 {
				return false
			}
		}
		return true
	}, 10*time.Second, 200*time.Millisecond, "epoch must bump on shards 1-3")

	require.Eventually(t, func() bool {
		for shard := 1; shard < numShards; shard++ {
			_, ok := registry.GetCampaign(campaignIDs[shard])
			if !ok {
				return false
			}
		}
		return true
	}, 10*time.Second, 200*time.Millisecond, "registry must reload from non-zero shards")

	assert.False(t, registry.IsStaleMode(), "stale-serve must not engage while shards 1-3 pubsub healthy")

	faultproof.Log(t, "symmetric_control_epoch", map[string]string{
		"shards": "4",
		"status": "shards_1_3_continue",
	})
}
