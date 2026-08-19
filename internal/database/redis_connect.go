package database

import (
	"context"
	"fmt"

	"github.com/bidshard/ad-event-processor/pkg/netaddr"

	redis "github.com/redis/go-redis/v9"
)

func ConnectRedis(ctx context.Context, addr, password string) (redis.UniversalClient, error) {
	return ConnectRedisWithBreaker(ctx, addr, password, nil)
}

func ConnectRedisWithBreaker(ctx context.Context, addr, password string, breaker *RedisBreaker) (redis.UniversalClient, error) {
	uopts := netaddr.RedisUniversalOptions(addr, password)

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

func BrokerRedisURL(addrs []string, password string) string {
	if len(addrs) == 0 {
		return ""
	}
	return netaddr.RedisURLFromAddr(addrs[0], password, 0)
}

func OpenRedisFromURLOrAddr(raw, password string) (redis.UniversalClient, error) {
	return netaddr.ParseRedisURL(raw, password)
}
