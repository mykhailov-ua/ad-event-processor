package identity

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/identity/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	Handler *Handler
	pool    *pgxpool.Pool
	rdbs    []redis.UniversalClient
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func (m *Module) Close() {
	if m == nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	if m.pool != nil {
		m.pool.Close()
	}
	for _, rdb := range m.rdbs {
		if rdb != nil {
			_ = rdb.Close()
		}
	}
}

func (m *Module) API() AuthAPI {
	if m == nil {
		return nil
	}
	return m.Handler.API()
}

func OpenModule(ctx context.Context, cfg *config.Config) (*Module, error) {
	if cfg == nil {
		return nil, nil
	}
	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return nil, err
	}
	rdb, err := database.ConnectRedisShard(ctx, cfg, 0, database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	})
	if err != nil {
		pool.Close()
		return nil, err
	}
	var controlRdbs []redis.UniversalClient
	controlRdbs, _, err = database.ConnectRedisShards(ctx, cfg, database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	})
	if err != nil {
		slog.Warn("multi-shard redis unavailable, using single shard for control", "error", err)
		controlRdbs = []redis.UniversalClient{rdb}
	}
	repo := db.NewStore(pool)
	tokenMaker, err := NewPasetoMaker(string(cfg.TokenSymmetricKey))
	if err != nil {
		pool.Close()
		rdb.Close()
		for _, shard := range controlRdbs {
			if shard != nil && shard != rdb {
				_ = shard.Close()
			}
		}
		return nil, err
	}
	lockoutLimiter := NewLockoutLimiter(controlRdbs...)
	hasher, err := NewPasswordHasher(
		uint32(cfg.Argon2Memory),
		uint32(cfg.Argon2Iterations),
		uint8(cfg.Argon2Parallelism),
	)
	if err != nil {
		pool.Close()
		rdb.Close()
		for _, shard := range controlRdbs {
			if shard != nil && shard != rdb {
				_ = shard.Close()
			}
		}
		return nil, err
	}
	authService := NewService(repo, tokenMaker, hasher, lockoutLimiter, rdb)
	authService.SetControlRedisShards(controlRdbs)
	workerCtx, workerCancel := context.WithCancel(ctx)
	mod := &Module{
		Handler: NewHandler(authService, cfg),
		pool:    pool,
		rdbs:    append([]redis.UniversalClient{rdb}, controlRdbs...),
		cancel:  workerCancel,
	}
	cleanupWorker := NewSessionCleanupWorker(authService)
	mod.wg.Add(1)
	go func() {
		defer mod.wg.Done()
		cleanupWorker.Start(workerCtx, time.Minute)
	}()
	return mod, nil
}

func OpenAPI(ctx context.Context, cfg *config.Config) (AuthAPI, func(), error) {
	noop := func() {}
	if cfg == nil {
		return nil, noop, nil
	}
	mod, err := OpenModule(ctx, cfg)
	if err != nil {
		return nil, noop, err
	}
	if mod == nil {
		return nil, noop, nil
	}
	return mod.API(), mod.Close, nil
}
