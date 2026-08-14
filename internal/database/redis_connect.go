package database

import (
	"context"
	"fmt"
	"net"
	"strings"

	redis "github.com/redis/go-redis/v9"
)

func ConnectRedis(ctx context.Context, addr string, password string) (redis.UniversalClient, error) {
	return ConnectRedisWithBreaker(ctx, addr, password, nil)
}

func ConnectRedisWithBreaker(ctx context.Context, addr string, password string, breaker *RedisBreaker) (redis.UniversalClient, error) {
	uopts := &redis.UniversalOptions{
		Addrs:    []string{addr},
		Password: password,
	}

	if strings.HasPrefix(addr, "/") || strings.HasSuffix(addr, ".sock") || strings.Contains(addr, ".sock") {
		uopts.Dialer = func(ctx context.Context, _, addr string) (net.Conn, error) {
			var netDialer net.Dialer
			return netDialer.DialContext(ctx, "unix", addr)
		}
	}

	rdb := redis.NewUniversalClient(uopts)

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	if breaker != nil {
		rdb.AddHook(NewRedisCircuitBreakerHook(breaker))
	}

	return rdb, nil
}
