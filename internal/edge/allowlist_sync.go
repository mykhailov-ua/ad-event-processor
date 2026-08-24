package edge

import (
	"context"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/redis/go-redis/v9"
)

const redisKeyAllowlistPartners = "allowlist:partners"

type setReader interface {
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
}

func SyncAllowlistFromRedis(ctx context.Context, redisClient setReader, v4Map, v6Map *ebpf.Map, store *AllowlistStore) (added, removed int, err error) {
	members, err := redisClient.SMembers(ctx, redisKeyAllowlistPartners).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("smembers %s: %w", redisKeyAllowlistPartners, err)
	}
	return store.ApplyDiff(v4Map, v6Map, members)
}
