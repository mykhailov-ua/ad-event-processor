package resilience_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/ingestion"
	"github.com/bidshard/ad-event-processor/internal/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const defaultTrackShards = 4

type multiShardTrackHarness struct {
	t           *testing.T
	ctx         context.Context
	cancel      context.CancelFunc
	Pool        *pgxpool.Pool
	ShardInfra  *testutil.RedisShardFaultInfra
	RDBs        []redis.UniversalClient
	Sharder     ingestion.Sharder
	CampaignIDs []uuid.UUID
	Registry    *ingestion.Registry
	Handler     *ingestion.AdsPacketHandler
}

type multiShardTrackOpts struct {
	NumShards        int
	StreamName       string
	FilterTimeoutMs  int
	CampaignChannel  string
	SkipPartitionMgr bool
	OnRedisClient    func(shard int, client *redis.Client)
}

func setupMultiShardTrackHarness(t *testing.T, opts multiShardTrackOpts) *multiShardTrackHarness {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test")
	}
	if opts.NumShards <= 0 {
		opts.NumShards = defaultTrackShards
	}
	if opts.StreamName == "" {
		opts.StreamName = "resilience-track-stream"
	}
	if opts.CampaignChannel == "" {
		opts.CampaignChannel = "campaigns:resilience-track"
	}
	if opts.FilterTimeoutMs <= 0 {
		opts.FilterTimeoutMs = 500
	}

	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	t.Cleanup(cleanupDB)

	shardInfra := testutil.SetupRedisShardsFault(t, opts.NumShards)
	rdbs := shardInfra.UniversalClients()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if opts.OnRedisClient != nil {
		for i, client := range shardInfra.Clients {
			opts.OnRedisClient(i, client)
		}
	}

	queries := db.New(pool)
	sharder := ingestion.NewStaticSlotSharder(opts.NumShards)
	campaignIDs := make([]uuid.UUID, opts.NumShards)
	for i := range campaignIDs {
		campaignIDs[i] = testutil.CampaignIDForShard(t, sharder, i)
	}

	customerID := uuid.New()
	_, err := pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		customerID, "Resilience Track Customer", 1_000_000_000,
	)
	require.NoError(t, err)

	for _, campaignID := range campaignIDs {
		_, err = pool.Exec(ctx,
			"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
			campaignID, "Resilience Track Campaign", "ACTIVE", customerID, 100_000_000,
		)
		require.NoError(t, err)
	}

	registry := testutil.NewAdsRegistry(t, queries)
	registry.ConfigureStaleMode(30 * time.Second)
	registry.SetBudgetWarmer(ingestion.NewBudgetCacheWarmer(rdbs, sharder))
	_, err = registry.Sync(ctx)
	require.NoError(t, err)

	cfg := &config.Config{
		EventBatchSize:        10,
		EventFlushMs:          100,
		MaxWorkers:            8,
		WriteTimeoutMs:        1000,
		FilterTimeoutMs:       opts.FilterTimeoutMs,
		MaxRequestBodySize:    1024 * 1024,
		StreamMaxLen:          100000,
		CampaignUpdateChannel: opts.CampaignChannel,
		RegistryStaleTTLSec:   30,
	}

	if !opts.SkipPartitionMgr {
		partManager := database.NewPartitionManager(pool, 7, 2)
		require.NoError(t, partManager.Run(ctx))
	}

	campaignRepo := ingestion.NewCampaignRepo(queries)
	unifiedFilter := ingestion.NewUnifiedFilter(
		rdbs,
		sharder,
		registry,
		campaignRepo,
		1000,
		time.Minute,
		45*time.Second,
		24*time.Hour,
		100_000,
		10_000,
		opts.StreamName,
		100000,
	)
	unifiedFilter.SetShardBreakers(shardInfra.Breakers)
	filterEngine := ingestion.NewFilterEngine(time.Duration(cfg.FilterTimeoutMs)*time.Millisecond, unifiedFilter)
	handler := ingestion.NewAdsPacketHandler(cfg, registry, filterEngine, pool, rdbs, sharder, cfg.FraudStreamName, nil)
	t.Cleanup(func() { handler.Stop(ctx) })

	return &multiShardTrackHarness{
		t:           t,
		ctx:         ctx,
		cancel:      cancel,
		Pool:        pool,
		ShardInfra:  shardInfra,
		RDBs:        rdbs,
		Sharder:     sharder,
		CampaignIDs: campaignIDs,
		Registry:    registry,
		Handler:     handler,
	}
}

func (h *multiShardTrackHarness) baselineLatency(shard int) time.Duration {
	h.t.Helper()
	_, elapsed := postClickCampaign(h.t, h.Handler, h.CampaignIDs[shard], uuid.NewString())
	return elapsed
}

func postClickCampaign(t *testing.T, h *ingestion.AdsPacketHandler, campaignID uuid.UUID, clickID string) (int, time.Duration) {
	t.Helper()
	status, _, elapsed := postClickCampaignFull(t, h, campaignID, clickID)
	return status, elapsed
}

func postClickCampaignBody(t *testing.T, h *ingestion.AdsPacketHandler, campaignID uuid.UUID, clickID string) (status int, body string) {
	t.Helper()
	status, body, _ = postClickCampaignFull(t, h, campaignID, clickID)
	return status, body
}

func postClickCampaignFull(t *testing.T, h *ingestion.AdsPacketHandler, campaignID uuid.UUID, clickID string) (int, string, time.Duration) {
	t.Helper()
	start := time.Now()
	payload := map[string]any{
		"campaign_id": campaignID,
		"type":        "click",
		"click_id":    clickID,
		"payload":     map[string]string{"fault": "resilience"},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	status, respBody := ingestion.PostTrackGnetJSON(h, body)
	return status, string(respBody), time.Since(start)
}

type redisProcessDelayHook struct {
	delay time.Duration
}

func (h *redisProcessDelayHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *redisProcessDelayHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.delay > 0 {
			time.Sleep(h.delay)
		}
		return next(ctx, cmd)
	}
}

func (h *redisProcessDelayHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if h.delay > 0 {
			time.Sleep(h.delay)
		}
		return next(ctx, cmds)
	}
}
