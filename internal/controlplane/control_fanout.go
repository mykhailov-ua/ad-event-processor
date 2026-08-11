package controlplane

import (
	"context"
	"fmt"
	"time"

	"espx/internal/domain"
	"espx/internal/metrics"

	"github.com/redis/go-redis/v9"
)

func publishControlChannelToAllShards(ctx context.Context, rdbs []redis.UniversalClient, channel, payload string) error {
	return forEachConnectedShard(ctx, rdbs, "publish_control_channel", func(_ int, rdb redis.UniversalClient) error {
		return rdb.Publish(ctx, channel, payload).Err()
	})
}

func publishCampaignControlToAllShards(ctx context.Context, rdbs []redis.UniversalClient, channel, campaignID string, queuedAt time.Time) error {
	return forEachConnectedShard(ctx, rdbs, "publish_campaign_control", func(i int, rdb redis.UniversalClient) error {
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
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

func publishControlMessagesToAllShards(ctx context.Context, rdbs []redis.UniversalClient, channel string, payloads []string) error {
	return forEachConnectedShard(ctx, rdbs, "publish_control_messages", func(_ int, rdb redis.UniversalClient) error {
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, domain.CampaignEpochKey)
			for _, payload := range payloads {
				pipe.Publish(ctx, channel, payload)
			}
			return nil
		})
		return err
	})
}

func setNXOnAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, value string, ttl time.Duration) (bool, error) {
	allNew := true
	err := forEachConnectedShard(ctx, rdbs, "setnx", func(_ int, rdb redis.UniversalClient) error {
		ok, err := rdb.SetNX(ctx, key, value, ttl).Result()
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

func PickHealthyControlShard(rdbs []redis.UniversalClient) redis.UniversalClient {
	for _, rdb := range rdbs {
		if rdb != nil {
			return rdb
		}
	}
	return nil
}
