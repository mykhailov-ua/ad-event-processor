package pgfailover

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/pkg/broker/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Promoter interface {
	Promote(ctx context.Context) (dsn string, err error)
}

type PromoteFunc func(ctx context.Context) (string, error)

func (f PromoteFunc) Promote(ctx context.Context) (string, error) {
	return f(ctx)
}

type HealthCheck func(ctx context.Context) error

type Config struct {
	NodeID         string
	RedisURL       string
	PrimaryDSN     string
	StandbyDSN     string
	HealthInterval time.Duration
	HealthTimeout  time.Duration
	FailThreshold  int
	MaxConns       int
	MinConns       int
	Coord          server.CoordConfig
}

func (c Config) normalized() Config {
	out := c
	if out.HealthInterval <= 0 {
		out.HealthInterval = 2 * time.Second
	}
	if out.HealthTimeout <= 0 {
		out.HealthTimeout = 2 * time.Second
	}
	if out.FailThreshold <= 0 {
		out.FailThreshold = 2
	}
	if out.MaxConns <= 0 {
		out.MaxConns = 4
	}
	if out.MinConns <= 0 {
		out.MinConns = 1
	}
	if out.Coord.LeaseTTL <= 0 {
		out.Coord = server.DefaultCoordConfig()
	}
	return out
}

type Coordinator struct {
	cfg       Config
	host      *CoordHost
	coord     *server.Coordinator
	promoter  Promoter
	health    HealthCheck
	redisClient       redis.UniversalClient
	failures  int
	failMu    sync.Mutex
	failover  atomic.Bool
	closeCh   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func NewCoordinator(cfg Config, promoter Promoter, health HealthCheck) (*Coordinator, error) {
	cfg = cfg.normalized()
	if cfg.NodeID == "" {
		return nil, errors.New("pg failover node id required")
	}
	if cfg.RedisURL == "" {
		return nil, errors.New("pg failover redis url required")
	}
	if promoter == nil {
		return nil, errors.New("pg failover promoter required")
	}
	if health == nil {
		health = PingDSN(cfg.PrimaryDSN)
	}
	host := NewCoordHost()
	coord, err := server.NewCoordinatorWithConfig(cfg.NodeID, "pgfailover:"+cfg.NodeID, cfg.RedisURL, host, cfg.Coord)
	if err != nil {
		return nil, err
	}
	return &Coordinator{
		cfg:      cfg,
		host:     host,
		coord:    coord,
		promoter: promoter,
		health:   health,
		redisClient:      coord.Redis(),
		closeCh:  make(chan struct{}),
	}, nil
}

func (c *Coordinator) Start(ctx context.Context) {
	c.coord.Start(ctx)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runHealthLoop(ctx)
	}()
}

func (c *Coordinator) Stop() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
	})
	c.coord.Stop()
	c.wg.Wait()
}

func (c *Coordinator) Redis() redis.UniversalClient {
	return c.redisClient
}

func (c *Coordinator) IsLeader() bool {
	return c.coord.IsLeader(c.host.TopicKey())
}

func (c *Coordinator) runHealthLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.HealthInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.closeCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !c.IsLeader() {
				c.resetFailures()
				continue
			}
			if c.failover.Load() {
				continue
			}
			healthCtx, cancel := context.WithTimeout(ctx, c.cfg.HealthTimeout)
			err := c.health(healthCtx)
			cancel()
			if err == nil {
				c.resetFailures()
				continue
			}
			if c.recordFailure() < c.cfg.FailThreshold {
				continue
			}
			if err := c.executeFailover(ctx); err != nil {
				slog.Warn("postgres failover failed", "error", err)
			}
		}
	}
}

func (c *Coordinator) recordFailure() int {
	c.failMu.Lock()
	defer c.failMu.Unlock()
	c.failures++
	return c.failures
}

func (c *Coordinator) resetFailures() {
	c.failMu.Lock()
	defer c.failMu.Unlock()
	c.failures = 0
}

func (c *Coordinator) executeFailover(ctx context.Context) error {
	if !c.failover.CompareAndSwap(false, true) {
		return nil
	}
	defer c.resetFailures()

	failoverCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	epoch, err := BumpEpoch(failoverCtx, c.redisClient)
	if err != nil {
		c.failover.Store(false)
		return fmt.Errorf("bump fencing epoch: %w", err)
	}

	promotedDSN, err := c.promoter.Promote(failoverCtx)
	if err != nil {
		c.failover.Store(false)
		return fmt.Errorf("promote standby: %w", err)
	}
	if promotedDSN == "" {
		promotedDSN = c.cfg.StandbyDSN
	}
	if promotedDSN == "" {
		c.failover.Store(false)
		return errors.New("promoted dsn empty")
	}

	if err := PublishDSN(failoverCtx, c.redisClient, promotedDSN, epoch); err != nil {
		c.failover.Store(false)
		return fmt.Errorf("publish dsn: %w", err)
	}

	slog.Info("postgres failover completed",
		"node_id", c.cfg.NodeID,
		"fencing_epoch", epoch,
		"dsn_host", redactDSNHost(promotedDSN),
	)
	return nil
}

func PingDSN(dsn string) HealthCheck {
	return func(ctx context.Context) error {
		if dsn == "" {
			return errors.New("dsn required")
		}
		cfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			return err
		}
		cfg.MaxConns = 1
		cfg.MinConns = 0
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			return err
		}
		defer pool.Close()
		return pool.Ping(ctx)
	}
}

func redactDSNHost(dsn string) string {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return "invalid"
	}
	return cfg.ConnConfig.Host
}
