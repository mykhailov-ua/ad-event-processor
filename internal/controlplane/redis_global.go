package controlplane

import (
	"context"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/edge"

	"github.com/redis/go-redis/v9"
)

const (
	redisConfigValuesKey  = "config:values"
	redisConfigVersionKey = "config:version"
)

func syncGlobalConfigToAllShards(ctx context.Context, redisShards []redis.UniversalClient, settings map[string]string, version int64) error {
	return forEachConnectedShard(ctx, redisShards, "sync_global_config", func(_ int, redisClient redis.UniversalClient) error {
		_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
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

func syncGlobalStringToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key, value string, ttl time.Duration) error {
	return database.SyncGlobalStringToAllShards(ctx, redisShards, key, value, ttl)
}

func deleteGlobalKeyFromAllShards(ctx context.Context, redisShards []redis.UniversalClient, key string) error {
	return database.DeleteGlobalKeyFromAllShards(ctx, redisShards, key)
}

func syncGlobalSetMemberToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key, member string, add bool) error {
	err := forEachConnectedShard(ctx, redisShards, "sync_global_set_member", func(_ int, redisClient redis.UniversalClient) error {
		if add {
			return redisClient.SAdd(ctx, key, member).Err()
		}
		return redisClient.SRem(ctx, key, member).Err()
	})
	if err != nil {
		return err
	}
	if len(redisShards) > 0 && redisShards[0] != nil {
		if changelogErr := edge.RecordBlacklistChangelog(ctx, redisShards[0], key, member, add); changelogErr != nil {
			return changelogErr
		}
	}
	return nil
}

func syncGlobalSetReplaceToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key string, members []interface{}) error {
	return forEachConnectedShard(ctx, redisShards, "sync_global_set_replace", func(_ int, redisClient redis.UniversalClient) error {
		_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			if len(members) > 0 {
				pipe.SAdd(ctx, key, members...)
			}
			return nil
		})
		return err
	})
}

func syncMLModelMetaOnShard(ctx context.Context, redisClient redis.UniversalClient, versionID, hash string, appliedAt int64) error {
	if redisClient == nil {
		return nil
	}
	_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, "ml:model:version", versionID, 0)
		pipe.Set(ctx, "ml:model:hash", hash, 0)
		pipe.Set(ctx, "ml:model:applied_at", appliedAt, 0)
		return nil
	})
	return err
}

func syncKeyToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key string, value interface{}, ttl time.Duration) error {
	return forEachConnectedShard(ctx, redisShards, "sync_key", func(_ int, redisClient redis.UniversalClient) error {
		return redisClient.Set(ctx, key, value, ttl).Err()
	})
}

func syncGlobalHashFieldToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key, field, value string, del bool) error {
	return forEachConnectedShard(ctx, redisShards, "sync_global_hash_field", func(_ int, redisClient redis.UniversalClient) error {
		if del {
			return redisClient.HDel(ctx, key, field).Err()
		}
		return redisClient.HSet(ctx, key, field, value).Err()
	})
}
