package management

import (
	"context"
	"fmt"
	"time"

	"espx/internal/ingestion"
	"espx/internal/metrics"

	"github.com/redis/go-redis/v9"
)

func publishControlChannelToAllShards(ctx context.Context, rdbs []redis.UniversalClient, channel, payload string) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		if err := rdb.Publish(ctx, channel, payload).Err(); err != nil {
			return fmt.Errorf("publish control channel on shard %d: %w", i, err)
		}
	}
	return nil
}

func publishCampaignControlToAllShards(ctx context.Context, rdbs []redis.UniversalClient, channel, campaignID string, queuedAt time.Time) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, ingestion.CampaignEpochKey)
			pipe.Publish(ctx, channel, campaignID)
			return nil
		})
		if err != nil {
			return fmt.Errorf("campaign control fan-out on shard %d: %w", i, err)
		}
		if !queuedAt.IsZero() {
			lag := time.Since(queuedAt).Seconds()
			if lag >= 0 {
				metrics.ControlFanoutLagSeconds.WithLabelValues(fmt.Sprintf("%d", i)).Observe(lag)
			}
		}
	}
	return nil
}

func publishControlMessagesToAllShards(ctx context.Context, rdbs []redis.UniversalClient, channel string, payloads []string) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, ingestion.CampaignEpochKey)
			for _, payload := range payloads {
				pipe.Publish(ctx, channel, payload)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("control message fan-out on shard %d: %w", i, err)
		}
	}
	return nil
}

func setNXOnAllShards(ctx context.Context, rdbs []redis.UniversalClient, key, value string, ttl time.Duration) (bool, error) {
	if len(rdbs) == 0 {
		return false, fmt.Errorf("no redis client available")
	}
	allNew := true
	for i, rdb := range rdbs {
		if rdb == nil {
			return false, fmt.Errorf("redis shard %d is nil", i)
		}
		ok, err := rdb.SetNX(ctx, key, value, ttl).Result()
		if err != nil {
			return false, fmt.Errorf("setnx on shard %d: %w", i, err)
		}
		if !ok {
			allNew = false
		}
	}
	return allNew, nil
}

func PickHealthyControlShard(rdbs []redis.UniversalClient) redis.UniversalClient {
	for _, rdb := range rdbs {
		if rdb != nil {
			return rdb
		}
	}
	return nil
}
