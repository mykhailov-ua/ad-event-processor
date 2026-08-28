package shardadmin

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/dedup"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type leaseTestHost struct {
	pool        *pgxpool.Pool
	redisShards []redis.UniversalClient
	cfg         *config.Config
}

func (h *leaseTestHost) Pool() *pgxpool.Pool {
	return h.pool
}

func (h *leaseTestHost) RedisShards() []redis.UniversalClient {
	return h.redisShards
}

func (h *leaseTestHost) ControlRedis() redis.UniversalClient {
	for _, shard := range h.redisShards {
		if shard != nil {
			return shard
		}
	}
	return nil
}

func (h *leaseTestHost) WithPostgresHigh(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (h *leaseTestHost) WithPostgresLow(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (h *leaseTestHost) DedupAdapter(ctx context.Context) *dedup.Adapter {
	if h.pool == nil || h.cfg == nil {
		return nil
	}
	epoch := dedup.LoadRoutingEpoch(ctx, h.pool)
	return dedup.NewAdapter(h.pool, h.cfg.RegionCode, epoch)
}

func (h *leaseTestHost) LeaseWorkerConfig() LeaseWorkerConfig {
	cfg := LeaseWorkerConfig{NodeRole: "management"}
	if h.cfg == nil {
		return cfg
	}
	cfg.NodeID = h.cfg.NodeID
	if h.cfg.NodeRole != "" {
		cfg.NodeRole = h.cfg.NodeRole
	}
	cfg.RegionCode = int16(h.cfg.RegionCode)
	cfg.OpLeaseTimeoutSec = h.cfg.OpLeaseTimeoutSec
	cfg.OpLeaseMaxRenewals = int32(h.cfg.OpLeaseMaxRenewals)
	cfg.OpLeaseFencingDir = h.cfg.OpLeaseFencingDir
	return cfg
}

func (h *leaseTestHost) LeaseReplicaNodes() []string {
	if h.cfg != nil && h.cfg.NodeID != "" {
		return []string{h.cfg.NodeID}
	}
	return []string{"management"}
}

func (h *leaseTestHost) SetPool(pool *pgxpool.Pool) {
	h.pool = pool
}

func newLeaseTestHost(t *testing.T, pool *pgxpool.Pool, redisShards []redis.UniversalClient, cfg *config.Config) LeaseHost {
	t.Helper()
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &leaseTestHost{pool: pool, redisShards: redisShards, cfg: cfg}
}
