package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

type RedisLimiter struct {
	rdb *redis.Client
}

func NewRedisLimiter(rdb *redis.Client) Limiter {
	return &RedisLimiter{rdb: rdb}
}

const rateLimitScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current = redis.call("INCR", key)
if current == 1 then
    redis.call("EXPIRE", key, window)
end

if current > limit then
    return 0
end
return 1
`

func (l *RedisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	res, err := l.rdb.Eval(ctx, rateLimitScript, []string{key}, limit, int(window.Seconds())).Result()
	if err != nil {
		return false, err
	}

	return res.(int64) == 1, nil
}
