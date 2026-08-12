package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/redis/go-redis/v9"
)

const globalBlacklistKeyPrefix = "blacklist:"

type mlModelMeta struct {
	version   string
	hash      string
	appliedAt int64
	present   bool
}

func pickGlobalReconcileSource(rdbs []redis.UniversalClient) redis.UniversalClient {
	for i := 1; i < len(rdbs); i++ {
		if rdbs[i] != nil {
			return rdbs[i]
		}
	}
	return PickHealthyControlShard(rdbs)
}

func replicateConfigVersionFromPrimary(ctx context.Context, rdbs []redis.UniversalClient) error {
	return replicateGlobalsFromPrimary(ctx, rdbs)
}

func replicateGlobalsFromPrimary(ctx context.Context, rdbs []redis.UniversalClient) error {
	if len(rdbs) < 2 {
		return nil
	}
	source := pickGlobalReconcileSource(rdbs)
	if source == nil {
		return nil
	}
	return forEachConnectedShard(ctx, rdbs, "replicate_globals", func(i int, rdb redis.UniversalClient) error {
		if rdb == source {
			return nil
		}
		return replicateGlobalsToTarget(ctx, source, rdb)
	})
}

func replicateGlobalsToTarget(ctx context.Context, source, target redis.UniversalClient) error {
	if source == nil || target == nil || source == target {
		return nil
	}

	settings, err := source.HGetAll(ctx, redisConfigValuesKey).Result()
	if err != nil {
		return fmt.Errorf("read %s from source: %w", redisConfigValuesKey, err)
	}

	version, versionErr := source.Get(ctx, redisConfigVersionKey).Int64()
	if versionErr != nil && versionErr != redis.Nil {
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
			pipe.Del(ctx, redisConfigValuesKey)
			pipe.HSet(ctx, redisConfigValuesKey, settings)
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
	if err == redis.Nil {
		return meta, nil
	}
	if err != nil {
		return meta, fmt.Errorf("read ml:model:version: %w", err)
	}
	hash, err := source.Get(ctx, "ml:model:hash").Result()
	if err != nil && err != redis.Nil {
		return meta, fmt.Errorf("read ml:model:hash: %w", err)
	}
	appliedAt, err := source.Get(ctx, "ml:model:applied_at").Int64()
	if err != nil && err != redis.Nil {
		return meta, fmt.Errorf("read ml:model:applied_at: %w", err)
	}
	meta.version = version
	meta.hash = hash
	meta.appliedAt = appliedAt
	meta.present = true
	return meta, nil
}

func scanRedisKeys(ctx context.Context, rdb redis.UniversalClient, pattern string) ([]string, error) {
	var keys []string
	iter := rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func validateEdgeBlacklistSource(ctx context.Context, rdbs []redis.UniversalClient) error {
	if len(rdbs) == 0 || rdbs[0] == nil {
		return fmt.Errorf("shard 0 not connected")
	}
	source := pickGlobalReconcileSource(rdbs)
	if source == nil {
		return fmt.Errorf("no healthy reconcile source shard")
	}
	if source == rdbs[0] {
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
		targetMembers, err := rdbs[0].SMembers(ctx, key).Result()
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

func shard0NeedsCatchup(rdbs []redis.UniversalClient) bool {
	if len(rdbs) == 0 || rdbs[0] == nil {
		return false
	}
	source := pickGlobalReconcileSource(rdbs)
	if source == nil || source == rdbs[0] {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sourceVersion, sourceErr := source.Get(ctx, redisConfigVersionKey).Int64()
	targetVersion, targetErr := rdbs[0].Get(ctx, redisConfigVersionKey).Int64()
	if sourceErr == nil && targetErr == nil && sourceVersion != targetVersion {
		return true
	}
	if sourceErr == nil && targetErr == redis.Nil {
		return true
	}

	keys, err := scanRedisKeys(ctx, source, globalBlacklistKeyPrefix+"*")
	if err != nil {
		return true
	}
	for _, key := range keys {
		sourceMembers, err := source.SMembers(ctx, key).Result()
		if err != nil {
			return true
		}
		targetMembers, err := rdbs[0].SMembers(ctx, key).Result()
		if err != nil {
			return true
		}
		sort.Strings(sourceMembers)
		sort.Strings(targetMembers)
		if !stringSlicesEqual(sourceMembers, targetMembers) {
			return true
		}
	}

	sourceSettings, err := source.HLen(ctx, redisConfigValuesKey).Result()
	if err != nil {
		return false
	}
	targetSettings, err := rdbs[0].HLen(ctx, redisConfigValuesKey).Result()
	if err != nil {
		return true
	}
	return sourceSettings != targetSettings
}

func (s *Service) RunShard0Catchup(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("service not configured")
	}
	rdbs := s.RedisShards()
	if len(rdbs) == 0 || rdbs[0] == nil {
		return fmt.Errorf("shard 0 not connected")
	}
	source := pickGlobalReconcileSource(rdbs)
	if source == nil {
		return fmt.Errorf("no healthy reconcile source shard")
	}
	if source == rdbs[0] {
		metrics.Shard0CatchupLastSuccessTimestamp.Set(float64(time.Now().Unix()))
		return nil
	}

	if err := replicateGlobalsToTarget(ctx, source, rdbs[0]); err != nil {
		return err
	}
	if err := validateEdgeBlacklistSource(ctx, rdbs); err != nil {
		return err
	}

	channel := s.campaignUpdateChannel()
	if channel == "" {
		channel = "campaigns:update"
	}
	if err := publishCampaignControlToAllShards(ctx, rdbs, channel, domain.RegistryFullSyncPayload, time.Time{}); err != nil {
		return fmt.Errorf("publish campaign full-sync: %w", err)
	}

	metrics.Shard0CatchupLastSuccessTimestamp.Set(float64(time.Now().Unix()))
	slog.Info("shard 0 global catch-up complete")
	return nil
}
