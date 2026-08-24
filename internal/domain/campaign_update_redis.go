package domain

import (
	"context"

	"ad-event-processor/internal/database"

	"github.com/redis/go-redis/v9"
)

func PublishCampaignUpdateRedis(ctx context.Context, redisShards []redis.UniversalClient, channel, campaignID string) error {
	if channel == "" {
		channel = "campaigns:update"
	}
	return database.ForEachConnectedShard(ctx, redisShards, "publish_campaign_control", func(_ int, redisClient redis.UniversalClient) error {
		_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, CampaignEpochKey)
			pipe.Publish(ctx, channel, campaignID)
			return nil
		})
		return err
	})
}

func DefaultCampaignUpdateChannel(channel string) string {
	if channel != "" {
		return channel
	}
	return "campaigns:update"
}
