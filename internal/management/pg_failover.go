package management

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"espx/internal/database"
	bserver "espx/pkg/broker/server"
	"espx/pkg/pgfailover"

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

func (s *Service) StartPgFailover(ctx context.Context) *PgFailoverRuntime {
	if s == nil || s.cfg == nil || !s.cfg.PgFailoverEnabled || len(s.rdbs) == 0 {
		return nil
	}
	rdb := PickHealthyControlShard(s.rdbs)
	rt := &PgFailoverRuntime{
		fencing: pgfailover.NewFencingGate(rdb),
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
		Interval: time.Duration(s.cfg.PgFailoverPollMs) * time.Millisecond,
	}
	rt.subscriber = pgfailover.NewSubscriber(rdb, rt.fencing, reconnect, subCfg)
	rt.subscriber.Start(ctx)
	for _, shardRDB := range s.rdbs {
		if shardRDB == nil || shardRDB == rdb {
			continue
		}
		extra := pgfailover.NewSubscriber(shardRDB, rt.fencing, reconnect, subCfg)
		extra.Start(ctx)
		rt.extraSubscribers = append(rt.extraSubscribers, extra)
	}

	runCoordinator := s.cfg.PgFailoverCoordinator || s.cfg.MultiRegionGlobal() || !s.cfg.MultiRegionEnabled
	if runCoordinator && s.cfg.PgStandbyDSN != "" {
		coordCfg := pgfailover.Config{
			NodeID:         s.cfg.NodeID,
			RedisURL:       s.cfg.PgFailoverRedisURL,
			PrimaryDSN:     string(s.cfg.PgPrimaryDSN),
			StandbyDSN:     string(s.cfg.PgStandbyDSN),
			HealthInterval: time.Duration(s.cfg.PgFailoverHealthMs) * time.Millisecond,
			FailThreshold:  s.cfg.PgFailoverFailThreshold,
			MaxConns:       s.cfg.DBTrackerMaxConns,
			MinConns:       s.cfg.DBMinConns,
			Coord: bserver.CoordConfig{
				LeaseTTL:           time.Duration(s.cfg.PgFailoverLeaseSec) * time.Second,
				Interval:           time.Duration(s.cfg.PgFailoverCoordMs) * time.Millisecond,
				RenewFailThreshold: 2,
				DebounceWindow:     time.Second,
			},
		}
		promoter := buildPgFailoverPromoter(s, reconnect)
		coord, err := pgfailover.NewCoordinator(coordCfg, promoter, pgfailover.PingDSN(string(s.cfg.PgPrimaryDSN)))
		if err != nil {
			slog.Warn("pg failover coordinator init failed", "error", err)
			return rt
		}
		rt.coord = coord
		coord.Start()
		slog.Info("pg failover coordinator started", "node_id", s.cfg.NodeID)
	}
	return rt
}

func buildPgFailoverPromoter(s *Service, reconnect func(*pgxpool.Pool)) pgfailover.Promoter {
	if s.cfg.PgPromoteCommand == "" && !s.cfg.PgFailoverSnapshotSync {
		return pgfailover.PromoteFunc(func(ctx context.Context) (string, error) {
			pool, err := database.Connect(ctx, string(s.cfg.PgStandbyDSN), s.cfg.DBTrackerMaxConns, s.cfg.DBMinConns)
			if err != nil {
				return "", err
			}
			if err := pool.Ping(ctx); err != nil {
				pool.Close()
				return "", err
			}
			reconnect(pool)
			return string(s.cfg.PgStandbyDSN), nil
		})
	}
	return &pgfailover.StandbyPromoter{
		StandbyDSN:     string(s.cfg.PgStandbyDSN),
		PrimaryDSN:     string(s.cfg.PgPrimaryDSN),
		PromoteCommand: s.cfg.PgPromoteCommand,
		SnapshotSync:   s.cfg.PgFailoverSnapshotSync,
		SyncConfig:     pgfailover.SnapshotSyncConfig{PageSize: s.cfg.PgFailoverSyncPageSize},
		MaxConns:       s.cfg.DBTrackerMaxConns,
		MinConns:       s.cfg.DBMinConns,
		OnReconnect:    reconnect,
	}
}

func (s *Service) AuditLedgerDuplicatesSinceFailover(ctx context.Context) (int, error) {
	if s == nil || s.pool == nil || s.cfg == nil {
		return 0, nil
	}
	cfg := pgfailover.AuditConfig{Window: time.Duration(s.cfg.PgFailoverAuditWindowSec) * time.Second}
	return pgfailover.CountLedgerDuplicatesSinceNow(ctx, s.pool, cfg)
}

func (rt *PgFailoverRuntime) ClosePgFailover() {
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
	if s == nil || s.pgFencing == nil || len(s.rdbs) == 0 {
		return nil
	}
	if err := s.pgFencing.Refresh(ctx); err != nil {
		return err
	}
	reader := newPgFailoverShardReader(s.rdbs)
	_, epoch, err := reader.activeDSN(ctx)
	if err != nil {
		return err
	}
	return s.pgFencing.Validate(epoch)
}
