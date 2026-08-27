package database

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime"
	"sync"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/netaddr"

	redis "github.com/redis/go-redis/v9"
)

const redisConnectRetries = 30

type RedisShardOptions struct {
	PoolSize         int
	FilterTimeoutMs  int
	StickyPinWorkers int
}

func ConnectRedisShards(ctx context.Context, cfg *config.Config, opts RedisShardOptions) ([]redis.UniversalClient, []*RedisBreaker, error) {
	names := cfg.ResolveRedisMasterNames()
	if cfg.RedisSentinelEnabled() && len(names) != len(cfg.RedisAddrs) {
		return nil, nil, fmt.Errorf("sentinel master name count (%d) must match REDIS_ADDRS (%d)", len(names), len(cfg.RedisAddrs))
	}

	clients := make([]redis.UniversalClient, 0, len(cfg.RedisAddrs))
	breakers := make([]*RedisBreaker, 0, len(cfg.RedisAddrs))
	for i := range cfg.RedisAddrs {
		redisClient, br, err := connectRedisShard(ctx, cfg, i, names, opts)
		if err != nil {
			if i == 0 && cfg.RedisShard0OptionalStartup {
				slog.Warn("redis shard 0 unavailable at startup; continuing degraded",
					"error", err,
					"hint", "registry replica + PG sync + broker fallback keep ingest alive",
				)
				open := NewRedisBreaker(1, 1, 24*time.Hour)
				open.RecordFailure()
				clients = append(clients, nil)
				breakers = append(breakers, open)
				continue
			}
			for _, c := range clients {
				_ = c.Close()
			}
			return nil, nil, err
		}
		clients = append(clients, redisClient)
		breakers = append(breakers, br)
	}
	setShard0ClientNilMetric(clients)
	return clients, breakers, nil
}

func WarmRedisShardPools(ctx context.Context, clients []redis.UniversalClient, pingsPerShard int) {
	if pingsPerShard <= 0 {
		pingsPerShard = 4
	}
	var wg sync.WaitGroup
	for _, client := range clients {
		if client == nil {
			continue
		}
		for range pingsPerShard {
			wg.Add(1)
			go func(c redis.UniversalClient) {
				defer wg.Done()
				_ = c.Ping(ctx).Err()
			}(client)
		}
	}
	wg.Wait()
}

func StartRedisPoolStatsReporter(ctx context.Context, clients []redis.UniversalClient, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for i, c := range clients {
					if c == nil {
						continue
					}
					if cli, ok := c.(*redis.Client); ok {
						metrics.RecordRedisPoolStats(i, cli.PoolStats())
					}
				}
			}
		}
	}()
}

func SetShard0ClientNilMetric(clients []redis.UniversalClient) {
	setShard0ClientNilMetric(clients)
}

func setShard0ClientNilMetric(clients []redis.UniversalClient) {
	if len(clients) > 0 && clients[0] == nil {
		metrics.Shard0ClientNil.Set(1)
		return
	}
	metrics.Shard0ClientNil.Set(0)
}

func ConnectRedisShard(ctx context.Context, cfg *config.Config, shardIdx int, opts RedisShardOptions) (redis.UniversalClient, error) {
	if shardIdx < 0 || shardIdx >= len(cfg.RedisAddrs) {
		return nil, fmt.Errorf("redis shard index %d out of range [0,%d)", shardIdx, len(cfg.RedisAddrs))
	}
	redisClient, _, err := connectRedisShard(ctx, cfg, shardIdx, cfg.ResolveRedisMasterNames(), opts)
	return redisClient, err
}

func connectRedisShard(ctx context.Context, cfg *config.Config, shardIdx int, masterNames []string, opts RedisShardOptions) (redis.UniversalClient, *RedisBreaker, error) {
	uopts := shardUniversalOptions(cfg, shardIdx, masterNames, opts)
	redisClient := redis.NewUniversalClient(uopts)

	dialLabel := cfg.RedisAddrs[shardIdx]
	if cfg.RedisSentinelEnabled() {
		dialLabel = masterNames[shardIdx]
	}

	var pingErr error
	for range redisConnectRetries {
		select {
		case <-ctx.Done():
			_ = redisClient.Close()
			return nil, nil, fmt.Errorf("redis shard %d (%s): %w", shardIdx, dialLabel, ctx.Err())
		default:
		}
		if pingErr = redisClient.Ping(ctx).Err(); pingErr == nil {
			break
		}
		slog.Warn("waiting for redis...", "shard", shardIdx, "target", dialLabel, "error", pingErr)
		select {
		case <-ctx.Done():
			_ = redisClient.Close()
			return nil, nil, fmt.Errorf("redis shard %d (%s): %w", shardIdx, dialLabel, ctx.Err())
		case <-time.After(time.Second):
		}
	}
	if pingErr != nil {
		_ = redisClient.Close()
		return nil, nil, fmt.Errorf("redis shard %d (%s): %w", shardIdx, dialLabel, pingErr)
	}

	breaker := NewRedisBreaker(
		int64(cfg.RedisBreakerFailThreshold),
		int64(cfg.RedisBreakerHalfOpen),
		time.Duration(cfg.RedisBreakerOpenTimeoutMs)*time.Millisecond,
	)
	redisClient.AddHook(NewRedisCircuitBreakerHook(breaker))
	return redisClient, breaker, nil
}

func shardUniversalOptions(cfg *config.Config, shardIdx int, masterNames []string, opts RedisShardOptions) *redis.UniversalOptions {
	poolSize := opts.PoolSize
	if poolSize <= 0 {
		poolSize = 10 * runtime.GOMAXPROCS(0)
	}
	if opts.StickyPinWorkers > 0 {
		poolSize += opts.StickyPinWorkers
	}
	maxActiveConns := poolSize
	if cfg != nil && cfg.RedisMaxActiveConns > maxActiveConns {
		maxActiveConns = cfg.RedisMaxActiveConns
	}
	uopts := &redis.UniversalOptions{
		Password:       string(cfg.RedisPassword),
		PoolSize:       poolSize,
		MinIdleConns:   poolSize,
		MaxActiveConns: maxActiveConns,
	}
	if opts.FilterTimeoutMs > 0 {
		d := time.Duration(opts.FilterTimeoutMs) * time.Millisecond
		uopts.ReadTimeout = d
		uopts.WriteTimeout = d
	}
	if cfg.RedisSentinelEnabled() {
		uopts.MasterName = masterNames[shardIdx]
		uopts.Addrs = cfg.RedisSentinelAddrs
		return uopts
	}
	addr := cfg.RedisAddrs[shardIdx]
	uopts.Addrs = []string{addr}
	if netaddr.IsUnixSocketPath(addr) {
		uopts.Dialer = func(ctx context.Context, _, addr string) (net.Conn, error) {
			var netDialer net.Dialer
			return netDialer.DialContext(ctx, "unix", addr)
		}
	}
	return uopts
}
