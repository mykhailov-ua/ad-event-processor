package identity

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/identity/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	Handler     *Handler
	pool        *pgxpool.Pool
	redisShards []redis.UniversalClient
	cancel      context.CancelFunc
	wg          sync.WaitGroup
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
	for _, redisClient := range m.redisShards {
		if redisClient != nil {
			_ = redisClient.Close()
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
	var controlRedisShards []redis.UniversalClient
	var errShards error
	controlRedisShards, _, errShards = database.ConnectRedisShards(ctx, cfg, database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	})
	var redisClient redis.UniversalClient
	if errShards != nil {
		redisClient, err = database.ConnectRedisShard(ctx, cfg, 0, database.RedisShardOptions{
			PoolSize: cfg.RedisPoolSize,
		})
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("connect redis shard 0 fallback: %w (shards err: %w)", err, errShards)
		}
		controlRedisShards = []redis.UniversalClient{redisClient}
	} else {
		if len(controlRedisShards) > 0 {
			redisClient = controlRedisShards[0]
		}
		if redisClient == nil {
			for _, shard := range controlRedisShards {
				if shard != nil {
					redisClient = shard
					break
				}
			}
		}
	}
	repo := db.NewStore(pool)
	tokenMaker, err := NewPasetoMaker(string(cfg.TokenSymmetricKey))
	if err != nil {
		pool.Close()
		_ = redisClient.Close()
		for _, shard := range controlRedisShards {
			if shard != nil && shard != redisClient {
				_ = shard.Close()
			}
		}
		return nil, err
	}
	lockoutLimiter := NewLockoutLimiter(controlRedisShards...)
	hasher, err := NewPasswordHasher(
		uint32(cfg.Argon2Memory),
		uint32(cfg.Argon2Iterations),
		uint8(cfg.Argon2Parallelism),
	)
	if err != nil {
		pool.Close()
		_ = redisClient.Close()
		for _, shard := range controlRedisShards {
			if shard != nil && shard != redisClient {
				_ = shard.Close()
			}
		}
		return nil, err
	}
	authService := NewService(repo, tokenMaker, hasher, lockoutLimiter, redisClient)
	authService.SetControlRedisShards(controlRedisShards)
	workerCtx, workerCancel := context.WithCancel(ctx)
	mod := &Module{
		Handler:     NewHandler(authService, cfg),
		pool:        pool,
		redisShards: append([]redis.UniversalClient{redisClient}, controlRedisShards...),
		cancel:      workerCancel,
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
