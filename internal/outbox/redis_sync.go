package outbox

import (
	"context"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/edge"

	"github.com/redis/go-redis/v9"
)

const (
	redisConfigValuesKey     = "config:values"
	redisConfigVersionKey    = "config:version"
	defaultFlowReloadChannel = "flow:reload"
)

func SyncGlobalConfigToAllShards(ctx context.Context, redisShards []redis.UniversalClient, settings map[string]string, version int64) error {
	return database.ForEachConnectedShard(ctx, redisShards, "sync_global_config", func(_ int, redisClient redis.UniversalClient) error {
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

func SyncGlobalStringToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key, value string, ttl time.Duration) error {
	return database.SyncGlobalStringToAllShards(ctx, redisShards, key, value, ttl)
}

func DeleteGlobalKeyFromAllShards(ctx context.Context, redisShards []redis.UniversalClient, key string) error {
	return database.DeleteGlobalKeyFromAllShards(ctx, redisShards, key)
}

// SyncGlobalSetMemberToAllShards: SADD/SREM on every Redis shard; changelog on shard[0] for edge-bpf-sync.
func SyncGlobalSetMemberToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key, member string, add bool) error {
	err := database.ForEachConnectedShard(ctx, redisShards, "sync_global_set_member", func(_ int, redisClient redis.UniversalClient) error {
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

func SyncGlobalHashFieldToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key, field, value string, del bool) error {
	return database.ForEachConnectedShard(ctx, redisShards, "sync_global_hash_field", func(_ int, redisClient redis.UniversalClient) error {
		if del {
			return redisClient.HDel(ctx, key, field).Err()
		}
		return redisClient.HSet(ctx, key, field, value).Err()
	})
}

func SyncKeyToAllShards(ctx context.Context, redisShards []redis.UniversalClient, key string, value interface{}, ttl time.Duration) error {
	return database.ForEachConnectedShard(ctx, redisShards, "sync_key", func(_ int, redisClient redis.UniversalClient) error {
		return redisClient.Set(ctx, key, value, ttl).Err()
	})
}

func PublishControlChannelToAllShards(ctx context.Context, redisShards []redis.UniversalClient, channel, payload string) error {
	return database.ForEachConnectedShard(ctx, redisShards, "publish_control_channel", func(_ int, redisClient redis.UniversalClient) error {
		return redisClient.Publish(ctx, channel, payload).Err()
	})
}

func PublishFraudQuarantineBatch(ctx context.Context, redisShards []redis.UniversalClient, ips []string) error {
	payload, err := edge.MarshalFraudQuarantinePayload(ips)
	if err != nil {
		return err
	}
	return PublishControlChannelToAllShards(ctx, redisShards, edge.FraudQuarantineChannel, payload)
}

func PublishControlMessagesToAllShards(ctx context.Context, redisShards []redis.UniversalClient, channel string, payloads []string) error {
	return database.ForEachConnectedShard(ctx, redisShards, "publish_control_messages", func(_ int, redisClient redis.UniversalClient) error {
		_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, domain.CampaignEpochKey)
			for _, payload := range payloads {
				pipe.Publish(ctx, channel, payload)
			}
			return nil
		})
		return err
	})
}

func PublishFlowReload(ctx context.Context, redisShards []redis.UniversalClient, channel string) error {
	if channel == "" {
		channel = defaultFlowReloadChannel
	}
	if len(redisShards) == 0 || redisShards[0] == nil {
		return nil
	}
	return redisShards[0].Publish(ctx, channel, "1").Err()
}
