package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func NewBareServiceForTest(t *testing.T, pool *pgxpool.Pool, redisShards []redis.UniversalClient, cfg *config.Config) *Service {
	t.Helper()
	if cfg == nil {
		cfg = &config.Config{}
	}
	shardCount := len(redisShards)
	if shardCount == 0 {
		shardCount = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	svc := &Service{
		redisShards: redisShards,
		sharder:     domain.NewStaticSlotSharder(shardCount),
		cfg:         cfg,
		ctx:         ctx,
		cancel:      cancel,
	}
	svc.SetPool(pool)
	if cfg.MultiRegionEnabled {
		svc.leaseWorker = NewOperationLeaseWorker(svc)
	}
	t.Cleanup(func() {
		cancel()
		svc.Close()
	})
	return svc
}

func NewRedisHostForOutboxTest(redisShards []redis.UniversalClient) *Service {
	shardCount := len(redisShards)
	if shardCount == 0 {
		shardCount = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		redisShards: redisShards,
		sharder:     domain.NewStaticSlotSharder(shardCount),
		cfg:         &config.Config{},
		ctx:         ctx,
		cancel:      cancel,
	}
}
