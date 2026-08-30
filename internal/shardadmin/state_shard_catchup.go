package shardadmin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/redis/go-redis/v9"
)

const globalBlacklistKeyPrefix = "blacklist:"

type mlModelMeta struct {
	version   string
	hash      string
	appliedAt int64
	present   bool
}

// PickGlobalReconcileSource: shard with highest config:version (or blacklist gen) for shard-0 catch-up.
func PickGlobalReconcileSource(ctx context.Context, redisShards []redis.UniversalClient) redis.UniversalClient {
	if len(redisShards) == 0 {
		return nil
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var (
		best      redis.UniversalClient
		bestIdx         = -1
		bestVer   int64 = -1
		bestBLGen int64 = -1
	)
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		version, verErr := redisClient.Get(checkCtx, redisConfigVersionKey).Int64()
		if verErr != nil {
			if !errors.Is(verErr, redis.Nil) {
				continue
			}
			version = -1
		}
		blGen, blErr := blacklistGenerationScore(checkCtx, redisClient)
		if blErr != nil {
			blGen = -1
		}
		if best == nil ||
			version > bestVer ||
			(version == bestVer && blGen > bestBLGen) ||
			(version == bestVer && blGen == bestBLGen && bestIdx > i) {
			best = redisClient
			bestIdx = i
			bestVer = version
			bestBLGen = blGen
		}
	}
	if best != nil {
		return best
	}
	return PickHealthyControlShard(redisShards)
}

func blacklistGenerationScore(ctx context.Context, redisClient redis.UniversalClient) (int64, error) {
	keys, err := scanRedisKeys(ctx, redisClient, globalBlacklistKeyPrefix+"*")
	if err != nil {
		return 0, err
	}
	var total int64
	for _, key := range keys {
		n, err := redisClient.SCard(ctx, key).Result()
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

func ReplicateConfigVersionFromPrimary(ctx context.Context, redisShards []redis.UniversalClient) error {
	return ReplicateGlobalsFromPrimary(ctx, redisShards)
}

func ReplicateGlobalsFromPrimary(ctx context.Context, redisShards []redis.UniversalClient) error {
	if len(redisShards) < 2 {
		return nil
	}
	source := PickGlobalReconcileSource(ctx, redisShards)
	if source == nil {
		return nil
	}
	return database.ForEachConnectedShard(ctx, redisShards, "replicate_globals", func(i int, redisClient redis.UniversalClient) error {
		if redisClient == source {
			return nil
		}
		return ReplicateGlobalsToTarget(ctx, source, redisClient)
	})
}

func ReplicateGlobalsToTarget(ctx context.Context, source, target redis.UniversalClient) error {
	if source == nil || target == nil || source == target {
		return nil
	}

	settings, err := source.HGetAll(ctx, RedisConfigValuesKey).Result()
	if err != nil {
		return fmt.Errorf("read %s from source: %w", RedisConfigValuesKey, err)
	}

	version, versionErr := source.Get(ctx, redisConfigVersionKey).Int64()
	if versionErr != nil && !errors.Is(versionErr, redis.Nil) {
		return fmt.Errorf("read %s from source: %w", redisConfigVersionKey, versionErr)
	}

	blacklistKeys, err := scanRedisKeys(ctx, source, globalBlacklistKeyPrefix+"*")
	if err != nil {
		return fmt.Errorf("scan blacklist keys: %w", err)
	}
	sort.Strings(blacklistKeys)

	blacklistMembers := make(map[string][]string, len(blacklistKeys))
	for _, key := range blacklistKeys {
		members, err := source.SMembers(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("read %s from source: %w", key, err)
		}
		sort.Strings(members)
		blacklistMembers[key] = members
	}

	mlMeta, err := readMLModelMeta(ctx, source)
	if err != nil {
		return err
	}

	_, err = target.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		if len(settings) > 0 {
			pipe.Del(ctx, RedisConfigValuesKey)
			pipe.HSet(ctx, RedisConfigValuesKey, settings)
		}
		if versionErr == nil {
			pipe.Set(ctx, redisConfigVersionKey, version, 0)
		}
		for _, key := range blacklistKeys {
			pipe.Del(ctx, key)
			if members := blacklistMembers[key]; len(members) > 0 {
				args := make([]interface{}, len(members))
				for i, m := range members {
					args[i] = m
				}
				pipe.SAdd(ctx, key, args...)
			}
		}
		if mlMeta.present {
			pipe.Set(ctx, "ml:model:version", mlMeta.version, 0)
			pipe.Set(ctx, "ml:model:hash", mlMeta.hash, 0)
			pipe.Set(ctx, "ml:model:applied_at", mlMeta.appliedAt, 0)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("write globals to target: %w", err)
	}
	return nil
}

func readMLModelMeta(ctx context.Context, source redis.UniversalClient) (mlModelMeta, error) {
	var meta mlModelMeta
	version, err := source.Get(ctx, "ml:model:version").Result()
	if errors.Is(err, redis.Nil) {
		return meta, nil
	}
	if err != nil {
		return meta, fmt.Errorf("read ml:model:version: %w", err)
	}
	hash, err := source.Get(ctx, "ml:model:hash").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return meta, fmt.Errorf("read ml:model:hash: %w", err)
	}
	appliedAt, err := source.Get(ctx, "ml:model:applied_at").Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return meta, fmt.Errorf("read ml:model:applied_at: %w", err)
	}
	meta.version = version
	meta.hash = hash
	meta.appliedAt = appliedAt
	meta.present = true
	return meta, nil
}

