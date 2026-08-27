package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/database"
	bserver "ad-event-processor/pkg/broker/server"
	"ad-event-processor/pkg/pgfailover"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStalePgFencingEpoch = pgfailover.ErrStalePgFencingEpoch

type PgFailoverRuntime struct {
	coord            *pgfailover.Coordinator
	subscriber       *pgfailover.Subscriber
	extraSubscribers []*pgfailover.Subscriber
	fencing          *pgfailover.FencingGate
	mu               sync.Mutex
	activePool       *pgxpool.Pool
}

func (s *Service) StartPostgresFailover(ctx context.Context) *PgFailoverRuntime {
	if s == nil || s.cfg == nil || !s.cfg.PostgresFailoverEnabled || len(s.redisShards) == 0 {
		return nil
	}
	redisClient := PickHealthyControlShard(s.redisShards)
	rt := &PgFailoverRuntime{
		fencing: pgfailover.NewFencingGate(redisClient),
	}
	s.pgFencing = rt.fencing

	reconnect := func(pool *pgxpool.Pool) {
		rt.mu.Lock()
		old := rt.activePool
		rt.activePool = pool
		rt.mu.Unlock()
		if old != nil && old != s.pool {
			old.Close()
		}
		s.SetPool(pool)
	}

	subCfg := pgfailover.SubscriberConfig{
		MaxConns: s.cfg.DBTrackerMaxConns,
		MinConns: s.cfg.DBMinConns,
		Interval: time.Duration(s.cfg.PostgresFailoverPollMs) * time.Millisecond,
	}
	rt.subscriber = pgfailover.NewSubscriber(redisClient, rt.fencing, reconnect, subCfg)
	rt.subscriber.Start(ctx)
	for _, shardRDB := range s.redisShards {
		if shardRDB == nil || shardRDB == redisClient {
			continue
		}
		extra := pgfailover.NewSubscriber(shardRDB, rt.fencing, reconnect, subCfg)
		extra.Start(ctx)
		rt.extraSubscribers = append(rt.extraSubscribers, extra)
	}

	runCoordinator := s.cfg.PostgresFailoverCoordinator || s.cfg.MultiRegionGlobal() || !s.cfg.MultiRegionEnabled
	if runCoordinator && s.cfg.PostgresStandbyDSN != "" {
		coordCfg := pgfailover.Config{
			NodeID:         s.cfg.NodeID,
			RedisURL:       s.cfg.PostgresFailoverRedisURL,
			PrimaryDSN:     string(s.cfg.PostgresPrimaryDSN),
			StandbyDSN:     string(s.cfg.PostgresStandbyDSN),
			HealthInterval: time.Duration(s.cfg.PostgresFailoverHealthMs) * time.Millisecond,
			FailThreshold:  s.cfg.PostgresFailoverFailThreshold,
			MaxConns:       s.cfg.DBTrackerMaxConns,
			MinConns:       s.cfg.DBMinConns,
			Coord: bserver.CoordConfig{
				LeaseTTL:           time.Duration(s.cfg.PostgresFailoverLeaseSec) * time.Second,
				Interval:           time.Duration(s.cfg.PostgresFailoverCoordMs) * time.Millisecond,
				RenewFailThreshold: 2,
				DebounceWindow:     time.Second,
			},
		}
		promoter := buildPgFailoverPromoter(s, reconnect)
		coord, err := pgfailover.NewCoordinator(coordCfg, promoter, pgfailover.PingDSN(string(s.cfg.PostgresPrimaryDSN)))
		if err != nil {
			slog.Warn("pg failover coordinator init failed", "error", err)
			return rt
		}
		rt.coord = coord
		coord.Start(ctx)
		slog.Info("pg failover coordinator started", "node_id", s.cfg.NodeID)
	}
	return rt
}

func buildPgFailoverPromoter(s *Service, reconnect func(*pgxpool.Pool)) pgfailover.Promoter {
	if s.cfg.PostgresPromoteCommand == "" && !s.cfg.PostgresFailoverSnapshotSync {
		return pgfailover.PromoteFunc(func(ctx context.Context) (string, error) {
			pool, err := database.Connect(ctx, string(s.cfg.PostgresStandbyDSN), s.cfg.DBTrackerMaxConns, s.cfg.DBMinConns)
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
			return string(s.cfg.PostgresStandbyDSN), nil
		})
	}
	return &pgfailover.StandbyPromoter{
		StandbyDSN:     string(s.cfg.PostgresStandbyDSN),
		PrimaryDSN:     string(s.cfg.PostgresPrimaryDSN),
		PromoteCommand: s.cfg.PostgresPromoteCommand,
		SnapshotSync:   s.cfg.PostgresFailoverSnapshotSync,
		SyncConfig:     pgfailover.SnapshotSyncConfig{PageSize: s.cfg.PostgresFailoverSyncPageSize},
		MaxConns:       s.cfg.DBTrackerMaxConns,
		MinConns:       s.cfg.DBMinConns,
		OnReconnect:    reconnect,
	}
}

func (s *Service) AuditLedgerDuplicatesSinceFailover(ctx context.Context) (int, error) {
	if s == nil || s.pool == nil || s.cfg == nil {
		return 0, nil
	}
	cfg := pgfailover.AuditConfig{Window: time.Duration(s.cfg.PostgresFailoverAuditWindowSec) * time.Second}
	return pgfailover.CountLedgerDuplicatesSinceNow(ctx, s.pool, cfg)
}

func (rt *PgFailoverRuntime) ClosePostgresFailover() {
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

func (rt *PgFailoverRuntime) CurrentDSN() string {
	if rt == nil || rt.subscriber == nil {
		return ""
	}
	return rt.subscriber.CurrentDSN()
}

func (s *Service) requirePgFencing(ctx context.Context) error {
	if s == nil || s.pgFencing == nil || len(s.redisShards) == 0 {
		return nil
	}
	if err := s.pgFencing.Refresh(ctx); err != nil {
		return err
	}
	reader := newPostgresFailoverShardReader(s.redisShards)
	_, epoch, err := reader.activeDSN(ctx)
	if err != nil {
		return err
	}
	return s.pgFencing.Validate(epoch)
}
