package edge

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisKeyBlacklistChangelogAdd    = "blacklist:changelog:add"
	redisKeyBlacklistChangelogRemove = "blacklist:changelog:remove"
	blacklistChangelogTTL            = 48 * time.Hour
)

func isEdgeBlacklistSetKey(key string) bool {
	switch key {
	case redisKeyBlacklistManual, redisKeyBlacklistAuto, redisKeyBlacklistFraud:
		return true
	default:
		return false
	}
}

func RecordBlacklistChangelog(ctx context.Context, rdb redis.Cmdable, setKey, member string, add bool) error {
	if rdb == nil || !isEdgeBlacklistSetKey(setKey) || member == "" {
		return nil
	}
	score := float64(time.Now().Unix())
	changelogKey := redisKeyBlacklistChangelogAdd
	if !add {
		changelogKey = redisKeyBlacklistChangelogRemove
	}
	pipe := rdb.Pipeline()
	pipe.ZAdd(ctx, changelogKey, redis.Z{Score: score, Member: member})
	pipe.Expire(ctx, changelogKey, blacklistChangelogTTL)
	_, err := pipe.Exec(ctx)
	return err
}
