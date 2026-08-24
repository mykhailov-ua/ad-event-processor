package ingestion

import (
	"context"
	"sync/atomic"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type evalCountRedis struct {
	redis.UniversalClient
	evals atomic.Int64
}

func (c *evalCountRedis) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *redis.Cmd {
	c.evals.Add(1)
	return c.UniversalClient.EvalSha(ctx, sha1, keys, args...)
}

func (c *evalCountRedis) Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd {
	c.evals.Add(1)
	return c.UniversalClient.Eval(ctx, script, keys, args...)
}

func benchRegistryForCampaign(camp *domain.Campaign) *Registry {
	reg := NewRegistry(nil)
	enrichMockCampaign(camp)
	reg.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		camp.ID: {campaign: camp},
	}})
	return reg
}