func scanRedisKeys(ctx context.Context, redisClient redis.UniversalClient, pattern string) ([]string, error) {
	var keys []string
	iter := redisClient.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func ValidateEdgeBlacklistSource(ctx context.Context, redisShards []redis.UniversalClient) error {
	if len(redisShards) == 0 || redisShards[0] == nil {
		return fmt.Errorf("shard 0 not connected")
	}
	source := PickGlobalReconcileSource(ctx, redisShards)
	if source == nil {
		return fmt.Errorf("no healthy reconcile source shard")
	}
	if source == redisShards[0] {
		return nil
	}

	keys, err := scanRedisKeys(ctx, source, globalBlacklistKeyPrefix+"*")
	if err != nil {
		return fmt.Errorf("scan source blacklist keys: %w", err)
	}
	sort.Strings(keys)

	for _, key := range keys {
		sourceMembers, err := source.SMembers(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("read %s from source: %w", key, err)
		}
		targetMembers, err := redisShards[0].SMembers(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("read %s from shard 0: %w", key, err)
		}
		sort.Strings(sourceMembers)
		sort.Strings(targetMembers)
		if !stringSlicesEqual(sourceMembers, targetMembers) {
			return fmt.Errorf("blacklist key %s mismatch after catch-up", key)
		}
	}
	return nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Shard0NeedsCatchup(ctx context.Context, redisShards []redis.UniversalClient) bool {
	if len(redisShards) == 0 || redisShards[0] == nil {
		return false
	}
	source := PickGlobalReconcileSource(ctx, redisShards)
	if source == nil || source == redisShards[0] {
		return false
	}

	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	sourceVersion, sourceErr := source.Get(checkCtx, redisConfigVersionKey).Int64()
	targetVersion, targetErr := redisShards[0].Get(checkCtx, redisConfigVersionKey).Int64()
	if sourceErr == nil && targetErr == nil && sourceVersion != targetVersion {
		return true
	}
	if sourceErr == nil && errors.Is(targetErr, redis.Nil) {
		return true
	}

	keys, err := scanRedisKeys(checkCtx, source, globalBlacklistKeyPrefix+"*")
	if err != nil {
		return true
	}
	for _, key := range keys {
		sourceMembers, err := source.SMembers(checkCtx, key).Result()
		if err != nil {
			return true
		}
		targetMembers, err := redisShards[0].SMembers(checkCtx, key).Result()
		if err != nil {
			return true
		}
		sort.Strings(sourceMembers)
		sort.Strings(targetMembers)
		if !stringSlicesEqual(sourceMembers, targetMembers) {
			return true
		}
	}

	sourceSettings, err := source.HLen(checkCtx, RedisConfigValuesKey).Result()
	if err != nil {
		return false
	}
	targetSettings, err := redisShards[0].HLen(checkCtx, RedisConfigValuesKey).Result()
	if err != nil {
		return true
	}
	return sourceSettings != targetSettings
}

// RunShard0Catchup: copy globals from healthiest shard to shard 0, then registry full-sync publish.
func RunShard0Catchup(ctx context.Context, host CatchupHost) error {
	if host == nil {
		return fmt.Errorf("service not configured")
	}
	redisShards := host.RedisShards()
	if len(redisShards) == 0 || redisShards[0] == nil {
		return fmt.Errorf("shard 0 not connected")
	}
	source := PickGlobalReconcileSource(ctx, redisShards)
	if source == nil {
		return fmt.Errorf("no healthy reconcile source shard")
	}
	if source == redisShards[0] {
		metrics.Shard0CatchupLastSuccessTimestamp.Set(float64(time.Now().Unix()))
		return nil
	}

	if err := ReplicateGlobalsToTarget(ctx, source, redisShards[0]); err != nil {
		return err
	}
	if err := ValidateEdgeBlacklistSource(ctx, redisShards); err != nil {
		return err
	}

	channel := domain.DefaultCampaignUpdateChannel(host.CampaignUpdateChannel())
	if err := domain.PublishCampaignUpdateRedis(ctx, redisShards, channel, domain.RegistryFullSyncPayload); err != nil {
		return fmt.Errorf("publish campaign full-sync: %w", err)
	}

	metrics.Shard0CatchupLastSuccessTimestamp.Set(float64(time.Now().Unix()))
	slog.Info("shard 0 global catch-up complete")
	return nil
}
