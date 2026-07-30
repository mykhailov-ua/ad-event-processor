package ingestion

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CampaignRedisKeyCatalog struct{}

var DefaultCampaignRedisKeyCatalog = NewCampaignRedisKeyCatalog()

func NewCampaignRedisKeyCatalog() *CampaignRedisKeyCatalog {
	return &CampaignRedisKeyCatalog{}
}

func (c *CampaignRedisKeyCatalog) FixedKeys(id uuid.UUID) []string {
	idStr := id.String()
	tag := campaignHashTag(id)
	return []string{
		budgetCampaignKey(id),
		tag + "budget:quota:" + idStr,
		tag + "budget:refill_lock:" + idStr,
		BudgetFrozenRedisKey(id),
		campaignSyncKey(id),
		"budget:inflight:campaign:" + idStr,
		"budget:lock:campaign:" + idStr,
		"budget:txid:campaign:" + idStr,
		"campaign:settings:" + idStr,
		PlacementBlacklistKey(id),
	}
}

func (c *CampaignRedisKeyCatalog) SourceOnlyKeys(id uuid.UUID) []string {
	return []string{MigrationFenceRedisKey(id)}
}

func (c *CampaignRedisKeyCatalog) PrefixPatterns(id uuid.UUID) []string {
	tag := campaignHashTag(id)
	return []string{
		dailySpendKeyPrefix(id),
		fcapKeyPrefix(id, ""),
		tag + "dup:",
		tag + "dedup/v2:",
		tag + "idempotency:click:",
		tag + "rl:ip:",
		tag + "imp_ts:",
	}
}

func (c *CampaignRedisKeyCatalog) ActivationRequiredKeys(id uuid.UUID) []string {
	return []string{budgetCampaignKey(id)}
}

func (c *CampaignRedisKeyCatalog) VerifyRequiredKeysExist(ctx context.Context, dst redis.Cmdable, id uuid.UUID) error {
	for _, key := range c.ActivationRequiredKeys(id) {
		n, err := dst.Exists(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("exists %q: %w", key, err)
		}
		if n == 0 {
			return fmt.Errorf("required key %q missing on target shard", key)
		}
	}
	return nil
}

func (c *CampaignRedisKeyCatalog) VerifySlotCampaignKeysExist(
	ctx context.Context,
	dst redis.Cmdable,
	campaignIDs []uuid.UUID,
) error {
	for _, id := range campaignIDs {
		if err := c.VerifyRequiredKeysExist(ctx, dst, id); err != nil {
			return err
		}
	}
	return nil
}
