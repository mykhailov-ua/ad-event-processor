package edge

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	redisKeyBlacklistManual  = "blacklist:manual"
	redisKeyBlacklistAuto    = "blacklist:auto"
	redisKeyBlacklistFraud   = "blacklist:fraud"
	redisKeyBlacklistAutoTTL = "blacklist:auto:ttl"
)

type denySetReader interface {
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
}

type autoBanReader interface {
	denySetReader
	ZScore(ctx context.Context, key, member string) *redis.FloatCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd
	ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd
}

func SyncBlocklistFromRedis(ctx context.Context, redisClient denySetReader, maps BlocklistMaps, store *BlocklistStore) (added, removed int, err error) {
	manual, err := redisClient.SMembers(ctx, redisKeyBlacklistManual).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("smembers %s: %w", redisKeyBlacklistManual, err)
	}
	auto, err := loadAutoBans(ctx, redisClient)
	if err != nil {
		return 0, 0, fmt.Errorf("active %s: %w", redisKeyBlacklistAuto, err)
	}
	fraud, err := redisClient.SMembers(ctx, redisKeyBlacklistFraud).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("smembers %s: %w", redisKeyBlacklistFraud, err)
	}
	return store.ApplyDiff(maps, manual, auto, fraud)
}
