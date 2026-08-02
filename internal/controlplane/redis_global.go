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
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			if len(settings) > 0 {
				pipe.HSet(ctx, redisConfigValuesKey, settings)
			}
			if version > 0 {
				pipe.Set(ctx, redisConfigVersionKey, version, 0)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("sync global config on shard %d: %w", i, err)
		}
	}
	return nil
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
	for i := 1; i < len(rdbs); i++ {
		if rdbs[i] == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		if err := rdbs[i].Set(ctx, redisConfigVersionKey, version, 0).Err(); err != nil {
			return fmt.Errorf("replicate config version on shard %d: %w", i, err)
		}
	}
	return nil
}

func syncGlobalStringToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, value string, ttl time.Duration) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		if err := rdb.Set(ctx, key, value, ttl).Err(); err != nil {
			return fmt.Errorf("set %s on shard %d: %w", key, i, err)
		}
	}
	return nil
}

func deleteGlobalKeyFromAllShards(ctx context.Context, rdbs []redis.UniversalClient, key string) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		if err := rdb.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("del %s on shard %d: %w", key, i, err)
		}
	}
	return nil
}

func syncGlobalSetMemberToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, member string, add bool) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		var err error
		if add {
			err = rdb.SAdd(ctx, key, member).Err()
		} else {
			err = rdb.SRem(ctx, key, member).Err()
		}
		if err != nil {
			return fmt.Errorf("set member sync on shard %d key %s: %w", i, key, err)
		}
	}
	return nil
}

func syncGlobalSetReplaceToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key string, members []interface{}) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			if len(members) > 0 {
				pipe.SAdd(ctx, key, members...)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("replace set on shard %d key %s: %w", i, key, err)
		}
	}
	return nil
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
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		if err := rdb.Set(ctx, key, value, ttl).Err(); err != nil {
			return fmt.Errorf("set %s on shard %d: %w", key, i, err)
		}
	}
	return nil
}

func syncGlobalHashFieldToAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, field, value string, del bool) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		var err error
		if del {
			err = rdb.HDel(ctx, key, field).Err()
		} else {
			err = rdb.HSet(ctx, key, field, value).Err()
		}
		if err != nil {
			return fmt.Errorf("hash field sync on shard %d key %s: %w", i, key, err)
		}
	}
	return nil
}
