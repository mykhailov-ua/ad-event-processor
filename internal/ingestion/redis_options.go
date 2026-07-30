package ingestion

import (
	"time"

	redis "github.com/redis/go-redis/v9"
)

func FilterRedisOptions(addrs []string, password string, poolSize, filterTimeoutMs int) *redis.UniversalOptions {
	opts := &redis.UniversalOptions{
		Addrs:    addrs,
		Password: password,
		PoolSize: poolSize,
	}
	if filterTimeoutMs > 0 {
		d := time.Duration(filterTimeoutMs) * time.Millisecond
		opts.ReadTimeout = d
		opts.WriteTimeout = d
	}
	return opts
}

func FilterRedisReadTimeoutMs(filterTimeoutMs int) int {
	if filterTimeoutMs <= 0 {
		return 0
	}
	return filterTimeoutMs
}
