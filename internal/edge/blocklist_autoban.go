package edge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func RecordAutoBan(ctx context.Context, redisClient redis.Cmdable, ip string, ttl time.Duration) error {
	if redisClient == nil || ip == "" {
		return fmt.Errorf("nil redis client or empty ip")
	}
	expiresAt := float64(time.Now().Add(ttl).Unix())
	pipe := redisClient.Pipeline()
	pipe.SAdd(ctx, redisKeyBlacklistAuto, ip)
	pipe.ZAdd(ctx, redisKeyBlacklistAutoTTL, redis.Z{Score: expiresAt, Member: ip})
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return RecordBlacklistChangelog(ctx, redisClient, redisKeyBlacklistAuto, ip, true)
}

func loadAutoBans(ctx context.Context, redisClient denySetReader) ([]string, error) {
	if ab, ok := redisClient.(autoBanReader); ok {
		return activeAutoBans(ctx, ab)
	}
	return redisClient.SMembers(ctx, redisKeyBlacklistAuto).Result()
}

func activeAutoBans(ctx context.Context, redisClient autoBanReader) ([]string, error) {
	members, err := redisClient.SMembers(ctx, redisKeyBlacklistAuto).Result()
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return members, nil
	}
	now := float64(time.Now().Unix())
	active := make([]string, 0, len(members))
	for _, ip := range members {
		score, err := redisClient.ZScore(ctx, redisKeyBlacklistAutoTTL, ip).Result()
		if errors.Is(err, redis.Nil) {
			active = append(active, ip)
			continue
		}
		if err != nil {
			return nil, err
		}
		if score > now {
			active = append(active, ip)
			continue
		}
		_ = redisClient.SRem(ctx, redisKeyBlacklistAuto, ip).Err()
		_ = redisClient.ZRem(ctx, redisKeyBlacklistAutoTTL, ip).Err()
	}
	return active, nil
}
