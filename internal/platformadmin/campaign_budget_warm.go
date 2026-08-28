package platformadmin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CampaignBudgetWarmHost interface {
	RedisClientForCampaign(campaignID uuid.UUID) redis.UniversalClient
	CampaignRemainingBudget(ctx context.Context, campaignID uuid.UUID) (int64, error)
	SetCampaignBudgetRemaining(ctx context.Context, pipe redis.Pipeliner, campaignIDStr string, campaignID uuid.UUID, payloadLimit int64) error
}

func WarmCampaignBudget(ctx context.Context, host CampaignBudgetWarmHost, campaignID uuid.UUID) (int64, error) {
	redisClient := host.RedisClientForCampaign(campaignID)
	if redisClient == nil {
		return 0, fmt.Errorf("no redis client available")
	}
	remaining, err := host.CampaignRemainingBudget(ctx, campaignID)
	if err != nil {
		return 0, err
	}
	if remaining <= 0 {
		return 0, nil
	}
	_, err = redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return host.SetCampaignBudgetRemaining(ctx, pipe, campaignID.String(), campaignID, 0)
	})
	if err != nil {
		return 0, err
	}
	return remaining, nil
}
