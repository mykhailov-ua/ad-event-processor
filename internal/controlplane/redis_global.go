package controlplane

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/edge"

	"github.com/redis/go-redis/v9"
)

const (
	redisConfigValuesKey  = "config:values"
	redisConfigVersionKey = "config:version"
)

func syncGlobalConfigToAllShards(ctx context.Context, rdbs []redis.UniversalClient, settings map[string]string, version int64) error {
	return forEachConnectedShard(ctx, rdbs, "sync_global_config", func(_ int, rdb redis.UniversalClient) error {
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			if len(settings) > 0 {
				pipe.HSet(ctx, redisConfigValuesKey, settings)
			}
			if version > 0 {
				pipe.Set(ctx, redisConfigVersionKey, version, 0)
			}
			return nil
		})
		return err
	})
}

func syncGlobalStringToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, value string, ttl time.Duration) error {
	return database.SyncGlobalStringToAllShards(ctx, rdbs, key, value, ttl)
}

func deleteGlobalKeyFromAllShards(ctx context.Context, rdbs []redis.UniversalClient, key string) error {
	return database.DeleteGlobalKeyFromAllShards(ctx, rdbs, key)
}

func syncGlobalSetMemberToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, member string, add bool) error {
	err := forEachConnectedShard(ctx, rdbs, "sync_global_set_member", func(_ int, rdb redis.UniversalClient) error {
		if add {
			return rdb.SAdd(ctx, key, member).Err()
		}
		return rdb.SRem(ctx, key, member).Err()
	})
	if err != nil {
		return err
	}
	if len(rdbs) > 0 && rdbs[0] != nil {
		if changelogErr := edge.RecordBlacklistChangelog(ctx, rdbs[0], key, member, add); changelogErr != nil {
			return changelogErr
		}
	}
	return nil
}

func syncGlobalSetReplaceToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key string, members []interface{}) error {
	return forEachConnectedShard(ctx, rdbs, "sync_global_set_replace", func(_ int, rdb redis.UniversalClient) error {
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			if len(members) > 0 {
				pipe.SAdd(ctx, key, members...)
			}
			return nil
		})
		return err
	})
}

func syncMLModelMetaOnShard(ctx context.Context, rdb redis.UniversalClient, versionID, hash string, appliedAt int64) error {
	if rdb == nil {
		return nil
	}
	_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, "ml:model:version", versionID, 0)
		pipe.Set(ctx, "ml:model:hash", hash, 0)
		pipe.Set(ctx, "ml:model:applied_at", appliedAt, 0)
		return nil
	})
	return err
}

func syncKeyToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key string, value interface{}, ttl time.Duration) error {
	return forEachConnectedShard(ctx, rdbs, "sync_key", func(_ int, rdb redis.UniversalClient) error {
		return rdb.Set(ctx, key, value, ttl).Err()
	})
}

func syncGlobalHashFieldToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, field, value string, del bool) error {
	return forEachConnectedShard(ctx, rdbs, "sync_global_hash_field", func(_ int, rdb redis.UniversalClient) error {
		if del {
			return rdb.HDel(ctx, key, field).Err()
		}
		return rdb.HSet(ctx, key, field, value).Err()
	})
}
