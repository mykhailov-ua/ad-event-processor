package domain

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/redis/go-redis/v9"
)

func PublishCampaignUpdateRedis(ctx context.Context, rdbs []redis.UniversalClient, channel, campaignID string) error {
	if channel == "" {
		channel = "campaigns:update"
	}
	return database.ForEachConnectedShard(ctx, rdbs, "publish_campaign_control", func(_ int, rdb redis.UniversalClient) error {
		_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
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
