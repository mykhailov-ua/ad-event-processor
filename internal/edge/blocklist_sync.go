package edge

import (
	"context"
	"fmt"

	"github.com/cilium/ebpf"
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

// SyncBlocklistFromRedis pulls manual/auto/fraud sets from Redis into the XDP blocklist map.
// Pub/sub cache refresh for nginx uses deploy/nginx/lua/edge-blacklist-sync.lua and
// edge.FraudQuarantineChannel payloads (see quarantine_pubsub.go).
func SyncBlocklistFromRedis(ctx context.Context, rdb denySetReader, v4Map, v6Map *ebpf.Map, store *BlocklistStore) (added, removed int, err error) {
	manual, err := rdb.SMembers(ctx, redisKeyBlacklistManual).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("smembers %s: %w", redisKeyBlacklistManual, err)
	}
	auto, err := loadAutoBans(ctx, rdb)
	if err != nil {
		return 0, 0, fmt.Errorf("active %s: %w", redisKeyBlacklistAuto, err)
	}
	fraud, err := rdb.SMembers(ctx, redisKeyBlacklistFraud).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("smembers %s: %w", redisKeyBlacklistFraud, err)
	}
	return store.ApplyDiff(v4Map, v6Map, manual, auto, fraud)
}
