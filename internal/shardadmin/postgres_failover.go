package shardadmin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	bserver "ad-event-processor/internal/broker"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/pgfailover"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStalePgFencingEpoch = pgfailover.ErrStalePgFencingEpoch

type PostgresFailoverRuntime struct {
	coord            *pgfailover.Coordinator
	subscriber       *pgfailover.Subscriber
	extraSubscribers []*pgfailover.Subscriber
	fencing          *pgfailover.FencingGate
	mu               sync.Mutex
	activePool       *pgxpool.Pool
}

func (rt *PostgresFailoverRuntime) Fencing() *pgfailover.FencingGate {
	if rt == nil {
		return nil
	}
	return rt.fencing
}

// StartPostgresFailover: Redis-coordinated PG role change; FencingGate blocks stale writers after promote.
func StartPostgresFailover(ctx context.Context, host PostgresFailoverHost) *PostgresFailoverRuntime {
	if host == nil {
		return nil
	}
	cfg := host.FailoverConfig()
	if cfg == nil || !cfg.PostgresFailoverEnabled || len(host.RedisShards()) == 0 {
		return nil
	}
	redisClient := host.ControlRedis()
	rt := &PostgresFailoverRuntime{
		fencing: pgfailover.NewFencingGate(redisClient),
	}
	host.SetPgFencing(rt.fencing)

	reconnect := func(pool *pgxpool.Pool) {
		rt.mu.Lock()
		old := rt.activePool
		rt.activePool = pool
		rt.mu.Unlock()
		if old != nil && old != host.Pool() {
			old.Close()
		}
		host.SetPool(pool)
	}

	subCfg := pgfailover.SubscriberConfig{
		MaxConns: cfg.DBTrackerMaxConns,
		MinConns: cfg.DBMinConns,
		Interval: time.Duration(cfg.PostgresFailoverPollMs) * time.Millisecond,
	}
	rt.subscriber = pgfailover.NewSubscriber(redisClient, rt.fencing, reconnect, subCfg)
	rt.subscriber.Start(ctx)
	for _, shardRDB := range host.RedisShards() {
		if shardRDB == nil || shardRDB == redisClient {
			continue
		}
		extra := pgfailover.NewSubscriber(shardRDB, rt.fencing, reconnect, subCfg)
		extra.Start(ctx)
		rt.extraSubscribers = append(rt.extraSubscribers, extra)
	}

	runCoordinator := cfg.PostgresFailoverCoordinator || cfg.MultiRegionGlobal() || !cfg.MultiRegionEnabled
	if runCoordinator && cfg.PostgresStandbyDSN != "" {
		coordCfg := pgfailover.Config{
			NodeID:         cfg.NodeID,
			RedisURL:       cfg.PostgresFailoverRedisURL,
			PrimaryDSN:     string(cfg.PostgresPrimaryDSN),
			StandbyDSN:     string(cfg.PostgresStandbyDSN),
			HealthInterval: time.Duration(cfg.PostgresFailoverHealthMs) * time.Millisecond,
			FailThreshold:  cfg.PostgresFailoverFailThreshold,
			MaxConns:       cfg.DBTrackerMaxConns,
			MinConns:       cfg.DBMinConns,
			Coord: bserver.CoordConfig{
				LeaseTTL:           time.Duration(cfg.PostgresFailoverLeaseSec) * time.Second,
				Interval:           time.Duration(cfg.PostgresFailoverCoordMs) * time.Millisecond,
				RenewFailThreshold: 2,
				DebounceWindow:     time.Second,
			},
		}
		promoter := buildPgFailoverPromoter(cfg, reconnect)
		coord, err := pgfailover.NewCoordinator(coordCfg, promoter, pgfailover.PingDSN(string(cfg.PostgresPrimaryDSN)))
		if err != nil {
			slog.Warn("pg failover coordinator init failed", "error", err)
			return rt
		}
		rt.coord = coord
		coord.Start(ctx)
		slog.Info("pg failover coordinator started", "node_id", cfg.NodeID)
	}
	return rt
}

func BuildPgFailoverPromoter(cfg *config.Config, reconnect func(*pgxpool.Pool)) pgfailover.Promoter {
	return buildPgFailoverPromoter(cfg, reconnect)
}

func buildPgFailoverPromoter(cfg *config.Config, reconnect func(*pgxpool.Pool)) pgfailover.Promoter {
	if cfg.PostgresPromoteCommand == "" && !cfg.PostgresFailoverSnapshotSync {
		return pgfailover.PromoteFunc(func(ctx context.Context) (string, error) {
			pool, err := database.Connect(ctx, string(cfg.PostgresStandbyDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
			if err != nil {
				return "", err
			}
			if err := pool.Ping(ctx); err != nil {
				pool.Close()
				return "", err
			}
			if err := pgfailover.EnsureWritablePrimary(ctx, pool); err != nil {
				pool.Close()
				return "", fmt.Errorf("pg failover promote: standby not writable (set PostgresPromoteCommand or promote out of band): %w", err)
			}
			reconnect(pool)
			return string(cfg.PostgresStandbyDSN), nil
		})
	}
	return &pgfailover.StandbyPromoter{
		StandbyDSN:     string(cfg.PostgresStandbyDSN),
		PrimaryDSN:     string(cfg.PostgresPrimaryDSN),
		PromoteCommand: cfg.PostgresPromoteCommand,
		SnapshotSync:   cfg.PostgresFailoverSnapshotSync,
		SyncConfig:     pgfailover.SnapshotSyncConfig{PageSize: cfg.PostgresFailoverSyncPageSize},
		MaxConns:       cfg.DBTrackerMaxConns,
		MinConns:       cfg.DBMinConns,
		OnReconnect:    reconnect,
	}
}

func AuditLedgerDuplicatesSinceFailover(ctx context.Context, host PostgresFailoverHost) (int, error) {
	if host == nil || host.Pool() == nil {
		return 0, nil
	}
	cfg := host.FailoverConfig()
	if cfg == nil {
		return 0, nil
	}
	auditCfg := pgfailover.AuditConfig{Window: time.Duration(cfg.PostgresFailoverAuditWindowSec) * time.Second}
	return pgfailover.CountLedgerDuplicatesSinceNow(ctx, host.Pool(), auditCfg)
}

func (rt *PostgresFailoverRuntime) ClosePostgresFailover() {
	if rt == nil {
		return
	}
	if rt.coord != nil {
		rt.coord.Stop()
	}
	if rt.subscriber != nil {
		rt.subscriber.Stop()
	}
	for _, sub := range rt.extraSubscribers {
		if sub != nil {
			sub.Stop()
		}
	}
}

func (rt *PostgresFailoverRuntime) CurrentDSN() string {
	if rt == nil || rt.subscriber == nil {
		return ""
	}
	return rt.subscriber.CurrentDSN()
}

func RequirePgFencing(ctx context.Context, host PostgresFailoverHost) error {
	if host == nil || host.PgFencing() == nil || len(host.RedisShards()) == 0 {
		return nil
	}
	if err := host.PgFencing().Refresh(ctx); err != nil {
		return err
	}
	reader := newPostgresFailoverShardReader(host.RedisShards())
	_, epoch, err := reader.activeDSN(ctx)
	if err != nil {
		return err
	}
	return host.PgFencing().Validate(epoch)
}
