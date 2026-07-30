package ingestion

import (
	"context"
	"fmt"
	"time"

	"espx/internal/campaignmodel"
	"espx/internal/metrics"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const budgetKeyTTL = 24 * time.Hour

func RemainingBudgetMicro(c *campaignmodel.Campaign) int64 {
	if c == nil {
		return 0
	}
	rem := c.BudgetLimit - c.CurrentSpend
	if rem < 0 {
		return 0
	}
	return rem
}

type BudgetCacheWarmer struct {
	rdbs    []redis.UniversalClient
	sharder Sharder
}

func NewBudgetCacheWarmer(rdbs []redis.UniversalClient, sharder Sharder) *BudgetCacheWarmer {
	return &BudgetCacheWarmer{rdbs: rdbs, sharder: sharder}
}

type budgetWarmItem struct {
	key string
	val int64
}

func (w *BudgetCacheWarmer) Warm(ctx context.Context, campaigns []*campaignmodel.Campaign) (int, error) {
	if w == nil || len(w.rdbs) == 0 || len(campaigns) == 0 {
		return 0, nil
	}

	byShard := make([][]budgetWarmItem, len(w.rdbs))
	for _, camp := range campaigns {
		if camp == nil || camp.BudgetCampaignKey == "" {
			continue
		}
		shard := w.sharder.GetShard(camp.ID)
		if shard < 0 || shard >= len(w.rdbs) {
			continue
		}
		byShard[shard] = append(byShard[shard], budgetWarmItem{
			key: camp.BudgetCampaignKey,
			val: RemainingBudgetMicro(camp),
		})
	}

	warmed := 0
	for shard, items := range byShard {
		if len(items) == 0 {
			continue
		}
		pipe := w.rdbs[shard].Pipeline()
		cmds := make([]*redis.BoolCmd, len(items))
		for i, item := range items {
			cmds[i] = pipe.SetNX(ctx, item.key, item.val, budgetKeyTTL)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return warmed, fmt.Errorf("budget warm pipeline shard %d: %w", shard, err)
		}
		for _, cmd := range cmds {
			if cmd.Val() {
				warmed++
			}
		}
	}
	metrics.BudgetCacheWarmTotal.WithLabelValues("full").Add(float64(warmed))
	return warmed, nil
}

func (w *BudgetCacheWarmer) WarmOne(ctx context.Context, camp *campaignmodel.Campaign) (bool, error) {
	if w == nil || len(w.rdbs) == 0 || camp == nil || camp.BudgetCampaignKey == "" {
		return false, nil
	}
	shard := w.sharder.GetShard(camp.ID)
	if shard < 0 || shard >= len(w.rdbs) {
		return false, fmt.Errorf("invalid shard index %d for campaign %s", shard, camp.ID)
	}

	rdb := w.rdbs[shard]
	remaining := RemainingBudgetMicro(camp)

	warmed, err := rdb.SetNX(ctx, camp.BudgetCampaignKey, remaining, budgetKeyTTL).Result()
	if err != nil {
		return false, fmt.Errorf("budget warm one shard %d: %w", shard, err)
	}

	if warmed {
		metrics.BudgetCacheWarmTotal.WithLabelValues("incremental").Inc()
	}
	return warmed, nil
}

func (w *BudgetCacheWarmer) WarmFromRegistry(ctx context.Context, reg *Registry) (int, error) {
	if reg == nil {
		return 0, nil
	}
	return w.Warm(ctx, reg.ActiveCampaigns())
}

func warmBudgetKeyNX(ctx context.Context, rdb redis.UniversalClient, key string, remaining int64) error {
	_, err := rdb.SetNX(ctx, key, remaining, budgetKeyTTL).Result()
	return err
}

func tryRecoverBudgetFromRegistry(
	ctx context.Context,
	rdb redis.UniversalClient,
	registry campaignmodel.CampaignRegistry,
	campaignID uuid.UUID,
	budgetKey string,
) (bool, error) {
	if registry == nil {
		return false, nil
	}
	camp, ok := registry.GetCampaign(campaignID)
	if !ok {
		return false, nil
	}
	if camp.BudgetLimit == 0 && camp.CurrentSpend == 0 {
		return false, nil
	}
	if err := warmBudgetKeyNX(ctx, rdb, budgetKey, RemainingBudgetMicro(camp)); err != nil {
		return false, err
	}
	metrics.BudgetCacheRegistryRecoverTotal.Inc()
	return true, nil
}
