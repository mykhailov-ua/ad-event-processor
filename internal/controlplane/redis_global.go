package controlplane

import (
	"context"
	"fmt"
	"time"

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

func replicateConfigVersionFromPrimary(ctx context.Context, rdbs []redis.UniversalClient) error {
	if len(rdbs) < 2 {
		return nil
	}
	primary := PickHealthyControlShard(rdbs)
	if primary == nil {
		return nil
	}
	version, err := primary.Get(ctx, redisConfigVersionKey).Int64()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config version from primary shard: %w", err)
	}
	return forEachConnectedShard(ctx, rdbs, "replicate_config_version", func(i int, rdb redis.UniversalClient) error {
		if rdb == primary {
			return nil
		}
		return rdb.Set(ctx, redisConfigVersionKey, version, 0).Err()
	})
}

func syncGlobalStringToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, value string, ttl time.Duration) error {
	return forEachConnectedShard(ctx, rdbs, "sync_global_string", func(_ int, rdb redis.UniversalClient) error {
		return rdb.Set(ctx, key, value, ttl).Err()
	})
}

func deleteGlobalKeyFromAllShards(ctx context.Context, rdbs []redis.UniversalClient, key string) error {
	return forEachConnectedShard(ctx, rdbs, "delete_global_key", func(_ int, rdb redis.UniversalClient) error {
		return rdb.Del(ctx, key).Err()
	})
}

func syncGlobalSetMemberToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, member string, add bool) error {
	return forEachConnectedShard(ctx, rdbs, "sync_global_set_member", func(_ int, rdb redis.UniversalClient) error {
		if add {
			return rdb.SAdd(ctx, key, member).Err()
		}
		return rdb.SRem(ctx, key, member).Err()
	})
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
