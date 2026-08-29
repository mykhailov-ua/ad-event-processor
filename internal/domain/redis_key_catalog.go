package domain

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain/shard"

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
	tag := shard.CampaignHashTag(id)
	return []string{
		shard.BudgetCampaignKey(id),
		tag + "budget:quota:" + idStr,
		tag + "budget:refill_lock:" + idStr,
		shard.BudgetFrozenRedisKey(id),
		shard.CampaignSyncKey(id),
		"budget:inflight:campaign:" + idStr,
		"budget:lock:campaign:" + idStr,
		"budget:txid:campaign:" + idStr,
		"campaign:settings:" + idStr,
		shard.PlacementBlacklistKey(id),
	}
}

func (c *CampaignRedisKeyCatalog) SourceOnlyKeys(id uuid.UUID) []string {
	return []string{shard.MigrationFenceRedisKey(id)}
}

func (c *CampaignRedisKeyCatalog) PrefixPatterns(id uuid.UUID) []string {
	tag := shard.CampaignHashTag(id)
	return []string{
		shard.DailySpendKeyPrefix(id),
		shard.FcapKeyPrefix(id, ""),
		tag + "dup:",
		tag + "dedup/v2:",
		tag + "idempotency:click:",
		tag + "rl:ip:",
		tag + "imp_ts:",
	}
}

func (c *CampaignRedisKeyCatalog) ActivationRequiredKeys(id uuid.UUID) []string {
	return []string{shard.BudgetCampaignKey(id)}
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
