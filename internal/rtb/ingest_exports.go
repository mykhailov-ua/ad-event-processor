package rtb

import (
	"context"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func LoadRedisDailySpend(
	ctx context.Context,
	redisShards []redis.UniversalClient,
	sharder domain.Sharder,
	camp *domain.Campaign,
) (int64, bool) {
	return loadRedisDailySpend(ctx, redisShards, sharder, camp)
}

func RunRtbCatalogReloadDebouncer(ctx context.Context, trigger <-chan struct{}, reload func(), debounce time.Duration) {
	runRtbCatalogReloadDebouncer(ctx, trigger, reload, debounce)
}

func FanOutRtbCatalogRows(camp *domain.Campaign, base RtbCampaignInput) []CampaignData {
	return fanOutRtbCatalogRows(camp, base)
}

const (
	RtbLiveGateInsufficient  = rtbLiveGateInsufficient
	RtbLiveGateMinParityRate = rtbLiveGateMinParityRate
)

type RtbShadowDiffBucket = rtbShadowDiffBucket

func RtbShadowDiffBucketNow() *RtbShadowDiffBucket {
	return &rtbShadowDiffRing[rtbShadowDiffBucketIdx(time.Now())]
}

func ResetGlobalRtbOutcomeWriterForTest() {
	globalRtbOutcomeWriter.Store(nil)
}

func (w *RtbBudgetReconcileWorker) Sample(ctx context.Context) {
	if w != nil {
		w.sample(ctx)
	}
}

func BuildCustomerBudgetPools(campaigns []*domain.Campaign) map[uuid.UUID]int64 {
	return buildCustomerBudgetPools(campaigns)
}

func RtbInputForCampaign(
	camp *domain.Campaign,
	cfg *config.Config,
	meta *CampaignMeta,
	customerBudget int64,
	hybrid CampaignWeighter,
	boosts *FraudBoostSnapshot,
) RtbCampaignInput {
	return rtbInputForCampaign(camp, cfg, meta, customerBudget, hybrid, boosts)
}

func (b *RtbShadowDiffBucket) RecordParityMatchForTest() {
	if b == nil {
		return
	}
	b.shadowEvals.Add(1)
	b.parityMatch.Add(1)
	b.shadowWinnerMatch.Add(1)
	b.liveWouldAccept.Add(1)
}

func (c *RtbCatalog) EnrichTargetingDeal(targeting RtbTargetingInput) RtbTargetingInput {
	return c.enrichTargetingDeal(targeting)
}
