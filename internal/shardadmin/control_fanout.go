package shardadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/metrics"

	"github.com/redis/go-redis/v9"
)

func PublishControlChannelToAllShards(ctx context.Context, redisShards []redis.UniversalClient, channel, payload string) error {
	return ForEachConnectedShard(ctx, redisShards, "publish_control_channel", func(_ int, redisClient redis.UniversalClient) error {
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

// PublishCampaignControlToAllShards: pipeline INCR campaign epoch + PUBLISH per shard (tracker registry reload).
func PublishCampaignControlToAllShards(ctx context.Context, redisShards []redis.UniversalClient, channel, campaignID string, queuedAt time.Time) error {
	return ForEachConnectedShard(ctx, redisShards, "publish_campaign_control", func(i int, redisClient redis.UniversalClient) error {
		_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, domain.CampaignEpochKey)
			pipe.Publish(ctx, channel, campaignID)
			return nil
		})
		if err != nil {
			return err
		}
		if !queuedAt.IsZero() {
			lag := time.Since(queuedAt).Seconds()
			if lag >= 0 {
				metrics.ControlFanoutLagSeconds.WithLabelValues(fmt.Sprintf("%d", i)).Observe(lag)
			}
		}
		return nil
	})
}

func PublishControlMessagesToAllShards(ctx context.Context, redisShards []redis.UniversalClient, channel string, payloads []string) error {
	return ForEachConnectedShard(ctx, redisShards, "publish_control_messages", func(_ int, redisClient redis.UniversalClient) error {
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

// SetNXOnAllShards: strict fanout; allNew false if any shard already holds the lease key.
func SetNXOnAllShards(ctx context.Context, redisShards []redis.UniversalClient, key, value string, ttl time.Duration) (bool, error) {
	allNew := true
	err := ForEachConnectedShardStrict(ctx, redisShards, "setnx", func(_ int, redisClient redis.UniversalClient) error {
		ok, err := redisClient.SetNX(ctx, key, value, ttl).Result()
		if err != nil {
			return err
		}
		if !ok {
			allNew = false
		}
		return nil
	})
	return allNew, err
}

func PickHealthyControlShard(redisShards []redis.UniversalClient) redis.UniversalClient {
	for _, redisClient := range redisShards {
		if redisClient != nil {
			return redisClient
		}
	}
	return nil
}
